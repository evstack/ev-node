package node

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/evstack/ev-node/block"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/raft"
	rpcserver "github.com/evstack/ev-node/pkg/rpc/server"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/store"
	evsync "github.com/evstack/ev-node/pkg/sync"
)

// failoverState collect the components to reset when switching modes.
type failoverState struct {
	logger zerolog.Logger

	p2pClient         *p2p.Client
	headerSyncService *evsync.HeaderSyncService
	dataSyncService   *evsync.DataSyncService
	rpcServer         *http.Server
	bc                *block.Components
	raftNode          *raft.Node
	isAggregator      bool

	// catchup fields — used when the aggregator needs to sync before producing
	catchupEnabled bool
	catchupTimeout time.Duration
	daBlockTime    time.Duration
	store          store.Store
}

func newSyncMode(
	nodeConfig config.Config,
	genesis genesispkg.Genesis,
	exec coreexecutor.Executor,
	da block.DAClient,
	logger zerolog.Logger,
	rktStore store.Store,
	blockMetrics *block.Metrics,
	nodeOpts NodeOptions,
	raftNode *raft.Node,
	p2pClient *p2p.Client,
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
			headerSyncService,
			dataSyncService,
			logger,
			blockMetrics,
			nodeOpts.BlockOptions,
			raftNode,
		)
	}
	return setupFailoverState(nodeConfig, genesis, logger, rktStore, blockComponentsFn, raftNode, p2pClient, false)
}

func newAggregatorMode(
	nodeConfig config.Config,
	signer signer.Signer,
	genesis genesispkg.Genesis,
	exec coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	da block.DAClient,
	logger zerolog.Logger,
	rktStore store.Store,
	blockMetrics *block.Metrics,
	nodeOpts NodeOptions,
	raftNode *raft.Node,
	p2pClient *p2p.Client,
) (*failoverState, error) {
	blockComponentsFn := func(headerSyncService *evsync.HeaderSyncService, dataSyncService *evsync.DataSyncService) (*block.Components, error) {
		return block.NewAggregatorWithCatchupComponents(
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
	return setupFailoverState(nodeConfig, genesis, logger, rktStore, blockComponentsFn, raftNode, p2pClient, true)
}

func setupFailoverState(
	nodeConfig config.Config,
	genesis genesispkg.Genesis,
	logger zerolog.Logger,
	rktStore store.Store,
	buildComponentsFn func(headerSyncService *evsync.HeaderSyncService, dataSyncService *evsync.DataSyncService) (*block.Components, error),
	raftNode *raft.Node,
	p2pClient *p2p.Client,
	isAggregator bool,
) (*failoverState, error) {
	headerSyncService, err := evsync.NewHeaderSyncService(rktStore, nodeConfig, genesis, p2pClient, logger.With().Str("component", "HeaderSyncService").Logger())
	if err != nil {
		return nil, fmt.Errorf("error while initializing HeaderSyncService: %w", err)
	}

	dataSyncService, err := evsync.NewDataSyncService(rktStore, nodeConfig, genesis, p2pClient, logger.With().Str("component", "DataSyncService").Logger())
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

	// Catchup only applies to aggregator nodes that need to sync before
	catchupEnabled := isAggregator && nodeConfig.Node.CatchupTimeout.Duration > 0
	if isAggregator && !catchupEnabled {
		bc.Syncer = nil
	}

	return &failoverState{
		logger:            logger,
		p2pClient:         p2pClient,
		headerSyncService: headerSyncService,
		dataSyncService:   dataSyncService,
		rpcServer:         rpcServer,
		bc:                bc,
		raftNode:          raftNode,
		isAggregator:      isAggregator,
		store:             rktStore,
		catchupEnabled:    catchupEnabled,
		catchupTimeout:    nodeConfig.Node.CatchupTimeout.Duration,
		daBlockTime:       nodeConfig.DA.BlockTime.Duration,
	}, nil
}

// shouldStartSyncInPublisherMode avoids startup deadlock when a raft leader boots
// with empty sync stores and no peer can serve height 1 yet.
func (f *failoverState) shouldStartSyncInPublisherMode(ctx context.Context) bool {
	if !f.isAggregator || f.raftNode == nil || !f.raftNode.IsLeader() {
		return false
	}

	storeHeight, err := f.store.Height(ctx)
	if err != nil {
		f.logger.Warn().Err(err).Msg("cannot determine store height; keeping blocking sync startup")
		return false
	}
	headerHeight := f.headerSyncService.Store().Height()
	dataHeight := f.dataSyncService.Store().Height()
	if headerHeight > 0 || dataHeight > 0 {
		return false
	}

	f.logger.Info().
		Uint64("store_height", storeHeight).
		Uint64("header_height", headerHeight).
		Uint64("data_height", dataHeight).
		Msg("raft-enabled aggregator with empty sync stores: starting sync services in publisher mode")
	return true
}

func (f *failoverState) Run(pCtx context.Context) (multiErr error) {
	stopService := func(stoppable func(context.Context) error, name string) { //nolint:contextcheck // shutdown uses context.Background intentionally
		// parent context is cancelled already, so we need to create a new one
		shutdownCtx, done := context.WithTimeout(context.Background(), 3*time.Second) //nolint:contextcheck // intentional: need fresh context for graceful shutdown after cancellation
		defer done()

		if err := stoppable(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
			multiErr = errors.Join(multiErr, fmt.Errorf("stopping %s: %w", name, err))
		}
	}
	cCtx, cancel := context.WithCancel(pCtx)
	defer cancel()
	wg, ctx := errgroup.WithContext(cCtx)
	wg.Go(func() (rerr error) { //nolint:contextcheck // block components stop API does not accept context
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

	// start header and data sync services concurrently to avoid cumulative startup delay.
	startSyncInPublisherMode := f.shouldStartSyncInPublisherMode(ctx)
	syncWg, syncCtx := errgroup.WithContext(ctx)
	syncWg.Go(func() error {
		var err error
		if startSyncInPublisherMode {
			err = f.headerSyncService.StartForPublishing(syncCtx)
		} else {
			err = f.headerSyncService.Start(syncCtx)
		}
		if err != nil {
			return fmt.Errorf("header sync service: %w", err)
		}
		return nil
	})
	syncWg.Go(func() error {
		var err error
		if startSyncInPublisherMode {
			err = f.dataSyncService.StartForPublishing(syncCtx)
		} else {
			err = f.dataSyncService.Start(syncCtx)
		}
		if err != nil {
			return fmt.Errorf("data sync service: %w", err)
		}
		return nil
	})
	if err := syncWg.Wait(); err != nil {
		return err
	}
	defer stopService(f.headerSyncService.Stop, "header sync")
	defer stopService(f.dataSyncService.Stop, "data sync")

	if f.catchupEnabled && f.bc.Syncer != nil {
		if err := f.runCatchupPhase(ctx); err != nil {
			return err
		}
	}

	wg.Go(func() error {
		defer func() { //nolint:contextcheck // shutdown uses context.Background intentionally
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

// runCatchupPhase starts the catchup syncer, waits until DA head is reached and P2P
// is caught up, then stops the syncer so the executor can take over.
func (f *failoverState) runCatchupPhase(ctx context.Context) error {
	f.logger.Info().Msg("catchup: syncing from DA and P2P before producing blocks")

	if err := f.bc.Syncer.Start(ctx); err != nil {
		return fmt.Errorf("catchup syncer start: %w", err)
	}
	defer f.bc.Syncer.Stop(context.Background()) // nolint:errcheck,contextcheck // not critical

	caughtUp, err := f.waitForCatchup(ctx)
	if err != nil {
		return err
	}
	if !caughtUp {
		return ctx.Err()
	}
	f.logger.Info().Msg("catchup: fully caught up, stopping syncer and starting block production")
	f.bc.Syncer = nil
	return nil
}

// waitForCatchup polls DA and P2P catchup status until both sources indicate the node is caught up.
func (f *failoverState) waitForCatchup(ctx context.Context) (bool, error) {
	pollInterval := f.daBlockTime
	if pollInterval <= 0 {
		pollInterval = time.Second / 10
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var timeoutCh <-chan time.Time
	if f.catchupTimeout > 0 {
		f.logger.Debug().Dur("p2p_timeout", f.catchupTimeout).Msg("P2P catchup timeout configured")
		timeoutCh = time.After(f.catchupTimeout)
	} else {
		f.logger.Debug().Msg("P2P catchup timeout disabled, relying on DA only")
	}
	ignoreP2P := false

	for {
		select {
		case <-ctx.Done():
			return false, nil
		case <-timeoutCh:
			f.logger.Info().Msg("catchup: P2P timeout reached, ignoring P2P status")
			ignoreP2P = true
			timeoutCh = nil
		case <-ticker.C:
			daCaughtUp := f.bc.Syncer != nil && f.bc.Syncer.HasReachedDAHead()

			storeHeight, err := f.store.Height(ctx)
			if err != nil {
				f.logger.Warn().Err(err).Msg("failed to get store height during catchup")
				continue
			}

			maxP2PHeight := max(
				f.headerSyncService.Store().Height(),
				f.dataSyncService.Store().Height(),
			)

			p2pCaughtUp := ignoreP2P || (maxP2PHeight > 0 && storeHeight >= maxP2PHeight)
			if !ignoreP2P && f.catchupTimeout == 0 && maxP2PHeight == 0 {
				p2pCaughtUp = true
			}

			pipelineDrained := f.bc.Syncer == nil || f.bc.Syncer.PendingCount() == 0
			if daCaughtUp && p2pCaughtUp && pipelineDrained {
				f.logger.Info().
					Uint64("store_height", storeHeight).
					Uint64("max_p2p_height", maxP2PHeight).
					Msg("catchup: fully caught up")
				return true, nil
			}
		}
	}
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
