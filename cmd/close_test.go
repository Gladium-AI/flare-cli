package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/paoloanzn/flare-cli/internal/session"
	"github.com/paoloanzn/flare-cli/internal/testutil"
)

func TestCloseSession(t *testing.T) {
	store := testutil.NewMockStore()
	svc, tunnelMgr, connector, accessMgr, dnsMgr := newTestServices(store)

	sess := testutil.MakeTestSession("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "test.example.com")
	store.Save(context.Background(), sess)

	SetTestServices(svc)
	defer ClearTestServices()

	_, err := executeCmd(t, "close", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if err != nil {
		t.Fatal(err)
	}

	if len(tunnelMgr.DeleteCalls) != 1 {
		t.Errorf("expected 1 tunnel delete, got %d", len(tunnelMgr.DeleteCalls))
	}
	if len(accessMgr.DeleteAppCalls) != 1 {
		t.Errorf("expected 1 access app delete, got %d", len(accessMgr.DeleteAppCalls))
	}
	if len(dnsMgr.DeleteRecordCalls) != 1 {
		t.Errorf("expected 1 DNS record delete, got %d", len(dnsMgr.DeleteRecordCalls))
	}
	if !connector.StopCalled {
		t.Error("expected connector stop to be called")
	}
}

func TestCloseAlreadyStopped(t *testing.T) {
	store := testutil.NewMockStore()
	svc, _, _, _, _ := newTestServices(store)

	sess := testutil.MakeTestSession("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "test.example.com")
	sess.State = session.StateStopped
	store.Save(context.Background(), sess)

	SetTestServices(svc)
	defer ClearTestServices()

	_, err := executeCmd(t, "close", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if err == nil {
		t.Error("expected error for already stopped session")
	}
	if err != nil && !strings.Contains(err.Error(), "already stopped") {
		t.Errorf("expected 'already stopped' in error, got: %v", err)
	}
}

func TestCloseForce(t *testing.T) {
	store := testutil.NewMockStore()
	svc, _, _, _, _ := newTestServices(store)

	sess := testutil.MakeTestSession("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "test.example.com")
	sess.State = session.StateStopped
	store.Save(context.Background(), sess)

	SetTestServices(svc)
	defer ClearTestServices()

	_, err := executeCmd(t, "close", "--force", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCloseNotFound(t *testing.T) {
	store := testutil.NewMockStore()
	svc, _, _, _, _ := newTestServices(store)

	SetTestServices(svc)
	defer ClearTestServices()

	_, err := executeCmd(t, "close", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}
