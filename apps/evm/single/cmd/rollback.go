package cmd

import (
    "context"
    "fmt"

    ds "github.com/ipfs/go-datastore"
    ktds "github.com/ipfs/go-datastore/keytransform"

    "github.com/evstack/ev-node/node"
    rollcmd "github.com/evstack/ev-node/pkg/cmd"
    "github.com/evstack/ev-node/pkg/store"
    "github.com/spf13/cobra"
)

// RollbackCmd rolls back ev-node's persisted state for the evm-single app.
// It always reverts exactly one height (currentHeight - 1).
var RollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback the evm-single node by one height",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeConfig, err := rollcmd.ParseConfig(cmd)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

        // Open the ev-node data store for evm-single
        datastore, err := store.NewDefaultKVStore(nodeConfig.RootDir, nodeConfig.DBPath, "evm-single")
        if err != nil {
            return err
        }

        // Use the same namespace/prefix as the running node (see node/full.go: EvPrefix)
        prefixStore := ktds.Wrap(datastore, ktds.PrefixTransform{Prefix: ds.NewKey(node.EvPrefix)})

		storeWrapper := store.New(prefixStore)

		cmd.Println("Starting rollback operation")

		currentHeight, err := storeWrapper.Height(ctx)
		if err != nil {
			return fmt.Errorf("failed to get current height: %w", err)
		}

		// Ensure there is at least one block to roll back
		if currentHeight == 0 {
			return fmt.Errorf("no blocks to rollback: current height is 0")
		}

		// Always roll back exactly one height
		targetHeight := currentHeight - 1

		// Roll back ev-node store only. The EVM execution client maintains its own state.
		if err := storeWrapper.Rollback(ctx, targetHeight); err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}

		cmd.Println("Rollback completed successfully")
		return nil
	},
}
