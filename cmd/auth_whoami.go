package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	cf "github.com/cloudflare/cloudflare-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/paoloanzn/flare-cli/internal/config"
	"github.com/paoloanzn/flare-cli/internal/ui"
)

var authWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current Cloudflare authentication info",
	Long: `Validates the current API token and displays information about
the authenticated identity, including token status and associated accounts.`,
	RunE: runAuthWhoami,
}

func init() {
	authCmd.AddCommand(authWhoamiCmd)
}

func runAuthWhoami(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	token := config.APIToken()
	if token == "" {
		return fmt.Errorf("no API token found — run 'flare auth login' to authenticate")
	}

	// Determine token source for display.
	tokenSource := detectTokenSource(token)

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

	if jsonOut {
		result := map[string]any{
			"token_id":     verify.ID,
			"status":       verify.Status,
			"token_source": tokenSource,
			"token_hint":   maskToken(token),
		}

		// Try to get account info.
		accounts, _, accErr := client.Accounts(ctx, cf.AccountsListParams{})
		if accErr == nil && len(accounts) > 0 {
			acctList := make([]map[string]string, len(accounts))
			for i, a := range accounts {
				acctList[i] = map[string]string{
					"id":   a.ID,
					"name": a.Name,
				}
			}
			result["accounts"] = acctList
		}

		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	// Human-readable output.
	ui.PrintHeader("Cloudflare Authentication")
	fmt.Println()
	ui.PrintKeyValue("Token ID", verify.ID)
	ui.PrintKeyValue("Status", verify.Status)
	ui.PrintKeyValue("Source", tokenSource)
	ui.PrintKeyValue("Token", maskToken(token))

	// Show accounts.
	accounts, _, accErr := client.Accounts(ctx, cf.AccountsListParams{})
	if accErr == nil && len(accounts) > 0 {
		fmt.Println()
		ui.PrintInfo("Accounts:")
		for _, a := range accounts {
			fmt.Printf("  %s (%s)\n", a.Name, a.ID)
		}
	}

	return nil
}

// detectTokenSource identifies where the current token came from.
func detectTokenSource(token string) string {
	// Check env var (highest priority in config.APIToken()).
	envName := viper.GetString(config.KeyAPITokenEnv)
	if envName == "" {
		envName = "CLOUDFLARE_API_TOKEN"
	}
	if envToken := os.Getenv(envName); envToken != "" && envToken == token {
		return fmt.Sprintf("environment variable (%s)", envName)
	}

	// Check inline config value.
	if cfgToken := viper.GetString("cloudflare.api_token"); cfgToken != "" && cfgToken == token {
		return "config file (cloudflare.api_token)"
	}

	// Check stored credentials file.
	if stored, err := config.LoadCredential(); err == nil && stored != "" && stored == token {
		return "credentials file (~/.config/flare-cli/credentials)"
	}

	return "unknown"
}

// maskToken shows only the first 4 and last 4 characters of a token.
func maskToken(token string) string {
	if len(token) <= 10 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
