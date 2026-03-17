package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/paoloanzn/flare-cli/internal/config"
	"github.com/paoloanzn/flare-cli/internal/pipeline"
	"github.com/paoloanzn/flare-cli/internal/session"
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
	f.String("grace-period", "10s", "Graceful shutdown timeout")
	f.Bool("force", false, "Force close even if resources are in error state")

	rootCmd.AddCommand(closeCmd)
}

func runClose(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	sessionID := args[0]

	store, err := getStore()
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

	// Get services (injected for tests, or create real ones).
	svc := getServices()
	if svc == nil {
		var buildErr error
		svc, buildErr = buildProductionServices()
		if buildErr != nil {
			return buildErr
		}
	}

	pipe := pipeline.New(svc.TunnelMgr, svc.Connector, svc.AccessMgr, svc.DNSMgr, store)

	return pipe.TeardownSession(ctx, sess, pipeline.TeardownOpts{
		DeleteAccessApp: deleteAccessApp,
		DeleteDNS:       deleteDNS,
		DeleteTunnel:    deleteTunnel,
		Force:           force,
	})
}

// loadSessionStore creates a FileStore from the configured sessions directory.
func loadSessionStore() (session.Store, error) {
	dir, err := config.SessionsDir()
	if err != nil {
		return nil, err
	}
	return session.NewFileStore(dir)
}
