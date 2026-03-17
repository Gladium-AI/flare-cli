package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/paoloanzn/flare-cli/internal/session"
	"github.com/paoloanzn/flare-cli/internal/testutil"
)

func TestLogsStoppedSession(t *testing.T) {
	store := testutil.NewMockStore()
	sess := testutil.MakeTestSession("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "test.example.com")
	sess.State = session.StateStopped
	store.Save(context.Background(), sess)

	SetTestServices(&Services{Store: store})
	defer ClearTestServices()

	_, err := executeCmd(t, "logs", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if err == nil {
		t.Error("expected error for stopped session")
	}
	if err != nil && !strings.Contains(err.Error(), "stopped") {
		t.Errorf("expected 'stopped' in error, got: %v", err)
	}
}

func TestLogsActiveSession(t *testing.T) {
	store := testutil.NewMockStore()
	sess := testutil.MakeTestSession("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "test.example.com")
	store.Save(context.Background(), sess)

	SetTestServices(&Services{Store: store})
	defer ClearTestServices()

	// The logs command prints to stdout via ui functions, not to cmd.OutOrStdout().
	// Just verify it doesn't error.
	_, err := executeCmd(t, "logs", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
