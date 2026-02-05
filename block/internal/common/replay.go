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

	// If execution layer is ahead, attempt automatic rollback or fail
	if execHeight > targetHeight {
		// Attempt automatic rollback if supported
		if rollbackable, ok := s.exec.(coreexecutor.Rollbackable); ok {
			s.logger.Warn().
				Uint64("exec_height", execHeight).
				Uint64("target_height", targetHeight).
				Msg("execution layer ahead, attempting automatic rollback")
			if err := rollbackable.Rollback(ctx, targetHeight); err != nil {
				return fmt.Errorf("failed to rollback execution layer from %d to %d: %w",
					execHeight, targetHeight, err)
			}
			s.logger.Info().Uint64("target_height", targetHeight).Msg("execution layer rollback successful")
			return nil
		}
		// Fallback to manual intervention if rollback not supported
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

		// Verify the target block exists in the store before attempting sync
		// This catches the case where Raft is ahead but blocks haven't been applied to the store yet
		if _, _, err := s.store.GetBlockData(ctx, targetHeight); err != nil {
			return fmt.Errorf("cannot sync to height %d: block not found in store (store may be behind Raft consensus): %w", targetHeight, err)
		}

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

	// DEBUG: Log full header details for double-signing investigation
	s.logger.Debug().
		Uint64("height", height).
		Str("chain_id", header.ChainID()).
		Int64("timestamp_unix", header.Time().Unix()).
		Str("timestamp", header.Time().String()).
		Str("app_hash", hex.EncodeToString(header.AppHash)).
		Str("data_hash", hex.EncodeToString(header.DataHash)).
		Str("last_header_hash", hex.EncodeToString(header.LastHeaderHash)).
		Str("proposer_address", hex.EncodeToString(header.ProposerAddress)).
		Int("tx_count", len(data.Txs)).
		Msg("replayBlock: loaded block from store")

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
		prevState, err = s.store.GetStateAtHeight(ctx, height-1)
		if err != nil {
			return fmt.Errorf("failed to get previous state: %w", err)
		}
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

	expectedState, stateErr := s.store.GetStateAtHeight(ctx, height)
	if stateErr != nil {
		s.logger.Warn().
			Uint64("height", height).
			Err(stateErr).
			Msg("replayBlock: failed to load expected state, skipping app hash verification")
	} else {
		// DEBUG: Log comparison of expected vs actual app hash
		s.logger.Debug().
			Uint64("height", height).
			Str("expected_app_hash", hex.EncodeToString(expectedState.AppHash)).
			Str("actual_app_hash", hex.EncodeToString(newAppHash)).
			Bool("hashes_match", bytes.Equal(newAppHash, expectedState.AppHash)).
			Msg("replayBlock: ExecuteTxs completed")

		// Verify the app hash matches the stored state at this height
		if !bytes.Equal(newAppHash, expectedState.AppHash) {
			err := fmt.Errorf("app hash mismatch: expected %s got %s",
				hex.EncodeToString(expectedState.AppHash),
				hex.EncodeToString(newAppHash),
			)
			s.logger.Error().
				Str("expected", hex.EncodeToString(expectedState.AppHash)).
				Str("got", hex.EncodeToString(newAppHash)).
				Uint64("height", height).
				Err(err).
				Msg("app hash mismatch during replay")
			return err
		}
	}

	// Calculate new state
	newState, err := prevState.NextState(header.Header, newAppHash)
	if err != nil {
		return fmt.Errorf("calculate next state: %w", err)
	}

	// Persist the new state
	batch, err := s.store.NewBatch(ctx)
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	if err := batch.UpdateState(newState); err != nil {
		return fmt.Errorf("update state in batch: %w", err)
	}
	// We don't need to SetHeight here because the store already has the block height (we fetched the block from it)
	// But we MUST persist the State blob so that GetState() can find it on restart.

	if err := batch.Commit(); err != nil {
		return fmt.Errorf("commit batch with new state: %w", err)
	}

	s.logger.Info().
		Uint64("height", height).
		Msg("successfully replayed block to execution layer")

	return nil
}
