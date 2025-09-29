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
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/store"

	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/p2p/key"
	rpcserver "github.com/evstack/ev-node/pkg/rpc/server"
	evsync "github.com/evstack/ev-node/pkg/sync"
	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"
)

type xxxx struct {
	p2pClient         *p2p.Client
	headerSyncService *evsync.HeaderSyncService
	dataSyncService   *evsync.DataSyncService
	rpcServer         *http.Server
	bc                *block.Components
	logger            zerolog.Logger
}

func NewSyncX(
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
) (*xxxx, error) {
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
	return new(nodeConfig, nodeKey, database, genesis, logger, mainKV, rktStore, blockComponentsFn)
}
func NewAggregatorX(
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
) (*xxxx, error) {

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

	return new(nodeConfig, nodeKey, database, genesis, logger, mainKV, rktStore, blockComponentsFn)
}

func new(
	nodeConfig config.Config,
	nodeKey *key.NodeKey,
	database ds.Batching,
	genesis genesispkg.Genesis,
	logger zerolog.Logger,
	mainKV ds.Batching,
	rktStore store.Store,
	yyy func(headerSyncService *evsync.HeaderSyncService, dataSyncService *evsync.DataSyncService) (*block.Components, error),
) (*xxxx, error) {
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

	return &xxxx{
		logger:            logger,
		p2pClient:         p2pClient,
		headerSyncService: headerSyncService,
		dataSyncService:   dataSyncService,
		rpcServer:         rpcServer,
		bc:                bc,
	}, nil
}
func (x *xxxx) Run(ctx context.Context) error {
	go func() {
		x.logger.Info().Str("addr", x.rpcServer.Addr).Msg("started RPC server")
		if err := x.rpcServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			x.logger.Error().Err(err).Msg("RPC server error")
		}
	}()
	defer x.rpcServer.Shutdown(context.Background()) // nolint: errcheck

	if err := x.p2pClient.Start(ctx); err != nil {
		return err
	}
	defer x.p2pClient.Close() // nolint: errcheck

	if err := x.headerSyncService.Start(ctx); err != nil {
		return fmt.Errorf("error while starting header sync service: %w", err)
	}
	defer x.headerSyncService.Stop(context.Background()) // nolint: errcheck

	if err := x.dataSyncService.Start(ctx); err != nil {
		return fmt.Errorf("error while starting data sync service: %w", err)
	}
	defer x.dataSyncService.Stop(context.Background()) // nolint: errcheck

	defer x.bc.Stop() // nolint: errcheck
	if err := x.bc.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("leader components started with error: %w", err)
	}
	return nil

}
