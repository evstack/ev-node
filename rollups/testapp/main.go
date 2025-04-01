package main

import (
	"context"
	"fmt"
	"os"

	"cosmossdk.io/log"

	coreda "github.com/rollkit/rollkit/core/da"
	coresequencer "github.com/rollkit/rollkit/core/sequencer"
	"github.com/rollkit/rollkit/da"
	rollcmd "github.com/rollkit/rollkit/pkg/cmd"
	commands "github.com/rollkit/rollkit/rollups/testapp/cmd"
	testExecutor "github.com/rollkit/rollkit/rollups/testapp/kv"
)

func main() {
	// Initiate the root command
	rootCmd := commands.RootCmd

	// Create context for the executor
	ctx := context.Background()

	// Create test implementations
	// TODO: we need to start the executor http server
	executor := testExecutor.CreateDirectKVExecutor(ctx)
	sequencer := coresequencer.NewDummySequencer()

	// Create DA client with dummy DA
	dummyDA := coreda.NewDummyDA(100_000, 0, 0)
	logger := log.NewLogger(os.Stdout)
	dac := da.NewDAClient(dummyDA, 0, 1.0, []byte("test"), []byte(""), logger)

	// Add subcommands to the root command
	rootCmd.AddCommand(
		rollcmd.NewDocsGenCmd(rootCmd, commands.AppName),
		rollcmd.NewRunNodeCmd(executor, sequencer, dac),
		rollcmd.VersionCmd,
		rollcmd.InitCmd,
	)

	if err := rootCmd.Execute(); err != nil {
		// Print to stderr and exit with error
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
