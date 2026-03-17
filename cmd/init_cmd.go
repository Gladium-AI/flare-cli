package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/paoloanzn/flare-cli/internal/config"
	flareexec "github.com/paoloanzn/flare-cli/internal/exec"
	"github.com/paoloanzn/flare-cli/internal/ui"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap flare-cli for a Cloudflare account and zone",
	Long: `Initialize flare-cli by providing your Cloudflare account ID, zone ID,
and domain. This stores the configuration so subsequent commands can
create tunnels, DNS records, and Access applications.

Tip: Use 'flare auth login' to authenticate via browser and auto-discover
your account ID and zone ID instead of providing them manually.`,
	RunE: runInit,
}

func init() {
	f := initCmd.Flags()
	f.String("account-id", "", "Cloudflare account ID (required)")
	f.String("zone-id", "", "Cloudflare zone ID (required)")
	f.String("domain", "", "Root domain for generated hostnames (required)")
	f.String("api-token-env", "CLOUDFLARE_API_TOKEN", "Environment variable containing the API token")
	f.String("team-domain", "", "Zero Trust team domain (e.g., yourteam.cloudflareaccess.com)")
	f.String("cloudflared-bin", "", "Path to cloudflared binary")
	f.String("state-dir", "", "Local state/config directory")
	f.String("default-auth", "otp", "Default auth mode: otp, idp, idp+mtls, service-token")
	f.String("default-session-duration", "30m", "Default Access session duration")
	f.String("default-hostname-template", "{app}-{id}.{domain}", "Hostname generation template")
	f.Bool("verify", false, "Run prerequisite checks after init")
	f.Bool("non-interactive", false, "Skip interactive prompts")

	initCmd.MarkFlagRequired("account-id")
	initCmd.MarkFlagRequired("zone-id")
	initCmd.MarkFlagRequired("domain")

	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	accountID, _ := cmd.Flags().GetString("account-id")
	zoneID, _ := cmd.Flags().GetString("zone-id")
	domain, _ := cmd.Flags().GetString("domain")
	apiTokenEnv, _ := cmd.Flags().GetString("api-token-env")
	teamDomain, _ := cmd.Flags().GetString("team-domain")
	cloudflaredBin, _ := cmd.Flags().GetString("cloudflared-bin")
	stateDir, _ := cmd.Flags().GetString("state-dir")
	defaultAuth, _ := cmd.Flags().GetString("default-auth")
	defaultSessionDur, _ := cmd.Flags().GetString("default-session-duration")
	defaultHostTemplate, _ := cmd.Flags().GetString("default-hostname-template")
	verify, _ := cmd.Flags().GetBool("verify")

	// Set values in viper.
	viper.Set(config.KeyAccountID, accountID)
	viper.Set(config.KeyZoneID, zoneID)
	viper.Set(config.KeyDomain, domain)

	if apiTokenEnv != "" {
		viper.Set(config.KeyAPITokenEnv, apiTokenEnv)
	}
	if teamDomain != "" {
		viper.Set(config.KeyTeamDomain, teamDomain)
	}
	if cloudflaredBin != "" {
		viper.Set(config.KeyCloudflaredBin, cloudflaredBin)
	}
	if stateDir != "" {
		viper.Set(config.KeyStateDir, stateDir)
	}
	if defaultAuth != "" {
		viper.Set(config.KeyDefaultAuth, defaultAuth)
	}
	if defaultSessionDur != "" {
		viper.Set(config.KeyDefaultSessionDur, defaultSessionDur)
	}
	if defaultHostTemplate != "" {
		viper.Set(config.KeyDefaultHostTemplate, defaultHostTemplate)
	}

	// Write config file.
	if err := config.SaveConfig(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	dir, _ := config.Dir()
	ui.PrintSuccess("Configuration saved to %s/config.yaml", dir)
	ui.PrintKeyValue("Account ID", accountID)
	ui.PrintKeyValue("Zone ID", zoneID)
	ui.PrintKeyValue("Domain", domain)
	if teamDomain != "" {
		ui.PrintKeyValue("Team Domain", teamDomain)
	}

	// Check API token is available.
	token := config.APIToken()
	if token == "" {
		ui.PrintWarning("No API token found.")
		ui.PrintInfo("Run 'flare auth login' to authenticate via browser, or set $%s.", apiTokenEnv)
	} else {
		ui.PrintSuccess("API token detected")
	}

	if verify {
		fmt.Fprintln(cmd.OutOrStdout())
		return runDoctorChecks(cmd)
	}

	return nil
}

// runDoctorChecks is shared between init --verify and the doctor command.
func runDoctorChecks(cmd *cobra.Command) error {
	ui.PrintHeader("Prerequisite checks")
	hasErrors := false

	// Check cloudflared.
	cfBin := viper.GetString(config.KeyCloudflaredBin)
	if cfBin == "" {
		cfBin = "cloudflared"
	}
	cfPath, err := flareexec.Which(cfBin)
	if err != nil {
		ui.PrintError("cloudflared: not found in PATH")
		hasErrors = true
	} else {
		ver, _ := flareexec.Run(cmd.Context(), cfPath, "version")
		ui.PrintSuccess("cloudflared: %s", trimFirstLine(ver))
	}

	// Check docker.
	dockerPath, err := flareexec.Which("docker")
	if err != nil {
		ui.PrintWarning("docker: not found (only needed for docker:* origins)")
	} else {
		ver, _ := flareexec.Run(cmd.Context(), dockerPath, "version", "--format", "{{.Client.Version}}")
		ui.PrintSuccess("docker: %s", trimFirstLine(ver))
	}

	// Check API token.
	token := config.APIToken()
	if token == "" {
		ui.PrintError("API token: not set")
		hasErrors = true
	} else {
		ui.PrintSuccess("API token: present (%d chars)", len(token))
	}

	// Check config values.
	if viper.GetString(config.KeyAccountID) == "" {
		ui.PrintError("account_id: not configured")
		hasErrors = true
	} else {
		ui.PrintSuccess("account_id: %s", viper.GetString(config.KeyAccountID))
	}

	if viper.GetString(config.KeyZoneID) == "" {
		ui.PrintError("zone_id: not configured")
		hasErrors = true
	} else {
		ui.PrintSuccess("zone_id: %s", viper.GetString(config.KeyZoneID))
	}

	if viper.GetString(config.KeyDomain) == "" {
		ui.PrintError("domain: not configured")
		hasErrors = true
	} else {
		ui.PrintSuccess("domain: %s", viper.GetString(config.KeyDomain))
	}

	// Check state directory.
	dir, err := config.Dir()
	if err != nil {
		ui.PrintError("state dir: %v", err)
		hasErrors = true
	} else {
		fi, err := os.Stat(dir)
		if err != nil {
			ui.PrintError("state dir: %v", err)
			hasErrors = true
		} else if !fi.IsDir() {
			ui.PrintError("state dir: %s is not a directory", dir)
			hasErrors = true
		} else {
			ui.PrintSuccess("state dir: %s", dir)
		}
	}

	if hasErrors {
		return fmt.Errorf("some checks failed")
	}
	return nil
}

func trimFirstLine(s string) string {
	for i, c := range s {
		if c == '\n' {
			return s[:i]
		}
	}
	return s
}
