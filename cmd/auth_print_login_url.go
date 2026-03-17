package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/paoloanzn/flare-cli/internal/session"
	"github.com/paoloanzn/flare-cli/internal/ui"
)

var authPrintLoginURLCmd = &cobra.Command{
	Use:   "print-login-url <session-id>",
	Short: "Print the public login URL for a session",
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
			ui.PrintWarning("Session %s is not active (state: %s)", sess.ShortID(), sess.State)
		}

		loginURL := sess.LoginURL
		if loginURL == "" {
			loginURL = fmt.Sprintf("https://%s", sess.Hostname)
		}

		showQR, _ := cmd.Flags().GetBool("qr")

		if jsonOut {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(map[string]string{
				"session_id": sess.ID,
				"hostname":   sess.Hostname,
				"login_url":  loginURL,
			})
		}

		fmt.Fprintln(cmd.OutOrStdout(), loginURL)

		if showQR {
			printQR(loginURL)
		}

		return nil
	},
}

func init() {
	authPrintLoginURLCmd.Flags().Bool("qr", false, "Display URL as QR code in terminal")
	authCmd.AddCommand(authPrintLoginURLCmd)
}

// printQR prints a simple text representation pointing to the URL.
// A full QR implementation would use a library like skip2/go-qrcode.
func printQR(url string) {
	fmt.Println()
	ui.PrintInfo("QR code generation requires 'qrencode' or a Go QR library.")
	ui.PrintInfo("Install qrencode and pipe: echo '%s' | qrencode -t UTF8", url)
}
