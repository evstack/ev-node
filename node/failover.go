package node

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/evstack/ev-node/block"
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
	"golang.org/x/sync/errgroup"
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
	rootDB ds.Batching,
	daStore store.Store,
	exec coreexecutor.Executor,
	da block.DAClient,
	logger zerolog.Logger,
	rktStore store.Store,
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
			headerSyncService,
			dataSyncService,
			logger,
			blockMetrics,
			nodeOpts.BlockOptions,
			raftNode,
		)
	}
	return setupFailoverState(nodeConfig, nodeKey, rootDB, daStore, genesis, logger, rktStore, blockComponentsFn, raftNode)
}

func newAggregatorMode(
	nodeConfig config.Config,
	nodeKey *key.NodeKey,
	signer signer.Signer,
	genesis genesispkg.Genesis,
	rootDB ds.Batching,
	daStore store.Store,
	exec coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	da block.DAClient,
	logger zerolog.Logger,
	rktStore store.Store,
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

	return setupFailoverState(nodeConfig, nodeKey, rootDB, daStore, genesis, logger, rktStore, blockComponentsFn, raftNode)
}

func setupFailoverState(
	nodeConfig config.Config,
	nodeKey *key.NodeKey,
	rootDB ds.Batching,
	daStore store.Store,
	genesis genesispkg.Genesis,
	logger zerolog.Logger,
	rktStore store.Store,
	buildComponentsFn func(headerSyncService *evsync.HeaderSyncService, dataSyncService *evsync.DataSyncService) (*block.Components, error),
	raftNode *raft.Node,
) (*failoverState, error) {
	p2pClient, err := p2p.NewClient(nodeConfig.P2P, nodeKey.PrivKey, rootDB, genesis.ChainID, logger, nil)
	if err != nil {
		return nil, err
	}

	headerSyncService, err := evsync.NewHeaderSyncService(daStore, nodeConfig, genesis, p2pClient, logger.With().Str("component", "HeaderSyncService").Logger())
	if err != nil {
		return nil, fmt.Errorf("error while initializing HeaderSyncService: %w", err)
	}

	dataSyncService, err := evsync.NewDataSyncService(daStore, nodeConfig, genesis, p2pClient, logger.With().Str("component", "DataSyncService").Logger())
	if err != nil {
		return nil, fmt.Errorf("error while initializing DataSyncService: %w", err)
	}

	bestKnownHeightProvider := func() uint64 {
		hHeight := headerSyncService.Store().Height()
		dHeight := dataSyncService.Store().Height()
		return min(hHeight, dHeight)
	}
	handler, err := rpcserver.NewServiceHandler(
		rktStore,
		headerSyncService.Store(),
		dataSyncService.Store(),
		p2pClient,
		genesis.ProposerAddress,
		logger,
		nodeConfig,
		bestKnownHeightProvider,
		raftNode,
	)
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

func (f *failoverState) Run(pCtx context.Context) (multiErr error) {
	stopService := func(stoppable func(context.Context) error, name string) {
		// parent context is cancelled already, so we need to create a new one
		shutdownCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		if err := stoppable(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
			multiErr = errors.Join(multiErr, fmt.Errorf("stopping %s: %w", name, err))
		}
	}
	cCtx, cancel := context.WithCancel(pCtx)
	defer cancel()
	wg, ctx := errgroup.WithContext(cCtx)
	wg.Go(func() (rerr error) {
		defer func() {
			if err := f.bc.Stop(); err != nil && !errors.Is(err, context.Canceled) {
				rerr = errors.Join(rerr, fmt.Errorf("stopping block components: %w", err))
			}
		}()

		f.logger.Info().Str("addr", f.rpcServer.Addr).Msg("Started RPC server")
		if err := f.rpcServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	if err := f.p2pClient.Start(ctx); err != nil {
		return fmt.Errorf("start p2p: %w", err)
	}
	defer f.p2pClient.Close() // nolint: errcheck

	if err := f.headerSyncService.Start(ctx); err != nil {
		return fmt.Errorf("error while starting header sync service: %w", err)
	}
	defer stopService(f.headerSyncService.Stop, "header sync")

	if err := f.dataSyncService.Start(ctx); err != nil {
		return fmt.Errorf("error while starting data sync service: %w", err)
	}
	defer stopService(f.dataSyncService.Stop, "data sync")

	wg.Go(func() error {
		defer func() {
			shutdownCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
			defer done()
			_ = f.rpcServer.Shutdown(shutdownCtx)
		}()
		if err := f.bc.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("components started with error: %w", err)
		}
		return nil
	})

	return wg.Wait()
}

func (f *failoverState) IsSynced(s *raft.RaftBlockState) (int, error) {
	if f.bc.Syncer != nil {
		return f.bc.Syncer.IsSyncedWithRaft(s)
	}
	if f.bc.Executor != nil {
		return f.bc.Executor.IsSyncedWithRaft(s)
	}
	return 0, errors.New("sync check not supported in this mode")
}

func (f *failoverState) Recover(ctx context.Context, state *raft.RaftBlockState) error {
	if f.bc.Syncer != nil {
		return f.bc.Syncer.RecoverFromRaft(ctx, state)
	}
	// For aggregator mode without syncer (e.g. based sequencer only?), recovery logic might differ or be implicit.
	// But failure to recover means we are stuck.
	return errors.New("recovery not supported in this mode")
}

var _ leaderElection = &singleRoleElector{}
var _ testSupportElection = &singleRoleElector{}

// singleRoleElector implements leaderElection but with a static role. No switchover.
type singleRoleElector struct {
	running  atomic.Bool
	runnable raft.Runnable
}

func newSingleRoleElector(factory func() (raft.Runnable, error)) (*singleRoleElector, error) {
	r, err := factory()
	if err != nil {
		return nil, err
	}
	return &singleRoleElector{runnable: r}, nil
}

func (a *singleRoleElector) Run(ctx context.Context) error {
	a.running.Store(true)
	defer a.running.Store(false)
	return a.runnable.Run(ctx)
}

func (a *singleRoleElector) IsRunning() bool {
	return a.running.Load()
}

// for testing purposes only
func (a *singleRoleElector) state() *failoverState {
	if v, ok := a.runnable.(*failoverState); ok {
		return v
	}
	return nil
}
