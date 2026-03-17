package session

import (
	"context"
	"testing"
	"time"
)

func makeSession(id string, state State) *Session {
	now := time.Now().UTC()
	return &Session{
		ID:         id,
		State:      state,
		CreatedAt:  now,
		UpdatedAt:  now,
		OriginType: "builtin:static",
		AccountID:  "test-acct",
		ZoneID:     "test-zone",
		Hostname:   "test.example.com",
	}
}

func newTestStore(t *testing.T) *FileStore {
	t.Helper()
	dir := t.TempDir()
	store, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestSaveLoadRoundtrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	sess := makeSession("aaa-bbb-ccc-ddd", StateActive)
	sess.TunnelID = "tun-123"
	sess.Hostname = "app.example.com"

	if err := store.Save(ctx, sess); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load(ctx, "aaa-bbb-ccc-ddd")
	if err != nil {
		t.Fatal(err)
	}

	if loaded.ID != sess.ID {
		t.Errorf("ID mismatch: got %s, want %s", loaded.ID, sess.ID)
	}
	if loaded.State != sess.State {
		t.Errorf("State mismatch: got %s, want %s", loaded.State, sess.State)
	}
	if loaded.TunnelID != sess.TunnelID {
		t.Errorf("TunnelID mismatch: got %s, want %s", loaded.TunnelID, sess.TunnelID)
	}
	if loaded.Hostname != sess.Hostname {
		t.Errorf("Hostname mismatch: got %s, want %s", loaded.Hostname, sess.Hostname)
	}
}

func TestListFilterByState(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	store.Save(ctx, makeSession("id-1", StateActive))
	store.Save(ctx, makeSession("id-2", StateStopped))
	store.Save(ctx, makeSession("id-3", StateActive))

	sessions, err := store.List(ctx, StateActive)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 active sessions, got %d", len(sessions))
	}
}

func TestListAll(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	store.Save(ctx, makeSession("id-1", StateActive))
	store.Save(ctx, makeSession("id-2", StateStopped))

	sessions, err := store.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestDelete(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	store.Save(ctx, makeSession("id-del", StateActive))

	if err := store.Delete(ctx, "id-del"); err != nil {
		t.Fatal(err)
	}

	_, err := store.Load(ctx, "id-del")
	if err == nil {
		t.Error("expected error loading deleted session")
	}
}

func TestResolveExactMatch(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	store.Save(ctx, makeSession("abcd1234-5678-9abc-def0-111111111111", StateActive))

	sess, err := store.Resolve(ctx, "abcd1234-5678-9abc-def0-111111111111")
	if err != nil {
		t.Fatal(err)
	}
	if sess.ID != "abcd1234-5678-9abc-def0-111111111111" {
		t.Errorf("unexpected ID: %s", sess.ID)
	}
}

func TestResolvePrefixMatch(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	store.Save(ctx, makeSession("abcd1234-5678-9abc-def0-111111111111", StateActive))

	sess, err := store.Resolve(ctx, "abcd1234")
	if err != nil {
		t.Fatal(err)
	}
	if sess.ID != "abcd1234-5678-9abc-def0-111111111111" {
		t.Errorf("unexpected ID: %s", sess.ID)
	}
}

func TestResolveAmbiguous(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	store.Save(ctx, makeSession("abcd1234-aaaa-0000-0000-000000000001", StateActive))
	store.Save(ctx, makeSession("abcd1234-bbbb-0000-0000-000000000002", StateActive))

	_, err := store.Resolve(ctx, "abcd1234")
	if err == nil {
		t.Error("expected ambiguous error")
	}
}

func TestResolveNotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.Resolve(ctx, "nonexistent")
	if err == nil {
		t.Error("expected not found error")
	}
}
