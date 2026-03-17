package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/paoloanzn/flare-cli/internal/config"
	"github.com/paoloanzn/flare-cli/internal/testutil"
)

func TestPrintLoginURL(t *testing.T) {
	store := testutil.NewMockStore()
	sess := testutil.MakeTestSession("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "test.example.com")
	sess.LoginURL = "https://test.example.com"
	store.Save(context.Background(), sess)

	SetTestServices(&Services{Store: store})
	defer ClearTestServices()

	buf, err := executeCmd(t, "auth", "print-login-url", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "https://test.example.com") {
		t.Errorf("expected login URL in output, got: %s", out)
	}
}

func TestPrintLoginURLJSON(t *testing.T) {
	store := testutil.NewMockStore()
	sess := testutil.MakeTestSession("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "test.example.com")
	sess.LoginURL = "https://test.example.com"
	store.Save(context.Background(), sess)

	SetTestServices(&Services{Store: store})
	defer ClearTestServices()

	buf, err := executeCmd(t, "auth", "print-login-url", "--json", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}

	if result["login_url"] != "https://test.example.com" {
		t.Errorf("expected login_url 'https://test.example.com', got %q", result["login_url"])
	}
}

func TestRotateMTLSNotImplemented(t *testing.T) {
	store := testutil.NewMockStore()
	sess := testutil.MakeTestSession("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "test.example.com")
	store.Save(context.Background(), sess)

	SetTestServices(&Services{Store: store})
	defer ClearTestServices()

	_, err := executeCmd(t, "auth", "rotate-mtls", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "--ca-id", "test-ca")
	if err == nil {
		t.Error("expected error for not yet implemented")
	}
	if err != nil && !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("expected 'not yet implemented' error, got: %v", err)
	}
}

func TestAuthLogoutRemovesCredential(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Save a credential first.
	if err := config.SaveCredential("test-token-for-logout"); err != nil {
		t.Fatalf("SaveCredential: %v", err)
	}

	// Verify it's there.
	tok, err := config.LoadCredential()
	if err != nil || tok != "test-token-for-logout" {
		t.Fatalf("credential not saved properly: %v, %q", err, tok)
	}

	// Run logout.
	_, err = executeCmd(t, "auth", "logout")
	if err != nil {
		t.Fatalf("auth logout: %v", err)
	}

	// Verify it's gone.
	_, err = config.LoadCredential()
	if err == nil {
		t.Error("expected credential to be deleted after logout")
	}
}

func TestAuthLogoutNoCredential(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Logout with no credential should not error (just warn).
	_, err := executeCmd(t, "auth", "logout")
	if err != nil {
		t.Errorf("expected no error for logout with no credentials, got: %v", err)
	}
}

func TestAuthLoginNoToken(t *testing.T) {
	// Login with --token="" and --no-browser should read from stdin.
	// Since stdin is empty in test, it should fail with "reading token" error.
	_, err := executeCmd(t, "auth", "login", "--no-browser", "--token", "")
	if err == nil {
		t.Error("expected error when no token provided")
	}
}

func TestAuthWhoamiNoToken(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("CLOUDFLARE_API_TOKEN", "")

	// Clear any cached token in viper so APIToken() returns empty.
	oldVal := viper.GetString("cloudflare.api_token")
	viper.Set("cloudflare.api_token", "")
	defer viper.Set("cloudflare.api_token", oldVal)

	_, err := executeCmd(t, "auth", "whoami")
	if err == nil {
		t.Error("expected error when no token available")
	}
	if err != nil && !strings.Contains(err.Error(), "no API token found") {
		t.Errorf("expected 'no API token found' error, got: %v", err)
	}
}

func TestMaskToken(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "****"},
		{"short", "****"},
		{"1234567890", "****"},
		{"12345678901", "1234...8901"},
		{"abcdefghijklmnop", "abcd...mnop"},
	}

	for _, tt := range tests {
		got := maskToken(tt.input)
		if got != tt.expected {
			t.Errorf("maskToken(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestDetectTokenSourceEnv(t *testing.T) {
	t.Setenv("CLOUDFLARE_API_TOKEN", "env-test-token")
	source := detectTokenSource("env-test-token")
	if !strings.Contains(source, "environment variable") {
		t.Errorf("expected 'environment variable' source, got: %s", source)
	}
}

func TestDetectTokenSourceCredFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("CLOUDFLARE_API_TOKEN", "")

	if err := config.SaveCredential("cred-file-token"); err != nil {
		t.Fatalf("SaveCredential: %v", err)
	}

	source := detectTokenSource("cred-file-token")
	if !strings.Contains(source, "credentials file") {
		t.Errorf("expected 'credentials file' source, got: %s", source)
	}
}
