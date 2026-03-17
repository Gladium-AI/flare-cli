package tunnel

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	flareexec "github.com/paoloanzn/flare-cli/internal/exec"
)

// Connector manages the cloudflared process.
type Connector interface {
	Run(ctx context.Context, token string) (pid int, err error)
	Stop(ctx context.Context) error
	Logs() io.ReadCloser
	Healthy() error
}

// ProcessConnector runs cloudflared as a subprocess.
type ProcessConnector struct {
	mu     sync.Mutex
	bin    string // Path to cloudflared binary.
	runner *flareexec.Runner
}

// NewProcessConnector creates a connector for the given cloudflared binary.
func NewProcessConnector(cloudflaredBin string) *ProcessConnector {
	if cloudflaredBin == "" {
		cloudflaredBin = "cloudflared"
	}
	return &ProcessConnector{bin: cloudflaredBin}
}

// Run starts `cloudflared tunnel run --token <token>`.
func (c *ProcessConnector) Run(ctx context.Context, token string) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	runner, err := flareexec.Start(ctx, flareexec.RunOpts{
		Name: c.bin,
		Args: []string{"tunnel", "run", "--token", token},
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
		return fmt.Errorf("cloudflared process exited")
	}
	return nil
}
