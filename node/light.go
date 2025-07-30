package node

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	ds "github.com/ipfs/go-datastore"
	"go.uber.org/zap"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	rpcserver "github.com/evstack/ev-node/pkg/rpc/server"
	"github.com/evstack/ev-node/pkg/service"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/pkg/sync"
)

var _ Node = &LightNode{}

// LightNode is a chain node that only needs the header service
type LightNode struct {
	service.BaseService

	P2P *p2p.Client

	hSyncService *sync.HeaderSyncService
	Store        store.Store
	rpcServer    *http.Server
	nodeConfig   config.Config

	running bool
}

func newLightNode(
	conf config.Config,
	genesis genesis.Genesis,
	p2pClient *p2p.Client,
	database ds.Batching,
	logger *zap.Logger,
) (ln *LightNode, err error) {
	headerSyncService, err := sync.NewHeaderSyncService(database, conf, genesis, p2pClient, logger)
	if err != nil {
		return nil, fmt.Errorf("error while initializing HeaderSyncService: %w", err)
	}

	store := store.New(database)

	node := &LightNode{
		P2P:          p2pClient,
		hSyncService: headerSyncService,
		Store:        store,
		nodeConfig:   conf,
	}

	node.BaseService = *service.NewBaseService(logger, "LightNode", node)

	return node, nil
}

// IsRunning returns true if the node is running.
func (ln *LightNode) IsRunning() bool {
	return ln.running
}

// Run implements the Service interface.
// It starts all subservices and manages the node's lifecycle.
func (ln *LightNode) Run(parentCtx context.Context) error {
	ctx, cancelNode := context.WithCancel(parentCtx)
	defer func() {
		ln.running = false
		cancelNode()
	}()

	ln.running = true
	// Start RPC server
	handler, err := rpcserver.NewServiceHandler(ln.Store, ln.P2P, ln.Logger)
	if err != nil {
		return fmt.Errorf("error creating RPC handler: %w", err)
	}

	ln.rpcServer = &http.Server{
		Addr:         ln.nodeConfig.RPC.Address,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		ln.Logger.Info("started RPC server", zap.String("addr", ln.nodeConfig.RPC.Address))
		if err := ln.rpcServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			ln.Logger.Error("RPC server error", zap.Error(err))
		}
	}()

	if err := ln.P2P.Start(ctx); err != nil {
		return fmt.Errorf("error while starting P2P client: %w", err)
	}

	if err := ln.hSyncService.Start(ctx); err != nil {
		return fmt.Errorf("error while starting header sync service: %w", err)
	}

	// Wait for the context to be canceled
	<-parentCtx.Done()
	ln.Logger.Info("context canceled, stopping node")
	cancelNode()

	ln.Logger.Info("halting light node and its sub services...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var multiErr error

	// Stop header sync service
	if ln.hSyncService != nil {
		if err := ln.hSyncService.Stop(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
			multiErr = errors.Join(multiErr, fmt.Errorf("stopping header sync service: %w", err))
		} else {
			ln.Logger.Debug("header sync service stop context ended", zap.Error(err))
		}
	}

	// Shutdown RPC server
	if ln.rpcServer != nil {
		if err := ln.rpcServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
			multiErr = errors.Join(multiErr, fmt.Errorf("shutting down RPC server: %w", err))
		} else {
			ln.Logger.Debug("RPC server shutdown context ended", zap.Error(err))
		}
	}

	// Close P2P client
	if ln.P2P != nil {
		if err := ln.P2P.Close(); err != nil {
			multiErr = errors.Join(multiErr, fmt.Errorf("closing P2P client: %w", err))
		}
	}

	// Close store
	if err := ln.Store.Close(); err != nil {
		multiErr = errors.Join(multiErr, fmt.Errorf("closing store: %w", err))
	} else {
		ln.Logger.Debug("store closed")
	}

	if multiErr != nil {
		if unwrapper, ok := multiErr.(interface{ Unwrap() []error }); ok {
			for _, err := range unwrapper.Unwrap() {
				ln.Logger.Error("error during shutdown", zap.Error(err))
			}
		} else {
			ln.Logger.Error("error during shutdown", zap.Error(multiErr))
		}
		ln.Logger.Error("error during shutdown", zap.Error(err))
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	return multiErr
}
