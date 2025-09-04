package cmd

import (
	"context"
	"fmt"
	"strconv"

	kvexecutor "github.com/evstack/ev-node/apps/testapp/kv"
	"github.com/evstack/ev-node/node"
	rollcmd "github.com/evstack/ev-node/pkg/cmd"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"

	goheaderstore "github.com/celestiaorg/go-header/store"
	ds "github.com/ipfs/go-datastore"
	kt "github.com/ipfs/go-datastore/keytransform"
	"github.com/spf13/cobra"
)

var RollbackCmd = &cobra.Command{
	Use:   "rollback <height>",
	Short: "Rollback the testapp node",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeConfig, err := rollcmd.ParseConfig(cmd)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		datastore, err := store.NewDefaultKVStore(nodeConfig.RootDir, nodeConfig.DBPath, "testapp")
		if err != nil {
			return err
		}

		// prefixed evolve db
		evolveDB := kt.Wrap(datastore, &kt.PrefixTransform{
			Prefix: ds.NewKey(node.EvPrefix),
		})

		storeWrapper := store.New(evolveDB)

		executor, err := kvexecutor.NewKVExecutor(nodeConfig.RootDir, nodeConfig.DBPath)
		if err != nil {
			return err
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

		cmd.Println("Starting rollback operation")
		currentHeight, err := storeWrapper.Height(ctx)
		if err != nil {
			return fmt.Errorf("failed to get current height: %w", err)
		}

		var targetHeight uint64 = currentHeight - 1
		if len(args) > 0 {
			targetHeight, err = strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse target height: %w", err)
			}
		}

		// rollback ev-node store
		if err := storeWrapper.Rollback(ctx, targetHeight); err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}

		// rollback sync services
		if err := headerStore.DeleteTo(ctx, targetHeight); err != nil {
			return fmt.Errorf("failed to rollback header sync service state: %w", err)
		}

		if err := dataStore.DeleteTo(ctx, targetHeight); err != nil {
			return fmt.Errorf("failed to rollback data sync service state: %w", err)
		}

		// rollback execution store
		if err := executor.Rollback(ctx, targetHeight); err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}

		cmd.Println("Rollback completed successfully")
		return nil
	},
}
