package cmd

import (
	"github.com/spf13/cobra"

	config "github.com/evstack/ev-node/pkg/config"
)

const (
	// AppName is the name of the application, the name of the command, and the name of the home directory.
	AppName = "testapp"

	// flagKVEndpoint is the flag for the KV endpoint
	flagKVEndpoint = "kv-endpoint"
)

func init() {
	config.AddGlobalFlags(RootCmd, AppName)
	config.AddFlags(RunCmd)
	// Add the KV endpoint flag specifically to the RunCmd
	RunCmd.Flags().String(flagKVEndpoint, "", "Address and port for the KV executor HTTP server")
}

// RootCmd is the root command for Evolve
var RootCmd = &cobra.Command{
	Use:   AppName,
	Short: "Testapp is a test application for Evolve, it consists of a simple key-value store and a single sequencer.",
}
