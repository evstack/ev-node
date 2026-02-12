package pruner

import (
	"context"
	"errors"
	"sync"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/store"
)

const (
	defaultPruneInterval = 15 * time.Minute
	// maxPruneBatch limits how many heights we prune per cycle to bound work.
	maxPruneBatch = uint64(1000)
)

// ExecPruner removes execution metadata at a given height.
type ExecPruner interface {
	PruneExec(ctx context.Context, height uint64) error
}

type stateDeleter interface {
	DeleteStateAtHeight(ctx context.Context, height uint64) error
}

// Pruner periodically removes old state and execution metadata entries.
type Pruner struct {
	store        store.Store
	stateDeleter stateDeleter
	execPruner   ExecPruner
	cfg          config.NodeConfig
	logger       zerolog.Logger
	lastPruned   uint64

	// Lifecycle
	ctx    context.Context
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// New creates a new Pruner instance.
func New(
	logger zerolog.Logger,
	store store.Store,
	execPruner ExecPruner,
	cfg config.NodeConfig,
) *Pruner {
	var deleter stateDeleter
	if store != nil {
		if sd, ok := store.(stateDeleter); ok {
			deleter = sd
		}
	}

	return &Pruner{
		store:        store,
		stateDeleter: deleter,
		execPruner:   execPruner,
		cfg:          cfg,
		logger:       logger.With().Str("component", "prune").Logger(),
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

			// TODO: add pruning of old blocks // https://github.com/evstack/ev-node/pull/2984
		case <-p.ctx.Done():
			return
		}
	}
}

// pruneRecoveryHistory prunes old state and execution metadata entries based on the configured retention depth.
// It does not prunes old blocks, as those are handled by the pruning logic.
// Pruning old state does not lose history but limit the ability to recover (replay or rollback) to the last HEAD-N blocks, where N is the retention depth.
func (p *Pruner) pruneRecoveryHistory(ctx context.Context, retention uint64) error {
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

	start := p.lastPruned + 1
	end := target
	if end-start+1 > maxPruneBatch {
		end = start + maxPruneBatch - 1
	}

	for h := start; h <= end; h++ {
		if p.stateDeleter != nil {
			if err := p.stateDeleter.DeleteStateAtHeight(ctx, h); err != nil && !errors.Is(err, ds.ErrNotFound) {
				return err
			}
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
