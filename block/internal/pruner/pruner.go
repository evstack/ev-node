package pruner

import (
	"context"
	"errors"
	"sync"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/pkg/store"
)

const (
	DefaultPruneInterval = time.Minute
	maxPruneBatch        = uint64(1000)
)

// ExecMetaPruner removes execution metadata at a given height.
type ExecMetaPruner interface {
	PruneExecMeta(ctx context.Context, height uint64) error
}

type stateDeleter interface {
	DeleteStateAtHeight(ctx context.Context, height uint64) error
}

// Pruner periodically removes old state and execution metadata entries.
type Pruner struct {
	store        store.Store
	stateDeleter stateDeleter
	execPruner   ExecMetaPruner
	retention    uint64
	interval     time.Duration
	logger       zerolog.Logger
	lastPruned   uint64

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// New creates a new Pruner instance.
func New(store store.Store, execMetaPruner ExecMetaPruner, retention uint64, interval time.Duration, logger zerolog.Logger) *Pruner {
	if interval <= 0 {
		interval = DefaultPruneInterval
	}

	var deleter stateDeleter
	if store != nil {
		if sd, ok := store.(stateDeleter); ok {
			deleter = sd
		}
	}

	return &Pruner{
		store:        store,
		stateDeleter: deleter,
		execPruner:   execMetaPruner,
		retention:    retention,
		interval:     interval,
		logger:       logger,
	}
}

// Start begins the pruning loop.
func (p *Pruner) Start(ctx context.Context) error {
	if p == nil || p.retention == 0 || (p.stateDeleter == nil && p.execPruner == nil) {
		return nil
	}

	loopCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	p.wg.Add(1)
	go p.pruneLoop(loopCtx)

	return nil
}

// Stop stops the pruning loop.
func (p *Pruner) Stop() error {
	if p == nil || p.cancel == nil {
		return nil
	}

	p.cancel()
	p.wg.Wait()
	return nil
}

func (p *Pruner) pruneLoop(ctx context.Context) {
	defer p.wg.Done()
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	if err := p.pruneOnce(ctx); err != nil {
		p.logger.Error().Err(err).Msg("failed to prune recovery history")
	}

	for {
		select {
		case <-ticker.C:
			if err := p.pruneOnce(ctx); err != nil {
				p.logger.Error().Err(err).Msg("failed to prune recovery history")
			}
		case <-ctx.Done():
			return
		}
	}
}

func (p *Pruner) pruneOnce(ctx context.Context) error {
	if p.retention == 0 || p.store == nil {
		return nil
	}

	height, err := p.store.Height(ctx)
	if err != nil {
		return err
	}

	if height <= p.retention {
		return nil
	}

	target := height - p.retention
	if target < p.lastPruned {
		return nil
	}
	if target == p.lastPruned {
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
			if err := p.execPruner.PruneExecMeta(ctx, h); err != nil && !errors.Is(err, ds.ErrNotFound) {
				return err
			}
		}
	}

	p.lastPruned = end
	return nil
}
