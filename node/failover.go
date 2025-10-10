package node

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/evstack/ev-node/block"
	coreda "github.com/evstack/ev-node/core/da"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/p2p/key"
	"github.com/evstack/ev-node/pkg/raft"
	rpcserver "github.com/evstack/ev-node/pkg/rpc/server"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/store"
	evsync "github.com/evstack/ev-node/pkg/sync"
	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"
)

// failoverState collect the components to reset when switching modes.
type failoverState struct {
	logger zerolog.Logger

	p2pClient         *p2p.Client
	headerSyncService *evsync.HeaderSyncService
	dataSyncService   *evsync.DataSyncService
	rpcServer         *http.Server
	bc                *block.Components
}

func newSyncMode(
	nodeConfig config.Config,
	nodeKey *key.NodeKey,
	genesis genesispkg.Genesis,
	database ds.Batching,
	exec coreexecutor.Executor,
	da coreda.DA,
	logger zerolog.Logger,
	rktStore store.Store,
	mainKV ds.Batching,
	blockMetrics *block.Metrics,
	nodeOpts NodeOptions,
	raftNode *raft.Node,
) (*failoverState, error) {
	blockComponentsFn := func(headerSyncService *evsync.HeaderSyncService, dataSyncService *evsync.DataSyncService) (*block.Components, error) {
		return block.NewSyncComponents(
			nodeConfig,
			genesis,
			rktStore,
			exec,
			da,
			headerSyncService.Store(),
			dataSyncService.Store(),
			logger,
			blockMetrics,
			nodeOpts.BlockOptions,
			raftNode,
		)
	}
	return setupFailoverState(nodeConfig, nodeKey, database, genesis, logger, mainKV, rktStore, blockComponentsFn, raftNode)
}
func newAggregatorMode(
	nodeConfig config.Config,
	nodeKey *key.NodeKey,
	signer signer.Signer,
	genesis genesispkg.Genesis,
	database ds.Batching,
	exec coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	da coreda.DA,
	logger zerolog.Logger,
	rktStore store.Store,
	mainKV ds.Batching,
	blockMetrics *block.Metrics,
	nodeOpts NodeOptions,
	raftNode *raft.Node,
) (*failoverState, error) {

	blockComponentsFn := func(headerSyncService *evsync.HeaderSyncService, dataSyncService *evsync.DataSyncService) (*block.Components, error) {
		return block.NewAggregatorComponents(
			nodeConfig,
			genesis,
			rktStore,
			exec,
			sequencer,
			da,
			signer,
			headerSyncService,
			dataSyncService,
			logger,
			blockMetrics,
			nodeOpts.BlockOptions,
			raftNode,
		)
	}

	return setupFailoverState(nodeConfig, nodeKey, database, genesis, logger, mainKV, rktStore, blockComponentsFn, raftNode)
}

func setupFailoverState(
	nodeConfig config.Config,
	nodeKey *key.NodeKey,
	database ds.Batching,
	genesis genesispkg.Genesis,
	logger zerolog.Logger,
	mainKV ds.Batching,
	rktStore store.Store,
	buildComponentsFn func(headerSyncService *evsync.HeaderSyncService, dataSyncService *evsync.DataSyncService) (*block.Components, error),
	raftNode *raft.Node,
) (*failoverState, error) {
	p2pClient, err := p2p.NewClient(nodeConfig.P2P, nodeKey.PrivKey, database, genesis.ChainID, logger, nil)
	if err != nil {
		return nil, err
	}

	headerSyncService, err := evsync.NewHeaderSyncService(mainKV, nodeConfig, genesis, p2pClient, logger.With().Str("component", "HeaderSyncService").Logger())
	if err != nil {
		return nil, fmt.Errorf("error while initializing HeaderSyncService: %w", err)
	}

	dataSyncService, err := evsync.NewDataSyncService(mainKV, nodeConfig, genesis, p2pClient, logger.With().Str("component", "DataSyncService").Logger())
	if err != nil {
		return nil, fmt.Errorf("error while initializing DataSyncService: %w", err)
	}

	bestKnownHeightProvider := func() uint64 {
		hHeight := headerSyncService.Store().Height()
		dHeight := dataSyncService.Store().Height()
		return min(hHeight, dHeight)
	}
	handler, err := rpcserver.NewServiceHandler(rktStore, p2pClient, genesis.ProposerAddress, logger, nodeConfig, bestKnownHeightProvider, raftNode)
	if err != nil {
		return nil, fmt.Errorf("error creating RPC handler: %w", err)
	}

	rpcServer := &http.Server{
		Addr:         nodeConfig.RPC.Address,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	bc, err := buildComponentsFn(headerSyncService, dataSyncService)
	if err != nil {
		return nil, fmt.Errorf("build follower components: %w", err)
	}

	return &failoverState{
		logger:            logger,
		p2pClient:         p2pClient,
		headerSyncService: headerSyncService,
		dataSyncService:   dataSyncService,
		rpcServer:         rpcServer,
		bc:                bc,
	}, nil
}

func (f *failoverState) Run(ctx context.Context) (multiErr error) {
	errChan := make(chan error)
	go func() {
		f.logger.Info().Str("addr", f.rpcServer.Addr).Msg("Started RPC server")
		if err := f.rpcServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			select {
			case errChan <- fmt.Errorf("RPC server error: %w", err):
			default:
				f.logger.Error().Err(err).Msg("RPC server error")
			}
		}
	}()
	defer f.rpcServer.Shutdown(context.Background()) // nolint: errcheck

	if err := f.p2pClient.Start(ctx); err != nil {
		return err
	}
	defer f.p2pClient.Close() // nolint: errcheck

	if err := f.headerSyncService.Start(ctx); err != nil {
		return fmt.Errorf("error while starting header sync service: %w", err)
	}
	defer func() {
		if err := f.headerSyncService.Stop(context.Background()); err != nil && !errors.Is(err, context.Canceled) {
			multiErr = errors.Join(multiErr, fmt.Errorf("stopping header sync: %w", err))
		}
	}()

	if err := f.dataSyncService.Start(ctx); err != nil {
		return fmt.Errorf("error while starting data sync service: %w", err)
	}
	defer func() {
		if err := f.dataSyncService.Stop(context.Background()); err != nil && !errors.Is(err, context.Canceled) {
			multiErr = errors.Join(multiErr, fmt.Errorf("stopping data sync: %w", err))
		}
	}()

	defer func() {
		if f.bc.Syncer != nil {
			for f.bc.Syncer.HasUnprocessedEvents() {
				// give it some time to gracefully complete
				time.Sleep(100 * time.Millisecond)
			}
		}
		if err := f.bc.Stop(); err != nil && !errors.Is(err, context.Canceled) {
			multiErr = errors.Join(multiErr, fmt.Errorf("stopping block components: %w", err))
		}
	}()

	go func() {
		if err := f.bc.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			select {
			case errChan <- fmt.Errorf("components started with error: %w", err):
			default:
				f.logger.Error().Err(err).Msg("Components start error")
			}
			return
		}
		select {
		case errChan <- nil:
		default:
		}
	}()

	return <-errChan
}

var _ leaderElection = AlwaysLeader{}

type AlwaysLeader struct{}

func (a AlwaysLeader) RunWithElection(ctx context.Context, leaderFunc, _ func(leaderCtx context.Context) error) error {
	return leaderFunc(ctx)
}

func (a AlwaysLeader) Start(ctx context.Context) error {
	return nil
}

func (a AlwaysLeader) Stop() error {
	return nil
}

var _ leaderElection = AlwaysFollower{}

type AlwaysFollower struct{}

func (a AlwaysFollower) RunWithElection(ctx context.Context, _, followerFunc func(leaderCtx context.Context) error) error {
	return followerFunc(ctx)
}

func (a AlwaysFollower) Start(ctx context.Context) error {
	return nil
}

func (a AlwaysFollower) Stop() error {
	return nil
}
