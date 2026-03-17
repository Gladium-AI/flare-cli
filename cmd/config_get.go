package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/paoloanzn/flare-cli/internal/ui"
)

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Read configuration values",
	Long: `Print one or all configuration values.

Examples:
  flare config get
  flare config get cloudflare.account_id
  flare config get defaults.auth`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			key := args[0]
			val := viper.Get(key)
			if val == nil {
				return fmt.Errorf("key %q is not set", key)
			}

			if jsonOut {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{key: val})
			}

			fmt.Fprintln(cmd.OutOrStdout(), val)
			return nil
		}

		// Print all keys.
		settings := viper.AllSettings()

		if jsonOut {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(settings)
		}

		// Flatten and print.
		flat := flattenMap("", settings)
		keys := make([]string, 0, len(flat))
		for k := range flat {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			ui.PrintKeyValue(k, fmt.Sprintf("%v", flat[k]))
		}
		return nil
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
}

func flattenMap(prefix string, m map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		if sub, ok := v.(map[string]any); ok {
			for sk, sv := range flattenMap(key, sub) {
				result[sk] = sv
			}
		} else {
			result[key] = v
		}
	}
	return result
}
