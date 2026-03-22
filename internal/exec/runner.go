package exec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Runner manages subprocess execution.
type Runner struct {
	cmd     *exec.Cmd
	logPipe io.ReadCloser

	// exitCh is closed when the process exits.
	exitCh chan struct{}
	// exitErr holds the result of cmd.Wait().
	exitErr error

	mu       sync.Mutex
	logBuf   *ringBuffer // Circular buffer that stores recent log output.
	waitDone bool
}

// ringBuffer is a fixed-size circular buffer that implements io.Writer.
// Older data is silently discarded when the buffer wraps.
type ringBuffer struct {
	mu  sync.Mutex
	buf []byte
	pos int    // next write position
	full bool  // whether the buffer has wrapped at least once
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{buf: make([]byte, size)}
}

func (rb *ringBuffer) Write(p []byte) (int, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	n := len(p)
	for len(p) > 0 {
		k := copy(rb.buf[rb.pos:], p)
		rb.pos += k
		p = p[k:]
		if rb.pos == len(rb.buf) {
			rb.pos = 0
			rb.full = true
		}
	}
	return n, nil
}

// Bytes returns the buffered data in chronological order.
func (rb *ringBuffer) Bytes() []byte {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if !rb.full {
		return bytes.Clone(rb.buf[:rb.pos])
	}
	out := make([]byte, len(rb.buf))
	n := copy(out, rb.buf[rb.pos:])
	copy(out[n:], rb.buf[:rb.pos])
	return out
}

// RunOpts configures a subprocess.
type RunOpts struct {
	Name string            // Binary name or path.
	Args []string          // Command arguments.
	Dir  string            // Working directory (optional).
	Env  map[string]string // Additional environment variables.
}

// Start launches a subprocess with its own process group.
// Stdout/stderr are continuously drained into a 1 MB ring buffer
// so the subprocess never blocks on log output.
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

	// Use OS pipes (buffered by kernel, typically 64KB+) instead of io.Pipe
	// (which has zero buffering and blocks the writer immediately).
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		stdoutR.Close()
		stdoutW.Close()
		return nil, fmt.Errorf("creating stderr pipe: %w", err)
	}

	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW

	if err := cmd.Start(); err != nil {
		stdoutR.Close()
		stdoutW.Close()
		stderrR.Close()
		stderrW.Close()
		return nil, fmt.Errorf("starting %s: %w", opts.Name, err)
	}

	// Close the write ends in our process — the child owns them now.
	stdoutW.Close()
	stderrW.Close()

	r := &Runner{
		cmd:    cmd,
		logBuf: newRingBuffer(1 << 20), // 1 MB ring buffer
		exitCh: make(chan struct{}),
	}

	// Continuously drain stdout and stderr into the ring buffer.
	// This runs until the read ends return EOF (i.e., the process exits).
	var drainWg sync.WaitGroup
	drainWg.Add(2)
	go func() { defer drainWg.Done(); io.Copy(r.logBuf, stdoutR); stdoutR.Close() }()
	go func() { defer drainWg.Done(); io.Copy(r.logBuf, stderrR); stderrR.Close() }()

	// Wait for the process to exit, then close exitCh to notify watchers.
	go func() {
		r.exitErr = cmd.Wait()
		drainWg.Wait() // ensure all output is captured before signalling
		close(r.exitCh)
	}()

	// logPipe provides a reader that returns the ring buffer contents on demand.
	// It's kept for backward compat with the Logs() method.
	r.logPipe = io.NopCloser(bytes.NewReader(nil))

	return r, nil
}

// PID returns the process ID.
func (r *Runner) PID() int {
	if r.cmd.Process != nil {
		return r.cmd.Process.Pid
	}
	return 0
}

// Logs returns a reader for the process's combined stdout/stderr.
// The returned reader contains a snapshot of the most recent log output
// (up to 1 MB) from the ring buffer.
func (r *Runner) Logs() io.ReadCloser {
	return io.NopCloser(bytes.NewReader(r.logBuf.Bytes()))
}

// ExitCh returns a channel that is closed when the process exits.
func (r *Runner) ExitCh() <-chan struct{} {
	return r.exitCh
}

// Stop sends SIGTERM to the process group, waits up to 10s, then SIGKILL.
func (r *Runner) Stop() error {
	if r.cmd.Process == nil {
		return nil
	}

	// If the process already exited, nothing to do.
	select {
	case <-r.exitCh:
		return nil
	default:
	}

	// Send SIGTERM to the process group.
	pgid := -r.cmd.Process.Pid
	if err := syscall.Kill(pgid, syscall.SIGTERM); err != nil {
		// Process may have already exited.
		return nil
	}

	select {
	case <-r.exitCh:
		return nil
	case <-time.After(10 * time.Second):
		// Force kill.
		_ = syscall.Kill(pgid, syscall.SIGKILL)
		<-r.exitCh
		return fmt.Errorf("process did not exit after SIGTERM; sent SIGKILL")
	}
}

// Wait blocks until the process exits and returns its error.
func (r *Runner) Wait() error {
	<-r.exitCh
	return r.exitErr
}

// Running returns true if the process is still alive.
func (r *Runner) Running() bool {
	select {
	case <-r.exitCh:
		return false
	default:
		return r.cmd.Process != nil
	}
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
