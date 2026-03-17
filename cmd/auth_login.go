package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	cf "github.com/cloudflare/cloudflare-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/paoloanzn/flare-cli/internal/config"
	"github.com/paoloanzn/flare-cli/internal/ui"
)

// tokenCreateURL is the Cloudflare dashboard URL for creating API tokens.
// We link to the custom token creation page; the user picks the permissions.
const tokenCreateURL = "https://dash.cloudflare.com/profile/api-tokens"

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Cloudflare via API token",
	Long: `Opens the Cloudflare dashboard in your browser so you can create an API
token, then prompts you to paste it. After validating the token, flare-cli
auto-discovers your accounts and zones and writes the configuration.

The token needs these permissions:
  • Account — Cloudflare Tunnel — Edit
  • Account — Access: Apps and Policies — Edit
  • Zone   — DNS — Edit

If you already have a token, pass it with --token or pipe it in.`,
	RunE: runAuthLogin,
}

func init() {
	f := authLoginCmd.Flags()
	f.String("token", "", "API token (skip browser flow)")
	f.Bool("no-browser", false, "Don't open the browser automatically")

	authCmd.AddCommand(authLoginCmd)
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	token, _ := cmd.Flags().GetString("token")
	noBrowser, _ := cmd.Flags().GetBool("no-browser")

	if token == "" {
		// Interactive flow: open browser, ask user to paste token.
		ui.PrintHeader("Cloudflare API Token Setup")
		fmt.Println()
		ui.PrintInfo("flare-cli needs an API token with these permissions:")
		fmt.Println("  • Account — Cloudflare Tunnel — Edit")
		fmt.Println("  • Account — Access: Apps and Policies — Edit")
		fmt.Println("  • Zone   — DNS — Edit")
		fmt.Println()

		if !noBrowser {
			ui.PrintInfo("Opening Cloudflare dashboard in your browser...")
			if err := openBrowser(tokenCreateURL); err != nil {
				ui.PrintWarning("Could not open browser: %v", err)
			}
		}

		ui.PrintInfo("Create a Custom Token at: %s", tokenCreateURL)
		fmt.Println()
		fmt.Print("Paste your API token here: ")

		reader := bufio.NewReader(cmd.InOrStdin())
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading token: %w", err)
		}
		token = strings.TrimSpace(line)
	}

	if token == "" {
		return fmt.Errorf("no token provided")
	}

	// Step 1: Validate the token.
	ui.PrintInfo("Validating token...")
	client, err := cf.NewWithAPIToken(token)
	if err != nil {
		return fmt.Errorf("invalid token format: %w", err)
	}

	verifyCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	verify, err := client.VerifyAPIToken(verifyCtx)
	if err != nil {
		return fmt.Errorf("token verification failed: %w", err)
	}
	if verify.Status != "active" {
		return fmt.Errorf("token is not active (status: %s)", verify.Status)
	}
	ui.PrintSuccess("Token is valid (ID: %s)", verify.ID)

	// Step 2: Save the token to credentials file.
	if err := config.SaveCredential(token); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}
	credPath, _ := config.CredentialPath()
	ui.PrintSuccess("Token saved to %s", credPath)

	// Step 3: Auto-discover accounts.
	ui.PrintInfo("Discovering accounts...")
	accounts, _, err := client.Accounts(ctx, cf.AccountsListParams{})
	if err != nil {
		ui.PrintWarning("Could not list accounts: %v", err)
		ui.PrintInfo("You can set account_id manually with: flare config set cloudflare.account_id <id>")
	} else if len(accounts) == 0 {
		ui.PrintWarning("No accounts found for this token")
	} else {
		selectedAccount := accounts[0]
		if len(accounts) > 1 {
			fmt.Println()
			ui.PrintInfo("Found %d accounts:", len(accounts))
			for i, acct := range accounts {
				fmt.Printf("  [%d] %s (%s)\n", i+1, acct.Name, acct.ID)
			}
			fmt.Print("\nSelect account number [1]: ")
			reader := bufio.NewReader(cmd.InOrStdin())
			line, _ := reader.ReadString('\n')
			line = strings.TrimSpace(line)
			if line != "" {
				var idx int
				if _, err := fmt.Sscanf(line, "%d", &idx); err == nil && idx >= 1 && idx <= len(accounts) {
					selectedAccount = accounts[idx-1]
				}
			}
		}
		viper.Set(config.KeyAccountID, selectedAccount.ID)
		ui.PrintSuccess("Account: %s (%s)", selectedAccount.Name, selectedAccount.ID)
	}

	// Step 4: Auto-discover zones.
	ui.PrintInfo("Discovering zones...")
	zones, err := client.ListZones(ctx)
	if err != nil {
		ui.PrintWarning("Could not list zones: %v", err)
		ui.PrintInfo("You can set zone_id manually with: flare config set cloudflare.zone_id <id>")
	} else if len(zones) == 0 {
		ui.PrintWarning("No zones found for this token")
	} else {
		selectedZone := zones[0]
		if len(zones) > 1 {
			fmt.Println()
			ui.PrintInfo("Found %d zones:", len(zones))
			for i, z := range zones {
				fmt.Printf("  [%d] %s (%s)\n", i+1, z.Name, z.ID)
			}
			fmt.Print("\nSelect zone number [1]: ")
			reader := bufio.NewReader(cmd.InOrStdin())
			line, _ := reader.ReadString('\n')
			line = strings.TrimSpace(line)
			if line != "" {
				var idx int
				if _, err := fmt.Sscanf(line, "%d", &idx); err == nil && idx >= 1 && idx <= len(zones) {
					selectedZone = zones[idx-1]
				}
			}
		}
		viper.Set(config.KeyZoneID, selectedZone.ID)
		viper.Set(config.KeyDomain, selectedZone.Name)
		ui.PrintSuccess("Zone: %s (%s)", selectedZone.Name, selectedZone.ID)
	}

	// Step 5: Save config.
	if err := config.SaveConfig(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println()
	ui.PrintSuccess("Setup complete! You can now use 'flare serve'.")
	dir, _ := config.Dir()
	ui.PrintKeyValue("Config", dir+"/config.yaml")
	ui.PrintKeyValue("Credentials", dir+"/credentials")
	ui.PrintKeyValue("Account ID", viper.GetString(config.KeyAccountID))
	ui.PrintKeyValue("Zone ID", viper.GetString(config.KeyZoneID))
	ui.PrintKeyValue("Domain", viper.GetString(config.KeyDomain))

	return nil
}

// openBrowser opens a URL in the default browser.
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return exec.Command(cmd, args...).Start()
}
