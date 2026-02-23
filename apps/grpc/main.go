package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/evstack/ev-node/apps/grpc/cmd"
	evcmd "github.com/evstack/ev-node/pkg/cmd"
	"github.com/evstack/ev-node/pkg/config"
)

func main() {
	// Initiate the root command
	rootCmd := &cobra.Command{
		Use:   "evgrpc",
		Short: "Evolve node with gRPC execution client; single sequencer",
		Long: `Run a Evolve node with a gRPC-based execution client.
This allows you to connect to any execution layer that implements
the Evolve execution gRPC interface.`,
	}

	config.AddGlobalFlags(rootCmd, "evgrpc")

	rootCmd.AddCommand(
		cmd.InitCmd(),
		cmd.RunCmd,
		evcmd.VersionCmd,
		evcmd.NetInfoCmd,
		evcmd.StoreUnsafeCleanCmd,
		evcmd.KeysCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		// Print to stderr and exit with error
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
