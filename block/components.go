package block

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	da "github.com/evstack/ev-node/block/internal/da"
	"github.com/evstack/ev-node/block/internal/executing"
	"github.com/evstack/ev-node/block/internal/pruner"
	"github.com/evstack/ev-node/block/internal/reaping"
	"github.com/evstack/ev-node/block/internal/submitting"
	"github.com/evstack/ev-node/block/internal/syncing"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/pkg/telemetry"
)

type Components struct {
	Executor  *executing.Executor
	Pruner    *pruner.Pruner
	Reaper    *reaping.Reaper
	Syncer    *syncing.Syncer
	Submitter *submitting.Submitter
	Cache     cache.Manager

	errorCh chan error
}

func (bc *Components) Start(ctx context.Context) error {
	ctxWithCancel, cancel := context.WithCancel(ctx)

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
	if bc.Pruner != nil {
		if err := bc.Pruner.Start(ctxWithCancel); err != nil {
			return fmt.Errorf("failed to start pruner: %w", err)
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

	<-ctxWithCancel.Done()

	select {
	case err := <-criticalErrCh:
		return fmt.Errorf("node stopped due to critical execution client failure: %w", err)
	default:
		return ctx.Err()
	}
}

func (bc *Components) Stop() error {
	var errs error
	if bc.Executor != nil {
		if err := bc.Executor.Stop(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to stop executor: %w", err))
		}
	}
	if bc.Pruner != nil {
		if err := bc.Pruner.Stop(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to stop pruner: %w", err))
		}
	}
	if bc.Reaper != nil {
		if err := bc.Reaper.Stop(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to stop reaper: %w", err))
		}
	}
	if bc.Syncer != nil {
		if err := bc.Syncer.Stop(context.Background()); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to stop syncer: %w", err))
		}
	}
	if bc.Submitter != nil {
		if err := bc.Submitter.Stop(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to stop submitter: %w", err))
		}
	}
	if bc.Cache != nil {
		if err := bc.Cache.SaveToStore(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to save caches: %w", err))
		}
	}

	return errs
}

func NewSyncComponents(
	config config.Config,
	genesis genesis.Genesis,
	store store.Store,
	exec coreexecutor.Executor,
	daClient da.Client,
	logger zerolog.Logger,
	metrics *Metrics,
	blockOpts BlockOptions,
	raftNode common.RaftNode,
) (*Components, error) {
	cacheManager, err := cache.NewManager(config, store, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache manager: %w", err)
	}

	errorCh := make(chan error, 1)

	syncer := syncing.NewSyncer(
		store,
		exec,
		daClient,
		cacheManager,
		metrics,
		config,
		genesis,
		logger,
		blockOpts,
		errorCh,
		raftNode,
	)

	if config.Instrumentation.IsTracingEnabled() {
		syncer.SetBlockSyncer(syncing.WithTracingBlockSyncer(syncer))
	}

	var execPruner coreexecutor.ExecPruner
	if p, ok := exec.(coreexecutor.ExecPruner); ok {
		execPruner = p
	}
	prunerObj := pruner.New(logger, store, execPruner, config.Pruning, config.Node.BlockTime.Duration, config.DA.Address)

	var daSubmitter submitting.DASubmitterAPI = submitting.NewDASubmitter(daClient, config, genesis, blockOpts, metrics, logger)
	if config.Instrumentation.IsTracingEnabled() {
		daSubmitter = submitting.WithTracingDASubmitter(daSubmitter)
	}
	submitter := submitting.NewSubmitter(
		store,
		exec,
		cacheManager,
		metrics,
		config,
		genesis,
		daSubmitter,
		nil,
		nil,
		logger,
		errorCh,
	)

	return &Components{
		Syncer:    syncer,
		Submitter: submitter,
		Cache:     cacheManager,
		Pruner:    prunerObj,
		errorCh:   errorCh,
	}, nil
}

func newAggregatorComponents(
	config config.Config,
	genesis genesis.Genesis,
	store store.Store,
	exec coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	daClient da.Client,
	signer signer.Signer,
	logger zerolog.Logger,
	metrics *Metrics,
	blockOpts BlockOptions,
	raftNode common.RaftNode,
) (*Components, error) {
	cacheManager, err := cache.NewManager(config, store, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache manager: %w", err)
	}

	errorCh := make(chan error, 1)

	if config.Instrumentation.IsTracingEnabled() {
		sequencer = telemetry.WithTracingSequencer(sequencer)
	}

	executor, err := executing.NewExecutor(
		store,
		exec,
		sequencer,
		signer,
		cacheManager,
		metrics,
		config,
		genesis,
		logger,
		blockOpts,
		errorCh,
		raftNode,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	if config.Instrumentation.IsTracingEnabled() {
		executor.SetBlockProducer(executing.WithTracingBlockProducer(executor))
	}

	var execPruner coreexecutor.ExecPruner
	if p, ok := exec.(coreexecutor.ExecPruner); ok {
		execPruner = p
	}
	prunerObj := pruner.New(logger, store, execPruner, config.Pruning, config.Node.BlockTime.Duration, config.DA.Address)

	reaper, err := reaping.NewReaper(
		exec,
		sequencer,
		genesis,
		logger,
		cacheManager,
		config.Node.ScrapeInterval.Duration,
		executor.NotifyNewTransactions,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create reaper: %w", err)
	}

	if config.Node.BasedSequencer {
		return &Components{
			Executor: executor,
			Pruner:   prunerObj,
			Reaper:   reaper,
			Cache:    cacheManager,
			errorCh:  errorCh,
		}, nil
	}

	var daSubmitter submitting.DASubmitterAPI = submitting.NewDASubmitter(daClient, config, genesis, blockOpts, metrics, logger)
	if config.Instrumentation.IsTracingEnabled() {
		daSubmitter = submitting.WithTracingDASubmitter(daSubmitter)
	}
	submitter := submitting.NewSubmitter(
		store,
		exec,
		cacheManager,
		metrics,
		config,
		genesis,
		daSubmitter,
		sequencer,
		signer,
		logger,
		errorCh,
	)

	return &Components{
		Executor:  executor,
		Pruner:    prunerObj,
		Reaper:    reaper,
		Submitter: submitter,
		Cache:     cacheManager,
		errorCh:   errorCh,
	}, nil
}

func NewAggregatorWithCatchupComponents(
	config config.Config,
	genesis genesis.Genesis,
	store store.Store,
	exec coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	daClient da.Client,
	signer signer.Signer,
	logger zerolog.Logger,
	metrics *Metrics,
	blockOpts BlockOptions,
	raftNode common.RaftNode,
) (*Components, error) {
	bc, err := newAggregatorComponents(
		config, genesis, store, exec, sequencer, daClient, signer,
		logger, metrics, blockOpts, raftNode,
	)
	if err != nil {
		return nil, err
	}

	catchupErrCh := make(chan error, 1)
	catchupSyncer := syncing.NewSyncer(
		store,
		exec,
		daClient,
		bc.Cache,
		metrics,
		config,
		genesis,
		logger,
		blockOpts,
		catchupErrCh,
		raftNode,
	)
	if config.Instrumentation.IsTracingEnabled() {
		catchupSyncer.SetBlockSyncer(syncing.WithTracingBlockSyncer(catchupSyncer))
	}

	bc.Syncer = catchupSyncer
	return bc, nil
}
