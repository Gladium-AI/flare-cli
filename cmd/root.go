package cmd

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/paoloanzn/flare-cli/internal/config"
)

// ctxKey is the context key for storing the app context.
type ctxKey struct{}

var (
	verbose bool
	jsonOut bool
)

var rootCmd = &cobra.Command{
	Use:   "flare",
	Short: "Serve local apps through Cloudflare Tunnel with Access protection",
	Long: `flare-cli serves arbitrary local or containerized applications and secures
them with Cloudflare Tunnel + Cloudflare Access.

It supports built-in file browsers, static sites, local HTTP services,
local commands, Docker containers, and Docker Compose services.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Set up signal-aware context.
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		cmd.SetContext(context.WithValue(ctx, ctxKey{}, cancel))

		// Initialize config (non-fatal if config file missing).
		_ = config.Init()

		if verbose {
			viper.Set("log_level", "debug")
		}

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output in JSON format")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// ContextCancel retrieves the cancel function from the command context.
func ContextCancel(cmd *cobra.Command) context.CancelFunc {
	if cancel, ok := cmd.Context().Value(ctxKey{}).(context.CancelFunc); ok {
		return cancel
	}
	return func() {}
}
