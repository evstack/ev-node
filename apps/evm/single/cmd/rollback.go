package cmd

import (
	"context"
	"errors"
	"fmt"

	ds "github.com/ipfs/go-datastore"
	kt "github.com/ipfs/go-datastore/keytransform"
	"github.com/spf13/cobra"

	goheaderstore "github.com/celestiaorg/go-header/store"
	"github.com/evstack/ev-node/node"
	rollcmd "github.com/evstack/ev-node/pkg/cmd"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
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
			rawEvolveDB, err := store.NewDefaultKVStore(nodeConfig.RootDir, nodeConfig.DBPath, "evm-single")
			if err != nil {
				return err
			}

			defer func() {
				if closeErr := rawEvolveDB.Close(); closeErr != nil {
					fmt.Printf("Warning: failed to close evolve database: %v\n", closeErr)
				}
			}()

			// prefixed evolve db
			evolveDB := kt.Wrap(rawEvolveDB, &kt.PrefixTransform{
				Prefix: ds.NewKey(node.EvPrefix),
			})

			evolveStore := store.New(evolveDB)
			if height == 0 {
				currentHeight, err := evolveStore.Height(goCtx)
				if err != nil {
					return err
				}

				height = currentHeight - 1
			}

			// rollback ev-node main state
			if err := evolveStore.Rollback(goCtx, height, !syncNode); err != nil {
				return fmt.Errorf("failed to rollback ev-node state: %w", err)
			}

			// rollback ev-node goheader state
			headerStore, err := goheaderstore.NewStore[*types.SignedHeader](
				evolveDB,
				goheaderstore.WithStorePrefix("headerSync"),
				goheaderstore.WithMetrics(),
			)
			if err != nil {
				return err
			}

			dataStore, err := goheaderstore.NewStore[*types.Data](
				evolveDB,
				goheaderstore.WithStorePrefix("dataSync"),
				goheaderstore.WithMetrics(),
			)
			if err != nil {
				return err
			}

			if err := headerStore.Start(goCtx); err != nil {
				return fmt.Errorf("failed to start header store: %w", err)
			}
			defer headerStore.Stop(goCtx)

			if err := dataStore.Start(goCtx); err != nil {
				return fmt.Errorf("failed to start data store: %w", err)
			}
			defer dataStore.Stop(goCtx)

			var errs error
			if err := headerStore.DeleteRange(goCtx, height+1, headerStore.Height()); err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to rollback header sync service state: %w", err))
			}

			if err := dataStore.DeleteRange(goCtx, height+1, dataStore.Height()); err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to rollback data sync service state: %w", err))
			}

			fmt.Printf("Rolled back ev-node state to height %d\n", height)
			if syncNode {
				fmt.Println("Restart the node with the `--clear-cache` flag")
			}

			return errs
		},
	}

	cmd.Flags().Uint64Var(&height, "height", 0, "rollback to a specific height")
	cmd.Flags().BoolVar(&syncNode, "sync-node", false, "sync node (no aggregator)")

	return cmd
}
