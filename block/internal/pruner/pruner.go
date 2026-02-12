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
	if !p.cfg.IsPruningEnabled() {
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

	lastPrunedState, err := p.store.GetLastPrunedStateHeight(p.ctx)
	if err != nil {
		return nil
	}

	if lastPrunedBlock, err := p.store.GetLastPrunedBlockHeight(p.ctx); err == nil && lastPrunedBlock > lastPrunedState {
		lastPrunedState = lastPrunedBlock
	}

	target := height - p.cfg.KeepRecent
	if target <= lastPrunedState {
		return nil
	}

	// maxPruneBatch limits how many heights we prune per cycle to bound work.
	maxPruneBatch := (target - lastPrunedState) / 20

	start := lastPrunedState + 1
	end := target
	if end-start+1 > maxPruneBatch {
		end = start + maxPruneBatch - 1
	}

	for h := start; h <= end; h++ {
		if err := p.store.DeleteStateAtHeight(p.ctx, h); err != nil && !errors.Is(err, ds.ErrNotFound) {
			return err
		}

		if p.execPruner != nil {
			if err := p.execPruner.PruneExec(p.ctx, h); err != nil && !errors.Is(err, ds.ErrNotFound) {
				return err
			}
		}
	}

	return nil
}
