// Package main is the fiber-bench tool: a single-sequencer ev-node wired
// to a remote Fibre network for throughput measurement.
//
// It deliberately runs in the simplest possible configuration:
//
//   - Solo sequencer (no based / no forced inclusion)
//   - Aggregator-only (no syncer, no P2P)
//   - In-memory executor with constant state root (no state computation
//     cost in the measurement)
//   - Bridge-bypass Fibre adapter (Upload directly via consensus gRPC + FSPs)
//
// The intent is a fail-fast baseline so we can isolate ev-node's batching
// + DA-submit pipeline from everything else.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	// Pull celestia-app params for its init() that sets the global SDK
	// bech32 prefix to "celestia" — must run before any keyring operation
	// that prints addresses.
	_ "github.com/celestiaorg/celestia-app/v9/app/params"
)

func main() {
	root := &cobra.Command{
		Use:   "fiber-bench",
		Short: "Single-sequencer ev-node throughput bench against a remote Fibre network",
	}

	root.AddCommand(
		keysCmd(),
		escrowCmd(),
		runCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
