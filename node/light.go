package node

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/log"
	ds "github.com/ipfs/go-datastore"

	"github.com/rollkit/rollkit/pkg/config"
	"github.com/rollkit/rollkit/pkg/genesis"
	"github.com/rollkit/rollkit/pkg/p2p"
	"github.com/rollkit/rollkit/pkg/service"
	"github.com/rollkit/rollkit/pkg/store"
	"github.com/rollkit/rollkit/pkg/sync"
)

var _ Node = &LightNode{}

// LightNode is a rollup node that only needs the header service
type LightNode struct {
	service.BaseService

	P2P *p2p.Client

	hSyncService *sync.HeaderSyncService
}

func newLightNode(
	conf config.Config,
	genesis genesis.Genesis,
	metricsProvider MetricsProvider,
	logger log.Logger,
) (ln *LightNode, err error) {

	_, p2pMetrics := metricsProvider(genesis.ChainID)

	datastore, err := openDatastore(conf, logger)
	if err != nil {
		return nil, err
	}
	client, err := p2p.NewClient(conf, genesis.ChainID, datastore, logger.With("module", "p2p"), p2pMetrics)
	if err != nil {
		return nil, err
	}

	headerSyncService, err := sync.NewHeaderSyncService(datastore, conf, genesis, client, logger.With("module", "HeaderSyncService"))
	if err != nil {
		return nil, fmt.Errorf("error while initializing HeaderSyncService: %w", err)
	}

	node := &LightNode{
		P2P:          client,
		hSyncService: headerSyncService,
	}

	node.BaseService = *service.NewBaseService(logger, "LightNode", node)

	return node, nil
}

func openDatastore(conf config.Config, logger log.Logger) (ds.Batching, error) {
	if conf.RootDir == "" && conf.DBPath == "" {
		logger.Info("WARNING: working in in-memory mode")
		return store.NewDefaultInMemoryKVStore()
	}
	return store.NewDefaultKVStore(conf.RootDir, conf.DBPath, "rollkit-light")
}

// OnStart starts the P2P and HeaderSync services
func (ln *LightNode) OnStart(ctx context.Context) error {
	if err := ln.P2P.Start(ctx); err != nil {
		return err
	}

	if err := ln.hSyncService.Start(ctx); err != nil {
		return fmt.Errorf("error while starting header sync service: %w", err)
	}

	return nil
}

// OnStop stops the light node
func (ln *LightNode) OnStop(ctx context.Context) {
	ln.Logger.Info("halting light node...")
	err := ln.P2P.Close()
	err = errors.Join(err, ln.hSyncService.Stop(ctx))
	ln.Logger.Error("errors while stopping node:", "errors", err)
}
