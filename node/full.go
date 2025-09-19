package node

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	ds "github.com/ipfs/go-datastore"
	ktds "github.com/ipfs/go-datastore/keytransform"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block"

	coreda "github.com/evstack/ev-node/core/da"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	rpcserver "github.com/evstack/ev-node/pkg/rpc/server"
	"github.com/evstack/ev-node/pkg/service"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/store"
	evsync "github.com/evstack/ev-node/pkg/sync"
)

// prefixes used in KV store to separate rollkit data from execution environment data (if the same data base is reused)
var EvPrefix = "0"

const (
	// genesisChunkSize is the maximum size, in bytes, of each
	// chunk in the genesis structure for the chunked API
	genesisChunkSize = 16 * 1024 * 1024 // 16 MiB
)

var _ Node = &FullNode{}

// FullNode represents a client node in Rollkit network.
// It connects all the components and orchestrates their work.
type FullNode struct {
	service.BaseService

	genesis genesispkg.Genesis
	// cache of chunked genesis data.
	genChunks []string

	nodeConfig config.Config

	da coreda.DA

	p2pClient       *p2p.Client
	hSyncService    *evsync.HeaderSyncService
	dSyncService    *evsync.DataSyncService
	Store           store.Store
	blockComponents *block.Components

	prometheusSrv *http.Server
	pprofSrv      *http.Server
	rpcServer     *http.Server
}

// newFullNode creates a new Rollkit full node.
func newFullNode(
	nodeConfig config.Config,
	p2pClient *p2p.Client,
	signer signer.Signer,
	genesis genesispkg.Genesis,
	database ds.Batching,
	exec coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	da coreda.DA,
	metricsProvider MetricsProvider,
	logger zerolog.Logger,
	nodeOpts NodeOptions,
) (fn *FullNode, err error) {
	logger.Debug().Bytes("address", genesis.ProposerAddress).Msg("Proposer address")

	blockMetrics, _ := metricsProvider(genesis.ChainID)

	mainKV := newPrefixKV(database, EvPrefix)
	rktStore := store.New(mainKV)

	headerSyncService, err := initHeaderSyncService(mainKV, nodeConfig, genesis, p2pClient, logger)
	if err != nil {
		return nil, err
	}

	dataSyncService, err := initDataSyncService(mainKV, nodeConfig, genesis, p2pClient, logger)
	if err != nil {
		return nil, err
	}

	var blockComponents *block.Components
	if nodeConfig.Node.Aggregator {
		blockComponents, err = block.NewAggregatorComponents(
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
	} else {
		blockComponents, err = block.NewSyncComponents(
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
	if err != nil {
		return nil, err
	}

	node := &FullNode{
		genesis:         genesis,
		nodeConfig:      nodeConfig,
		p2pClient:       p2pClient,
		blockComponents: blockComponents,
		da:              da,
		Store:           rktStore,
		hSyncService:    headerSyncService,
		dSyncService:    dataSyncService,
	}

	node.BaseService = *service.NewBaseService(logger, "Node", node)

	return node, nil
}

func initHeaderSyncService(
	mainKV ds.Batching,
	nodeConfig config.Config,
	genesis genesispkg.Genesis,
	p2pClient *p2p.Client,
	logger zerolog.Logger,
) (*evsync.HeaderSyncService, error) {
	headerSyncService, err := evsync.NewHeaderSyncService(mainKV, nodeConfig, genesis, p2pClient, logger.With().Str("component", "HeaderSyncService").Logger())
	if err != nil {
		return nil, fmt.Errorf("error while initializing HeaderSyncService: %w", err)
	}
	return headerSyncService, nil
}

func initDataSyncService(
	mainKV ds.Batching,
	nodeConfig config.Config,
	genesis genesispkg.Genesis,
	p2pClient *p2p.Client,
	logger zerolog.Logger,
) (*evsync.DataSyncService, error) {
	dataSyncService, err := evsync.NewDataSyncService(mainKV, nodeConfig, genesis, p2pClient, logger.With().Str("component", "DataSyncService").Logger())
	if err != nil {
		return nil, fmt.Errorf("error while initializing DataSyncService: %w", err)
	}
	return dataSyncService, nil
}

// initGenesisChunks creates a chunked format of the genesis document to make it easier to
// iterate through larger genesis structures.
func (n *FullNode) initGenesisChunks() error {
	if n.genChunks != nil {
		return nil
	}

	data, err := json.Marshal(n.genesis)
	if err != nil {
		return err
	}

	for i := 0; i < len(data); i += genesisChunkSize {
		end := min(i+genesisChunkSize, len(data))

		n.genChunks = append(n.genChunks, base64.StdEncoding.EncodeToString(data[i:end]))
	}

	return nil
}

// startInstrumentationServer starts HTTP servers for instrumentation (Prometheus metrics and pprof).
// Returns the primary server (Prometheus if enabled, otherwise pprof) and optionally a secondary server.
func (n *FullNode) startInstrumentationServer() (*http.Server, *http.Server) {
	var prometheusServer, pprofServer *http.Server

	// Check if Prometheus is enabled
	if n.nodeConfig.Instrumentation.IsPrometheusEnabled() {
		prometheusMux := http.NewServeMux()

		// Register Prometheus metrics handler
		prometheusMux.Handle("/metrics", promhttp.InstrumentMetricHandler(
			prometheus.DefaultRegisterer, promhttp.HandlerFor(
				prometheus.DefaultGatherer,
				promhttp.HandlerOpts{MaxRequestsInFlight: n.nodeConfig.Instrumentation.MaxOpenConnections},
			),
		))

		prometheusServer = &http.Server{
			Addr:              n.nodeConfig.Instrumentation.PrometheusListenAddr,
			Handler:           prometheusMux,
			ReadHeaderTimeout: readHeaderTimeout,
		}

		go func() {
			if err := prometheusServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				n.Logger.Error().Err(err).Msg("Prometheus HTTP server ListenAndServe")
			}
		}()

		n.Logger.Info().Str("addr", n.nodeConfig.Instrumentation.PrometheusListenAddr).Msg("Started Prometheus HTTP server")
	}

	// Check if pprof is enabled
	if n.nodeConfig.Instrumentation.IsPprofEnabled() {
		pprofMux := http.NewServeMux()

		// Register pprof handlers
		pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
		pprofMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		pprofMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		pprofMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		pprofMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		// Register other pprof handlers
		pprofMux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		pprofMux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		pprofMux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
		pprofMux.Handle("/debug/pprof/block", pprof.Handler("block"))
		pprofMux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
		pprofMux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))

		pprofServer = &http.Server{
			Addr:              n.nodeConfig.Instrumentation.GetPprofListenAddr(),
			Handler:           pprofMux,
			ReadHeaderTimeout: readHeaderTimeout,
		}

		go func() {
			if err := pprofServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				n.Logger.Error().Err(err).Msg("pprof HTTP server ListenAndServe")
			}
		}()

		n.Logger.Info().Str("addr", n.nodeConfig.Instrumentation.GetPprofListenAddr()).Msg("Started pprof HTTP server")
	}

	// Return the primary server (for backward compatibility) and the secondary server
	if prometheusServer != nil {
		return prometheusServer, pprofServer
	}
	return pprofServer, nil
}

// Run implements the Service interface.
// It starts all subservices and manages the node's lifecycle.
func (n *FullNode) Run(parentCtx context.Context) error {
	ctx, cancelNode := context.WithCancel(parentCtx)
	defer cancelNode() // safety net

	// begin prometheus metrics gathering if it is enabled
	if n.nodeConfig.Instrumentation != nil &&
		(n.nodeConfig.Instrumentation.IsPrometheusEnabled() || n.nodeConfig.Instrumentation.IsPprofEnabled()) {
		n.prometheusSrv, n.pprofSrv = n.startInstrumentationServer()
	}

	// Start RPC server
	bestKnownHeightProvider := func() uint64 {
		hHeight := n.hSyncService.Store().Height()
		dHeight := n.dSyncService.Store().Height()
		return min(hHeight, dHeight)
	}

	handler, err := rpcserver.NewServiceHandler(n.Store, n.p2pClient, n.genesis.ProposerAddress, n.Logger, n.nodeConfig, bestKnownHeightProvider)
	if err != nil {
		return fmt.Errorf("error creating RPC handler: %w", err)
	}

	n.rpcServer = &http.Server{
		Addr:         n.nodeConfig.RPC.Address,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		n.Logger.Info().Str("addr", n.nodeConfig.RPC.Address).Msg("started RPC server")
		if err := n.rpcServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			n.Logger.Error().Err(err).Msg("RPC server error")
		}
	}()

	n.Logger.Info().Msg("starting P2P client")
	err = n.p2pClient.Start(ctx)
	if err != nil {
		return fmt.Errorf("error while starting P2P client: %w", err)
	}

	if err = n.hSyncService.Start(ctx); err != nil {
		return fmt.Errorf("error while starting header sync service: %w", err)
	}

	if err = n.dSyncService.Start(ctx); err != nil {
		return fmt.Errorf("error while starting data sync service: %w", err)
	}

	// Start the block components (blocking)
	if err := n.blockComponents.Start(ctx); err != nil {
		if !errors.Is(err, context.Canceled) {
			n.Logger.Error().Err(err).Msg("unrecoverable error in block components")
		} else {
			n.Logger.Info().Msg("context canceled, stopping node")
		}
	}

	// blocking components start exited, propagate shutdown to all other processes
	cancelNode()
	n.Logger.Info().Msg("halting full node and its sub services...")

	// Use a timeout context to ensure shutdown doesn't hang
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 9*time.Second)
	defer cancel()

	var multiErr error // Use a multierror variable

	// Stop block components
	if err := n.blockComponents.Stop(); err != nil {
		n.Logger.Error().Err(err).Msg("error stopping block components")
		multiErr = errors.Join(multiErr, fmt.Errorf("stopping block components: %w", err))
	}

	// Stop Header Sync Service
	err = n.hSyncService.Stop(shutdownCtx)
	if err != nil {
		// Log context canceled errors at a lower level if desired, or handle specific non-cancel errors
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			n.Logger.Error().Err(err).Msg("error stopping header sync service")
			multiErr = errors.Join(multiErr, fmt.Errorf("stopping header sync service: %w", err))
		} else {
			n.Logger.Debug().Err(err).Msg("header sync service stop context ended") // Log cancellation as debug
		}
	}

	// Stop Data Sync Service
	err = n.dSyncService.Stop(shutdownCtx)
	if err != nil {
		// Log context canceled errors at a lower level if desired, or handle specific non-cancel errors
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			n.Logger.Error().Err(err).Msg("error stopping data sync service")
			multiErr = errors.Join(multiErr, fmt.Errorf("stopping data sync service: %w", err))
		} else {
			n.Logger.Debug().Err(err).Msg("data sync service stop context ended") // Log cancellation as debug
		}
	}

	// Stop P2P Client
	err = n.p2pClient.Close()
	if err != nil {
		multiErr = errors.Join(multiErr, fmt.Errorf("closing P2P client: %w", err))
	}

	// Shutdown Prometheus Server
	if n.prometheusSrv != nil {
		err = n.prometheusSrv.Shutdown(shutdownCtx)
		// http.ErrServerClosed is expected on graceful shutdown
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			multiErr = errors.Join(multiErr, fmt.Errorf("shutting down Prometheus server: %w", err))
		} else {
			n.Logger.Debug().Err(err).Msg("Prometheus server shutdown context ended")
		}
	}

	// Shutdown Pprof Server
	if n.pprofSrv != nil {
		err = n.pprofSrv.Shutdown(shutdownCtx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			multiErr = errors.Join(multiErr, fmt.Errorf("shutting down pprof server: %w", err))
		} else {
			n.Logger.Debug().Err(err).Msg("pprof server shutdown context ended")
		}
	}

	// Shutdown RPC Server
	if n.rpcServer != nil {
		err = n.rpcServer.Shutdown(shutdownCtx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			multiErr = errors.Join(multiErr, fmt.Errorf("shutting down RPC server: %w", err))
		} else {
			n.Logger.Debug().Err(err).Msg("RPC server shutdown context ended")
		}
	}

	// Ensure Store.Close is called last to maximize chance of data flushing
	if err = n.Store.Close(); err != nil {
		multiErr = errors.Join(multiErr, fmt.Errorf("closing store: %w", err))
	} else {
		n.Logger.Debug().Msg("store closed")
	}

	// Save caches if needed
	if n.blockComponents != nil && n.blockComponents.Cache != nil {
		if err := n.blockComponents.Cache.SaveToDisk(); err != nil {
			multiErr = errors.Join(multiErr, fmt.Errorf("saving caches: %w", err))
		} else {
			n.Logger.Debug().Msg("caches saved")
		}
	}

	// Log final status
	if multiErr != nil {
		for _, err := range multiErr.(interface{ Unwrap() []error }).Unwrap() {
			n.Logger.Error().Err(err).Msg("error during shutdown")
		}
	} else {
		n.Logger.Info().Msg("full node halted successfully")
	}

	// Return the original context error if it exists (e.g., context cancelled)
	// or the combined shutdown error if the context cancellation was clean.
	if ctx.Err() != nil {
		return ctx.Err()
	}

	return multiErr // Return shutdown errors if context was okay
}

// GetGenesis returns entire genesis doc.
func (n *FullNode) GetGenesis() genesispkg.Genesis {
	return n.genesis
}

// GetGenesisChunks returns chunked version of genesis.
func (n *FullNode) GetGenesisChunks() ([]string, error) {
	err := n.initGenesisChunks()
	if err != nil {
		return nil, err
	}
	return n.genChunks, nil
}

// IsRunning returns true if the node is running.
func (n *FullNode) IsRunning() bool {
	return n.blockComponents != nil
}

func newPrefixKV(kvStore ds.Batching, prefix string) ds.Batching {
	return ktds.Wrap(kvStore, ktds.PrefixTransform{Prefix: ds.NewKey(prefix)})
}
