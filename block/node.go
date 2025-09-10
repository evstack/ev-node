package block

import (
	"context"
	"fmt"

	goheader "github.com/celestiaorg/go-header"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/block/internal/executing"
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
	Executor *executing.Executor
	Syncer   *syncing.Syncer
	Cache    cache.Manager
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

// broadcaster interface for P2P broadcasting
type broadcaster[T any] interface {
	WriteToStoreAndBroadcast(ctx context.Context, payload T) error
}

// BlockOptions defines the options for creating block components
type BlockOptions = common.BlockOptions

// DefaultBlockOptions returns the default block options
func DefaultBlockOptions() BlockOptions {
	return common.DefaultBlockOptions()
}

// NewLightNode creates components for a light node that can only sync blocks.
// Light nodes have minimal capabilities - they only sync from P2P and DA,
// but cannot produce blocks or submit to DA. No signer required.
func NewLightNode(
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
	// Create shared cache manager
	cacheManager, err := cache.NewManager(config, store, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache manager: %w", err)
	}

	// Light nodes only have syncer, no executor, no signer
	syncer := syncing.NewSyncer(
		store,
		exec,
		da,
		cacheManager,
		metrics,
		config,
		genesis,
		nil, // Light nodes don't have signers
		headerStore,
		dataStore,
		logger,
		blockOpts,
	)

	return &BlockComponents{
		Executor: nil, // Light nodes don't have executors
		Syncer:   syncer,
		Cache:    cacheManager,
	}, nil
}

// NewFullNode creates components for a non-aggregator full node that can only sync blocks.
// Non-aggregator full nodes can sync from P2P and DA but cannot produce blocks or submit to DA.
// They have more sync capabilities than light nodes but no block production. No signer required.
func NewFullNode(
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
	// Create shared cache manager
	cacheManager, err := cache.NewManager(config, store, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache manager: %w", err)
	}

	// Non-aggregator full nodes have only syncer, no executor, no signer
	syncer := syncing.NewSyncer(
		store,
		exec,
		da,
		cacheManager,
		metrics,
		config,
		genesis,
		nil, // Non-aggregator nodes don't have signers
		headerStore,
		dataStore,
		logger,
		blockOpts,
	)

	return &BlockComponents{
		Executor: nil, // Non-aggregator full nodes don't have executors
		Syncer:   syncer,
		Cache:    cacheManager,
	}, nil
}

// NewFullNodeAggregator creates components for an aggregator full node that can produce and sync blocks.
// Aggregator nodes have full capabilities - they can produce blocks, sync from P2P and DA,
// and submit headers/data to DA. Requires a signer for block production and DA submission.
func NewFullNodeAggregator(
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
	if signer == nil {
		return nil, fmt.Errorf("aggregator nodes require a signer")
	}

	// Create shared cache manager
	cacheManager, err := cache.NewManager(config, store, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache manager: %w", err)
	}

	// Aggregator nodes have both executor and syncer with signer
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

	syncer := syncing.NewSyncer(
		store,
		exec,
		da,
		cacheManager,
		metrics,
		config,
		genesis,
		signer,
		headerStore,
		dataStore,
		logger,
		blockOpts,
	)

	return &BlockComponents{
		Executor: executor,
		Syncer:   syncer,
		Cache:    cacheManager,
	}, nil
}
