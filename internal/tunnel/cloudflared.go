package tunnel

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	flareexec "github.com/paoloanzn/flare-cli/internal/exec"
)

// Connector manages the cloudflared process.
type Connector interface {
	Run(ctx context.Context, token string) (pid int, err error)
	Stop(ctx context.Context) error
	Logs() io.ReadCloser
	Healthy() error
	// ExitCh returns a channel that is closed when the cloudflared process exits.
	// Returns nil if the process is not running.
	ExitCh() <-chan struct{}
	// LogFilePath returns the path to the persistent log file on disk.
	LogFilePath() string
}

// ProcessConnector runs cloudflared as a subprocess.
type ProcessConnector struct {
	mu     sync.Mutex
	bin    string // Path to cloudflared binary.
	runner *flareexec.Runner
	logDir string // Directory for persistent log files.
}

// NewProcessConnector creates a connector for the given cloudflared binary.
func NewProcessConnector(cloudflaredBin string) *ProcessConnector {
	if cloudflaredBin == "" {
		cloudflaredBin = "cloudflared"
	}

	// Use the flare-cli config dir for log files.
	logDir := ""
	if home, err := os.UserHomeDir(); err == nil {
		logDir = filepath.Join(home, ".config", "flare-cli", "logs")
		os.MkdirAll(logDir, 0700)
	}

	return &ProcessConnector{bin: cloudflaredBin, logDir: logDir}
}

// Run starts `cloudflared tunnel --no-autoupdate run --token <token>`.
func (c *ProcessConnector) Run(ctx context.Context, token string) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Build log file path with timestamp for easy identification.
	logFile := ""
	if c.logDir != "" {
		ts := time.Now().Format("20060102-150405")
		logFile = filepath.Join(c.logDir, fmt.Sprintf("cloudflared-%s.log", ts))
	}

	runner, err := flareexec.Start(ctx, flareexec.RunOpts{
		Name: c.bin,
		Args: []string{
			"tunnel",
			"--no-autoupdate",
			"run",
			"--token", token,
		},
		LogFile: logFile,
	})
	if err != nil {
		return 0, fmt.Errorf("starting cloudflared: %w", err)
	}

	c.runner = runner
	return runner.PID(), nil
}

// Stop gracefully stops the cloudflared process.
func (c *ProcessConnector) Stop(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.runner == nil {
		return nil
	}
	return c.runner.Stop()
}

// Logs returns a reader for cloudflared's output.
func (c *ProcessConnector) Logs() io.ReadCloser {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.runner == nil {
		return io.NopCloser(strings.NewReader(""))
	}
	return c.runner.Logs()
}

// Healthy returns nil if cloudflared is still running.
func (c *ProcessConnector) Healthy() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.runner == nil {
		return fmt.Errorf("cloudflared not started")
	}
	if !c.runner.Running() {
		exitErr := c.runner.ExitError()
		if exitErr != nil {
			return fmt.Errorf("cloudflared process exited: %w", exitErr)
		}
		return fmt.Errorf("cloudflared process exited")
	}
	return nil
}

// ExitCh returns a channel that is closed when the cloudflared process exits.
func (c *ProcessConnector) ExitCh() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.runner == nil {
		// Return a closed channel so callers don't block forever.
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return c.runner.ExitCh()
}

// LogFilePath returns the path to the persistent log file on disk.
func (c *ProcessConnector) LogFilePath() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.runner == nil {
		return ""
	}
	return c.runner.LogFilePath()
}
