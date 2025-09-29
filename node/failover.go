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
		)
	}
	return newFailoverState(nodeConfig, nodeKey, database, genesis, logger, mainKV, rktStore, blockComponentsFn)
}
func NewAggregatorMode(
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
		)
	}

	return newFailoverState(nodeConfig, nodeKey, database, genesis, logger, mainKV, rktStore, blockComponentsFn)
}

func newFailoverState(
	nodeConfig config.Config,
	nodeKey *key.NodeKey,
	database ds.Batching,
	genesis genesispkg.Genesis,
	logger zerolog.Logger,
	mainKV ds.Batching,
	rktStore store.Store,
	yyy func(headerSyncService *evsync.HeaderSyncService, dataSyncService *evsync.DataSyncService) (*block.Components, error),
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
	handler, err := rpcserver.NewServiceHandler(rktStore, p2pClient, genesis.ProposerAddress, logger, nodeConfig, bestKnownHeightProvider)
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
	bc, err := yyy(headerSyncService, dataSyncService)
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
		if err := f.bc.Stop(); err != nil && !errors.Is(err, context.Canceled) {
			multiErr = errors.Join(multiErr, fmt.Errorf("stopping block components: %w", err))
		}
	}()

	errChan := make(chan error)
	go func() {
		f.logger.Info().Str("addr", f.rpcServer.Addr).Msg("started RPC server")
		if err := f.rpcServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			select {
			case errChan <- fmt.Errorf("RPC server error: %w", err):
			default:
				f.logger.Error().Err(err).Msg("RPC server error")
			}
		}
	}()

	go func() {
		if err := f.bc.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			select {
			case errChan <- fmt.Errorf("components started with error: %w", err):
			default:
				f.logger.Error().Err(err).Msg("Components start error")
			}
		} else {
			select {
			case errChan <- nil:
			default:
			}
		}
	}()

	return <-errChan
}
