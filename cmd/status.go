package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/paoloanzn/flare-cli/internal/ui"
)

var statusCmd = &cobra.Command{
	Use:   "status <session-id>",
	Short: "Show the state of a session",
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

		if jsonOut {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(sess)
		}

		ui.PrintHeader("Session " + sess.ShortID())
		ui.PrintKeyValue("ID", sess.ID)
		if sess.Name != "" {
			ui.PrintKeyValue("Name", sess.Name)
		}
		ui.PrintKeyValue("State", string(sess.State))
		ui.PrintKeyValue("Origin Type", sess.OriginType)
		if sess.LoopbackAddr != "" {
			ui.PrintKeyValue("Local URL", sess.LoopbackAddr)
		}
		ui.PrintKeyValue("Hostname", fmt.Sprintf("https://%s", sess.Hostname))
		if sess.LoginURL != "" {
			ui.PrintKeyValue("Login URL", sess.LoginURL)
		}
		if sess.TunnelID != "" {
			ui.PrintKeyValue("Tunnel ID", sess.TunnelID)
		}
		if sess.TunnelName != "" {
			ui.PrintKeyValue("Tunnel Name", sess.TunnelName)
		}
		if sess.CloudflaredPID > 0 {
			ui.PrintKeyValue("Connector PID", fmt.Sprintf("%d", sess.CloudflaredPID))
		}
		if sess.AuthMode != "" {
			ui.PrintKeyValue("Auth Mode", sess.AuthMode)
		}
		if sess.AccessAppID != "" {
			ui.PrintKeyValue("Access App ID", sess.AccessAppID)
		}
		if len(sess.AllowedEmails) > 0 {
			ui.PrintKeyValue("Allowed Emails", strings.Join(sess.AllowedEmails, ", "))
		}
		if len(sess.AllowedDomains) > 0 {
			ui.PrintKeyValue("Allowed Domains", strings.Join(sess.AllowedDomains, ", "))
		}
		if sess.SessionDuration != "" {
			ui.PrintKeyValue("Session Duration", sess.SessionDuration)
		}
		if sess.DNSRecordID != "" {
			ui.PrintKeyValue("DNS Record ID", sess.DNSRecordID)
		}
		ui.PrintKeyValue("Created", sess.CreatedAt.Format(time.RFC3339))
		ui.PrintKeyValue("Updated", sess.UpdatedAt.Format(time.RFC3339))
		if sess.ExpiresAt != nil {
			ui.PrintKeyValue("Expires", sess.ExpiresAt.Format(time.RFC3339))
		}
		if sess.LastError != "" {
			ui.PrintKeyValue("Last Error", sess.LastError)
		}

		return nil
	},
}

func init() {
	statusCmd.Flags().Bool("verbose", false, "Show all fields")
	rootCmd.AddCommand(statusCmd)
}
