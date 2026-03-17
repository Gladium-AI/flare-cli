package cmd

import (
	"github.com/paoloanzn/flare-cli/internal/access"
	"github.com/paoloanzn/flare-cli/internal/dns"
	"github.com/paoloanzn/flare-cli/internal/session"
	"github.com/paoloanzn/flare-cli/internal/tunnel"
)

// Services holds all injectable dependencies for commands.
type Services struct {
	TunnelMgr tunnel.Manager
	Connector tunnel.Connector
	AccessMgr access.Manager
	DNSMgr    dns.Manager
	Store     session.Store
}

// overrideServices is set by tests to inject mock dependencies.
var overrideServices *Services

// SetTestServices injects mock services for testing.
func SetTestServices(svc *Services) {
	overrideServices = svc
}

// ClearTestServices removes injected test services.
func ClearTestServices() {
	overrideServices = nil
}

// getServices returns the override services if set, nil otherwise.
func getServices() *Services {
	return overrideServices
}

// getStore returns the injected store if available, else creates a FileStore.
func getStore() (session.Store, error) {
	if svc := getServices(); svc != nil && svc.Store != nil {
		return svc.Store, nil
	}
	return loadSessionStore()
}
