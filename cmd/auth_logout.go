package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/paoloanzn/flare-cli/internal/config"
	"github.com/paoloanzn/flare-cli/internal/ui"
)

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored Cloudflare credentials",
	Long: `Removes the API token stored by 'flare auth login'.

This does NOT revoke the token on Cloudflare's side — you can do that
from the Cloudflare dashboard. It only deletes the local credentials file.`,
	RunE: runAuthLogout,
}

func init() {
	authCmd.AddCommand(authLogoutCmd)
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	credPath, err := config.CredentialPath()
	if err != nil {
		return fmt.Errorf("finding credential path: %w", err)
	}

	token, _ := config.LoadCredential()
	if token == "" {
		ui.PrintWarning("No stored credentials found at %s", credPath)
		return nil
	}

	if err := config.DeleteCredential(); err != nil {
		return fmt.Errorf("removing credentials: %w", err)
	}

	ui.PrintSuccess("Credentials removed from %s", credPath)
	ui.PrintInfo("Your API token has NOT been revoked on Cloudflare.")
	ui.PrintInfo("To revoke it, visit: https://dash.cloudflare.com/profile/api-tokens")
	return nil
}
