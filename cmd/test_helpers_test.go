package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/pflag"

	"github.com/paoloanzn/flare-cli/internal/testutil"
)

// newTestServices creates a Services struct with all mocks wired up.
func newTestServices(store *testutil.MockStore) (*Services, *testutil.MockTunnelManager, *testutil.MockConnector, *testutil.MockAccessManager, *testutil.MockDNSManager) {
	tunnelMgr := &testutil.MockTunnelManager{}
	connector := &testutil.MockConnector{}
	accessMgr := &testutil.MockAccessManager{}
	dnsMgr := &testutil.MockDNSManager{}

	svc := &Services{
		TunnelMgr: tunnelMgr,
		Connector: connector,
		AccessMgr: accessMgr,
		DNSMgr:    dnsMgr,
		Store:     store,
	}

	return svc, tunnelMgr, connector, accessMgr, dnsMgr
}

// resetFlags marks all flags as unchanged and resets their values to defaults.
func resetFlags() {
	// Reset persistent flags on root.
	jsonOut = false
	verbose = false

	resetFlagSet := func(fs *pflag.FlagSet) {
		fs.VisitAll(func(f *pflag.Flag) {
			f.Changed = false
			// For slice flags, DefValue is "[]" but Set("[]") adds "[]" as element.
			// Use the underlying SliceValue interface if available.
			if sv, ok := f.Value.(pflag.SliceValue); ok {
				sv.Replace(nil)
			} else {
				f.Value.Set(f.DefValue)
			}
		})
	}

	// Reset "Changed" state on all subcommand flags to prevent leaking between tests.
	for _, cmd := range rootCmd.Commands() {
		resetFlagSet(cmd.Flags())
		for _, sub := range cmd.Commands() {
			resetFlagSet(sub.Flags())
		}
	}
}

// executeCmd runs a command and returns the output buffer.
// It resets global flag state before each execution.
func executeCmd(t *testing.T, args ...string) (*bytes.Buffer, error) {
	t.Helper()
	resetFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return buf, err
}
