package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

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
