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

const defaultPruneInterval = 15 * time.Minute

// Pruner periodically removes old state and execution metadata entries.
type Pruner struct {
	store      store.Store
	execPruner coreexecutor.ExecPruner
	cfg        config.NodeConfig
	logger     zerolog.Logger
	lastPruned uint64

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
	cfg config.NodeConfig,
) *Pruner {
	return &Pruner{
		store:      store,
		execPruner: execPruner,
		cfg:        cfg,
		logger:     logger.With().Str("component", "prune").Logger(),
	}
}

// Start begins the pruning loop.
func (p *Pruner) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	// Start pruner loop
	p.wg.Go(p.pruneLoop)

	p.logger.Info().Msg("pruner started")
	return nil
}

// Stop stops the pruning loop.
func (p *Pruner) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()

	p.logger.Info().Msg("pruner stopped")
	return nil
}

func (p *Pruner) pruneLoop() {
	ticker := time.NewTicker(defaultPruneInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := p.pruneRecoveryHistory(p.ctx, p.cfg.RecoveryHistoryDepth); err != nil {
				p.logger.Error().Err(err).Msg("failed to prune recovery history")
			}

			if err := p.pruneBlocks(); err != nil {
				p.logger.Error().Err(err).Msg("failed to prune old blocks")
			}

			// TODO: add pruning of old blocks // https://github.com/evstack/ev-node/pull/2984
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *Pruner) pruneBlocks() error {
	if !p.cfg.PruningEnabled || p.cfg.PruningKeepRecent == 0 || p.cfg.PruningInterval == 0 {
		return nil
	}

	var currentDAIncluded uint64
	currentDAIncludedBz, err := p.store.GetMetadata(p.ctx, store.DAIncludedHeightKey)
	if err == nil && len(currentDAIncludedBz) == 8 {
		currentDAIncluded = binary.LittleEndian.Uint64(currentDAIncludedBz)
	} else {
		// if we cannot get the current DA height, we cannot safely prune, so we skip pruning until we can get it.
		return nil
	}

	var lastPruned uint64
	if bz, err := p.store.GetMetadata(p.ctx, store.LastPrunedBlockHeightKey); err == nil && len(bz) == 8 {
		lastPruned = binary.LittleEndian.Uint64(bz)
	}

	storeHeight, err := p.store.Height(p.ctx)
	if err != nil {
		return fmt.Errorf("failed to get store height for pruning: %w", err)
	}
	if storeHeight <= lastPruned+p.cfg.PruningInterval {
		return nil
	}

	// Never prune blocks that are not DA included
	upperBound := min(storeHeight, currentDAIncluded)
	if upperBound <= p.cfg.PruningKeepRecent {
		// Not enough fully included blocks to prune
		return nil
	}

	targetHeight := upperBound - p.cfg.PruningKeepRecent

	if err := p.store.PruneBlocks(p.ctx, targetHeight); err != nil {
		p.logger.Error().Err(err).Uint64("target_height", targetHeight).Msg("failed to prune old block data")
	}

	if p.execPruner != nil {
		if err := p.execPruner.PruneExec(p.ctx, targetHeight); err != nil && !errors.Is(err, ds.ErrNotFound) {
			return err
		}
	}

	return nil
}

// pruneRecoveryHistory prunes old state and execution metadata entries based on the configured retention depth.
// It does not prunes old blocks, as those are handled by the pruning logic.
// Pruning old state does not lose history but limit the ability to recover (replay or rollback) to the last HEAD-N blocks, where N is the retention depth.
func (p *Pruner) pruneRecoveryHistory(ctx context.Context, retention uint64) error {
	if p.cfg.RecoveryHistoryDepth == 0 {
		return nil
	}

	height, err := p.store.Height(ctx)
	if err != nil {
		return err
	}

	if height <= retention {
		return nil
	}

	target := height - retention
	if target <= p.lastPruned {
		return nil
	}

	// maxPruneBatch limits how many heights we prune per cycle to bound work.
	// it is callibrated to prune the last N blocks in one cycle, where N is the number of blocks produced in the defaultPruneInterval.
	blockTime := p.cfg.BlockTime.Duration
	if blockTime == 0 {
		blockTime = 1
	}

	maxPruneBatch := max(uint64(defaultPruneInterval/blockTime), (target-p.lastPruned)/5)

	start := p.lastPruned + 1
	end := target
	if end-start+1 > maxPruneBatch {
		end = start + maxPruneBatch - 1
	}

	for h := start; h <= end; h++ {
		if err := p.store.DeleteStateAtHeight(ctx, h); err != nil && !errors.Is(err, ds.ErrNotFound) {
			return err
		}

		if p.execPruner != nil {
			if err := p.execPruner.PruneExec(ctx, h); err != nil && !errors.Is(err, ds.ErrNotFound) {
				return err
			}
		}
	}

	p.lastPruned = end
	return nil
}
