package cmd

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"

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
func NewForceRollbackCmd() *cobra.Command {
	var (
		syncNode bool
	)

	cmd := &cobra.Command{
		Use:   "force-rollback [from] [to]",
		Args:  cobra.ExactArgs(2),
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

			fromStr, toStr := args[0], args[1]
			from, err := strconv.ParseUint(fromStr, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse from height: %w", err)
			}
			to, err := strconv.ParseUint(toStr, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse to height: %w", err)
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

			// rollback ev-node main state
			if err := Rollback(evolveStore, evolveDB, goCtx, from, to, !syncNode); err != nil {
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
			if err := headerStore.DeleteRange(goCtx, to+1, headerStore.Height()); err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to rollback header sync service state: %w", err))
			}

			if err := dataStore.DeleteRange(goCtx, to+1, dataStore.Height()); err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to rollback data sync service state: %w", err))
			}

			fmt.Printf("Rolled back ev-node state to height %d\n", to)
			if syncNode {
				fmt.Println("Restart the node with the `--clear-cache` flag")
			}

			return errs
		},
	}

	cmd.Flags().BoolVar(&syncNode, "sync-node", false, "sync node (no aggregator)")

	return cmd
}

func Rollback(s store.Store, db ds.Batching, ctx context.Context, from, to uint64, aggregator bool) error {
	batch, err := db.Batch(ctx)
	if err != nil {
		return fmt.Errorf("failed to create a new batch: %w", err)
	}

	daIncludedHeightBz, err := s.GetMetadata(ctx, store.DAIncludedHeightKey)
	if err != nil && !errors.Is(err, ds.ErrNotFound) {
		return fmt.Errorf("failed to get DA included height: %w", err)
	} else if len(daIncludedHeightBz) == 8 { // valid height stored, so able to check
		daIncludedHeight := binary.LittleEndian.Uint64(daIncludedHeightBz)
		if daIncludedHeight > to {
			// an aggregator must not rollback a finalized height, DA is the source of truth
			if aggregator {
				return fmt.Errorf("DA included height is greater than the rollback height: cannot rollback a finalized height")
			} else { // in case of syncing issues, rollback the included height is OK.
				bz := make([]byte, 8)
				binary.LittleEndian.PutUint64(bz, to)
				if err := batch.Put(ctx, ds.NewKey(getMetaKey(DAIncludedHeightKey)), bz); err != nil {
					return fmt.Errorf("failed to update DA included height: %w", err)
				}
			}
		}
	}

	for from > to {
		header, err := s.GetHeader(ctx, from)
		if err != nil {
			return fmt.Errorf("failed to get header at height %d: %w", from, err)
		}

		if err := batch.Delete(ctx, ds.NewKey(getHeaderKey(from))); err != nil {
			return fmt.Errorf("failed to delete header blob in batch: %w", err)
		}

		if err := batch.Delete(ctx, ds.NewKey(getDataKey(from))); err != nil {
			return fmt.Errorf("failed to delete data blob in batch: %w", err)
		}

		if err := batch.Delete(ctx, ds.NewKey(getSignatureKey(from))); err != nil {
			return fmt.Errorf("failed to delete signature of block blob in batch: %w", err)
		}

		hash := header.Hash()
		if err := batch.Delete(ctx, ds.NewKey(getIndexKey(hash))); err != nil {
			return fmt.Errorf("failed to delete index key in batch: %w", err)
		}

		if err := batch.Delete(ctx, ds.NewKey(getStateAtHeightKey(from))); err != nil {
			return fmt.Errorf("failed to delete state in batch: %w", err)
		}

		from--
	}

	// set height -- using set height checks the current height
	// so we cannot use that
	heightBytes := encodeHeight(to)
	if err := batch.Put(ctx, ds.NewKey(getHeightKey()), heightBytes); err != nil {
		return fmt.Errorf("failed to set height: %w", err)
	}

	if err := batch.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}

	return nil
}

const (
	// GenesisDAHeightKey is the key used for persisting the first DA included height in store.
	// It avoids to walk over the HeightToDAHeightKey to find the first DA included height.
	GenesisDAHeightKey = "gdh"

	// HeightToDAHeightKey is the key prefix used for persisting the mapping from a Evolve height
	// to the DA height where the block's header/data was included.
	// Full keys are like: rhb/<evolve_height>/h and rhb/<evolve_height>/d
	HeightToDAHeightKey = "rhb"

	// DAIncludedHeightKey is the key used for persisting the da included height in store.
	DAIncludedHeightKey = "d"

	// LastBatchDataKey is the key used for persisting the last batch data in store.
	LastBatchDataKey = "l"

	// LastSubmittedHeaderHeightKey is the key used for persisting the last submitted header height in store.
	LastSubmittedHeaderHeightKey = "last-submitted-header-height"

	headerPrefix    = "h"
	dataPrefix      = "d"
	signaturePrefix = "c"
	statePrefix     = "s"
	metaPrefix      = "m"
	indexPrefix     = "i"
	heightPrefix    = "t"
)

func getHeaderKey(height uint64) string {
	return store.GenerateKey([]string{headerPrefix, strconv.FormatUint(height, 10)})
}

func getDataKey(height uint64) string {
	return store.GenerateKey([]string{dataPrefix, strconv.FormatUint(height, 10)})
}

func getSignatureKey(height uint64) string {
	return store.GenerateKey([]string{signaturePrefix, strconv.FormatUint(height, 10)})
}

func getStateAtHeightKey(height uint64) string {
	return store.GenerateKey([]string{statePrefix, strconv.FormatUint(height, 10)})
}

func getMetaKey(key string) string {
	return store.GenerateKey([]string{metaPrefix, key})
}

func getIndexKey(hash types.Hash) string {
	return store.GenerateKey([]string{indexPrefix, hash.String()})
}

func getHeightKey() string {
	return store.GenerateKey([]string{heightPrefix})
}

const heightLength = 8

func encodeHeight(height uint64) []byte {
	heightBytes := make([]byte, heightLength)
	binary.LittleEndian.PutUint64(heightBytes, height)
	return heightBytes
}
