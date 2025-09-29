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

	"github.com/evstack/ev-node/pkg/p2p/key"
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
	"github.com/evstack/ev-node/pkg/lease"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/service"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/store"
)

// EvPrefix prefixes used in KV store to separate rollkit data from execution environment data (if the same data base is reused)
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

	Store                   store.Store
	blockComponents         *block.Components
	leaderElection          lease.LeaderElector
	SyncComponentFunc       func(c context.Context) error
	AggregatorComponentFunc func(c context.Context) error

	prometheusSrv *http.Server
	pprofSrv      *http.Server
}

// newFullNode creates a new Rollkit full node.
func newFullNode(
	nodeConfig config.Config,
	nodeKey *key.NodeKey,
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

	// Initialize leader election if enabled
	var leaderElection lease.LeaderElector
	if nodeConfig.LeaderElection.Enabled {
		nodeID, err := p2p.ClientID(nodeKey.PubKey)
		if err != nil {
			return nil, fmt.Errorf("error getting node id: %w", err)
		}
		leaseName := nodeConfig.LeaderElection.LeaseName
		if leaseName == "" {
			leaseName = fmt.Sprintf("leader-%s", genesis.ChainID)
		}

		leaseTerm := 30 * time.Second // default
		if nodeConfig.LeaderElection.LeaseTerm.Duration > 0 {
			leaseTerm = nodeConfig.LeaderElection.LeaseTerm.Duration
		}

		// Create lease backend based on configuration
		var leaseImpl lease.Lease
		switch nodeConfig.LeaderElection.Backend {
		case "memory", "":
			leaseImpl = lease.NewMemoryLease(leaseName)
		case "http":
			if nodeConfig.LeaderElection.BackendAddr == "" {
				return nil, fmt.Errorf("http lease backend requires backend_addr")
			}
			leaseImpl = lease.NewHTTPLease(nodeConfig.LeaderElection.BackendAddr, leaseName)
		default:
			return nil, fmt.Errorf("unsupported lease backend: %s", nodeConfig.LeaderElection.Backend)
		}
		logger.Info().Str("backend", nodeConfig.LeaderElection.Backend).Msg("using leader election backend")
		leaderElection = lease.NewLeaderElection(
			leaseImpl,
			nodeID,
			leaseName,
			leaseTerm,
			logger.With().Str("component", "leader-election").Logger(),
		)
	} else {
		leaderElection = &lease.AlwaysLeaderElection{}
	}

	node := &FullNode{
		genesis:    genesis,
		nodeConfig: nodeConfig,
		//p2pClient:  nodeKey,
		da:    da,
		Store: rktStore,
		//hSyncService:   headerSyncService,
		//dSyncService:   dataSyncService,
		leaderElection: leaderElection,
		// block components will be constructed dynamically based on leadership state
	}

	node.BaseService = *service.NewBaseService(logger, "Node", node)

	node.AggregatorComponentFunc = func(ctx context.Context) error {
		logger.Info().Msg("ALEX: Starting aggregator-MODE")

		nodeConfig.Node.Aggregator = true
		nodeConfig.P2P.Peers = ""

		x, err := NewAggregatorX(nodeConfig, nodeKey, signer, genesis, database, exec, sequencer, da, logger, rktStore, mainKV, blockMetrics, nodeOpts)
		if err != nil {
			return err
		}
		return x.Run(ctx)
	}

	node.SyncComponentFunc = func(ctx context.Context) error {
		logger.Info().Msg("ALEX: Starting sync-MODE")
		nodeConfig.Node.Aggregator = false
		x, err := NewSyncX(nodeConfig, nodeKey, genesis, database, exec, da, logger, rktStore, mainKV, blockMetrics, nodeOpts)
		if err != nil {
			return err
		}
		return x.Run(ctx)
	}

	return node, nil
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

	// Start leader election if enabled
	if err := n.leaderElection.Start(ctx); err != nil {
		return fmt.Errorf("error while starting leader election: %w", err)
	}

	if err := n.leaderElection.RunWithElection(ctx, n.AggregatorComponentFunc, n.SyncComponentFunc); err != nil && !errors.Is(err, context.Canceled) {
		n.Logger.Warn().Err(err).Msg("leadership change detected, restarting components")
	}

	// blocking components start exited, propagate shutdown to all other processes
	cancelNode()
	n.Logger.Info().Msg("halting full node and its sub services...")

	// Use a timeout context to ensure shutdown doesn't hang
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 9*time.Second)
	defer cancel()

	var multiErr error // Use a multierror variable

	// Stop leader election
	if err := n.leaderElection.Stop(); err != nil && !errors.Is(err, context.Canceled) {
		multiErr = errors.Join(multiErr, fmt.Errorf("stopping leader election: %w", err))
	} else {
		n.Logger.Debug().Msg("leader election stopped")
	}

	// Shutdown Prometheus Server
	if n.prometheusSrv != nil {
		err := n.prometheusSrv.Shutdown(shutdownCtx)
		// http.ErrServerClosed is expected on graceful shutdown
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			multiErr = errors.Join(multiErr, fmt.Errorf("shutting down Prometheus server: %w", err))
		} else {
			n.Logger.Debug().Err(err).Msg("Prometheus server shutdown context ended")
		}
	}

	// Shutdown Pprof Server
	if n.pprofSrv != nil {
		err := n.pprofSrv.Shutdown(shutdownCtx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			multiErr = errors.Join(multiErr, fmt.Errorf("shutting down pprof server: %w", err))
		} else {
			n.Logger.Debug().Err(err).Msg("pprof server shutdown context ended")
		}
	}

	// Ensure Store.Close is called last to maximize chance of data flushing
	if err := n.Store.Close(); err != nil {
		multiErr = errors.Join(multiErr, fmt.Errorf("closing store: %w", err))
	} else {
		n.Logger.Debug().Msg("store closed")
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
