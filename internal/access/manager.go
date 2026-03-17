package access

import (
	"context"
	"fmt"

	cf "github.com/cloudflare/cloudflare-go"
)

// AppInfo holds the created Access application details.
type AppInfo struct {
	AppID    string
	PolicyID string
	LoginURL string
}

// Policy defines who can access the application.
type Policy struct {
	AllowedEmails   []string
	AllowedDomains  []string
	SessionDuration string // e.g., "30m", "1h"
	AuthMode        string // otp, idp, idp+mtls, service-token
}

// Manager manages Cloudflare Access applications and policies.
type Manager interface {
	CreateApp(ctx context.Context, accountID, hostname string, policy Policy) (*AppInfo, error)
	UpdatePolicy(ctx context.Context, accountID, appID, policyID string, policy Policy) error
	DeleteApp(ctx context.Context, accountID, appID string) error
}

// APIManager implements Manager using the Cloudflare API.
type APIManager struct {
	client     *cf.API
	teamDomain string
}

// NewAPIManager creates a new Access manager.
func NewAPIManager(client *cf.API, teamDomain string) *APIManager {
	return &APIManager{client: client, teamDomain: teamDomain}
}

// CreateApp creates a self-hosted Access application with an allow policy.
func (m *APIManager) CreateApp(ctx context.Context, accountID, hostname string, policy Policy) (*AppInfo, error) {
	rc := cf.AccountIdentifier(accountID)

	sessionDur := policy.SessionDuration
	if sessionDur == "" {
		sessionDur = "30m"
	}

	// Create the Access application.
	app, err := m.client.CreateAccessApplication(ctx, rc, cf.CreateAccessApplicationParams{
		Name:            fmt.Sprintf("flare-%s", hostname),
		Domain:          hostname,
		Type:            cf.SelfHosted,
		SessionDuration: sessionDur,
	})
	if err != nil {
		return nil, fmt.Errorf("creating Access application: %w", err)
	}

	// Build the include rules for the allow policy.
	include := buildIncludeRules(policy)

	// Create an allow policy on the application.
	accessPolicy, err := m.client.CreateAccessPolicy(ctx, rc, cf.CreateAccessPolicyParams{
		ApplicationID: app.ID,
		Name:          fmt.Sprintf("flare-allow-%s", hostname),
		Decision:      "allow",
		Precedence:    1,
		Include:       include,
	})
	if err != nil {
		// Attempt cleanup of the app we just created.
		_ = m.client.DeleteAccessApplication(ctx, rc, app.ID)
		return nil, fmt.Errorf("creating Access policy: %w", err)
	}

	loginURL := fmt.Sprintf("https://%s", hostname)

	return &AppInfo{
		AppID:    app.ID,
		PolicyID: accessPolicy.ID,
		LoginURL: loginURL,
	}, nil
}

// UpdatePolicy modifies the allow policy on an existing Access application.
func (m *APIManager) UpdatePolicy(ctx context.Context, accountID, appID, policyID string, policy Policy) error {
	rc := cf.AccountIdentifier(accountID)

	include := buildIncludeRules(policy)

	params := cf.UpdateAccessPolicyParams{
		ApplicationID: appID,
		PolicyID:      policyID,
		Name:          "flare-allow-policy",
		Decision:      "allow",
		Precedence:    1,
		Include:       include,
	}

	if policy.SessionDuration != "" {
		params.SessionDuration = &policy.SessionDuration
	}

	_, err := m.client.UpdateAccessPolicy(ctx, rc, params)
	if err != nil {
		return fmt.Errorf("updating Access policy: %w", err)
	}
	return nil
}

// DeleteApp removes an Access application (cascades to its policies).
func (m *APIManager) DeleteApp(ctx context.Context, accountID, appID string) error {
	rc := cf.AccountIdentifier(accountID)
	if err := m.client.DeleteAccessApplication(ctx, rc, appID); err != nil {
		return fmt.Errorf("deleting Access application: %w", err)
	}
	return nil
}

// buildIncludeRules constructs Access policy include rules from a Policy.
func buildIncludeRules(policy Policy) []any {
	var include []any

	for _, email := range policy.AllowedEmails {
		include = append(include, cf.AccessGroupEmail{
			Email: struct {
				Email string `json:"email"`
			}{Email: email},
		})
	}

	for _, domain := range policy.AllowedDomains {
		include = append(include, cf.AccessGroupEmailDomain{
			EmailDomain: struct {
				Domain string `json:"domain"`
			}{Domain: domain},
		})
	}

	// If no specific rules, allow everyone (fallback).
	if len(include) == 0 {
		include = append(include, cf.AccessGroupEveryone{})
	}

	return include
}
