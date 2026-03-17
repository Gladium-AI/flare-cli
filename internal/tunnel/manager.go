package tunnel

import (
	"context"
	"encoding/base64"
	"fmt"

	cf "github.com/cloudflare/cloudflare-go"
	"github.com/google/uuid"
)

// Info holds the result of creating a tunnel.
type Info struct {
	TunnelID   string
	TunnelName string
	Token      string // For cloudflared tunnel run --token
}

// Manager manages Cloudflare Tunnel lifecycle via the API.
type Manager interface {
	Create(ctx context.Context, accountID, name string) (*Info, error)
	ConfigureIngress(ctx context.Context, accountID, tunnelID, hostname, originURL string) error
	GetToken(ctx context.Context, accountID, tunnelID string) (string, error)
	Delete(ctx context.Context, accountID, tunnelID string) error
}

// APIManager implements Manager using the Cloudflare API.
type APIManager struct {
	client *cf.API
}

// NewAPIManager creates a new tunnel manager with the given API client.
func NewAPIManager(client *cf.API) *APIManager {
	return &APIManager{client: client}
}

// Create creates a remotely-managed Cloudflare Tunnel.
func (m *APIManager) Create(ctx context.Context, accountID, name string) (*Info, error) {
	// Generate a random tunnel secret.
	secret := base64.StdEncoding.EncodeToString([]byte(uuid.New().String()))

	rc := cf.AccountIdentifier(accountID)
	tunnel, err := m.client.CreateTunnel(ctx, rc, cf.TunnelCreateParams{
		Name:      name,
		Secret:    secret,
		ConfigSrc: "cloudflare", // Remotely managed.
	})
	if err != nil {
		return nil, fmt.Errorf("creating tunnel: %w", err)
	}

	// Retrieve the run token.
	token, err := m.client.GetTunnelToken(ctx, rc, tunnel.ID)
	if err != nil {
		return nil, fmt.Errorf("getting tunnel token: %w", err)
	}

	return &Info{
		TunnelID:   tunnel.ID,
		TunnelName: tunnel.Name,
		Token:      token,
	}, nil
}

// ConfigureIngress sets the tunnel's ingress rules.
func (m *APIManager) ConfigureIngress(ctx context.Context, accountID, tunnelID, hostname, originURL string) error {
	rc := cf.AccountIdentifier(accountID)

	_, err := m.client.UpdateTunnelConfiguration(ctx, rc, cf.TunnelConfigurationParams{
		TunnelID: tunnelID,
		Config: cf.TunnelConfiguration{
			Ingress: []cf.UnvalidatedIngressRule{
				{
					Hostname: hostname,
					Service:  originURL,
				},
				{
					// Mandatory catch-all rule.
					Service: "http_status:404",
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("configuring tunnel ingress: %w", err)
	}
	return nil
}

// GetToken retrieves the tunnel connector run token.
func (m *APIManager) GetToken(ctx context.Context, accountID, tunnelID string) (string, error) {
	rc := cf.AccountIdentifier(accountID)
	token, err := m.client.GetTunnelToken(ctx, rc, tunnelID)
	if err != nil {
		return "", fmt.Errorf("getting tunnel token: %w", err)
	}
	return token, nil
}

// Delete removes a Cloudflare Tunnel.
func (m *APIManager) Delete(ctx context.Context, accountID, tunnelID string) error {
	rc := cf.AccountIdentifier(accountID)
	if err := m.client.DeleteTunnel(ctx, rc, tunnelID); err != nil {
		return fmt.Errorf("deleting tunnel: %w", err)
	}
	return nil
}
