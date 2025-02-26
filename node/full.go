package node

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	execproxy "github.com/rollkit/go-execution/proxy/grpc"

	ds "github.com/ipfs/go-datastore"
	ktds "github.com/ipfs/go-datastore/keytransform"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
	cmtypes "github.com/cometbft/cometbft/types"

	proxyda "github.com/rollkit/go-da/proxy"
	"github.com/rollkit/go-execution"
	seqGRPC "github.com/rollkit/go-sequencing/proxy/grpc"

	"github.com/rollkit/rollkit/block"
	"github.com/rollkit/rollkit/config"
	"github.com/rollkit/rollkit/da"
	"github.com/rollkit/rollkit/p2p"
	"github.com/rollkit/rollkit/store"
	"github.com/rollkit/rollkit/types"
)

// prefixes used in KV store to separate main node data from DALC data
var (
	mainPrefix    = "0"
	indexerPrefix = "1" // indexPrefix uses "i", so using "0-2" to avoid clash
)

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

	genesis *cmtypes.GenesisDoc
	// cache of chunked genesis data.
	genChunks []string

	nodeConfig config.NodeConfig

	eventBus     *cmtypes.EventBus
	dalc         *da.DAClient
	p2pClient    *p2p.Client
	hSyncService *block.HeaderSyncService
	dSyncService *block.DataSyncService
	Store        store.Store
	blockManager *block.Manager

	// Preserves cometBFT compatibility
	prometheusSrv *http.Server

	// keep context here only because of API compatibility
	// - it's used in `OnStart` (defined in service.Service interface)
	ctx           context.Context
	cancel        context.CancelFunc
	threadManager *types.ThreadManager
	seqClient     *seqGRPC.Client
}

// newFullNode creates a new Rollkit full node.
func newFullNode(
	ctx context.Context,
	nodeConfig config.NodeConfig,
	p2pKey crypto.PrivKey,
	signingKey crypto.PrivKey,
	genesis *cmtypes.GenesisDoc,
	metricsProvider MetricsProvider,
	logger log.Logger,
) (fn *FullNode, err error) {
	// Create context with cancel so that all services using the context can
	// catch the cancel signal when the node shutdowns
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		// If there is an error, cancel the context
		if err != nil {
			cancel()
		}
	}()

	seqMetrics, p2pMetrics := metricsProvider(genesis.ChainID)

	eventBus, err := initEventBus(logger)
	if err != nil {
		return nil, err
	}

	baseKV, err := initBaseKV(nodeConfig, logger)
	if err != nil {
		return nil, err
	}

	dalc, err := initDALC(nodeConfig, logger)
	if err != nil {
		return nil, err
	}

	p2pClient, err := p2p.NewClient(nodeConfig.P2P, p2pKey, genesis.ChainID, baseKV, logger.With("module", "p2p"), p2pMetrics)
	if err != nil {
		return nil, err
	}

	mainKV := newPrefixKV(baseKV, mainPrefix)
	headerSyncService, err := initHeaderSyncService(mainKV, nodeConfig, genesis, p2pClient, logger)
	if err != nil {
		return nil, err
	}

	dataSyncService, err := initDataSyncService(mainKV, nodeConfig, genesis, p2pClient, logger)
	if err != nil {
		return nil, err
	}

	seqClient := seqGRPC.NewClient()

	store := store.New(mainKV)

	blockManager, err := initBlockManager(signingKey, nodeConfig, genesis, store, seqClient, dalc, eventBus, logger, headerSyncService, dataSyncService, seqMetrics)
	if err != nil {
		return nil, err
	}

	node := &FullNode{
		eventBus:      eventBus,
		genesis:       genesis,
		nodeConfig:    nodeConfig,
		p2pClient:     p2pClient,
		blockManager:  blockManager,
		dalc:          dalc,
		seqClient:     seqClient,
		Store:         store,
		hSyncService:  headerSyncService,
		dSyncService:  dataSyncService,
		ctx:           ctx,
		cancel:        cancel,
		threadManager: types.NewThreadManager(),
	}

	node.BaseService = *service.NewBaseService(logger, "Node", node)

	return node, nil
}

func initEventBus(logger log.Logger) (*cmtypes.EventBus, error) {
	eventBus := cmtypes.NewEventBus()
	eventBus.SetLogger(logger.With("module", "events"))
	if err := eventBus.Start(); err != nil {
		return nil, err
	}
	return eventBus, nil
}

// initBaseKV initializes the base key-value store.
func initBaseKV(nodeConfig config.NodeConfig, logger log.Logger) (ds.TxnDatastore, error) {
	if nodeConfig.RootDir == "" && nodeConfig.DBPath == "" { // this is used for testing
		logger.Info("WARNING: working in in-memory mode")
		return store.NewDefaultInMemoryKVStore()
	}
	return store.NewDefaultKVStore(nodeConfig.RootDir, nodeConfig.DBPath, "rollkit")
}

func initDALC(nodeConfig config.NodeConfig, logger log.Logger) (*da.DAClient, error) {
	namespace := make([]byte, len(nodeConfig.DANamespace)/2)
	_, err := hex.Decode(namespace, []byte(nodeConfig.DANamespace))
	if err != nil {
		return nil, fmt.Errorf("error decoding namespace: %w", err)
	}

	if nodeConfig.DAGasMultiplier < 0 {
		return nil, fmt.Errorf("gas multiplier must be greater than or equal to zero")
	}

	client, err := proxyda.NewClient(nodeConfig.DAAddress, nodeConfig.DAAuthToken)
	if err != nil {
		return nil, fmt.Errorf("error while establishing connection to DA layer: %w", err)
	}

	var submitOpts []byte
	if nodeConfig.DASubmitOptions != "" {
		submitOpts = []byte(nodeConfig.DASubmitOptions)
	}
	return da.NewDAClient(client, nodeConfig.DAGasPrice, nodeConfig.DAGasMultiplier,
		namespace, submitOpts, logger.With("module", "da_client")), nil
}

func initHeaderSyncService(mainKV ds.TxnDatastore, nodeConfig config.NodeConfig, genesis *cmtypes.GenesisDoc, p2pClient *p2p.Client, logger log.Logger) (*block.HeaderSyncService, error) {
	headerSyncService, err := block.NewHeaderSyncService(mainKV, nodeConfig, genesis, p2pClient, logger.With("module", "HeaderSyncService"))
	if err != nil {
		return nil, fmt.Errorf("error while initializing HeaderSyncService: %w", err)
	}
	return headerSyncService, nil
}

func initDataSyncService(mainKV ds.TxnDatastore, nodeConfig config.NodeConfig, genesis *cmtypes.GenesisDoc, p2pClient *p2p.Client, logger log.Logger) (*block.DataSyncService, error) {
	dataSyncService, err := block.NewDataSyncService(mainKV, nodeConfig, genesis, p2pClient, logger.With("module", "DataSyncService"))
	if err != nil {
		return nil, fmt.Errorf("error while initializing DataSyncService: %w", err)
	}
	return dataSyncService, nil
}

func initBlockManager(signingKey crypto.PrivKey, nodeConfig config.NodeConfig, genesis *cmtypes.GenesisDoc, store store.Store, seqClient *seqGRPC.Client, dalc *da.DAClient, eventBus *cmtypes.EventBus, logger log.Logger, headerSyncService *block.HeaderSyncService, dataSyncService *block.DataSyncService, seqMetrics *block.Metrics) (*block.Manager, error) {
	exec, err := initExecutor(nodeConfig)
	if err != nil {
		return nil, fmt.Errorf("error while initializing executor: %w", err)
	}

	logger.Debug("Proposer address", "address", genesis.Validators[0].Address.Bytes())

	rollGen := &block.RollkitGenesis{
		GenesisTime:     genesis.GenesisTime,
		InitialHeight:   uint64(genesis.InitialHeight),
		ChainID:         genesis.ChainID,
		ProposerAddress: genesis.Validators[0].Address.Bytes(),
	}
	blockManager, err := block.NewManager(context.TODO(), signingKey, nodeConfig.BlockManagerConfig, rollGen, store, exec, seqClient, dalc, logger.With("module", "BlockManager"), headerSyncService.Store(), dataSyncService.Store(), seqMetrics)
	if err != nil {
		return nil, fmt.Errorf("error while initializing BlockManager: %w", err)
	}
	return blockManager, nil
}

func initExecutor(cfg config.NodeConfig) (execution.Executor, error) {
	client := execproxy.NewClient()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	err := client.Start(cfg.ExecutorAddress, opts...)
	return client, err
}

// initGenesisChunks creates a chunked format of the genesis document to make it easier to
// iterate through larger genesis structures.
func (n *FullNode) initGenesisChunks() error {
	if n.genChunks != nil {
		return nil
	}

	if n.genesis == nil {
		return nil
	}

	data, err := json.Marshal(n.genesis)
	if err != nil {
		return err
	}

	for i := 0; i < len(data); i += genesisChunkSize {
		end := i + genesisChunkSize

		if end > len(data) {
			end = len(data)
		}

		n.genChunks = append(n.genChunks, base64.StdEncoding.EncodeToString(data[i:end]))
	}

	return nil
}

func (n *FullNode) headerPublishLoop(ctx context.Context) {
	for {
		select {
		case signedHeader := <-n.blockManager.HeaderCh:
			err := n.hSyncService.WriteToStoreAndBroadcast(ctx, signedHeader)
			if err != nil {
				// failed to init or start headerstore
				n.Logger.Error(err.Error())
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (n *FullNode) dataPublishLoop(ctx context.Context) {
	for {
		select {
		case data := <-n.blockManager.DataCh:
			err := n.dSyncService.WriteToStoreAndBroadcast(ctx, data)
			if err != nil {
				// failed to init or start blockstore
				n.Logger.Error(err.Error())
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// Cancel calls the underlying context's cancel function.
func (n *FullNode) Cancel() {
	n.cancel()
}

// startPrometheusServer starts a Prometheus HTTP server, listening for metrics
// collectors on addr.
func (n *FullNode) startPrometheusServer() *http.Server {
	srv := &http.Server{
		Addr: n.nodeConfig.Instrumentation.PrometheusListenAddr,
		Handler: promhttp.InstrumentMetricHandler(
			prometheus.DefaultRegisterer, promhttp.HandlerFor(
				prometheus.DefaultGatherer,
				promhttp.HandlerOpts{MaxRequestsInFlight: n.nodeConfig.Instrumentation.MaxOpenConnections},
			),
		),
		ReadHeaderTimeout: readHeaderTimeout,
	}
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			// Error starting or closing listener:
			n.Logger.Error("Prometheus HTTP server ListenAndServe", "err", err)
		}
	}()
	return srv
}

// OnStart is a part of Service interface.
func (n *FullNode) OnStart() error {
	// begin prometheus metrics gathering if it is enabled
	if n.nodeConfig.Instrumentation != nil && n.nodeConfig.Instrumentation.IsPrometheusEnabled() {
		n.prometheusSrv = n.startPrometheusServer()
	}
	n.Logger.Info("starting P2P client")
	err := n.p2pClient.Start(n.ctx)
	if err != nil {
		return fmt.Errorf("error while starting P2P client: %w", err)
	}

	if err = n.hSyncService.Start(n.ctx); err != nil {
		return fmt.Errorf("error while starting header sync service: %w", err)
	}

	if err = n.dSyncService.Start(n.ctx); err != nil {
		return fmt.Errorf("error while starting data sync service: %w", err)
	}

	if err := n.seqClient.Start(
		n.nodeConfig.SequencerAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	); err != nil {
		return err
	}

	if n.nodeConfig.Aggregator {
		n.Logger.Info("working in aggregator mode", "block time", n.nodeConfig.BlockTime)

		n.threadManager.Go(func() { n.blockManager.BatchRetrieveLoop(n.ctx) })
		n.threadManager.Go(func() { n.blockManager.AggregationLoop(n.ctx) })
		n.threadManager.Go(func() { n.blockManager.HeaderSubmissionLoop(n.ctx) })
		n.threadManager.Go(func() { n.headerPublishLoop(n.ctx) })
		n.threadManager.Go(func() { n.dataPublishLoop(n.ctx) })
		return nil
	}
	n.threadManager.Go(func() { n.blockManager.RetrieveLoop(n.ctx) })
	n.threadManager.Go(func() { n.blockManager.HeaderStoreRetrieveLoop(n.ctx) })
	n.threadManager.Go(func() { n.blockManager.DataStoreRetrieveLoop(n.ctx) })
	n.threadManager.Go(func() { n.blockManager.SyncLoop(n.ctx, n.cancel) })
	return nil
}

// GetGenesis returns entire genesis doc.
func (n *FullNode) GetGenesis() *cmtypes.GenesisDoc {
	return n.genesis
}

// GetGenesisChunks returns chunked version of genesis.
func (n *FullNode) GetGenesisChunks() ([]string, error) {
	err := n.initGenesisChunks()
	if err != nil {
		return nil, err
	}
	return n.genChunks, err
}

// OnStop is a part of Service interface.
//
// p2pClient and sync services stop first, ceasing network activities. Then rest of services are halted.
// Context is cancelled to signal goroutines managed by thread manager to stop.
// Store is closed last because it's used by other services/goroutines.
func (n *FullNode) OnStop() {
	n.Logger.Info("halting full node...")
	n.Logger.Info("shutting down full node sub services...")
	err := errors.Join(
		n.p2pClient.Close(),
		n.hSyncService.Stop(n.ctx),
		n.dSyncService.Stop(n.ctx),
		n.seqClient.Stop(),
	)
	if n.prometheusSrv != nil {
		err = errors.Join(err, n.prometheusSrv.Shutdown(n.ctx))
	}

	n.cancel()
	n.threadManager.Wait()
	err = errors.Join(err, n.Store.Close())
	n.Logger.Error("errors while stopping node:", "errors", err)
}

// OnReset is a part of Service interface.
func (n *FullNode) OnReset() error {
	panic("OnReset - not implemented!")
}

// SetLogger sets the logger used by node.
func (n *FullNode) SetLogger(logger log.Logger) {
	n.Logger = logger
}

// GetLogger returns logger.
func (n *FullNode) GetLogger() log.Logger {
	return n.Logger
}

// EventBus gives access to Node's event bus.
func (n *FullNode) EventBus() *cmtypes.EventBus {
	return n.eventBus
}

func newPrefixKV(kvStore ds.Datastore, prefix string) ds.TxnDatastore {
	return (ktds.Wrap(kvStore, ktds.PrefixTransform{Prefix: ds.NewKey(prefix)}).Children()[0]).(ds.TxnDatastore)
}

// Start implements NodeLifecycle
func (fn *FullNode) Start() error {
	return fn.BaseService.Start()
}

// Stop implements NodeLifecycle
func (fn *FullNode) Stop() error {
	return fn.BaseService.Stop()
}

// IsRunning implements NodeLifecycle
func (fn *FullNode) IsRunning() bool {
	return fn.BaseService.IsRunning()
}
