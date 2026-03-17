package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paoloanzn/flare-cli/internal/access"
	"github.com/paoloanzn/flare-cli/internal/origin"
	"github.com/paoloanzn/flare-cli/internal/session"
	"github.com/paoloanzn/flare-cli/internal/testutil"
	"github.com/paoloanzn/flare-cli/internal/tunnel"
)

// originConfigForTest creates a builtin:static origin config using a temp dir.
func originConfigForTest(t *testing.T) origin.Config {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>test</h1>"), 0644)
	return origin.Config{
		Type: origin.TypeBuiltinStatic,
		Path: dir,
	}
}

func TestServeHappyPath(t *testing.T) {
	store := testutil.NewMockStore()
	tunnelMgr := &testutil.MockTunnelManager{}
	connector := &testutil.MockConnector{}
	accessMgr := &testutil.MockAccessManager{}
	dnsMgr := &testutil.MockDNSManager{}

	pipe := New(tunnelMgr, connector, accessMgr, dnsMgr, store)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := pipe.Serve(ctx, ServeParams{
		OriginConfig: originConfigForTest(t),
		Hostname:     "test.example.com",
		AccountID:    "test-acct",
		ZoneID:       "test-zone",
		CreateDNS:    true,
		Policy: access.Policy{
			AuthMode:        "otp",
			SessionDuration: "30m",
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	if len(tunnelMgr.CreateCalls) != 1 {
		t.Errorf("expected 1 tunnel Create call, got %d", len(tunnelMgr.CreateCalls))
	}
	if len(tunnelMgr.ConfigureIngressCalls) != 1 {
		t.Errorf("expected 1 ConfigureIngress call, got %d", len(tunnelMgr.ConfigureIngressCalls))
	}
	if len(dnsMgr.CreateCNAMECalls) != 1 {
		t.Errorf("expected 1 CreateCNAME call, got %d", len(dnsMgr.CreateCNAMECalls))
	}
	if len(accessMgr.CreateAppCalls) != 1 {
		t.Errorf("expected 1 CreateApp call, got %d", len(accessMgr.CreateAppCalls))
	}
	if len(connector.RunCalls) != 1 {
		t.Errorf("expected 1 connector Run call, got %d", len(connector.RunCalls))
	}

	sessions := store.Sessions()
	for _, s := range sessions {
		if s.State != session.StateStopped {
			t.Errorf("expected session state stopped, got %s", s.State)
		}
	}
}

func TestServeTunnelCreateFails(t *testing.T) {
	store := testutil.NewMockStore()
	tunnelMgr := &testutil.MockTunnelManager{
		FnCreate: func(ctx context.Context, accountID, name string) (*tunnel.Info, error) {
			return nil, fmt.Errorf("tunnel creation failed")
		},
	}
	connector := &testutil.MockConnector{}
	accessMgr := &testutil.MockAccessManager{}
	dnsMgr := &testutil.MockDNSManager{}

	pipe := New(tunnelMgr, connector, accessMgr, dnsMgr, store)

	err := pipe.Serve(context.Background(), ServeParams{
		OriginConfig: originConfigForTest(t),
		Hostname:     "test.example.com",
		AccountID:    "test-acct",
		ZoneID:       "test-zone",
		CreateDNS:    true,
		Policy:       access.Policy{AuthMode: "otp"},
	})

	if err == nil {
		t.Error("expected error from tunnel create failure")
	}

	if len(accessMgr.CreateAppCalls) != 0 {
		t.Error("CreateApp should not be called after tunnel failure")
	}
	if len(connector.RunCalls) != 0 {
		t.Error("connector Run should not be called after tunnel failure")
	}

	sessions := store.Sessions()
	for _, s := range sessions {
		if s.State != session.StateError {
			t.Errorf("expected error state, got %s", s.State)
		}
	}
}

func TestServeAccessCreateFails(t *testing.T) {
	store := testutil.NewMockStore()
	tunnelMgr := &testutil.MockTunnelManager{}
	connector := &testutil.MockConnector{}
	accessMgr := &testutil.MockAccessManager{
		FnCreateApp: func(ctx context.Context, accountID, hostname string, policy access.Policy) (*access.AppInfo, error) {
			return nil, fmt.Errorf("access app creation failed")
		},
	}
	dnsMgr := &testutil.MockDNSManager{}

	pipe := New(tunnelMgr, connector, accessMgr, dnsMgr, store)

	err := pipe.Serve(context.Background(), ServeParams{
		OriginConfig: originConfigForTest(t),
		Hostname:     "test.example.com",
		AccountID:    "test-acct",
		ZoneID:       "test-zone",
		CreateDNS:    true,
		Policy:       access.Policy{AuthMode: "otp"},
	})

	if err == nil {
		t.Error("expected error from access creation failure")
	}

	// Tunnel should be deleted in rollback.
	if len(tunnelMgr.DeleteCalls) != 1 {
		t.Errorf("expected 1 tunnel Delete (rollback), got %d", len(tunnelMgr.DeleteCalls))
	}
}

func TestServeNoDNS(t *testing.T) {
	store := testutil.NewMockStore()
	tunnelMgr := &testutil.MockTunnelManager{}
	connector := &testutil.MockConnector{}
	accessMgr := &testutil.MockAccessManager{}
	dnsMgr := &testutil.MockDNSManager{}

	pipe := New(tunnelMgr, connector, accessMgr, dnsMgr, store)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := pipe.Serve(ctx, ServeParams{
		OriginConfig: originConfigForTest(t),
		Hostname:     "test.example.com",
		AccountID:    "test-acct",
		ZoneID:       "test-zone",
		CreateDNS:    false,
		Policy:       access.Policy{AuthMode: "otp"},
	})

	if err != nil {
		t.Fatal(err)
	}

	if len(dnsMgr.CreateCNAMECalls) != 0 {
		t.Errorf("CreateCNAME should not be called when CreateDNS=false, got %d calls", len(dnsMgr.CreateCNAMECalls))
	}
}

func TestTeardownSession(t *testing.T) {
	store := testutil.NewMockStore()
	tunnelMgr := &testutil.MockTunnelManager{}
	connector := &testutil.MockConnector{}
	accessMgr := &testutil.MockAccessManager{}
	dnsMgr := &testutil.MockDNSManager{}

	pipe := New(tunnelMgr, connector, accessMgr, dnsMgr, store)

	sess := testutil.MakeTestSession("sess-001", "test.example.com")
	store.Save(context.Background(), sess)

	err := pipe.TeardownSession(context.Background(), sess, TeardownOpts{
		DeleteAccessApp: true,
		DeleteDNS:       true,
		DeleteTunnel:    true,
	})

	if err != nil {
		t.Fatal(err)
	}

	if len(accessMgr.DeleteAppCalls) != 1 {
		t.Errorf("expected 1 DeleteApp call, got %d", len(accessMgr.DeleteAppCalls))
	}
	if len(dnsMgr.DeleteRecordCalls) != 1 {
		t.Errorf("expected 1 DeleteRecord call, got %d", len(dnsMgr.DeleteRecordCalls))
	}
	if len(tunnelMgr.DeleteCalls) != 1 {
		t.Errorf("expected 1 tunnel Delete call, got %d", len(tunnelMgr.DeleteCalls))
	}
	if !connector.StopCalled {
		t.Error("expected connector Stop to be called")
	}

	if sess.State != session.StateStopped {
		t.Errorf("expected stopped state, got %s", sess.State)
	}
}
