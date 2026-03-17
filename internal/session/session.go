package session

import "time"

// State represents the session lifecycle state.
type State string

const (
	StateInitializing State = "initializing"
	StateProvisioning State = "provisioning"
	StateActive       State = "active"
	StateStopping     State = "stopping"
	StateStopped      State = "stopped"
	StateError        State = "error"
)

// Session is the central state object persisted to disk.
type Session struct {
	ID        string     `json:"id"`
	Name      string     `json:"name,omitempty"`
	State     State      `json:"state"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// Origin info.
	OriginType   string            `json:"origin_type"`
	OriginConfig map[string]string `json:"origin_config,omitempty"`
	LoopbackAddr string            `json:"loopback_addr,omitempty"`

	// Cloudflare resource IDs.
	AccountID      string `json:"account_id"`
	ZoneID         string `json:"zone_id"`
	TunnelID       string `json:"tunnel_id,omitempty"`
	TunnelName     string `json:"tunnel_name,omitempty"`
	TunnelToken    string `json:"tunnel_token,omitempty"`
	DNSRecordID    string `json:"dns_record_id,omitempty"`
	AccessAppID    string `json:"access_app_id,omitempty"`
	AccessPolicyID string `json:"access_policy_id,omitempty"`

	// Public endpoint.
	Hostname string `json:"hostname"`
	LoginURL string `json:"login_url,omitempty"`

	// Access policy config.
	AuthMode        string   `json:"auth_mode,omitempty"`
	AllowedEmails   []string `json:"allowed_emails,omitempty"`
	AllowedDomains  []string `json:"allowed_domains,omitempty"`
	SessionDuration string   `json:"session_duration,omitempty"`
	RequireMTLS     bool     `json:"require_mtls,omitempty"`

	// Runtime.
	CloudflaredPID int    `json:"cloudflared_pid,omitempty"`
	LastError      string `json:"last_error,omitempty"`

	// Ephemeral flags (controls cleanup behavior).
	EphemeralAccessApp bool `json:"ephemeral_access_app,omitempty"`
	EphemeralDNS       bool `json:"ephemeral_dns,omitempty"`
}

// IsTerminal returns true if the session is in a final state.
func (s *Session) IsTerminal() bool {
	return s.State == StateStopped || s.State == StateError
}

// SetError transitions the session to error state with a message.
func (s *Session) SetError(err error) {
	s.State = StateError
	s.LastError = err.Error()
	s.UpdatedAt = time.Now().UTC()
}

// Transition updates the session state and timestamp.
func (s *Session) Transition(state State) {
	s.State = state
	s.UpdatedAt = time.Now().UTC()
}

// ShortID returns the first 8 characters of the session ID.
func (s *Session) ShortID() string {
	if len(s.ID) >= 8 {
		return s.ID[:8]
	}
	return s.ID
}
