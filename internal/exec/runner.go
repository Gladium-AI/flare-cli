package exec

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Runner manages subprocess execution.
type Runner struct {
	cmd     *exec.Cmd
	logPipe io.ReadCloser
}

// RunOpts configures a subprocess.
type RunOpts struct {
	Name string            // Binary name or path.
	Args []string          // Command arguments.
	Dir  string            // Working directory (optional).
	Env  map[string]string // Additional environment variables.
}

// Start launches a subprocess with its own process group.
func Start(ctx context.Context, opts RunOpts) (*Runner, error) {
	cmd := exec.CommandContext(ctx, opts.Name, opts.Args...)

	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}

	// Inherit parent env + add extras.
	cmd.Env = os.Environ()
	for k, v := range opts.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	// Own process group so we can kill the tree.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Capture combined output.
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		return nil, fmt.Errorf("starting %s: %w", opts.Name, err)
	}

	// Close the write end when the process exits so readers see EOF.
	go func() {
		cmd.Wait()
		pw.Close()
	}()

	return &Runner{cmd: cmd, logPipe: pr}, nil
}

// PID returns the process ID.
func (r *Runner) PID() int {
	if r.cmd.Process != nil {
		return r.cmd.Process.Pid
	}
	return 0
}

// Logs returns a reader for the process's combined stdout/stderr.
func (r *Runner) Logs() io.ReadCloser {
	return r.logPipe
}

// Stop sends SIGTERM to the process group, waits up to 10s, then SIGKILL.
func (r *Runner) Stop() error {
	if r.cmd.Process == nil {
		return nil
	}

	// Send SIGTERM to the process group.
	pgid := -r.cmd.Process.Pid
	if err := syscall.Kill(pgid, syscall.SIGTERM); err != nil {
		// Process may have already exited.
		return nil
	}

	// Wait for exit with timeout.
	done := make(chan error, 1)
	go func() {
		done <- r.cmd.Wait()
	}()

	select {
	case <-done:
		return nil
	case <-time.After(10 * time.Second):
		// Force kill.
		_ = syscall.Kill(pgid, syscall.SIGKILL)
		<-done // Wait for the process to be reaped.
		return fmt.Errorf("process did not exit after SIGTERM; sent SIGKILL")
	}
}

// Wait blocks until the process exits and returns its error.
func (r *Runner) Wait() error {
	return r.cmd.Wait()
}

// Running returns true if the process is still alive.
func (r *Runner) Running() bool {
	if r.cmd.Process == nil {
		return false
	}
	// Signal 0 checks if the process exists.
	return r.cmd.Process.Signal(syscall.Signal(0)) == nil
}

// Run executes a command synchronously and returns its combined output.
func Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Which checks if a binary is in PATH and returns its path.
func Which(name string) (string, error) {
	return exec.LookPath(name)
}
