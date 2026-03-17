package cmd

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/paoloanzn/flare-cli/internal/testutil"
)

func TestStatusJSON(t *testing.T) {
	store := testutil.NewMockStore()
	sess := testutil.MakeTestSession("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "test.example.com")
	store.Save(context.Background(), sess)

	SetTestServices(&Services{Store: store})
	defer ClearTestServices()

	buf, err := executeCmd(t, "status", "--json", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}

	if result["hostname"] != "test.example.com" {
		t.Errorf("expected hostname test.example.com, got %v", result["hostname"])
	}
	if result["state"] != "active" {
		t.Errorf("expected state active, got %v", result["state"])
	}
	if result["id"] != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Errorf("expected id, got %v", result["id"])
	}
}

func TestStatusNotFound(t *testing.T) {
	store := testutil.NewMockStore()

	SetTestServices(&Services{Store: store})
	defer ClearTestServices()

	_, err := executeCmd(t, "status", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}
