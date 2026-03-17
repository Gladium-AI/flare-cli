package origin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// LocalHTTP proxies to an already-running local HTTP service.
type LocalHTTP struct {
	cfg       Config
	originURL string
	healthURL string
}

// NewLocalHTTP creates a LocalHTTP origin.
func NewLocalHTTP(cfg Config) (*LocalHTTP, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("--url is required for local:http origin")
	}

	u, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid --url: %w", err)
	}

	healthURL := cfg.HealthURL
	if healthURL == "" {
		healthURL = cfg.URL
		if cfg.HealthPath != "" {
			healthURL = cfg.URL + cfg.HealthPath
		}
	}

	return &LocalHTTP{
		cfg:       cfg,
		originURL: u.String(),
		healthURL: healthURL,
	}, nil
}

func (l *LocalHTTP) Type() Type {
	return TypeLocalHTTP
}

// Start validates the upstream is reachable and returns its URL.
// For local:http, the service is already running; we just confirm reachability.
func (l *LocalHTTP) Start(ctx context.Context) (string, error) {
	// Poll for readiness.
	timeout := 10 * time.Second
	if l.cfg.WaitForReady != "" {
		d, err := time.ParseDuration(l.cfg.WaitForReady)
		if err == nil {
			timeout = d
		}
	}

	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for {
		if err := l.healthCheck(ctx, client); err == nil {
			return l.originURL, nil
		}

		if time.Now().After(deadline) {
			return "", fmt.Errorf("origin at %s not reachable after %s", l.originURL, timeout)
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// Stop is a no-op for local:http since we don't own the process.
func (l *LocalHTTP) Stop(_ context.Context) error {
	return nil
}

// Logs returns nil since we don't own the process.
func (l *LocalHTTP) Logs() io.ReadCloser {
	return nil
}

// Healthy checks if the upstream is responding.
func (l *LocalHTTP) Healthy(ctx context.Context) error {
	client := &http.Client{Timeout: 5 * time.Second}
	return l.healthCheck(ctx, client)
}

func (l *LocalHTTP) healthCheck(ctx context.Context, client *http.Client) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.healthURL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("health check returned %d", resp.StatusCode)
	}
	return nil
}
