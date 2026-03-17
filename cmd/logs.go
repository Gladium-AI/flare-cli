package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/paoloanzn/flare-cli/internal/session"
	"github.com/paoloanzn/flare-cli/internal/ui"
)

var logsCmd = &cobra.Command{
	Use:   "logs <session-id>",
	Short: "Stream origin and tunnel connector logs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		store, err := getStore()
		if err != nil {
			return err
		}

		sess, err := store.Resolve(ctx, args[0])
		if err != nil {
			return err
		}

		if sess.State == session.StateStopped {
			return fmt.Errorf("session %s is stopped; no live logs available", sess.ShortID())
		}

		source, _ := cmd.Flags().GetString("source")

		// For now, show session info since live log streaming requires
		// attaching to the running process (which is in a different PID space
		// when serve is running in another terminal).
		ui.PrintHeader("Logs for session " + sess.ShortID())
		ui.PrintKeyValue("Origin Type", sess.OriginType)
		ui.PrintKeyValue("Hostname", sess.Hostname)
		ui.PrintKeyValue("State", string(sess.State))

		if sess.CloudflaredPID > 0 {
			ui.PrintKeyValue("Connector PID", fmt.Sprintf("%d", sess.CloudflaredPID))
		}

		// Attempt to read from the session log file if it exists.
		logPath := sessionLogPath(sess.ID)
		if _, statErr := os.Stat(logPath); statErr == nil {
			return streamLogFile(ctx, logPath, source)
		}

		ui.PrintInfo("No stored logs available. Live logs are only visible in the terminal running 'flare serve'.")
		return nil
	},
}

func init() {
	f := logsCmd.Flags()
	f.String("source", "all", "Log source: all, origin, cloudflared")
	f.Bool("follow", false, "Follow log output")
	f.String("since", "", "Show logs since duration (e.g., 10m)")

	rootCmd.AddCommand(logsCmd)
}

func sessionLogPath(sessionID string) string {
	dir, _ := os.UserHomeDir()
	return fmt.Sprintf("%s/.config/flare-cli/logs/%s.log", dir, sessionID)
}

func streamLogFile(_ interface{}, path string, _ string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("reading log file: %w", err)
	}
	return nil
}
