package cmd

import (
	"context"
	"errors"
	"fmt"

	kvexecutor "github.com/evstack/ev-node/apps/testapp/kv"
	rollcmd "github.com/evstack/ev-node/pkg/cmd"
	"github.com/evstack/ev-node/pkg/store"

	"github.com/spf13/cobra"
)

// NewRollbackCmd creates a command to rollback ev-node state by one height.
func NewRollbackCmd() *cobra.Command {
	var (
		height   uint64
		syncNode bool
	)

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "rollback ev-node state by one height. Pass --height to specify another height to rollback to.",
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeConfig, err := rollcmd.ParseConfig(cmd)
			if err != nil {
				return err
			}

			goCtx := cmd.Context()
			if goCtx == nil {
				goCtx = context.Background()
			}

			// evolve db
			rawEvolveDB, err := store.NewDefaultKVStore(nodeConfig.RootDir, nodeConfig.DBPath, testDbName)
			if err != nil {
				return err
			}

			defer func() {
				if closeErr := rawEvolveDB.Close(); closeErr != nil {
					fmt.Printf("Warning: failed to close evolve database: %v\n", closeErr)
				}
			}()

			// prefixed evolve db
			evolveDB := store.NewEvNodeKVStore(rawEvolveDB)
			evolveStore := store.New(evolveDB)
			if height == 0 {
				currentHeight, err := evolveStore.Height(goCtx)
				if err != nil {
					return err
				}

				height = currentHeight - 1
			}

			executor, err := kvexecutor.NewKVExecutor(nodeConfig.RootDir, nodeConfig.DBPath)
			if err != nil {
				return err
			}

			var errs error

			// rollback ev-node main state
			// Note: With the unified store approach, the ev-node store is the single source of truth.
			// The store adapters (HeaderStoreAdapter/DataStoreAdapter) read from this store,
			// so rolling back the ev-node store automatically affects P2P sync operations.
			if err := evolveStore.Rollback(goCtx, height, !syncNode); err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to rollback ev-node state: %w", err))
			}

			// rollback execution store
			if err := executor.Rollback(goCtx, height); err != nil {
				errs = errors.Join(errs, fmt.Errorf("rollback failed: %w", err))
			}

			fmt.Printf("Rolled back ev-node state to height %d\n", height)
			if syncNode {
				fmt.Println("Restart the node with the `--evnode.clear_cache` flag")
			}

			return errs
		},
	}

	cmd.Flags().Uint64Var(&height, "height", 0, "rollback to a specific height")
	cmd.Flags().BoolVar(&syncNode, "sync-node", false, "sync node (no aggregator)")
	return cmd
}
