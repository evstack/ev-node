package cmd

import (
	"context"
	"fmt"
	"strconv"

	kvexecutor "github.com/evstack/ev-node/apps/testapp/kv"
	rollcmd "github.com/evstack/ev-node/pkg/cmd"
	"github.com/evstack/ev-node/pkg/store"
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
		storeWrapper := store.New(datastore)

		executor, err := kvexecutor.NewKVExecutor(nodeConfig.RootDir, nodeConfig.DBPath)
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

		// rollback execution store
		if err := executor.Rollback(ctx, targetHeight); err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}

		cmd.Println("Rollback completed successfully")
		return nil
	},
}
