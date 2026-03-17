package cmd

import (
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Validate prerequisites, API token, and runtime environment",
	Long: `Check that cloudflared is installed, the API token is valid,
configuration is complete, and Docker is available if needed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctorChecks(cmd)
	},
}

func init() {
	f := doctorCmd.Flags()
	f.Bool("network", false, "Include network connectivity checks")
	f.Bool("dns", false, "Include DNS resolution checks")
	f.Bool("access", false, "Include Access/Zero Trust checks")
	f.Bool("cloudflared", false, "Include detailed cloudflared checks")
	f.Bool("docker", false, "Include Docker checks")

	rootCmd.AddCommand(doctorCmd)
}
