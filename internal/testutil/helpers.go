package testutil

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/paoloanzn/flare-cli/cmd"
	"github.com/paoloanzn/flare-cli/internal/session"
)

// ExecuteCommand runs a flare CLI command with the given args and returns stdout.
func ExecuteCommand(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd := cmd.RootCmdForTest()
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return buf.String(), err
}

// SetupTestConfig creates a temp config directory with a valid config.yaml
// and sets viper + env for the test. Cleanup is automatic.
func SetupTestConfig(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "flare-cli")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}

	configContent := `cloudflare:
  account_id: "test-account-id"
  zone_id: "test-zone-id"
  domain: "test.example.com"
  api_token_env: "CLOUDFLARE_API_TOKEN"
defaults:
  auth: "otp"
  session_duration: "30m"
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Set viper to use our test config.
	viper.Reset()
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatal(err)
	}

	// Set the API token env.
	t.Setenv("CLOUDFLARE_API_TOKEN", "test-api-token")

	t.Cleanup(func() {
		viper.Reset()
		cmd.ClearTestServices()
	})

	return tmpDir
}

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

// NewTestServices creates a full Services struct with all mocks wired up.
func NewTestServices(store *MockStore) (*cmd.Services, *MockTunnelManager, *MockConnector, *MockAccessManager, *MockDNSManager) {
	tunnelMgr := &MockTunnelManager{}
	connector := &MockConnector{}
	accessMgr := &MockAccessManager{}
	dnsMgr := &MockDNSManager{}

	svc := &cmd.Services{
		TunnelMgr: tunnelMgr,
		Connector: connector,
		AccessMgr: accessMgr,
		DNSMgr:    dnsMgr,
		Store:     store,
	}

	return svc, tunnelMgr, connector, accessMgr, dnsMgr
}
