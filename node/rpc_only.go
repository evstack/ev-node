package node

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	rpcserver "github.com/evstack/ev-node/pkg/rpc/server"
	"github.com/evstack/ev-node/pkg/service"
	"github.com/evstack/ev-node/pkg/store"
)

var _ Node = &RPCOnlyNode{}

// RPCOnlyNode is a chain node that only provides RPC endpoints
// without block production or synchronization capabilities.
// This mode keeps the RPC server available for debugging even 
// when the main node processes halt due to errors.
type RPCOnlyNode struct {
	service.BaseService

	P2P        *p2p.Client
	Store      store.Store
	rpcServer  *http.Server
	nodeConfig config.Config
	running    bool
}

func newRPCOnlyNode(
	conf config.Config,
	genesis genesis.Genesis,
	p2pClient *p2p.Client,
	database ds.Batching,
	logger zerolog.Logger,
) (rn *RPCOnlyNode, err error) {
	store := store.New(database)

	node := &RPCOnlyNode{
		P2P:        p2pClient,
		Store:      store,
		nodeConfig: conf,
	}

	node.BaseService = *service.NewBaseService(logger, "RPCOnlyNode", node)

	return node, nil
}

// IsRunning returns true if the node is running.
func (rn *RPCOnlyNode) IsRunning() bool {
	return rn.running
}

// Run implements the Service interface.
// It starts only the RPC server without any block production or syncing.
func (rn *RPCOnlyNode) Run(parentCtx context.Context) error {
	ctx, cancelNode := context.WithCancel(parentCtx)
	defer func() {
		rn.running = false
		cancelNode()
	}()

	rn.running = true
	rn.Logger.Info().Msg("starting RPC-only node mode - no block production or syncing")

	// Start RPC server
	handler, err := rpcserver.NewServiceHandler(rn.Store, rn.P2P, rn.Logger)
	if err != nil {
		return fmt.Errorf("error creating RPC handler: %w", err)
	}

	rn.rpcServer = &http.Server{
		Addr:         rn.nodeConfig.RPC.Address,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		rn.Logger.Info().Str("addr", rn.nodeConfig.RPC.Address).Msg("started RPC server")
		if err := rn.rpcServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			rn.Logger.Error().Err(err).Msg("RPC server error")
		}
	}()

	// Start minimal P2P for connectivity (but no syncing services)
	if err := rn.P2P.Start(ctx); err != nil {
		return fmt.Errorf("error while starting P2P client: %w", err)
	}

	// Wait for shutdown signal - no background processing loops
	<-parentCtx.Done()
	rn.Logger.Info().Msg("context canceled, stopping RPC-only node")
	cancelNode()

	rn.Logger.Info().Msg("halting RPC-only node and its services...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var multiErr error

	// Shutdown RPC Server
	if rn.rpcServer != nil {
		err = rn.rpcServer.Shutdown(shutdownCtx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			multiErr = errors.Join(multiErr, fmt.Errorf("shutting down RPC server: %w", err))
		} else {
			rn.Logger.Debug().Msg("RPC server shutdown completed")
		}
	}

	// Stop P2P Client
	err = rn.P2P.Close()
	if err != nil {
		multiErr = errors.Join(multiErr, fmt.Errorf("closing P2P client: %w", err))
	}

	if err = rn.Store.Close(); err != nil {
		multiErr = errors.Join(multiErr, fmt.Errorf("closing store: %w", err))
	} else {
		rn.Logger.Debug().Msg("store closed")
	}

	// Log final status
	if multiErr != nil {
		for _, err := range multiErr.(interface{ Unwrap() []error }).Unwrap() {
			rn.Logger.Error().Err(err).Msg("error during shutdown")
		}
	} else {
		rn.Logger.Info().Msg("RPC-only node halted successfully")
	}

	return multiErr
}