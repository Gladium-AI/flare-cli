package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/paoloanzn/flare-cli/internal/access"
	"github.com/paoloanzn/flare-cli/internal/dns"
	"github.com/paoloanzn/flare-cli/internal/origin"
	"github.com/paoloanzn/flare-cli/internal/session"
	"github.com/paoloanzn/flare-cli/internal/tunnel"
	"github.com/paoloanzn/flare-cli/internal/ui"
)

// ServeParams holds the inputs for the serve pipeline.
type ServeParams struct {
	// Origin configuration.
	OriginConfig origin.Config

	// Hostname and naming.
	Hostname string
	Name     string // Human-readable session name.

	// Access policy.
	Policy access.Policy

	// TTL (0 = no auto-expiry).
	TTL time.Duration

	// Cloudflare identifiers.
	AccountID string
	ZoneID    string

	// Ephemeral flags.
	EphemeralAccessApp bool
	EphemeralDNS       bool

	// Tunnel reuse.
	TunnelID    string // Force use of existing tunnel.
	ReuseTunnel bool
	CreateDNS   bool
}

// Pipeline orchestrates the serve lifecycle.
type Pipeline struct {
	tunnelMgr  tunnel.Manager
	connector  tunnel.Connector
	accessMgr  access.Manager
	dnsMgr     dns.Manager
	store      session.Store
}

// New creates a new Pipeline with all required dependencies.
func New(
	tunnelMgr tunnel.Manager,
	connector tunnel.Connector,
	accessMgr access.Manager,
	dnsMgr dns.Manager,
	store session.Store,
) *Pipeline {
	return &Pipeline{
		tunnelMgr:  tunnelMgr,
		connector:  connector,
		accessMgr:  accessMgr,
		dnsMgr:     dnsMgr,
		store:      store,
	}
}

// Serve runs the full serve pipeline. It blocks until the context is cancelled
// or the TTL expires, then performs teardown.
func (p *Pipeline) Serve(ctx context.Context, params ServeParams) error {
	// Step 1: Create session.
	sess := &session.Session{
		ID:                 uuid.New().String(),
		Name:               params.Name,
		State:              session.StateInitializing,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
		OriginType:         string(params.OriginConfig.Type),
		AccountID:          params.AccountID,
		ZoneID:             params.ZoneID,
		Hostname:           params.Hostname,
		AuthMode:           params.Policy.AuthMode,
		AllowedEmails:      params.Policy.AllowedEmails,
		AllowedDomains:     params.Policy.AllowedDomains,
		SessionDuration:    params.Policy.SessionDuration,
		EphemeralAccessApp: params.EphemeralAccessApp,
		EphemeralDNS:       params.EphemeralDNS,
	}

	if params.TTL > 0 {
		exp := time.Now().UTC().Add(params.TTL)
		sess.ExpiresAt = &exp
	}

	if err := p.store.Save(ctx, sess); err != nil {
		return fmt.Errorf("saving initial session: %w", err)
	}

	// Track cleanup functions for rollback on failure.
	var cleanups []func(context.Context)

	rollback := func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i](cleanupCtx)
		}
		sess.SetError(fmt.Errorf("serve pipeline failed"))
		_ = p.store.Save(context.Background(), sess)
	}

	// Step 2: Start origin.
	ui.PrintInfo("Starting origin (%s)...", params.OriginConfig.Type)
	orig, err := origin.New(params.OriginConfig)
	if err != nil {
		rollback()
		return fmt.Errorf("creating origin: %w", err)
	}

	loopbackURL, err := orig.Start(ctx)
	if err != nil {
		rollback()
		return fmt.Errorf("starting origin: %w", err)
	}
	sess.LoopbackAddr = loopbackURL
	sess.Transition(session.StateProvisioning)
	_ = p.store.Save(ctx, sess)
	cleanups = append(cleanups, func(cleanupCtx context.Context) {
		orig.Stop(cleanupCtx)
	})
	ui.PrintSuccess("Origin ready at %s", loopbackURL)

	// Step 3: Create tunnel.
	ui.PrintInfo("Creating Cloudflare Tunnel...")
	tunnelName := fmt.Sprintf("flare-%s", sess.ShortID())
	info, err := p.tunnelMgr.Create(ctx, params.AccountID, tunnelName)
	if err != nil {
		rollback()
		return fmt.Errorf("creating tunnel: %w", err)
	}
	sess.TunnelID = info.TunnelID
	sess.TunnelName = info.TunnelName
	sess.TunnelToken = info.Token
	_ = p.store.Save(ctx, sess)
	cleanups = append(cleanups, func(cleanupCtx context.Context) {
		ui.PrintInfo("Deleting tunnel %s...", info.TunnelID)
		p.tunnelMgr.Delete(cleanupCtx, params.AccountID, info.TunnelID)
	})
	ui.PrintSuccess("Tunnel created: %s (%s)", info.TunnelName, info.TunnelID[:8])

	// Step 4: Configure tunnel ingress.
	ui.PrintInfo("Configuring tunnel ingress...")
	if err := p.tunnelMgr.ConfigureIngress(ctx, params.AccountID, info.TunnelID, params.Hostname, loopbackURL); err != nil {
		rollback()
		return fmt.Errorf("configuring ingress: %w", err)
	}
	ui.PrintSuccess("Ingress: %s -> %s", params.Hostname, loopbackURL)

	// Step 5: Create DNS record.
	if params.CreateDNS {
		ui.PrintInfo("Creating DNS record...")
		recordID, err := p.dnsMgr.CreateCNAME(ctx, params.ZoneID, params.Hostname, info.TunnelID)
		if err != nil {
			rollback()
			return fmt.Errorf("creating DNS record: %w", err)
		}
		sess.DNSRecordID = recordID
		_ = p.store.Save(ctx, sess)
		cleanups = append(cleanups, func(cleanupCtx context.Context) {
			if sess.EphemeralDNS {
				ui.PrintInfo("Deleting DNS record...")
				p.dnsMgr.DeleteRecord(cleanupCtx, params.ZoneID, recordID)
			}
		})
		ui.PrintSuccess("DNS record created for %s", params.Hostname)
	}

	// Step 6: Create Access application + policy.
	ui.PrintInfo("Creating Access application...")
	appInfo, err := p.accessMgr.CreateApp(ctx, params.AccountID, params.Hostname, params.Policy)
	if err != nil {
		rollback()
		return fmt.Errorf("creating Access app: %w", err)
	}
	sess.AccessAppID = appInfo.AppID
	sess.AccessPolicyID = appInfo.PolicyID
	sess.LoginURL = appInfo.LoginURL
	_ = p.store.Save(ctx, sess)
	cleanups = append(cleanups, func(cleanupCtx context.Context) {
		if sess.EphemeralAccessApp {
			ui.PrintInfo("Deleting Access application...")
			p.accessMgr.DeleteApp(cleanupCtx, params.AccountID, appInfo.AppID)
		}
	})
	ui.PrintSuccess("Access application created (auth: %s)", params.Policy.AuthMode)

	// Step 7: Start cloudflared connector.
	ui.PrintInfo("Starting cloudflared connector...")
	pid, err := p.connector.Run(ctx, info.Token)
	if err != nil {
		rollback()
		return fmt.Errorf("starting cloudflared: %w", err)
	}
	sess.CloudflaredPID = pid
	sess.Transition(session.StateActive)
	_ = p.store.Save(ctx, sess)
	cleanups = append(cleanups, func(cleanupCtx context.Context) {
		ui.PrintInfo("Stopping cloudflared...")
		p.connector.Stop(cleanupCtx)
	})
	ui.PrintSuccess("cloudflared running (PID %d)", pid)

	// Step 8: Print success summary.
	fmt.Println()
	ui.PrintHeader("Session Active")
	ui.PrintKeyValue("Session ID", sess.ID)
	ui.PrintKeyValue("Origin", sess.LoopbackAddr)
	ui.PrintKeyValue("Hostname", fmt.Sprintf("https://%s", sess.Hostname))
	ui.PrintKeyValue("Login URL", sess.LoginURL)
	ui.PrintKeyValue("Auth", sess.AuthMode)
	if sess.ExpiresAt != nil {
		ui.PrintKeyValue("Expires", sess.ExpiresAt.Format(time.RFC3339))
	}
	fmt.Println()
	ui.PrintInfo("Press Ctrl+C to stop")

	// Step 9: Wait for shutdown.
	return p.waitForShutdown(ctx, sess, params, cleanups)
}

func (p *Pipeline) waitForShutdown(ctx context.Context, sess *session.Session, params ServeParams, cleanups []func(context.Context)) error {
	var ttlTimer <-chan time.Time
	if params.TTL > 0 {
		ttlTimer = time.After(params.TTL)
	}

	select {
	case <-ctx.Done():
		ui.PrintInfo("Shutdown signal received...")
	case <-ttlTimer:
		ui.PrintInfo("TTL expired, shutting down...")
	}

	return p.Teardown(sess, cleanups)
}

// Teardown performs reverse-order cleanup of all resources.
func (p *Pipeline) Teardown(sess *session.Session, cleanups []func(context.Context)) error {
	sess.Transition(session.StateStopping)
	_ = p.store.Save(context.Background(), sess)

	fmt.Println()
	ui.PrintHeader("Tearing down")

	// Use a fresh context with timeout for cleanup.
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Execute cleanups in reverse order.
	for i := len(cleanups) - 1; i >= 0; i-- {
		cleanups[i](cleanupCtx)
	}

	sess.Transition(session.StateStopped)
	_ = p.store.Save(context.Background(), sess)

	ui.PrintSuccess("Session %s stopped", sess.ShortID())
	return nil
}

// TeardownSession loads a session and tears down its resources.
// Used by `flare close`.
func (p *Pipeline) TeardownSession(ctx context.Context, sess *session.Session, opts TeardownOpts) error {
	sess.Transition(session.StateStopping)
	_ = p.store.Save(ctx, sess)

	ui.PrintHeader("Tearing down session " + sess.ShortID())

	var errs []error

	// Stop cloudflared.
	if sess.CloudflaredPID > 0 {
		ui.PrintInfo("Stopping cloudflared...")
		if err := p.connector.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("stopping cloudflared: %w", err))
		}
	}

	// Delete Access application.
	if opts.DeleteAccessApp && sess.AccessAppID != "" {
		ui.PrintInfo("Deleting Access application...")
		if err := p.accessMgr.DeleteApp(ctx, sess.AccountID, sess.AccessAppID); err != nil {
			errs = append(errs, fmt.Errorf("deleting Access app: %w", err))
		} else {
			ui.PrintSuccess("Access application deleted")
		}
	}

	// Delete DNS record.
	if opts.DeleteDNS && sess.DNSRecordID != "" {
		ui.PrintInfo("Deleting DNS record...")
		if err := p.dnsMgr.DeleteRecord(ctx, sess.ZoneID, sess.DNSRecordID); err != nil {
			errs = append(errs, fmt.Errorf("deleting DNS record: %w", err))
		} else {
			ui.PrintSuccess("DNS record deleted")
		}
	}

	// Delete tunnel.
	if opts.DeleteTunnel && sess.TunnelID != "" {
		ui.PrintInfo("Deleting tunnel...")
		if err := p.tunnelMgr.Delete(ctx, sess.AccountID, sess.TunnelID); err != nil {
			errs = append(errs, fmt.Errorf("deleting tunnel: %w", err))
		} else {
			ui.PrintSuccess("Tunnel deleted")
		}
	}

	if len(errs) > 0 {
		sess.SetError(errs[0])
		_ = p.store.Save(ctx, sess)
		return fmt.Errorf("teardown completed with %d errors", len(errs))
	}

	sess.Transition(session.StateStopped)
	_ = p.store.Save(ctx, sess)
	ui.PrintSuccess("Session %s stopped", sess.ShortID())
	return nil
}

// TeardownOpts controls what gets cleaned up.
type TeardownOpts struct {
	DeleteAccessApp bool
	DeleteDNS       bool
	DeleteTunnel    bool
	Force           bool
}
