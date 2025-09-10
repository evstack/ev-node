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

// Dependencies contains all the dependencies needed to create a node
type Dependencies struct {
	Store             store.Store
	Executor          coreexecutor.Executor
	Sequencer         coresequencer.Sequencer
	DA                coreda.DA
	HeaderStore       goheader.Store[*types.SignedHeader]
	DataStore         goheader.Store[*types.Data]
	HeaderBroadcaster broadcaster[*types.SignedHeader]
	DataBroadcaster   broadcaster[*types.Data]
	Signer            signer.Signer // Optional - only needed for full nodes
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

// NewFullNodeComponents creates components for a full node that can produce and sync blocks
func NewFullNodeComponents(
	config config.Config,
	genesis genesis.Genesis,
	deps Dependencies,
	logger zerolog.Logger,
	opts ...common.BlockOptions,
) (*BlockComponents, error) {
	if deps.Signer == nil {
		return nil, fmt.Errorf("signer is required for full nodes")
	}

	blockOpts := common.DefaultBlockOptions()
	if len(opts) > 0 {
		blockOpts = opts[0]
	}

	// Create metrics
	metrics := common.PrometheusMetrics("ev_node")

	// Create shared cache manager
	cacheManager, err := cache.NewManager(config, deps.Store, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache manager: %w", err)
	}

	// Create executing component
	executor := executing.NewExecutor(
		deps.Store,
		deps.Executor,
		deps.Sequencer,
		deps.Signer,
		cacheManager,
		metrics,
		config,
		genesis,
		deps.HeaderBroadcaster,
		deps.DataBroadcaster,
		logger,
		blockOpts,
	)

	// Create syncing component
	syncer := syncing.NewSyncer(
		deps.Store,
		deps.Executor,
		deps.DA,
		cacheManager,
		metrics,
		config,
		genesis,
		deps.Signer,
		deps.HeaderStore,
		deps.DataStore,
		logger,
		blockOpts,
	)

	return &BlockComponents{
		Executor: executor,
		Syncer:   syncer,
		Cache:    cacheManager,
	}, nil
}

// NewLightNodeComponents creates components for a light node that can only sync blocks
func NewLightNodeComponents(
	config config.Config,
	genesis genesis.Genesis,
	deps Dependencies,
	logger zerolog.Logger,
	opts ...common.BlockOptions,
) (*BlockComponents, error) {
	blockOpts := common.DefaultBlockOptions()
	if len(opts) > 0 {
		blockOpts = opts[0]
	}

	// Create metrics
	metrics := common.PrometheusMetrics("ev_node")

	// Create shared cache manager
	cacheManager, err := cache.NewManager(config, deps.Store, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache manager: %w", err)
	}

	// Create syncing component only
	syncer := syncing.NewSyncer(
		deps.Store,
		deps.Executor,
		deps.DA,
		cacheManager,
		metrics,
		config,
		genesis,
		deps.Signer,
		deps.HeaderStore,
		deps.DataStore,
		logger,
		blockOpts,
	)

	return &BlockComponents{
		Executor: nil, // Light nodes don't have executors
		Syncer:   syncer,
		Cache:    cacheManager,
	}, nil
}
