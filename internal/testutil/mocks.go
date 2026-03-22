package testutil

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/paoloanzn/flare-cli/internal/access"
	"github.com/paoloanzn/flare-cli/internal/session"
	"github.com/paoloanzn/flare-cli/internal/tunnel"
)

// --- MockTunnelManager ---

type MockTunnelManager struct {
	mu sync.Mutex

	FnCreate           func(ctx context.Context, accountID, name string) (*tunnel.Info, error)
	FnConfigureIngress func(ctx context.Context, accountID, tunnelID, hostname, originURL string) error
	FnGetToken         func(ctx context.Context, accountID, tunnelID string) (string, error)
	FnDelete           func(ctx context.Context, accountID, tunnelID string) error

	CreateCalls           []struct{ AccountID, Name string }
	ConfigureIngressCalls []struct{ AccountID, TunnelID, Hostname, OriginURL string }
	GetTokenCalls         []struct{ AccountID, TunnelID string }
	DeleteCalls           []struct{ AccountID, TunnelID string }
}

func (m *MockTunnelManager) Create(ctx context.Context, accountID, name string) (*tunnel.Info, error) {
	m.mu.Lock()
	m.CreateCalls = append(m.CreateCalls, struct{ AccountID, Name string }{accountID, name})
	m.mu.Unlock()

	if m.FnCreate != nil {
		return m.FnCreate(ctx, accountID, name)
	}
	return &tunnel.Info{
		TunnelID:   "tun-test-uuid-0001-0001-000000000001",
		TunnelName: "flare-test",
		Token:      "eyJ0ZXN0IjoidG9rZW4ifQ==",
	}, nil
}

func (m *MockTunnelManager) ConfigureIngress(ctx context.Context, accountID, tunnelID, hostname, originURL string) error {
	m.mu.Lock()
	m.ConfigureIngressCalls = append(m.ConfigureIngressCalls, struct{ AccountID, TunnelID, Hostname, OriginURL string }{accountID, tunnelID, hostname, originURL})
	m.mu.Unlock()

	if m.FnConfigureIngress != nil {
		return m.FnConfigureIngress(ctx, accountID, tunnelID, hostname, originURL)
	}
	return nil
}

func (m *MockTunnelManager) GetToken(ctx context.Context, accountID, tunnelID string) (string, error) {
	m.mu.Lock()
	m.GetTokenCalls = append(m.GetTokenCalls, struct{ AccountID, TunnelID string }{accountID, tunnelID})
	m.mu.Unlock()

	if m.FnGetToken != nil {
		return m.FnGetToken(ctx, accountID, tunnelID)
	}
	return "eyJ0ZXN0IjoidG9rZW4ifQ==", nil
}

func (m *MockTunnelManager) Delete(ctx context.Context, accountID, tunnelID string) error {
	m.mu.Lock()
	m.DeleteCalls = append(m.DeleteCalls, struct{ AccountID, TunnelID string }{accountID, tunnelID})
	m.mu.Unlock()

	if m.FnDelete != nil {
		return m.FnDelete(ctx, accountID, tunnelID)
	}
	return nil
}

// --- MockConnector ---

type MockConnector struct {
	mu sync.Mutex

	FnRun     func(ctx context.Context, token string) (int, error)
	FnStop    func(ctx context.Context) error
	FnLogs    func() io.ReadCloser
	FnHealthy func() error
	FnExitCh  func() <-chan struct{}

	RunCalls   []string
	StopCalled bool
}

func (m *MockConnector) Run(ctx context.Context, token string) (int, error) {
	m.mu.Lock()
	m.RunCalls = append(m.RunCalls, token)
	m.mu.Unlock()

	if m.FnRun != nil {
		return m.FnRun(ctx, token)
	}
	return 12345, nil
}

func (m *MockConnector) Stop(ctx context.Context) error {
	m.mu.Lock()
	m.StopCalled = true
	m.mu.Unlock()

	if m.FnStop != nil {
		return m.FnStop(ctx)
	}
	return nil
}

func (m *MockConnector) Logs() io.ReadCloser {
	if m.FnLogs != nil {
		return m.FnLogs()
	}
	return io.NopCloser(strings.NewReader(""))
}

func (m *MockConnector) Healthy() error {
	if m.FnHealthy != nil {
		return m.FnHealthy()
	}
	return nil
}

func (m *MockConnector) ExitCh() <-chan struct{} {
	if m.FnExitCh != nil {
		return m.FnExitCh()
	}
	// Default: return a channel that never closes (process stays alive).
	return make(chan struct{})
}

// --- MockAccessManager ---

type MockAccessManager struct {
	mu sync.Mutex

	FnCreateApp    func(ctx context.Context, accountID, hostname string, policy access.Policy) (*access.AppInfo, error)
	FnUpdatePolicy func(ctx context.Context, accountID, appID, policyID string, policy access.Policy) error
	FnDeleteApp    func(ctx context.Context, accountID, appID string) error

	CreateAppCalls    []struct{ AccountID, Hostname string; Policy access.Policy }
	UpdatePolicyCalls []struct{ AccountID, AppID, PolicyID string; Policy access.Policy }
	DeleteAppCalls    []struct{ AccountID, AppID string }
}

func (m *MockAccessManager) CreateApp(ctx context.Context, accountID, hostname string, policy access.Policy) (*access.AppInfo, error) {
	m.mu.Lock()
	m.CreateAppCalls = append(m.CreateAppCalls, struct{ AccountID, Hostname string; Policy access.Policy }{accountID, hostname, policy})
	m.mu.Unlock()

	if m.FnCreateApp != nil {
		return m.FnCreateApp(ctx, accountID, hostname, policy)
	}
	return &access.AppInfo{
		AppID:    "app-test-uuid-0001-0001-000000000001",
		PolicyID: "pol-test-uuid-0001-0001-000000000001",
		LoginURL: fmt.Sprintf("https://%s", hostname),
	}, nil
}

func (m *MockAccessManager) UpdatePolicy(ctx context.Context, accountID, appID, policyID string, policy access.Policy) error {
	m.mu.Lock()
	m.UpdatePolicyCalls = append(m.UpdatePolicyCalls, struct{ AccountID, AppID, PolicyID string; Policy access.Policy }{accountID, appID, policyID, policy})
	m.mu.Unlock()

	if m.FnUpdatePolicy != nil {
		return m.FnUpdatePolicy(ctx, accountID, appID, policyID, policy)
	}
	return nil
}

func (m *MockAccessManager) DeleteApp(ctx context.Context, accountID, appID string) error {
	m.mu.Lock()
	m.DeleteAppCalls = append(m.DeleteAppCalls, struct{ AccountID, AppID string }{accountID, appID})
	m.mu.Unlock()

	if m.FnDeleteApp != nil {
		return m.FnDeleteApp(ctx, accountID, appID)
	}
	return nil
}

// --- MockDNSManager ---

type MockDNSManager struct {
	mu sync.Mutex

	FnCreateCNAME  func(ctx context.Context, zoneID, hostname, tunnelID string) (string, error)
	FnDeleteRecord func(ctx context.Context, zoneID, recordID string) error

	CreateCNAMECalls  []struct{ ZoneID, Hostname, TunnelID string }
	DeleteRecordCalls []struct{ ZoneID, RecordID string }
}

func (m *MockDNSManager) CreateCNAME(ctx context.Context, zoneID, hostname, tunnelID string) (string, error) {
	m.mu.Lock()
	m.CreateCNAMECalls = append(m.CreateCNAMECalls, struct{ ZoneID, Hostname, TunnelID string }{zoneID, hostname, tunnelID})
	m.mu.Unlock()

	if m.FnCreateCNAME != nil {
		return m.FnCreateCNAME(ctx, zoneID, hostname, tunnelID)
	}
	return "dns-rec-test-uuid-0001-0001-000000000001", nil
}

func (m *MockDNSManager) DeleteRecord(ctx context.Context, zoneID, recordID string) error {
	m.mu.Lock()
	m.DeleteRecordCalls = append(m.DeleteRecordCalls, struct{ ZoneID, RecordID string }{zoneID, recordID})
	m.mu.Unlock()

	if m.FnDeleteRecord != nil {
		return m.FnDeleteRecord(ctx, zoneID, recordID)
	}
	return nil
}

// --- MockStore ---

type MockStore struct {
	mu       sync.Mutex
	sessions map[string]*session.Session
}

func NewMockStore() *MockStore {
	return &MockStore{sessions: make(map[string]*session.Session)}
}

func (ms *MockStore) Save(_ context.Context, s *session.Session) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.sessions[s.ID] = s
	return nil
}

func (ms *MockStore) Load(_ context.Context, id string) (*session.Session, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	s, ok := ms.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session %q not found", id)
	}
	return s, nil
}

func (ms *MockStore) List(_ context.Context, stateFilter ...session.State) ([]*session.Session, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	filterSet := make(map[session.State]bool, len(stateFilter))
	for _, st := range stateFilter {
		filterSet[st] = true
	}

	var result []*session.Session
	for _, s := range ms.sessions {
		if len(filterSet) > 0 && !filterSet[s.State] {
			continue
		}
		result = append(result, s)
	}
	return result, nil
}

func (ms *MockStore) Delete(_ context.Context, id string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	delete(ms.sessions, id)
	return nil
}

func (ms *MockStore) Resolve(_ context.Context, idOrPrefix string) (*session.Session, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Exact match first.
	if s, ok := ms.sessions[idOrPrefix]; ok {
		return s, nil
	}

	// Prefix match.
	var matches []*session.Session
	for id, s := range ms.sessions {
		if strings.HasPrefix(id, idOrPrefix) {
			matches = append(matches, s)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no session matching %q", idOrPrefix)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("ambiguous session prefix %q matches %d sessions", idOrPrefix, len(matches))
	}
}

// Sessions returns a snapshot of all stored sessions (for test assertions).
func (ms *MockStore) Sessions() map[string]*session.Session {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	cp := make(map[string]*session.Session, len(ms.sessions))
	for k, v := range ms.sessions {
		cp[k] = v
	}
	return cp
}
