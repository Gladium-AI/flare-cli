package dns

import (
	"context"
	"fmt"

	cf "github.com/cloudflare/cloudflare-go"
)

// Manager manages Cloudflare DNS records.
type Manager interface {
	CreateCNAME(ctx context.Context, zoneID, hostname, tunnelID string) (recordID string, err error)
	DeleteRecord(ctx context.Context, zoneID, recordID string) error
}

// APIManager implements Manager using the Cloudflare API.
type APIManager struct {
	client *cf.API
}

// NewAPIManager creates a new DNS manager.
func NewAPIManager(client *cf.API) *APIManager {
	return &APIManager{client: client}
}

// CreateCNAME creates a proxied CNAME record pointing to the tunnel.
func (m *APIManager) CreateCNAME(ctx context.Context, zoneID, hostname, tunnelID string) (string, error) {
	rc := cf.ZoneIdentifier(zoneID)

	proxied := true
	record, err := m.client.CreateDNSRecord(ctx, rc, cf.CreateDNSRecordParams{
		Type:    "CNAME",
		Name:    hostname,
		Content: fmt.Sprintf("%s.cfargotunnel.com", tunnelID),
		TTL:     1, // Auto TTL.
		Proxied: &proxied,
		Comment: "Created by flare-cli",
	})
	if err != nil {
		return "", fmt.Errorf("creating CNAME record: %w", err)
	}

	return record.ID, nil
}

// DeleteRecord removes a DNS record.
func (m *APIManager) DeleteRecord(ctx context.Context, zoneID, recordID string) error {
	rc := cf.ZoneIdentifier(zoneID)
	if err := m.client.DeleteDNSRecord(ctx, rc, recordID); err != nil {
		return fmt.Errorf("deleting DNS record: %w", err)
	}
	return nil
}
