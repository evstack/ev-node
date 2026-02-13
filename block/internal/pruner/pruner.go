package pruner

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"

	coreexecutor "github.com/evstack/ev-node/core/execution"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/store"
)

// Pruner periodically removes old state and execution metadata entries.
type Pruner struct {
	store      store.Store
	execPruner coreexecutor.ExecPruner
	cfg        config.PruningConfig
	blockTime  time.Duration
	logger     zerolog.Logger

	// Lifecycle
	ctx    context.Context
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// New creates a new Pruner instance.
func New(
	logger zerolog.Logger,
	store store.Store,
	execPruner coreexecutor.ExecPruner,
	cfg config.PruningConfig,
	blockTime time.Duration,
) *Pruner {
	return &Pruner{
		store:      store,
		execPruner: execPruner,
		cfg:        cfg,
		blockTime:  blockTime,
		logger:     logger.With().Str("component", "pruner").Logger(),
	}
}

// Start begins the pruning loop.
func (p *Pruner) Start(ctx context.Context) error {
	if !p.cfg.IsPruningEnabled() {
		p.logger.Info().Msg("pruning is disabled, not starting pruner")
		return nil
	}

	p.ctx, p.cancel = context.WithCancel(ctx)

	// Start pruner loop
	p.wg.Go(p.pruneLoop)

	p.logger.Info().Msg("pruner started")
	return nil
}

// Stop stops the pruning loop.
func (p *Pruner) Stop() error {
	if !p.cfg.IsPruningEnabled() {
		return nil
	}

	if p.cancel != nil {
		p.cancel()
	}

	p.wg.Wait()

	p.logger.Info().Msg("pruner stopped")
	return nil
}

func (p *Pruner) pruneLoop() {
	ticker := time.NewTicker(p.cfg.Interval.Duration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			switch p.cfg.Mode {
			case config.PruningModeMetadata:
				if err := p.pruneMetadata(); err != nil {
					p.logger.Error().Err(err).Msg("failed to prune blocks metadata")
				}
			case config.PruningModeAll:
				if err := p.pruneBlocks(); err != nil {
					p.logger.Error().Err(err).Msg("failed to prune blocks")
				}
			}
		case <-p.ctx.Done():
			return
		}
	}
}

// pruneBlocks prunes blocks and their metadatas.
func (p *Pruner) pruneBlocks() error {
	var currentDAIncluded uint64
	currentDAIncludedBz, err := p.store.GetMetadata(p.ctx, store.DAIncludedHeightKey)
	if err == nil && len(currentDAIncludedBz) == 8 {
		currentDAIncluded = binary.LittleEndian.Uint64(currentDAIncludedBz)
	} else {
		// if we cannot get the current DA height, we cannot safely prune, so we skip pruning until we can get it.
		return nil
	}

	storeHeight, err := p.store.Height(p.ctx)
	if err != nil {
		return fmt.Errorf("failed to get store height for pruning: %w", err)
	}

	// Never prune blocks that are not DA included
	upperBound := min(storeHeight, currentDAIncluded)
	if upperBound <= p.cfg.KeepRecent {
		// Not enough fully included blocks to prune
		return nil
	}

	targetHeight := upperBound - p.cfg.KeepRecent

	// Get the last pruned height to determine batch size
	lastPruned, err := p.getLastPrunedBlockHeight(p.ctx)
	if err != nil {
		return fmt.Errorf("failed to get last pruned block height: %w", err)
	}

	// prune in batches to avoid overwhelming the system
	batchSize := p.calculateBatchSize()
	batchEnd := min(lastPruned+batchSize, targetHeight)

	if err := p.store.PruneBlocks(p.ctx, batchEnd); err != nil {
		p.logger.Error().Err(err).Uint64("target_height", batchEnd).Msg("failed to prune old block data")
		return err
	}

	if p.execPruner != nil {
		if err := p.execPruner.PruneExec(p.ctx, batchEnd); err != nil && !errors.Is(err, ds.ErrNotFound) {
			return err
		}
	}

	p.logger.Debug().Uint64("pruned_up_to_height", batchEnd).Msg("pruned blocks up to height")
	return nil
}

// calculateBatchSize returns the appropriate batch size for pruning operations.
// The batch size is based on the pruning interval and block time to ensure reasonable progress
// without overwhelming the node.
func (p *Pruner) calculateBatchSize() uint64 {
	// Calculate batch size based on pruning interval and block time.
	// We use 4x the blocks produced during one pruning interval as the batch size.
	// This ensures we catch up at 3x the block production rate when there's a backlog.
	// Example: With 100ms blocks and 15min interval: 15*60/0.1 = 9000 blocks/interval
	//   - Batch size: 36,000 blocks (prunes 36k, chain grows 9k = net 27k catch-up per interval)
	blocksPerInterval := uint64(p.cfg.Interval.Duration / p.blockTime)

	// Ensure reasonable minimum
	return min(blocksPerInterval*4, 10000)
}

// pruneMetadata prunes old state and execution metadata entries based on the configured retention depth.
// It does not prunes old blocks, as those are handled by the pruning logic.
// Pruning old state does not lose history but limit the ability to recover (replay or rollback) to the last HEAD-N blocks, where N is the retention depth.
func (p *Pruner) pruneMetadata() error {
	height, err := p.store.Height(p.ctx)
	if err != nil {
		return err
	}

	if height <= p.cfg.KeepRecent {
		return nil
	}

	lastPrunedState, err := p.getLastPrunedStateHeight(p.ctx)
	if err != nil {
		return fmt.Errorf("failed to get last pruned state height: %w", err)
	}

	if lastPrunedBlock, err := p.getLastPrunedBlockHeight(p.ctx); err == nil && lastPrunedBlock > lastPrunedState {
		lastPrunedState = lastPrunedBlock
	}

	target := height - p.cfg.KeepRecent
	if target <= lastPrunedState {
		return nil
	}

	batchSize := p.calculateBatchSize()

	// prune in batches to avoid overwhelming the system
	batchEnd := min(lastPrunedState+batchSize, target)

	for h := lastPrunedState + 1; h <= batchEnd; h++ {
		if err := p.store.DeleteStateAtHeight(p.ctx, h); err != nil && !errors.Is(err, ds.ErrNotFound) {
			return err
		}
	}

	if p.execPruner != nil {
		if err := p.execPruner.PruneExec(p.ctx, batchEnd); err != nil && !errors.Is(err, ds.ErrNotFound) {
			return err
		}
	}

	if err := p.setLastPrunedStateHeight(p.ctx, batchEnd); err != nil {
		return fmt.Errorf("failed to set last pruned block height: %w", err)
	}

	p.logger.Debug().Uint64("pruned_to", batchEnd).Msg("pruned state height metadata up to height")
	return nil
}

// getLastPrunedBlockHeight returns the height of the last block that was pruned using PruneBlocks.
func (p *Pruner) getLastPrunedBlockHeight(ctx context.Context) (uint64, error) {
	lastPrunedBlockHeightBz, err := p.store.GetMetadata(ctx, store.LastPrunedBlockHeightKey)
	if errors.Is(err, ds.ErrNotFound) {
		// If not found, it means we haven't pruned any blocks yet, so we return 0.
		return 0, nil
	}

	if err != nil || len(lastPrunedBlockHeightBz) != 8 {
		return 0, fmt.Errorf("failed to get last pruned block height or invalid format: %w", err)
	}

	lastPrunedBlockHeight := binary.LittleEndian.Uint64(lastPrunedBlockHeightBz)
	if lastPrunedBlockHeight == 0 {
		return 0, fmt.Errorf("invalid last pruned block height")
	}

	return lastPrunedBlockHeight, nil
}

// getLastPrunedStateHeight returns the height of the last state that was pruned using DeleteStateAtHeight.
func (p *Pruner) getLastPrunedStateHeight(ctx context.Context) (uint64, error) {
	lastPrunedStateHeightBz, err := p.store.GetMetadata(ctx, store.LastPrunedStateHeightKey)
	if errors.Is(err, ds.ErrNotFound) {
		// If not found, it means we haven't pruned any state yet, so we return 0.
		return 0, nil
	}

	if err != nil || len(lastPrunedStateHeightBz) != 8 {
		return 0, fmt.Errorf("failed to get last pruned block height or invalid format: %w", err)
	}

	lastPrunedStateHeight := binary.LittleEndian.Uint64(lastPrunedStateHeightBz)
	if lastPrunedStateHeight == 0 {
		return 0, fmt.Errorf("invalid last pruned block height")
	}

	return lastPrunedStateHeight, nil
}

func (p *Pruner) setLastPrunedStateHeight(ctx context.Context, height uint64) error {
	bz := make([]byte, 8)
	binary.LittleEndian.PutUint64(bz, height)
	return p.store.SetMetadata(ctx, store.LastPrunedStateHeightKey, bz)
}
