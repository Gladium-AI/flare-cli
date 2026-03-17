package cmd

import (
	"fmt"
	"strings"
	"time"

	cf "github.com/cloudflare/cloudflare-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	accesspkg "github.com/paoloanzn/flare-cli/internal/access"
	"github.com/paoloanzn/flare-cli/internal/config"
	dnspkg "github.com/paoloanzn/flare-cli/internal/dns"
	"github.com/paoloanzn/flare-cli/internal/origin"
	"github.com/paoloanzn/flare-cli/internal/pipeline"
	"github.com/paoloanzn/flare-cli/internal/session"
	tunnelpkg "github.com/paoloanzn/flare-cli/internal/tunnel"
)

var serveCmd = &cobra.Command{
	Use:   "serve <origin-type>",
	Short: "Start an origin and expose it through Cloudflare Tunnel with Access",
	Long: `Serve starts or attaches to a local application, exposes it through a
Cloudflare Tunnel hostname, and protects it with Cloudflare Access.

Supported origin types:
  local:http              Expose an already-running local HTTP service
  local:command           Launch a command and expose it
  docker:container        Run a Docker container and expose it
  docker:compose-service  Start a Compose service and expose it
  builtin:static          Serve a static directory
  builtin:file-browser    Expose a built-in file browser`,
	Args: cobra.ExactArgs(1),
	RunE: runServe,
}

func init() {
	f := serveCmd.Flags()

	// Shared flags.
	f.String("name", "", "Human-readable session name")
	f.String("hostname", "", "Full public hostname (e.g., app.example.com)")
	f.String("subdomain", "", "Subdomain (combined with configured domain)")
	f.String("ttl", "", "Session lifetime (e.g., 30m, 2h)")
	f.String("session-duration", "", "Access session duration (e.g., 30m, 1h)")
	f.String("auth", "", "Auth mode: otp, idp, idp+mtls, service-token")
	f.StringSlice("allow-email", nil, "Allowed email addresses (repeatable)")
	f.StringSlice("allow-domain", nil, "Allowed email domains (repeatable)")
	f.Bool("require-mtls", false, "Require mTLS client certificate")
	f.Bool("create-dns", true, "Create DNS record automatically")
	f.Bool("ephemeral-access-app", true, "Delete Access app on close")
	f.Bool("ephemeral-dns", true, "Delete DNS record on close")
	f.String("tunnel-id", "", "Force use of existing tunnel")
	f.Bool("reuse-tunnel", false, "Reuse shared tunnel")
	f.String("health-path", "", "HTTP health check path")
	f.String("wait-for-ready", "", "Startup timeout (e.g., 60s, 90s)")
	f.String("idempotency-key", "", "Idempotency key for agent workflows")

	// local:http flags.
	f.String("url", "", "Local origin URL (for local:http)")
	f.String("health-url", "", "Override health check URL")

	// local:command flags.
	f.String("cmd", "", "Command to execute (for local:command)")
	f.String("cwd", "", "Working directory")
	f.StringSlice("env", nil, "Environment variables (KEY=VALUE)")
	f.Int("port", 0, "Port the app listens on")

	// builtin:static flags.
	f.String("path", "", "Root directory (for builtin:static, builtin:file-browser)")
	f.String("index", "index.html", "Index file")
	f.Bool("spa", false, "SPA fallback routing")
	f.String("cache-control", "", "Cache-Control header")

	// builtin:file-browser flags.
	f.Bool("read-only", false, "Read-only mode")
	f.Bool("allow-upload", false, "Allow file uploads")
	f.Bool("allow-delete", false, "Allow file deletion")
	f.Bool("allow-rename", false, "Allow file renaming")
	f.Bool("show-hidden", false, "Show hidden files")
	f.Bool("download", true, "Allow downloads")

	// docker:container flags.
	f.String("image", "", "Docker image")
	f.Int("container-port", 0, "Container port")
	f.String("publish-port", "", "Local bind port (e.g., 127.0.0.1:38080)")
	f.String("entrypoint", "", "Container entrypoint")
	f.String("network", "", "Docker network")
	f.Bool("rm", true, "Remove container on stop")
	f.StringSlice("mount", nil, "Bind mounts (src:dst:mode)")

	// docker:compose-service flags.
	f.String("file", "", "Compose file path")
	f.String("service", "", "Compose service name")
	f.String("project-name", "", "Compose project name")
	f.Bool("build", false, "Build images before starting")
	f.Bool("up-detached", true, "Run compose in detached mode")

	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	originType := origin.Type(args[0])

	// Validate config is ready.
	if err := config.Validate(); err != nil {
		return err
	}

	accountID := viper.GetString(config.KeyAccountID)
	zoneID := viper.GetString(config.KeyZoneID)
	domain := viper.GetString(config.KeyDomain)

	// Resolve hostname.
	hostname, _ := cmd.Flags().GetString("hostname")
	subdomain, _ := cmd.Flags().GetString("subdomain")
	if hostname == "" && subdomain != "" {
		hostname = subdomain + "." + domain
	}
	if hostname == "" {
		return fmt.Errorf("--hostname or --subdomain is required")
	}

	// Build origin config from flags.
	originCfg, err := buildOriginConfig(cmd, originType)
	if err != nil {
		return err
	}

	// Build Access policy from flags.
	policy, err := buildAccessPolicy(cmd)
	if err != nil {
		return err
	}

	// Parse TTL.
	var ttl time.Duration
	if ttlStr, _ := cmd.Flags().GetString("ttl"); ttlStr != "" {
		ttl, err = time.ParseDuration(ttlStr)
		if err != nil {
			return fmt.Errorf("invalid --ttl: %w", err)
		}
	}

	createDNS, _ := cmd.Flags().GetBool("create-dns")
	ephemeralAccessApp, _ := cmd.Flags().GetBool("ephemeral-access-app")
	ephemeralDNS, _ := cmd.Flags().GetBool("ephemeral-dns")
	name, _ := cmd.Flags().GetString("name")

	// Get services (injected for tests, or create real ones).
	svc := getServices()
	if svc == nil {
		var buildErr error
		svc, buildErr = buildProductionServices()
		if buildErr != nil {
			return buildErr
		}
	}

	// Build and run pipeline.
	pipe := pipeline.New(svc.TunnelMgr, svc.Connector, svc.AccessMgr, svc.DNSMgr, svc.Store)

	return pipe.Serve(ctx, pipeline.ServeParams{
		OriginConfig:       originCfg,
		Hostname:           hostname,
		Name:               name,
		Policy:             policy,
		TTL:                ttl,
		AccountID:          accountID,
		ZoneID:             zoneID,
		CreateDNS:          createDNS,
		EphemeralAccessApp: ephemeralAccessApp,
		EphemeralDNS:       ephemeralDNS,
	})
}

func buildOriginConfig(cmd *cobra.Command, t origin.Type) (origin.Config, error) {
	cfg := origin.Config{Type: t}

	getString := func(name string) string {
		v, _ := cmd.Flags().GetString(name)
		return v
	}
	getInt := func(name string) int {
		v, _ := cmd.Flags().GetInt(name)
		return v
	}
	getBool := func(name string) bool {
		v, _ := cmd.Flags().GetBool(name)
		return v
	}

	cfg.URL = getString("url")
	cfg.HealthURL = getString("health-url")
	cfg.HealthPath = getString("health-path")
	cfg.WaitForReady = getString("wait-for-ready")
	cfg.Command = getString("cmd")
	cfg.Dir = getString("cwd")
	cfg.Port = getInt("port")
	cfg.Path = getString("path")
	cfg.Index = getString("index")
	cfg.SPA = getBool("spa")
	cfg.CacheControl = getString("cache-control")
	cfg.ReadOnly = getBool("read-only")
	cfg.AllowUpload = getBool("allow-upload")
	cfg.AllowDelete = getBool("allow-delete")
	cfg.AllowRename = getBool("allow-rename")
	cfg.ShowHidden = getBool("show-hidden")
	cfg.Download = getBool("download")
	cfg.Image = getString("image")
	cfg.ContainerPort = getInt("container-port")
	cfg.PublishPort = getString("publish-port")
	cfg.Entrypoint = getString("entrypoint")
	cfg.Network = getString("network")
	cfg.Remove = getBool("rm")
	cfg.ComposeFile = getString("file")
	cfg.ServiceName = getString("service")
	cfg.ProjectName = getString("project-name")
	cfg.Build = getBool("build")
	cfg.UpDetached = getBool("up-detached")

	// Parse env flags.
	envSlice, _ := cmd.Flags().GetStringSlice("env")
	if len(envSlice) > 0 {
		cfg.Env = make(map[string]string)
		for _, e := range envSlice {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) != 2 {
				return cfg, fmt.Errorf("invalid --env format: %q (expected KEY=VALUE)", e)
			}
			cfg.Env[parts[0]] = parts[1]
		}
	}

	return cfg, nil
}

func buildProductionServices() (*Services, error) {
	token := config.APIToken()
	cfClient, err := cf.NewWithAPIToken(token)
	if err != nil {
		return nil, fmt.Errorf("creating Cloudflare client: %w", err)
	}

	tunnelMgr := tunnelpkg.NewAPIManager(cfClient)
	connector := tunnelpkg.NewProcessConnector(viper.GetString(config.KeyCloudflaredBin))
	teamDomain := viper.GetString(config.KeyTeamDomain)
	accessMgr := accesspkg.NewAPIManager(cfClient, teamDomain)
	dnsMgr := dnspkg.NewAPIManager(cfClient)

	sessDir, err := config.SessionsDir()
	if err != nil {
		return nil, err
	}
	store, err := session.NewFileStore(sessDir)
	if err != nil {
		return nil, err
	}

	return &Services{
		TunnelMgr: tunnelMgr,
		Connector: connector,
		AccessMgr: accessMgr,
		DNSMgr:    dnsMgr,
		Store:     store,
	}, nil
}

func buildAccessPolicy(cmd *cobra.Command) (accesspkg.Policy, error) {
	allowEmails, _ := cmd.Flags().GetStringSlice("allow-email")
	allowDomains, _ := cmd.Flags().GetStringSlice("allow-domain")
	sessionDuration, _ := cmd.Flags().GetString("session-duration")
	auth, _ := cmd.Flags().GetString("auth")

	if sessionDuration == "" {
		sessionDuration = viper.GetString(config.KeyDefaultSessionDur)
	}
	if auth == "" {
		auth = viper.GetString(config.KeyDefaultAuth)
	}

	return accesspkg.Policy{
		AllowedEmails:   allowEmails,
		AllowedDomains:  allowDomains,
		SessionDuration: sessionDuration,
		AuthMode:        auth,
	}, nil
}
