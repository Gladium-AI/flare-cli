package cmd

import (
	"context"
	"testing"

	"github.com/paoloanzn/flare-cli/internal/testutil"
)

func TestUpdateAddEmail(t *testing.T) {
	store := testutil.NewMockStore()
	svc, _, _, accessMgr, _ := newTestServices(store)

	sess := testutil.MakeTestSession("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "test.example.com")
	store.Save(context.Background(), sess)

	SetTestServices(svc)
	defer ClearTestServices()

	_, err := executeCmd(t, "update", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "--allow-email", "add:foo@bar.com")
	if err != nil {
		t.Fatal(err)
	}

	if len(accessMgr.UpdatePolicyCalls) != 1 {
		t.Errorf("expected 1 UpdatePolicy call, got %d", len(accessMgr.UpdatePolicyCalls))
	}

	updated, _ := store.Load(context.Background(), sess.ID)
	found := false
	for _, e := range updated.AllowedEmails {
		if e == "foo@bar.com" {
			found = true
		}
	}
	if !found {
		t.Error("expected foo@bar.com in allowed emails")
	}
}

func TestUpdateTTL(t *testing.T) {
	store := testutil.NewMockStore()
	svc, _, _, _, _ := newTestServices(store)

	sess := testutil.MakeTestSession("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "test.example.com")
	store.Save(context.Background(), sess)

	SetTestServices(svc)
	defer ClearTestServices()

	_, err := executeCmd(t, "update", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "--ttl", "1h")
	if err != nil {
		t.Fatal(err)
	}

	updated, _ := store.Load(context.Background(), sess.ID)
	if updated.ExpiresAt == nil {
		t.Error("expected ExpiresAt to be set")
	}
}

func TestUpdateNoChanges(t *testing.T) {
	store := testutil.NewMockStore()
	svc, _, _, _, _ := newTestServices(store)

	sess := testutil.MakeTestSession("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "test.example.com")
	store.Save(context.Background(), sess)

	SetTestServices(svc)
	defer ClearTestServices()

	_, err := executeCmd(t, "update", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if err == nil {
		t.Error("expected error for no changes")
	}
}
