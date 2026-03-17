package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/paoloanzn/flare-cli/internal/ui"
)

// Build-time variables injected via -ldflags.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version and build information",
	RunE: func(cmd *cobra.Command, args []string) error {
		info := map[string]string{
			"version":    Version,
			"commit":     Commit,
			"built":      Date,
			"go":         runtime.Version(),
			"os/arch":    runtime.GOOS + "/" + runtime.GOARCH,
			"cloudflared": detectCloudflared(),
			"docker":     detectDocker(),
		}

		if jsonOut {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(info)
		}

		fmt.Fprintln(cmd.OutOrStdout(), ui.Bold.Render("flare-cli"))
		fmt.Fprintf(cmd.OutOrStdout(), "  Version:     %s\n", info["version"])
		fmt.Fprintf(cmd.OutOrStdout(), "  Commit:      %s\n", info["commit"])
		fmt.Fprintf(cmd.OutOrStdout(), "  Built:       %s\n", info["built"])
		fmt.Fprintf(cmd.OutOrStdout(), "  Go:          %s\n", info["go"])
		fmt.Fprintf(cmd.OutOrStdout(), "  OS/Arch:     %s\n", info["os/arch"])
		fmt.Fprintf(cmd.OutOrStdout(), "  cloudflared: %s\n", info["cloudflared"])
		fmt.Fprintf(cmd.OutOrStdout(), "  docker:      %s\n", info["docker"])

		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func detectCloudflared() string {
	out, err := exec.Command("cloudflared", "version").CombinedOutput()
	if err != nil {
		return "not found"
	}
	return strings.TrimSpace(strings.Split(string(out), "\n")[0])
}

func detectDocker() string {
	out, err := exec.Command("docker", "version", "--format", "{{.Client.Version}}").CombinedOutput()
	if err != nil {
		return "not found"
	}
	return strings.TrimSpace(string(out))
}
