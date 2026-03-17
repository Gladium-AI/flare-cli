package origin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// LocalCommand launches a local command and exposes its HTTP port.
type LocalCommand struct {
	cfg    Config
	cmd    *exec.Cmd
	stdout io.ReadCloser
}

// NewLocalCommand creates a LocalCommand origin.
func NewLocalCommand(cfg Config) (*LocalCommand, error) {
	if cfg.Command == "" {
		return nil, fmt.Errorf("--cmd is required for local:command origin")
	}
	if cfg.Port == 0 {
		return nil, fmt.Errorf("--port is required for local:command origin")
	}
	return &LocalCommand{cfg: cfg}, nil
}

func (l *LocalCommand) Type() Type {
	return TypeLocalCommand
}

// Start launches the command and waits until the port is ready.
func (l *LocalCommand) Start(ctx context.Context) (string, error) {
	// Build the command.
	var cmd *exec.Cmd
	if strings.Contains(l.cfg.Command, " ") {
		// Shell mode: wrap in sh -c.
		cmd = exec.CommandContext(ctx, "sh", "-c", l.cfg.Command)
	} else {
		cmd = exec.CommandContext(ctx, l.cfg.Command)
	}

	if l.cfg.Dir != "" {
		cmd.Dir = l.cfg.Dir
	}

	// Set environment.
	cmd.Env = os.Environ()
	for k, v := range l.cfg.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	// Capture output.
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	l.stdout = pr

	if err := cmd.Start(); err != nil {
		pw.Close()
		return "", fmt.Errorf("starting command: %w", err)
	}
	l.cmd = cmd

	// Close write end when process exits.
	go func() {
		cmd.Wait()
		pw.Close()
	}()

	// Wait for the port to be ready.
	originURL := fmt.Sprintf("http://127.0.0.1:%d", l.cfg.Port)
	healthURL := originURL
	if l.cfg.HealthPath != "" {
		healthURL = originURL + l.cfg.HealthPath
	}

	timeout := 60 * time.Second
	if l.cfg.WaitForReady != "" {
		if d, err := time.ParseDuration(l.cfg.WaitForReady); err == nil {
			timeout = d
		}
	}

	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return originURL, nil
			}
		}

		// Check if the process exited.
		if l.cmd.ProcessState != nil && l.cmd.ProcessState.Exited() {
			return "", fmt.Errorf("command exited before becoming ready (exit code %d)", l.cmd.ProcessState.ExitCode())
		}

		if time.Now().After(deadline) {
			l.Stop(ctx)
			return "", fmt.Errorf("command not ready on port %d after %s", l.cfg.Port, timeout)
		}

		select {
		case <-ctx.Done():
			l.Stop(ctx)
			return "", ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (l *LocalCommand) Stop(_ context.Context) error {
	if l.cmd != nil && l.cmd.Process != nil {
		l.cmd.Process.Signal(os.Interrupt)
		done := make(chan error, 1)
		go func() { done <- l.cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			l.cmd.Process.Kill()
		}
	}
	return nil
}

func (l *LocalCommand) Logs() io.ReadCloser {
	return l.stdout
}

func (l *LocalCommand) Healthy(ctx context.Context) error {
	if l.cmd == nil || l.cmd.Process == nil {
		return fmt.Errorf("command not running")
	}
	if l.cmd.ProcessState != nil && l.cmd.ProcessState.Exited() {
		return fmt.Errorf("command exited (code %d)", l.cmd.ProcessState.ExitCode())
	}
	return nil
}
