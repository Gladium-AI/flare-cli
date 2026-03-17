package cmd

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/paoloanzn/flare-cli/internal/session"
	"github.com/paoloanzn/flare-cli/internal/testutil"
)

func TestListJSON(t *testing.T) {
	store := testutil.NewMockStore()
	s1 := testutil.MakeTestSession("id-1-aaaa-bbbb-cccc-ddddeeeeeeee", "a.example.com")
	s2 := testutil.MakeTestSession("id-2-aaaa-bbbb-cccc-ddddeeeeeeee", "b.example.com")
	s2.State = session.StateStopped

	store.Save(context.Background(), s1)
	store.Save(context.Background(), s2)

	SetTestServices(&Services{Store: store})
	defer ClearTestServices()

	// Default list (no --all): should only include active sessions.
	buf, err := executeCmd(t, "list", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var sessions []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &sessions); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}

	if len(sessions) != 1 {
		t.Errorf("expected 1 active session, got %d", len(sessions))
	}
}

func TestListAllJSON(t *testing.T) {
	store := testutil.NewMockStore()
	s1 := testutil.MakeTestSession("id-1-aaaa-bbbb-cccc-ddddeeeeeeee", "a.example.com")
	s2 := testutil.MakeTestSession("id-2-aaaa-bbbb-cccc-ddddeeeeeeee", "b.example.com")
	s2.State = session.StateStopped

	store.Save(context.Background(), s1)
	store.Save(context.Background(), s2)

	SetTestServices(&Services{Store: store})
	defer ClearTestServices()

	buf, err := executeCmd(t, "list", "--all", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var sessions []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &sessions); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}

	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions with --all, got %d", len(sessions))
	}
}

func TestListEmptyJSON(t *testing.T) {
	store := testutil.NewMockStore()

	SetTestServices(&Services{Store: store})
	defer ClearTestServices()

	buf, err := executeCmd(t, "list", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var sessions []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &sessions); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}

	if sessions != nil && len(sessions) != 0 {
		t.Errorf("expected empty list, got %d sessions", len(sessions))
	}
}
