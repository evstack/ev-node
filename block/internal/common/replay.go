package common

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"

	"github.com/rs/zerolog"

	coreexecutor "github.com/evstack/ev-node/core/execution"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

// Replayer handles synchronization of the execution layer with ev-node's state.
// It replays blocks from the store to bring the execution layer up to date.
type Replayer struct {
	store   store.Store
	exec    coreexecutor.Executor
	genesis genesis.Genesis
	logger  zerolog.Logger
}

// NewReplayer creates a new execution layer replayer.
func NewReplayer(
	store store.Store,
	exec coreexecutor.Executor,
	genesis genesis.Genesis,
	logger zerolog.Logger,
) *Replayer {
	return &Replayer{
		store:   store,
		exec:    exec,
		genesis: genesis,
		logger:  logger.With().Str("component", "execution_replayer").Logger(),
	}
}

// SyncToHeight checks if the execution layer is behind ev-node and syncs it to the target height.
// This is useful for crash recovery scenarios where ev-node is ahead of the execution layer.
//
// Returns:
// - error if sync fails
func (s *Replayer) SyncToHeight(ctx context.Context, targetHeight uint64) error {
	// Check if the executor implements HeightProvider
	execHeightProvider, ok := s.exec.(coreexecutor.HeightProvider)
	if !ok {
		s.logger.Debug().Msg("executor does not implement HeightProvider, skipping sync")
		return nil
	}

	// Skip sync check if we're at genesis
	if targetHeight < s.genesis.InitialHeight {
		s.logger.Debug().Msg("at genesis height, skipping execution layer sync check")
		return nil
	}

	execHeight, err := execHeightProvider.GetLatestHeight(ctx)
	if err != nil {
		return fmt.Errorf("failed to get execution layer height: %w", err)
	}

	s.logger.Info().
		Uint64("target_height", targetHeight).
		Uint64("exec_layer_height", execHeight).
		Msg("execution layer height check")

	// If execution layer is ahead, we cannot proceed safely as this indicates state divergence.
	// The execution layer must be rolled back before the node can continue.
	if execHeight > targetHeight {
		return fmt.Errorf("execution layer height (%d) ahead of target height (%d): manually rollback execution layer to height %d",
			execHeight, targetHeight, targetHeight)
	}

	// If execution layer is behind, sync the missing blocks
	if execHeight < targetHeight {
		s.logger.Info().
			Uint64("target_height", targetHeight).
			Uint64("exec_layer_height", execHeight).
			Uint64("blocks_to_sync", targetHeight-execHeight).
			Msg("execution layer is behind, syncing blocks")

		// Sync blocks from execHeight+1 to targetHeight
		for height := execHeight + 1; height <= targetHeight; height++ {
			if err := s.replayBlock(ctx, height); err != nil {
				return fmt.Errorf("failed to replay block %d to execution layer: %w", height, err)
			}
		}

		s.logger.Info().
			Uint64("synced_blocks", targetHeight-execHeight).
			Msg("successfully synced execution layer")
	} else {
		s.logger.Info().Msg("execution layer is in sync")
	}

	return nil
}

// replayBlock replays a specific block from the store to the execution layer.
//
// Validation assumptions:
// - Blocks in the store have already been fully validated (signatures, timestamps, etc.)
// - We only verify the AppHash matches to detect state divergence
// - We skip re-validating signatures and consensus rules since this is a replay
// - This is safe because we're re-executing transactions against a known-good state
func (s *Replayer) replayBlock(ctx context.Context, height uint64) error {
	s.logger.Info().Uint64("height", height).Msg("replaying block to execution layer")

	// Get the block from store
	header, data, err := s.store.GetBlockData(ctx, height)
	if err != nil {
		return fmt.Errorf("failed to get block data from store: %w", err)
	}

	// Get the previous state
	var prevState types.State
	if height == s.genesis.InitialHeight {
		// For the first block, use genesis state
		prevState = types.State{
			ChainID:         s.genesis.ChainID,
			InitialHeight:   s.genesis.InitialHeight,
			LastBlockHeight: s.genesis.InitialHeight - 1,
			LastBlockTime:   s.genesis.StartTime,
			AppHash:         header.AppHash, // This will be updated by InitChain
		}
	} else {
		// Get previous state from store
		prevState, err = s.store.GetState(ctx)
		if err != nil {
			return fmt.Errorf("failed to get previous state: %w", err)
		}
		// We need the state at height-1, so load that block's app hash
		prevHeader, _, err := s.store.GetBlockData(ctx, height-1)
		if err != nil {
			return fmt.Errorf("failed to get previous block header: %w", err)
		}
		prevState.AppHash = prevHeader.AppHash
		prevState.LastBlockHeight = height - 1
	}

	// Prepare transactions
	rawTxs := make([][]byte, len(data.Txs))
	for i, tx := range data.Txs {
		rawTxs[i] = []byte(tx)
	}

	// Execute transactions on the execution layer
	s.logger.Debug().
		Uint64("height", height).
		Int("tx_count", len(rawTxs)).
		Msg("executing transactions on execution layer")

	newAppHash, err := s.exec.ExecuteTxs(ctx, rawTxs, height, header.Time(), prevState.AppHash)
	if err != nil {
		return fmt.Errorf("failed to execute transactions: %w", err)
	}

	// Verify the app hash matches
	if !bytes.Equal(newAppHash, header.AppHash) {
		err := fmt.Errorf("app hash mismatch: expected %s got %s",
			hex.EncodeToString(header.AppHash),
			hex.EncodeToString(newAppHash),
		)
		s.logger.Error().
			Str("expected", hex.EncodeToString(header.AppHash)).
			Str("got", hex.EncodeToString(newAppHash)).
			Uint64("height", height).
			Err(err).
			Msg("app hash mismatch during replay")
		return err
	}

	s.logger.Info().
		Uint64("height", height).
		Msg("successfully replayed block to execution layer")

	return nil
}
