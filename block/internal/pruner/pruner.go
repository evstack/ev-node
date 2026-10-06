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

// Pruner periodically removes old blocks, state snapshots, and execution metadata.
type Pruner struct {
	store      store.Store
	execPruner coreexecutor.ExecPruner
	cfg        config.PruningConfig
	blockTime  time.Duration
	daEnabled  bool
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
	daAddress string,
) *Pruner {
	return &Pruner{
		store:      store,
		execPruner: execPruner,
		cfg:        cfg,
		blockTime:  blockTime,
		daEnabled:  daAddress != "", // DA is enabled if address is provided
		logger:     logger.With().Str("component", "pruner").Logger(),
	}
}

// Start begins the pruning loop.
func (p *Pruner) Start(ctx context.Context) error {
	if p.cancel != nil {
		return errors.New("pruner already started")
	}
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
	storeHeight, err := p.store.Height(p.ctx)
	if err != nil {
		return fmt.Errorf("failed to get store height for pruning: %w", err)
	}

	upperBound := storeHeight

	// If DA is enabled, only prune blocks that are DA included
	if p.daEnabled {
		var currentDAIncluded uint64
		currentDAIncludedBz, err := p.store.GetMetadata(p.ctx, store.DAIncludedHeightKey)
		if err == nil && len(currentDAIncludedBz) == 8 {
			currentDAIncluded = binary.LittleEndian.Uint64(currentDAIncludedBz)
		} else {
			p.logger.Debug().Msg("skipping pruning: DA is enabled but DA included height is not available yet")
			return nil
		}

		// Never prune blocks that are not DA included
		upperBound = min(storeHeight, currentDAIncluded)
	}

	if upperBound <= p.cfg.KeepRecent {
		// Not enough fully included blocks to prune
		return nil
	}

	targetHeight := upperBound - p.cfg.KeepRecent

	lastBlock, err := p.getLastPrunedBlockHeight(p.ctx)
	if err != nil {
		return fmt.Errorf("failed to get last pruned block height: %w", err)
	}
	lastState, err := p.getLastPrunedStateHeight(p.ctx)
	if err != nil {
		return fmt.Errorf("failed to get last pruned state height: %w", err)
	}

	// State may lag behind blocks after upgrading from a version that only
	// deleted blocks in all mode, or after a failed pruning operation.
	start := min(lastBlock, lastState)
	if start >= targetHeight {
		return nil
	}
	end := start + min(p.calculateBatchSize(), targetHeight-start)
	for start < end {
		if err := p.ctx.Err(); err != nil {
			return err
		}
		batchEnd := start + min(maxPruningBatchSize, end-start)
		if lastBlock < batchEnd {
			if err := p.store.PruneBlocks(p.ctx, batchEnd); err != nil {
				return fmt.Errorf("failed to prune blocks through height %d: %w", batchEnd, err)
			}
			lastBlock = batchEnd
		}
		if err := p.pruneState(lastState, batchEnd); err != nil {
			return err
		}
		lastState = max(lastState, batchEnd)
		start = batchEnd
	}

	p.logger.Debug().Uint64("pruned_up_to_height", end).Bool("da_enabled", p.daEnabled).Msg("pruned blocks and state snapshots")
	return nil
}

// maxPruningBatchSize bounds each datastore batch, rather than the total
// progress per interval, so fast chains can still catch up with old history.
const maxPruningBatchSize uint64 = 10000

// calculateBatchSize returns a per-interval work budget of four times the
// expected block production. Each datastore batch is bounded independently.
func (p *Pruner) calculateBatchSize() uint64 {
	if p.blockTime <= 0 || p.cfg.Interval.Duration <= 0 {
		return 1
	}
	blocksPerInterval := max(uint64(p.cfg.Interval.Duration/p.blockTime), 1)
	return min(blocksPerInterval, ^uint64(0)/4) * 4
}

// pruneMetadata prunes old state and execution metadata entries based on the configured retention depth.
// It does not prune old blocks, as those are handled by the pruning logic.
// Pruning old state does not lose history but limits the ability to recover (replay or rollback) to the last HEAD-N blocks, where N is the retention depth.
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

	target := height - p.cfg.KeepRecent
	if target <= lastPrunedState {
		return nil
	}

	end := lastPrunedState + min(p.calculateBatchSize(), target-lastPrunedState)
	for lastPrunedState < end {
		if err := p.ctx.Err(); err != nil {
			return err
		}
		batchEnd := lastPrunedState + min(maxPruningBatchSize, end-lastPrunedState)
		if err := p.pruneState(lastPrunedState, batchEnd); err != nil {
			return err
		}
		lastPrunedState = batchEnd
	}

	p.logger.Debug().Uint64("pruned_to", end).Msg("pruned state height metadata up to height")
	return nil
}

// pruneState retries execution pruning before advancing the state cursor, so
// failures remain recoverable even if block deletion has already committed.
func (p *Pruner) pruneState(lastPruned, end uint64) error {
	for h := lastPruned; h < end; {
		h++
		if err := p.ctx.Err(); err != nil {
			return err
		}
		if err := p.store.DeleteStateAtHeight(p.ctx, h); err != nil && !errors.Is(err, ds.ErrNotFound) {
			return fmt.Errorf("failed to prune state at height %d: %w", h, err)
		}
	}
	if p.execPruner != nil {
		if err := p.execPruner.PruneExec(p.ctx, end); err != nil && !errors.Is(err, ds.ErrNotFound) {
			return fmt.Errorf("failed to prune execution metadata through height %d: %w", end, err)
		}
	}
	if end > lastPruned {
		if err := p.setLastPrunedStateHeight(p.ctx, end); err != nil {
			return fmt.Errorf("failed to set last pruned state height: %w", err)
		}
	}
	return nil
}

// getLastPrunedBlockHeight returns the height of the last block that was pruned using PruneBlocks.
func (p *Pruner) getLastPrunedBlockHeight(ctx context.Context) (uint64, error) {
	lastPrunedBlockHeightBz, err := p.store.GetMetadata(ctx, store.LastPrunedBlockHeightKey)
	if errors.Is(err, ds.ErrNotFound) {
		// If not found, it means we haven't pruned any blocks yet, so we return 0.
		return 0, nil
	}

	if err != nil {
		return 0, fmt.Errorf("failed to get last pruned block height: %w", err)
	}
	if len(lastPrunedBlockHeightBz) != 8 {
		return 0, errors.New("invalid last pruned block height format")
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

	if err != nil {
		return 0, fmt.Errorf("failed to get last pruned state height: %w", err)
	}
	if len(lastPrunedStateHeightBz) != 8 {
		return 0, errors.New("invalid last pruned state height format")
	}

	lastPrunedStateHeight := binary.LittleEndian.Uint64(lastPrunedStateHeightBz)
	if lastPrunedStateHeight == 0 {
		return 0, fmt.Errorf("invalid last pruned state height")
	}

	return lastPrunedStateHeight, nil
}

func (p *Pruner) setLastPrunedStateHeight(ctx context.Context, height uint64) error {
	bz := make([]byte, 8)
	binary.LittleEndian.PutUint64(bz, height)
	return p.store.SetMetadata(ctx, store.LastPrunedStateHeightKey, bz)
}
