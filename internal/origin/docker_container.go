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

// DockerContainer runs a Docker container and exposes its port.
type DockerContainer struct {
	cfg           Config
	containerName string
	runner        *flareexec.Runner
	publishAddr   string
}

// NewDockerContainer creates a DockerContainer origin.
func NewDockerContainer(cfg Config) (*DockerContainer, error) {
	if cfg.Image == "" {
		return nil, fmt.Errorf("--image is required for docker:container origin")
	}
	if cfg.ContainerPort == 0 {
		return nil, fmt.Errorf("--container-port is required for docker:container origin")
	}
	return &DockerContainer{cfg: cfg}, nil
}

func (d *DockerContainer) Type() Type {
	return TypeDockerContainer
}

func (d *DockerContainer) Start(ctx context.Context) (string, error) {
	dockerBin := d.cfg.DockerBin
	if dockerBin == "" {
		dockerBin = "docker"
	}

	// Generate a container name.
	d.containerName = fmt.Sprintf("flare-%d", time.Now().UnixNano()%100000)

	// Determine publish address.
	publishPort := d.cfg.PublishPort
	if publishPort == "" {
		publishPort = fmt.Sprintf("127.0.0.1:0:%d", d.cfg.ContainerPort)
	} else if !strings.Contains(publishPort, ":") {
		publishPort = fmt.Sprintf("127.0.0.1:%s:%d", publishPort, d.cfg.ContainerPort)
	}

	// Build docker run args.
	args := []string{"run", "--name", d.containerName, "-p", publishPort}

	if d.cfg.Remove {
		args = append(args, "--rm")
	}

	for k, v := range d.cfg.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	if d.cfg.Entrypoint != "" {
		args = append(args, "--entrypoint", d.cfg.Entrypoint)
	}

	if d.cfg.Network != "" {
		args = append(args, "--network", d.cfg.Network)
	}

	args = append(args, d.cfg.Image)

	// Start the container.
	runner, err := flareexec.Start(ctx, flareexec.RunOpts{
		Name: dockerBin,
		Args: args,
	})
	if err != nil {
		return "", fmt.Errorf("starting docker container: %w", err)
	}
	d.runner = runner

	// Get the actual mapped port.
	time.Sleep(2 * time.Second) // Give Docker time to map ports.

	out, err := exec.CommandContext(ctx, dockerBin, "port", d.containerName, fmt.Sprintf("%d", d.cfg.ContainerPort)).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("getting container port mapping: %w (output: %s)", err, string(out))
	}

	// Parse output like "0.0.0.0:32768" or "127.0.0.1:32768".
	mapped := strings.TrimSpace(string(out))
	if idx := strings.LastIndex(mapped, "\n"); idx >= 0 {
		mapped = mapped[idx+1:]
	}
	// Ensure it's on loopback.
	if strings.HasPrefix(mapped, "0.0.0.0:") {
		mapped = "127.0.0.1:" + strings.TrimPrefix(mapped, "0.0.0.0:")
	}

	d.publishAddr = mapped
	originURL := fmt.Sprintf("http://%s", mapped)

	// Wait for the container to be ready.
	healthURL := originURL
	if d.cfg.HealthPath != "" {
		healthURL = originURL + d.cfg.HealthPath
	}

	timeout := 30 * time.Second
	if d.cfg.WaitForReady != "" {
		if dur, err := time.ParseDuration(d.cfg.WaitForReady); err == nil {
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
			d.Stop(ctx)
			return "", fmt.Errorf("container not ready on port %s after %s", mapped, timeout)
		}

		select {
		case <-ctx.Done():
			d.Stop(ctx)
			return "", ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}

func (d *DockerContainer) Stop(_ context.Context) error {
	dockerBin := d.cfg.DockerBin
	if dockerBin == "" {
		dockerBin = "docker"
	}

	if d.containerName != "" {
		exec.Command(dockerBin, "stop", d.containerName).Run()
		if !d.cfg.Remove {
			exec.Command(dockerBin, "rm", "-f", d.containerName).Run()
		}
	}
	return nil
}

func (d *DockerContainer) Logs() io.ReadCloser {
	if d.runner != nil {
		return d.runner.Logs()
	}
	return nil
}

func (d *DockerContainer) Healthy(ctx context.Context) error {
	dockerBin := d.cfg.DockerBin
	if dockerBin == "" {
		dockerBin = "docker"
	}

	out, err := exec.CommandContext(ctx, dockerBin, "inspect", "-f", "{{.State.Running}}", d.containerName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("container not running: %w", err)
	}
	if strings.TrimSpace(string(out)) != "true" {
		return fmt.Errorf("container is not running")
	}
	return nil
}
