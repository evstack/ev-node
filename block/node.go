package block

import (
	"context"
	"fmt"

	goheader "github.com/celestiaorg/go-header"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/executing"
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

// BlockComponents represents the block-related components
type BlockComponents struct {
	Executor  *executing.Executor
	Syncer    *syncing.Syncer
	Submitter *submitting.Submitter
	Cache     cache.Manager
}

// GetLastState returns the current blockchain state
func (bc *BlockComponents) GetLastState() types.State {
	if bc.Executor != nil {
		return bc.Executor.GetLastState()
	}
	if bc.Syncer != nil {
		return bc.Syncer.GetLastState()
	}
	return types.State{}
}

// Start starts all components
func (bc *BlockComponents) Start(ctx context.Context) error {
	if bc.Executor != nil {
		if err := bc.Executor.Start(ctx); err != nil {
			return fmt.Errorf("failed to start executor: %w", err)
		}
	}
	if bc.Syncer != nil {
		if err := bc.Syncer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start syncer: %w", err)
		}
	}
	if bc.Submitter != nil {
		if err := bc.Submitter.Start(ctx); err != nil {
			return fmt.Errorf("failed to start submitter: %w", err)
		}
	}
	return nil
}

// Stop stops all components
func (bc *BlockComponents) Stop() error {
	var errs []error
	if bc.Executor != nil {
		if err := bc.Executor.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop executor: %w", err))
		}
	}
	if bc.Syncer != nil {
		if err := bc.Syncer.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop syncer: %w", err))
		}
	}
	if bc.Submitter != nil {
		if err := bc.Submitter.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop submitter: %w", err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors stopping components: %v", errs)
	}
	return nil
}

// broadcaster interface for P2P broadcasting
type broadcaster[T any] interface {
	WriteToStoreAndBroadcast(ctx context.Context, payload T) error
}

// NewSyncNode creates components for a non-aggregator full node that can only sync blocks.
// Non-aggregator full nodes can sync from P2P and DA but cannot produce blocks or submit to DA.
// They have more sync capabilities than light nodes but no block production. No signer required.
func NewSyncNode(
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
) (*BlockComponents, error) {
	cacheManager, err := cache.NewManager(config, store, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache manager: %w", err)
	}

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
	)

	// Create DA submitter for sync nodes (no signer, only DA inclusion processing)
	daSubmitter := submitting.NewDASubmitter(da, config, genesis, blockOpts, logger)
	submitter := submitting.NewSubmitter(
		store,
		exec,
		cacheManager,
		config,
		genesis,
		daSubmitter,
		nil, // No signer for sync nodes
		logger,
	)

	return &BlockComponents{
		Syncer:    syncer,
		Submitter: submitter,
		Cache:     cacheManager,
	}, nil
}

// NewAggregatorNode creates components for an aggregator full node that can produce and sync blocks.
// Aggregator nodes have full capabilities - they can produce blocks, sync from P2P and DA,
// and submit headers/data to DA. Requires a signer for block production and DA submission.
func NewAggregatorNode(
	config config.Config,
	genesis genesis.Genesis,
	store store.Store,
	exec coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	da coreda.DA,
	signer signer.Signer,
	headerStore goheader.Store[*types.SignedHeader],
	dataStore goheader.Store[*types.Data],
	headerBroadcaster broadcaster[*types.SignedHeader],
	dataBroadcaster broadcaster[*types.Data],
	logger zerolog.Logger,
	metrics *Metrics,
	blockOpts BlockOptions,
) (*BlockComponents, error) {
	cacheManager, err := cache.NewManager(config, store, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache manager: %w", err)
	}

	executor := executing.NewExecutor(
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
	)

	// Create DA submitter for aggregator nodes (with signer for submission)
	daSubmitter := submitting.NewDASubmitter(da, config, genesis, blockOpts, logger)
	submitter := submitting.NewSubmitter(
		store,
		exec,
		cacheManager,
		config,
		genesis,
		daSubmitter,
		signer, // Signer for aggregator nodes to submit to DA
		logger,
	)

	return &BlockComponents{
		Executor:  executor,
		Submitter: submitter,
		Cache:     cacheManager,
	}, nil
}
