package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadDeleteCredential(t *testing.T) {
	// Override home dir so we don't write to real config.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	token := "test-api-token-abc123"

	// Save.
	if err := SaveCredential(token); err != nil {
		t.Fatalf("SaveCredential: %v", err)
	}

	// Verify file permissions.
	credPath, err := CredentialPath()
	if err != nil {
		t.Fatalf("CredentialPath: %v", err)
	}
	fi, err := os.Stat(credPath)
	if err != nil {
		t.Fatalf("Stat credential: %v", err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 permissions, got %o", fi.Mode().Perm())
	}

	// Load.
	loaded, err := LoadCredential()
	if err != nil {
		t.Fatalf("LoadCredential: %v", err)
	}
	if loaded != token {
		t.Errorf("expected %q, got %q", token, loaded)
	}

	// Delete.
	if err := DeleteCredential(); err != nil {
		t.Fatalf("DeleteCredential: %v", err)
	}

	// Load after delete should fail.
	_, err = LoadCredential()
	if err == nil {
		t.Error("expected error loading deleted credential")
	}
}

func TestLoadCredentialNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	_, err := LoadCredential()
	if err == nil {
		t.Error("expected error when no credentials file exists")
	}
}

func TestDeleteCredentialIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Delete when no file exists should not error.
	if err := DeleteCredential(); err != nil {
		t.Errorf("DeleteCredential on missing file: %v", err)
	}
}

func TestCredentialPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	path, err := CredentialPath()
	if err != nil {
		t.Fatalf("CredentialPath: %v", err)
	}

	expected := filepath.Join(tmpDir, ".config", "flare-cli", "credentials")
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestAPITokenPriority(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Clear any existing env token.
	t.Setenv("CLOUDFLARE_API_TOKEN", "")

	// No token anywhere — should return empty.
	if tok := APIToken(); tok != "" {
		t.Errorf("expected empty token, got %q", tok)
	}

	// Save a credential — should pick it up.
	if err := SaveCredential("cred-file-token"); err != nil {
		t.Fatalf("SaveCredential: %v", err)
	}
	if tok := APIToken(); tok != "cred-file-token" {
		t.Errorf("expected 'cred-file-token', got %q", tok)
	}

	// Set env var — should take precedence over credential file.
	t.Setenv("CLOUDFLARE_API_TOKEN", "env-token")
	if tok := APIToken(); tok != "env-token" {
		t.Errorf("expected 'env-token', got %q", tok)
	}
}
