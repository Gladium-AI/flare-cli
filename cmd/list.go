package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/paoloanzn/flare-cli/internal/session"
	"github.com/paoloanzn/flare-cli/internal/ui"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		store, err := loadSessionStore()
		if err != nil {
			return err
		}

		showAll, _ := cmd.Flags().GetBool("all")
		staleOnly, _ := cmd.Flags().GetBool("stale")

		var filters []session.State
		if !showAll {
			filters = []session.State{
				session.StateInitializing,
				session.StateProvisioning,
				session.StateActive,
				session.StateStopping,
				session.StateError,
			}
		}

		sessions, err := store.List(ctx, filters...)
		if err != nil {
			return err
		}

		if staleOnly {
			var stale []*session.Session
			for _, s := range sessions {
				if isStale(s) {
					stale = append(stale, s)
				}
			}
			sessions = stale
		}

		if jsonOut {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(sessions)
		}

		if len(sessions) == 0 {
			ui.PrintInfo("No sessions found")
			return nil
		}

		headers := []string{"ID", "STATE", "ORIGIN", "HOSTNAME", "CREATED", "EXPIRES"}
		var rows [][]string
		for _, s := range sessions {
			expires := "-"
			if s.ExpiresAt != nil {
				expires = s.ExpiresAt.Format(time.RFC3339)
			}
			rows = append(rows, []string{
				s.ShortID(),
				string(s.State),
				s.OriginType,
				s.Hostname,
				s.CreatedAt.Format("2006-01-02 15:04"),
				expires,
			})
		}

		ui.PrintTable(headers, rows)
		fmt.Fprintf(cmd.OutOrStdout(), "\n%d session(s)\n", len(sessions))
		return nil
	},
}

func init() {
	f := listCmd.Flags()
	f.Bool("all", false, "Include stopped sessions")
	f.Bool("cloudflare", false, "Verify Cloudflare-side resources")
	f.Bool("stale", false, "Show only stale/leaked sessions")

	rootCmd.AddCommand(listCmd)
}

// isStale returns true if a session appears stuck (not terminal, old).
func isStale(s *session.Session) bool {
	if s.IsTerminal() {
		return false
	}
	return time.Since(s.UpdatedAt) > 5*time.Minute
}
