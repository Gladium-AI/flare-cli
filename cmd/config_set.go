package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/paoloanzn/flare-cli/internal/config"
	"github.com/paoloanzn/flare-cli/internal/ui"
)

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Write a configuration value",
	Long: `Set a persistent configuration value.

Examples:
  flare config set cloudflare.account_id abc123
  flare config set cloudflare.zone_id def456
  flare config set defaults.auth otp
  flare config set defaults.hostname_template "{app}-{id}.example.com"`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		viper.Set(key, value)

		if err := config.SaveConfig(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		ui.PrintSuccess("%s = %s", key, value)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
}
