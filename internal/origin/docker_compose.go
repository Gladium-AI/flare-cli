package origin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	flareexec "github.com/paoloanzn/flare-cli/internal/exec"
)

// DockerCompose starts a Compose service and exposes one port.
type DockerCompose struct {
	cfg         Config
	runner      *flareexec.Runner
	publishAddr string
}

// NewDockerCompose creates a DockerCompose origin.
func NewDockerCompose(cfg Config) (*DockerCompose, error) {
	if cfg.ComposeFile == "" {
		return nil, fmt.Errorf("--file is required for docker:compose-service origin")
	}
	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("--service is required for docker:compose-service origin")
	}
	if cfg.ContainerPort == 0 {
		return nil, fmt.Errorf("--container-port is required for docker:compose-service origin")
	}
	return &DockerCompose{cfg: cfg}, nil
}

func (dc *DockerCompose) Type() Type {
	return TypeDockerCompose
}

func (dc *DockerCompose) Start(ctx context.Context) (string, error) {
	// Build the docker compose up command.
	args := []string{"compose", "-f", dc.cfg.ComposeFile}

	if dc.cfg.ProjectName != "" {
		args = append(args, "-p", dc.cfg.ProjectName)
	}

	upArgs := append(args, "up")
	if dc.cfg.Build {
		upArgs = append(upArgs, "--build")
	}
	if dc.cfg.UpDetached {
		upArgs = append(upArgs, "-d")
	}
	upArgs = append(upArgs, dc.cfg.ServiceName)

	// Start the compose service.
	runner, err := flareexec.Start(ctx, flareexec.RunOpts{
		Name: "docker",
		Args: upArgs,
	})
	if err != nil {
		return "", fmt.Errorf("starting compose service: %w", err)
	}
	dc.runner = runner

	// Wait for the service to be up.
	time.Sleep(3 * time.Second)

	// Get the port mapping.
	publishPort := dc.cfg.PublishPort
	if publishPort == "" {
		// Try to get the port from docker compose.
		portArgs := append(args, "port", dc.cfg.ServiceName, fmt.Sprintf("%d", dc.cfg.ContainerPort))
		out, err := exec.CommandContext(ctx, "docker", portArgs...).CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("getting compose port mapping: %w (output: %s)", err, string(out))
		}
		mapped := strings.TrimSpace(string(out))
		if strings.HasPrefix(mapped, "0.0.0.0:") {
			mapped = "127.0.0.1:" + strings.TrimPrefix(mapped, "0.0.0.0:")
		}
		dc.publishAddr = mapped
	} else {
		dc.publishAddr = publishPort
	}

	originURL := fmt.Sprintf("http://%s", dc.publishAddr)

	// Health check.
	healthURL := originURL
	if dc.cfg.HealthPath != "" {
		healthURL = originURL + dc.cfg.HealthPath
	}

	timeout := 30 * time.Second
	if dc.cfg.WaitForReady != "" {
		if dur, err := time.ParseDuration(dc.cfg.WaitForReady); err == nil {
			timeout = dur
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

		if time.Now().After(deadline) {
			dc.Stop(ctx)
			return "", fmt.Errorf("compose service not ready after %s", timeout)
		}

		select {
		case <-ctx.Done():
			dc.Stop(ctx)
			return "", ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}

func (dc *DockerCompose) Stop(_ context.Context) error {
	args := []string{"compose", "-f", dc.cfg.ComposeFile}
	if dc.cfg.ProjectName != "" {
		args = append(args, "-p", dc.cfg.ProjectName)
	}
	args = append(args, "down")

	exec.Command("docker", args...).Run()
	return nil
}

func (dc *DockerCompose) Logs() io.ReadCloser {
	if dc.runner != nil {
		return dc.runner.Logs()
	}
	return nil
}

func (dc *DockerCompose) Healthy(ctx context.Context) error {
	args := []string{"compose", "-f", dc.cfg.ComposeFile}
	if dc.cfg.ProjectName != "" {
		args = append(args, "-p", dc.cfg.ProjectName)
	}
	args = append(args, "ps", "--format", "{{.State}}", dc.cfg.ServiceName)

	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("service not running: %w", err)
	}
	state := strings.TrimSpace(string(out))
	if !strings.Contains(state, "running") {
		return fmt.Errorf("service state: %s", state)
	}
	return nil
}
