package cmd

import (
	"fmt"
	"time"

	cf "github.com/cloudflare/cloudflare-go"
	"github.com/spf13/cobra"

	accesspkg "github.com/paoloanzn/flare-cli/internal/access"
	"github.com/paoloanzn/flare-cli/internal/config"
	dnspkg "github.com/paoloanzn/flare-cli/internal/dns"
	"github.com/paoloanzn/flare-cli/internal/pipeline"
	"github.com/paoloanzn/flare-cli/internal/session"
	tunnelpkg "github.com/paoloanzn/flare-cli/internal/tunnel"
)

var closeCmd = &cobra.Command{
	Use:   "close <session-id>",
	Short: "Stop a session and clean up Cloudflare resources",
	Args:  cobra.ExactArgs(1),
	RunE:  runClose,
}

func init() {
	f := closeCmd.Flags()
	f.Bool("delete-access-app", true, "Delete the Access application")
	f.Bool("delete-dns", true, "Delete the DNS record")
	f.Bool("delete-tunnel", true, "Delete the tunnel")
	f.Bool("stop-origin", true, "Stop the origin process")
	f.String("grace-period", "10s", "Graceful shutdown timeout")
	f.Bool("force", false, "Force close even if resources are in error state")

	rootCmd.AddCommand(closeCmd)
}

func runClose(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	sessionID := args[0]

	// Load session.
	store, err := loadSessionStore()
	if err != nil {
		return err
	}

	sess, err := store.Resolve(ctx, sessionID)
	if err != nil {
		return err
	}

	force, _ := cmd.Flags().GetBool("force")
	if sess.State == session.StateStopped && !force {
		return fmt.Errorf("session %s is already stopped", sess.ShortID())
	}

	deleteAccessApp, _ := cmd.Flags().GetBool("delete-access-app")
	deleteDNS, _ := cmd.Flags().GetBool("delete-dns")
	deleteTunnel, _ := cmd.Flags().GetBool("delete-tunnel")

	gracePeriodStr, _ := cmd.Flags().GetString("grace-period")
	gracePeriod, _ := time.ParseDuration(gracePeriodStr)
	if gracePeriod == 0 {
		gracePeriod = 10 * time.Second
	}

	// Create Cloudflare client and managers.
	token := config.APIToken()
	if token == "" {
		return fmt.Errorf("CLOUDFLARE_API_TOKEN not set")
	}

	cfClient, err := cf.NewWithAPIToken(token)
	if err != nil {
		return fmt.Errorf("creating Cloudflare client: %w", err)
	}

	pipe := pipeline.New(
		tunnelpkg.NewAPIManager(cfClient),
		tunnelpkg.NewProcessConnector(""),
		accesspkg.NewAPIManager(cfClient, ""),
		dnspkg.NewAPIManager(cfClient),
		store,
	)

	return pipe.TeardownSession(ctx, sess, pipeline.TeardownOpts{
		DeleteAccessApp: deleteAccessApp,
		DeleteDNS:       deleteDNS,
		DeleteTunnel:    deleteTunnel,
		Force:           force,
	})
}

// loadSessionStore creates a FileStore from the configured sessions directory.
func loadSessionStore() (*session.FileStore, error) {
	dir, err := config.SessionsDir()
	if err != nil {
		return nil, err
	}
	return session.NewFileStore(dir)
}
