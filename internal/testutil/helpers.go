package testutil

import (
	"time"

	"github.com/paoloanzn/flare-cli/internal/session"
)

// MakeTestSession creates a session with sensible defaults for testing.
func MakeTestSession(id, hostname string) *session.Session {
	now := time.Now().UTC()
	return &session.Session{
		ID:              id,
		State:           session.StateActive,
		CreatedAt:       now,
		UpdatedAt:       now,
		OriginType:      "builtin:static",
		AccountID:       "test-account-id",
		ZoneID:          "test-zone-id",
		TunnelID:        "tun-test-uuid-0001-0001-000000000001",
		TunnelName:      "flare-test",
		TunnelToken:     "eyJ0ZXN0IjoidG9rZW4ifQ==",
		AccessAppID:     "app-test-uuid-0001-0001-000000000001",
		AccessPolicyID:  "pol-test-uuid-0001-0001-000000000001",
		DNSRecordID:     "dns-rec-test-uuid-0001-0001-000000000001",
		Hostname:        hostname,
		LoginURL:        "https://" + hostname,
		AuthMode:        "otp",
		CloudflaredPID:  12345,
		LoopbackAddr:    "http://127.0.0.1:8080",
		SessionDuration: "30m",
	}
}
