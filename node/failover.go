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
	"github.com/evstack/ev-node/pkg/raft"
	rpcserver "github.com/evstack/ev-node/pkg/rpc/server"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/store"
	evsync "github.com/evstack/ev-node/pkg/sync"
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
	return setupFailoverState(nodeConfig, genesis, logger, rktStore, blockComponentsFn, raftNode, p2pClient)
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

	return setupFailoverState(nodeConfig, genesis, logger, rktStore, blockComponentsFn, raftNode, p2pClient)
}

func setupFailoverState(
	nodeConfig config.Config,
	genesis genesispkg.Genesis,
	logger zerolog.Logger,
	rktStore store.Store,
	buildComponentsFn func(headerSyncService *evsync.HeaderSyncService, dataSyncService *evsync.DataSyncService) (*block.Components, error),
	raftNode *raft.Node,
	p2pClient *p2p.Client,
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

	// start header and data sync services concurrently to avoid cumulative startup delay.
	syncWg, syncCtx := errgroup.WithContext(ctx)
	syncWg.Go(func() error {
		if err := f.headerSyncService.Start(syncCtx); err != nil {
			return fmt.Errorf("header sync service: %w", err)
		}
		return nil
	})
	syncWg.Go(func() error {
		if err := f.dataSyncService.Start(syncCtx); err != nil {
			return fmt.Errorf("data sync service: %w", err)
		}
		return nil
	})
	if err := syncWg.Wait(); err != nil {
		return err
	}
	defer stopService(f.headerSyncService.Stop, "header sync")
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

var _ leaderElection = &sequencerRecoveryElector{}
var _ testSupportElection = &sequencerRecoveryElector{}

// sequencerRecoveryElector implements leaderElection for disaster recovery.
// It starts in sync mode (follower), catches up from DA and P2P, then switches to aggregator (leader) mode.
// This is for single-sequencer setups that don't use raft.
type sequencerRecoveryElector struct {
	running         atomic.Bool
	logger          zerolog.Logger
	followerFactory func() (raft.Runnable, error)
	leaderFactory   func() (raft.Runnable, error)
	store           store.Store
	daBlockTime     time.Duration
	p2pTimeout      time.Duration

	// activeState tracks the current failoverState for test access
	activeState atomic.Pointer[failoverState]
}

func newSequencerRecoveryElector(
	logger zerolog.Logger,
	leaderFactory func() (raft.Runnable, error),
	followerFactory func() (raft.Runnable, error),
	store store.Store,
	daBlockTime time.Duration,
	p2pTimeout time.Duration,
) (*sequencerRecoveryElector, error) {
	return &sequencerRecoveryElector{
		logger:          logger.With().Str("component", "sequencer-recovery").Logger(),
		followerFactory: followerFactory,
		leaderFactory:   leaderFactory,
		store:           store,
		daBlockTime:     daBlockTime,
		p2pTimeout:      p2pTimeout,
	}, nil
}

func (s *sequencerRecoveryElector) Run(pCtx context.Context) error {
	s.running.Store(true)
	defer s.running.Store(false)

	syncCtx, cancel := context.WithCancel(pCtx)
	defer cancel()
	syncState, syncErrCh, err := s.startSyncPhase(syncCtx)
	if err != nil {
		return err
	}

	s.logger.Info().Msg("monitoring catchup status from DA and P2P")
	caughtUp, err := s.waitForCatchup(syncCtx, syncState, syncErrCh)
	if err != nil {
		return err
	}
	if !caughtUp {
		return <-syncErrCh
	}
	s.logger.Info().Msg("caught up with DA and P2P, stopping sync mode")
	cancel()

	if err := <-syncErrCh; err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("sync mode failed before switchover completed: %w", err)
	}

	return s.startAggregatorPhase(pCtx)
}

func (s *sequencerRecoveryElector) startSyncPhase(ctx context.Context) (*failoverState, <-chan error, error) {
	s.logger.Info().Msg("starting sequencer recovery: syncing from DA and P2P")

	syncRunnable, err := s.followerFactory()
	if err != nil {
		return nil, nil, fmt.Errorf("create sync mode: %w", err)
	}

	syncState, ok := syncRunnable.(*failoverState)
	if !ok {
		return nil, nil, fmt.Errorf("unexpected runnable type from follower factory")
	}

	s.activeState.Store(syncState)

	syncErrCh := make(chan error, 1)
	go func() {
		syncErrCh <- syncState.Run(ctx)
	}()

	return syncState, syncErrCh, nil
}

func (s *sequencerRecoveryElector) startAggregatorPhase(ctx context.Context) error {
	s.logger.Info().Msg("starting aggregator mode after recovery")

	aggRunnable, err := s.leaderFactory()
	if err != nil {
		return fmt.Errorf("create aggregator mode after recovery: %w", err)
	}

	if aggState, ok := aggRunnable.(*failoverState); ok {
		s.activeState.Store(aggState)
	}

	return aggRunnable.Run(ctx)
}

// waitForCatchup polls DA and P2P catchup status until both sources indicate the node is caught up.
// Returns (true, nil) when caught up, (false, nil) if context cancelled, or (false, err) on error.
func (s *sequencerRecoveryElector) waitForCatchup(ctx context.Context, syncState *failoverState, syncErrCh <-chan error) (bool, error) {
	pollInterval := s.daBlockTime
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var timeoutCh <-chan time.Time
	if s.p2pTimeout > 0 {
		s.logger.Debug().Dur("p2p_timeout", s.p2pTimeout).Msg("P2P catchup timeout configured")
		timeoutCh = time.After(s.p2pTimeout)
	} else {
		s.logger.Debug().Msg("P2P catchup timeout disabled, relying on DA only")
	}
	ignoreP2P := false

	for {
		select {
		case <-ctx.Done():
			return false, nil
		case err := <-syncErrCh:
			return false, fmt.Errorf("sync mode exited during recovery: %w", err)
		case <-timeoutCh:
			s.logger.Info().Msg("sequencer recovery: P2P catchup timeout reached, ignoring P2P status")
			ignoreP2P = true
			timeoutCh = nil
		case <-ticker.C:
			// Check DA caught up
			daCaughtUp := syncState.bc.Syncer != nil && syncState.bc.Syncer.HasReachedDAHead()

			// Check P2P caught up: store height >= best known height from P2P
			storeHeight, err := s.store.Height(ctx)
			if err != nil {
				s.logger.Warn().Err(err).Msg("failed to get store height during recovery")
				continue
			}

			maxP2PHeight := max(
				syncState.headerSyncService.Store().Height(),
				syncState.dataSyncService.Store().Height(),
			)

			p2pCaughtUp := ignoreP2P || (maxP2PHeight == 0 || storeHeight >= maxP2PHeight)

			if daCaughtUp && p2pCaughtUp && storeHeight > 0 {
				s.logger.Info().
					Uint64("store_height", storeHeight).
					Uint64("max_p2p_height", maxP2PHeight).
					Msg("sequencer recovery: fully caught up")
				return true, nil
			}
		}
	}
}

func (s *sequencerRecoveryElector) IsRunning() bool {
	return s.running.Load()
}

// for testing purposes only
func (s *sequencerRecoveryElector) state() *failoverState {
	return s.activeState.Load()
}
