package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/paoloanzn/flare-cli/internal/session"
	"github.com/paoloanzn/flare-cli/internal/ui"
)

var authRotateMTLSCmd = &cobra.Command{
	Use:   "rotate-mtls <session-id>",
	Short: "Rotate the mTLS certificate configuration for a session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		store, err := loadSessionStore()
		if err != nil {
			return err
		}

		sess, err := store.Resolve(ctx, args[0])
		if err != nil {
			return err
		}

		if sess.State != session.StateActive {
			return fmt.Errorf("session %s is not active (state: %s)", sess.ShortID(), sess.State)
		}

		caID, _ := cmd.Flags().GetString("ca-id")
		if caID == "" {
			return fmt.Errorf("--ca-id is required")
		}

		// TODO: Implement mTLS certificate rotation via Cloudflare Access API.
		// This requires:
		// 1. Upload new CA cert via Access mTLS certificates API
		// 2. Update the Access application to reference the new CA
		// 3. Optionally update the policy to require mTLS
		ui.PrintWarning("mTLS rotation is not yet implemented")
		ui.PrintInfo("CA ID: %s", caID)
		ui.PrintInfo("Session: %s", sess.ShortID())

		return fmt.Errorf("not yet implemented")
	},
}

func init() {
	f := authRotateMTLSCmd.Flags()
	f.String("ca-id", "", "Access mTLS CA ID (required)")
	f.Bool("require-idp", true, "Require IdP login in addition to mTLS")

	authCmd.AddCommand(authRotateMTLSCmd)
}
