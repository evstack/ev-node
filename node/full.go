package node

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block"

	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p/key"
	raftpkg "github.com/evstack/ev-node/pkg/raft"
	"github.com/evstack/ev-node/pkg/service"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/store"
)

const (
	// genesisChunkSize is the maximum size, in bytes, of each
	// chunk in the genesis structure for the chunked API
	genesisChunkSize = 16 * 1024 * 1024 // 16 MiB
)

var _ Node = &FullNode{}

type leaderElection interface {
	Run(ctx context.Context) error
	IsRunning() bool
}

// FullNode represents a client node in Rollkit network.
// It connects all the components and orchestrates their work.
type FullNode struct {
	service.BaseService

	genesis genesispkg.Genesis
	// cache of chunked genesis data.
	genChunks []string

	nodeConfig config.Config

	daClient block.DAClient

	Store    store.Store
	raftNode *raftpkg.Node

	prometheusSrv  *http.Server
	pprofSrv       *http.Server
	leaderElection leaderElection
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
	daClient block.DAClient,
	metricsProvider MetricsProvider,
	logger zerolog.Logger,
	nodeOpts NodeOptions,
) (fn *FullNode, err error) {
	logger.Debug().Hex("address", genesis.ProposerAddress).Msg("Proposer address")

	blockMetrics, _ := metricsProvider(genesis.ChainID)

	mainKV := store.NewEvNodeKVStore(database)
	baseStore := store.New(mainKV)

	// Wrap with cached store for LRU caching of headers and block data
	cachedStore, err := store.NewCachedStore(baseStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create cached store: %w", err)
	}

	var evstore store.Store = cachedStore
	if nodeConfig.Instrumentation.IsTracingEnabled() {
		evstore = store.WithTracingStore(cachedStore)
	}

	var raftNode *raftpkg.Node
	if nodeConfig.Node.Aggregator && nodeConfig.Raft.Enable {
		raftNode, err = initRaftNode(nodeConfig, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize raft node: %w", err)
		}
	}

	leaderFactory := func() (raftpkg.Runnable, error) {
		logger.Info().Msg("Starting aggregator-MODE")
		nodeConfig.Node.Aggregator = true
		nodeConfig.P2P.Peers = "" // peers are not supported in aggregator mode
		return newAggregatorMode(nodeConfig, nodeKey, signer, genesis, database, evstore, exec, sequencer, daClient, logger, evstore, mainKV, blockMetrics, nodeOpts, raftNode)
	}
	followerFactory := func() (raftpkg.Runnable, error) {
		logger.Info().Msg("Starting sync-MODE")
		nodeConfig.Node.Aggregator = false
		return newSyncMode(nodeConfig, nodeKey, genesis, database, evstore, exec, daClient, logger, evstore, mainKV, blockMetrics, nodeOpts, raftNode)
	}

	// Initialize raft node if enabled (for both aggregator and sync nodes)
	var leaderElection leaderElection
	switch {
	case nodeConfig.Node.Aggregator && nodeConfig.Raft.Enable:
		leaderElection = raftpkg.NewDynamicLeaderElection(logger, leaderFactory, followerFactory, raftNode)
	case nodeConfig.Node.Aggregator && !nodeConfig.Raft.Enable:
		if leaderElection, err = newSingleRoleElector(leaderFactory); err != nil {
			return nil, err
		}
	case !nodeConfig.Node.Aggregator && !nodeConfig.Raft.Enable:
		if leaderElection, err = newSingleRoleElector(followerFactory); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("raft config must be used in sequencer setup only")
	}

	node := &FullNode{
		genesis:        genesis,
		nodeConfig:     nodeConfig,
		daClient:       daClient,
		Store:          evstore,
		leaderElection: leaderElection,
		raftNode:       raftNode,
	}

	node.BaseService = *service.NewBaseService(logger, "Node", node)

	return node, nil
}

func initRaftNode(nodeConfig config.Config, logger zerolog.Logger) (*raftpkg.Node, error) {
	raftCfg := &raftpkg.Config{
		NodeID:             nodeConfig.Raft.NodeID,
		RaftAddr:           nodeConfig.Raft.RaftAddr,
		RaftDir:            nodeConfig.Raft.RaftDir,
		Bootstrap:          nodeConfig.Raft.Bootstrap,
		SnapCount:          nodeConfig.Raft.SnapCount,
		SendTimeout:        nodeConfig.Raft.SendTimeout,
		HeartbeatTimeout:   nodeConfig.Raft.HeartbeatTimeout,
		LeaderLeaseTimeout: nodeConfig.Raft.LeaderLeaseTimeout,
	}

	if nodeConfig.Raft.Peers != "" {
		raftCfg.Peers = strings.Split(nodeConfig.Raft.Peers, ",")
	}
	raftNode, err := raftpkg.NewNode(raftCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("create raft node: %w", err)
	}

	logger.Info().
		Str("node_id", nodeConfig.Raft.NodeID).
		Str("addr", nodeConfig.Raft.RaftAddr).
		Bool("bootstrap", nodeConfig.Raft.Bootstrap).
		Msg("initialized raft node")

	return raftNode, nil
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
	// Start leader election
	if n.raftNode != nil {
		if err := n.raftNode.Start(ctx); err != nil {
			return fmt.Errorf("error while starting leader election: %w", err)
		}
	}

	var runtimeErr error
	if err := n.leaderElection.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		runtimeErr = err
	}

	// blocking components start exited, propagate shutdown to all other processes
	cancelNode()
	n.Logger.Info().Msg("halting full node and its sub services...")

	// Use a timeout context to ensure shutdown doesn't hang
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 9*time.Second)
	defer cancel()

	var shutdownMultiErr error // Variable to accumulate multiple errors

	// Stop leader election
	if n.raftNode != nil {
		if err := n.raftNode.Stop(); err != nil && !errors.Is(err, context.Canceled) {
			shutdownMultiErr = errors.Join(shutdownMultiErr, fmt.Errorf("stopping leader election: %w", err))
		} else {
			n.Logger.Debug().Msg("leader election stopped")
		}
	}

	// Shutdown Prometheus Server
	if n.prometheusSrv != nil {
		err := n.prometheusSrv.Shutdown(shutdownCtx)
		// http.ErrServerClosed is expected on graceful shutdown
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			shutdownMultiErr = errors.Join(shutdownMultiErr, fmt.Errorf("shutting down Prometheus server: %w", err))
		} else {
			n.Logger.Debug().Err(err).Msg("Prometheus server shutdown context ended")
		}
	}

	// Shutdown Pprof Server
	if n.pprofSrv != nil {
		err := n.pprofSrv.Shutdown(shutdownCtx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			shutdownMultiErr = errors.Join(shutdownMultiErr, fmt.Errorf("shutting down pprof server: %w", err))
		} else {
			n.Logger.Debug().Err(err).Msg("pprof server shutdown context ended")
		}
	}

	// Ensure Store.Close is called last to maximize chance of data flushing
	if err := n.Store.Close(); err != nil {
		shutdownMultiErr = errors.Join(shutdownMultiErr, fmt.Errorf("closing store: %w", err))
	} else {
		n.Logger.Debug().Msg("store closed")
	}

	// Log final status
	if shutdownMultiErr != nil {
		for _, err := range shutdownMultiErr.(interface{ Unwrap() []error }).Unwrap() {
			n.Logger.Error().Err(err).Msg("error during shutdown")
		}
	} else {
		n.Logger.Info().Msg("full node halted successfully")
	}
	if runtimeErr != nil {
		return runtimeErr
	}
	if shutdownMultiErr != nil {
		return shutdownMultiErr
	}
	return ctx.Err() // context canceled
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
	return n.leaderElection.IsRunning()
}
