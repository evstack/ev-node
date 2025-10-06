package block

import (
	"context"
	"errors"
	"fmt"

	goheader "github.com/celestiaorg/go-header"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/executing"
	"github.com/evstack/ev-node/block/internal/reaping"
	"github.com/evstack/ev-node/block/internal/submitting"
	"github.com/evstack/ev-node/block/internal/syncing"
	coreda "github.com/evstack/ev-node/core/da"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

// Components represents the block-related components
type Components struct {
	Executor  *executing.Executor
	Reaper    *reaping.Reaper
	Syncer    *syncing.Syncer
	Submitter *submitting.Submitter
	Cache     cache.Manager

	// Error channel for critical failures that should stop the node
	errorCh chan error
}

// GetLastState returns the current blockchain state
func (bc *Components) GetLastState() types.State {
	if bc.Executor != nil {
		return bc.Executor.GetLastState()
	}
	if bc.Syncer != nil {
		return bc.Syncer.GetLastState()
	}
	return types.State{}
}

// Start starts all components and monitors for critical errors
func (bc *Components) Start(ctx context.Context) error {
	ctxWithCancel, cancel := context.WithCancel(ctx)

	// error monitoring goroutine
	criticalErrCh := make(chan error, 1)
	go func() {
		select {
		case err := <-bc.errorCh:
			criticalErrCh <- err
			cancel()
		case <-ctx.Done():
			return
		}
	}()

	if bc.Executor != nil {
		if err := bc.Executor.Start(ctxWithCancel); err != nil {
			return fmt.Errorf("failed to start executor: %w", err)
		}
	}
	if bc.Reaper != nil {
		if err := bc.Reaper.Start(ctxWithCancel); err != nil {
			return fmt.Errorf("failed to start reaper: %w", err)
		}
	}
	if bc.Syncer != nil {
		if err := bc.Syncer.Start(ctxWithCancel); err != nil {
			return fmt.Errorf("failed to start syncer: %w", err)
		}
	}
	if bc.Submitter != nil {
		if err := bc.Submitter.Start(ctxWithCancel); err != nil {
			return fmt.Errorf("failed to start submitter: %w", err)
		}
	}

	// wait for context cancellation (either from parent or critical error)
	<-ctxWithCancel.Done()

	// if we got here due to a critical error, return that error
	select {
	case err := <-criticalErrCh:
		return fmt.Errorf("node stopped due to critical execution client failure: %w", err)
	default:
		return ctx.Err()
	}
}

// Stop stops all components
func (bc *Components) Stop() error {
	var errs error
	if bc.Executor != nil {
		if err := bc.Executor.Stop(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to stop executor: %w", err))
		}
	}
	if bc.Reaper != nil {
		if err := bc.Reaper.Stop(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to stop reaper: %w", err))
		}
	}
	if bc.Syncer != nil {
		if err := bc.Syncer.Stop(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to stop syncer: %w", err))
		}
	}
	if bc.Submitter != nil {
		if err := bc.Submitter.Stop(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to stop submitter: %w", err))
		}
	}

	return errs
}

// broadcaster interface for P2P broadcasting
type broadcaster[T any] interface {
	WriteToStoreAndBroadcast(ctx context.Context, payload T) error
}

// NewSyncComponents creates components for a non-aggregator full node that can only sync blocks.
// Non-aggregator full nodes can sync from P2P and DA but cannot produce blocks or submit to DA.
// They have more sync capabilities than light nodes but no block production. No signer required.
func NewSyncComponents(
	config config.Config,
	genesis genesis.Genesis,
	store store.Store,
	exec coreexecutor.Executor,
	da coreda.DA,
	headerStore goheader.Store[*types.SignedHeader],
	dataStore goheader.Store[*types.Data],
	logger zerolog.Logger,
	metrics *Metrics,
	blockOpts BlockOptions,
) (*Components, error) {
	cacheManager, err := cache.NewManager(config, store, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache manager: %w", err)
	}

	// error channel for critical failures
	errorCh := make(chan error, 1)

	syncer := syncing.NewSyncer(
		store,
		exec,
		da,
		cacheManager,
		metrics,
		config,
		genesis,
		headerStore,
		dataStore,
		logger,
		blockOpts,
		errorCh,
	)

	// Create DA submitter for sync nodes (no signer, only DA inclusion processing)
	daSubmitter := submitting.NewDASubmitter(da, config, genesis, blockOpts, logger)
	submitter := submitting.NewSubmitter(
		store,
		exec,
		cacheManager,
		metrics,
		config,
		genesis,
		daSubmitter,
		nil, // No signer for sync nodes
		logger,
		errorCh,
	)

	return &Components{
		Syncer:    syncer,
		Submitter: submitter,
		Cache:     cacheManager,
		errorCh:   errorCh,
	}, nil
}

// NewAggregatorComponents creates components for an aggregator full node that can produce and sync blocks.
// Aggregator nodes have full capabilities - they can produce blocks, sync from P2P and DA,
// and submit headers/data to DA. Requires a signer for block production and DA submission.
func NewAggregatorComponents(
	config config.Config,
	genesis genesis.Genesis,
	store store.Store,
	exec coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	da coreda.DA,
	signer signer.Signer,
	headerBroadcaster broadcaster[*types.SignedHeader],
	dataBroadcaster broadcaster[*types.Data],
	logger zerolog.Logger,
	metrics *Metrics,
	blockOpts BlockOptions,
) (*Components, error) {
	cacheManager, err := cache.NewManager(config, store, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache manager: %w", err)
	}

	// error channel for critical failures
	errorCh := make(chan error, 1)

	executor, err := executing.NewExecutor(
		store,
		exec,
		sequencer,
		signer,
		cacheManager,
		metrics,
		config,
		genesis,
		headerBroadcaster,
		dataBroadcaster,
		logger,
		blockOpts,
		errorCh,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	reaper, err := reaping.NewReaper(
		exec,
		sequencer,
		genesis,
		logger,
		executor,
		reaping.DefaultInterval,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create reaper: %w", err)
	}

	// Create DA submitter for aggregator nodes (with signer for submission)
	daSubmitter := submitting.NewDASubmitter(da, config, genesis, blockOpts, logger)
	submitter := submitting.NewSubmitter(
		store,
		exec,
		cacheManager,
		metrics,
		config,
		genesis,
		daSubmitter,
		signer, // Signer for aggregator nodes to submit to DA
		logger,
		errorCh,
	)

	return &Components{
		Executor:  executor,
		Reaper:    reaper,
		Submitter: submitter,
		Cache:     cacheManager,
		errorCh:   errorCh,
	}, nil
}
