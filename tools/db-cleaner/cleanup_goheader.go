package main

import (
	"context"
	"fmt"
	"os"

	ds "github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	"github.com/spf13/cobra"

	rollcmd "github.com/evstack/ev-node/pkg/cmd"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/store"
)

const evmDbName = "evm-single"

const (
	// go-header store prefixes used prior to the unified store migration
	headerSyncPrefix = "/headerSync"
	dataSyncPrefix   = "/dataSync"
)

func main() {
	// Initiate the root command
	rootCmd := &cobra.Command{
		Use: "db-cleaner",
	}

	config.AddGlobalFlags(rootCmd, "evm")

	rootCmd.AddCommand(
		NewCleanupGoHeaderCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		// Print to stderr and exit with error
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// NewCleanupGoHeaderCmd creates a command to delete the legacy go-header store data.
// This data is no longer needed after the migration to the unified store approach
// where HeaderStoreAdapter and DataStoreAdapter read directly from the ev-node store.
func NewCleanupGoHeaderCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "clean-evm",
		Short: "Delete legacy go-header store data from disk",
		Long: `Delete the legacy go-header store data (headerSync and dataSync prefixes) from the database.

This command removes data that was previously duplicated by the go-header library
for P2P sync operations. After the migration to the unified store approach,
this data is no longer needed as the HeaderStoreAdapter and DataStoreAdapter
now read directly from the ev-node store.

WARNING: Make sure the node is stopped before running this command.
This operation is irreversible.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeConfig, err := rollcmd.ParseConfig(cmd)
			if err != nil {
				return err
			}

			goCtx := cmd.Context()
			if goCtx == nil {
				goCtx = context.Background()
			}

			// Open the database
			rawDB, err := store.NewDefaultKVStore(nodeConfig.RootDir, nodeConfig.DBPath, evmDbName)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				if closeErr := rawDB.Close(); closeErr != nil {
					cmd.Printf("Warning: failed to close database: %v\n", closeErr)
				}
			}()

			// Delete headerSync prefix
			headerCount, err := deletePrefix(goCtx, rawDB, headerSyncPrefix, dryRun)
			if err != nil {
				return fmt.Errorf("failed to delete headerSync data: %w", err)
			}

			// Delete dataSync prefix
			dataCount, err := deletePrefix(goCtx, rawDB, dataSyncPrefix, dryRun)
			if err != nil {
				return fmt.Errorf("failed to delete dataSync data: %w", err)
			}

			totalCount := headerCount + dataCount

			if dryRun {
				cmd.Printf("Dry run: would delete %d keys (%d headerSync, %d dataSync)\n",
					totalCount, headerCount, dataCount)
			} else {
				if totalCount == 0 {
					cmd.Println("No legacy go-header store data found to delete.")
				} else {
					cmd.Printf("Successfully deleted %d keys (%d headerSync, %d dataSync)\n",
						totalCount, headerCount, dataCount)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be deleted without actually deleting")

	return cmd
}

// deletePrefix deletes all keys with the given prefix from the datastore.
// Returns the number of keys deleted.
func deletePrefix(ctx context.Context, db ds.Batching, prefix string, dryRun bool) (int, error) {
	results, err := db.Query(ctx, dsq.Query{
		Prefix:   prefix,
		KeysOnly: true,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to query keys with prefix %s: %w", prefix, err)
	}
	defer results.Close()

	count := 0
	batch, err := db.Batch(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to create batch: %w", err)
	}

	for result := range results.Next() {
		if result.Error != nil {
			return count, fmt.Errorf("error iterating results: %w", result.Error)
		}

		if !dryRun {
			if err := batch.Delete(ctx, ds.NewKey(result.Key)); err != nil {
				return count, fmt.Errorf("failed to delete key %s: %w", result.Key, err)
			}
		}
		count++
	}

	if !dryRun && count > 0 {
		if err := batch.Commit(ctx); err != nil {
			return count, fmt.Errorf("failed to commit batch delete: %w", err)
		}
	}

	return count, nil
}
