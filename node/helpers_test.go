//nolint:unused
package node

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	testutils "github.com/celestiaorg/utils/test"
	"github.com/evstack/ev-node/block"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/test/testda"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	evconfig "github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/p2p"
	remote_signer "github.com/evstack/ev-node/pkg/signer/noop"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

const (
	// TestDAAddress is the address used by the dummy gRPC service
	// NOTE: this should be unique per test package to avoid
	// "bind: listen address already in use" because multiple packages
	// are tested in parallel
	TestDAAddress = "grpc://localhost:7990"

	// TestDANamespace is a sample namespace used by the dummy DA client
	TestDANamespace = "mock-namespace"

	// MockExecutorAddress is a sample address used by the mock executor
	MockExecutorAddress = "127.0.0.1:40041"
)

// sharedDummyDA is a shared DummyDA instance for multi-node tests.
var (
	sharedDummyDA     *testda.DummyDA
	sharedDummyDAOnce sync.Once
)

func getSharedDummyDA(maxBlobSize uint64) *testda.DummyDA {
	sharedDummyDAOnce.Do(func() {
		sharedDummyDA = testda.New(testda.WithMaxBlobSize(maxBlobSize))
	})
	return sharedDummyDA
}

func resetSharedDummyDA() {
	if sharedDummyDA != nil {
		sharedDummyDA.Reset()
	}
}

func newDummyDAClient(maxBlobSize uint64) *testda.DummyDA {
	if maxBlobSize == 0 {
		maxBlobSize = testda.DefaultMaxBlobSize
	}
	return getSharedDummyDA(maxBlobSize)
}

func createTestComponents(t *testing.T, config evconfig.Config) (coreexecutor.Executor, coresequencer.Sequencer, block.DAClient, crypto.PrivKey, datastore.Batching, func()) {
	executor := coreexecutor.NewDummyExecutor()
	sequencer := coresequencer.NewDummySequencer()
	daClient := newDummyDAClient(0)

	// Create genesis and keys for P2P client
	_, genesisValidatorKey, _ := types.GetGenesisWithPrivkey("test-chain")
	ds, err := store.NewTestInMemoryKVStore()
	require.NoError(t, err)

	stop := daClient.StartHeightTicker(config.DA.BlockTime.Duration)
	return executor, sequencer, daClient, genesisValidatorKey, ds, stop
}

func getTestConfig(t *testing.T, n int) evconfig.Config {
	// Use a higher base port to reduce chances of conflicts with system services
	startPort := 40000 // Spread port ranges further apart
	return evconfig.Config{
		RootDir: t.TempDir(),
		Node: evconfig.NodeConfig{
			Aggregator:               true,
			BlockTime:                evconfig.DurationWrapper{Duration: 100 * time.Millisecond},
			MaxPendingHeadersAndData: 1000,
			LazyBlockInterval:        evconfig.DurationWrapper{Duration: 5 * time.Second},
			ScrapeInterval:           evconfig.DurationWrapper{Duration: time.Second},
		},
		DA: evconfig.DAConfig{
			BlockTime:         evconfig.DurationWrapper{Duration: 200 * time.Millisecond},
			Address:           TestDAAddress,
			Namespace:         TestDANamespace,
			MaxSubmitAttempts: 30,
		},
		P2P: evconfig.P2PConfig{
			ListenAddress: fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", startPort+n),
		},
		RPC: evconfig.RPCConfig{
			Address: fmt.Sprintf("127.0.0.1:%d", 8000+n),
		},
		Instrumentation: &evconfig.InstrumentationConfig{},
	}
}

// newTestNode is a private helper that creates a node and returns it with a unified cleanup function.
func newTestNode(
	t *testing.T,
	config evconfig.Config,
	executor coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	daClient block.DAClient,
	privKey crypto.PrivKey,
	ds datastore.Batching,
	stopDAHeightTicker func(),
) (*FullNode, func()) {
	// Generate genesis and keys
	genesis, genesisValidatorKey, _ := types.GetGenesisWithPrivkey("test-chain")
	remoteSigner, err := remote_signer.NewNoopSigner(genesisValidatorKey)
	require.NoError(t, err)

	logger := zerolog.Nop()
	if testing.Verbose() {
		logger = zerolog.New(zerolog.NewTestWriter(t))
	}

	p2pClient, err := newTestP2PClient(config, privKey, ds, genesis.ChainID, logger)
	require.NoError(t, err)

	node, err := NewNode(
		config,
		executor,
		sequencer,
		daClient,
		remoteSigner,
		p2pClient,
		genesis,
		ds,
		DefaultMetricsProvider(evconfig.DefaultInstrumentationConfig()),
		logger,
		NodeOptions{},
	)
	require.NoError(t, err)

	cleanup := func() {
		if stopDAHeightTicker != nil {
			stopDAHeightTicker()
		}
	}

	return node.(*FullNode), cleanup
}

func createNodeWithCleanup(t *testing.T, config evconfig.Config) (*FullNode, func()) {
	resetSharedDummyDA()
	executor, sequencer, daClient, privKey, ds, stopDAHeightTicker := createTestComponents(t, config)
	return newTestNode(t, config, executor, sequencer, daClient, privKey, ds, stopDAHeightTicker)
}

func createNodeWithCustomComponents(
	t *testing.T,
	config evconfig.Config,
	executor coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	daClient block.DAClient,
	privKey crypto.PrivKey,
	ds datastore.Batching,
	stopDAHeightTicker func(),
) (*FullNode, func()) {
	return newTestNode(t, config, executor, sequencer, daClient, privKey, ds, stopDAHeightTicker)
}

// Creates the given number of nodes the given nodes using the given wait group to synchronize them
func createNodesWithCleanup(t *testing.T, num int, config evconfig.Config) ([]*FullNode, []func()) {
	t.Helper()
	require := require.New(t)
	resetSharedDummyDA()

	nodes := make([]*FullNode, num)
	cleanups := make([]func(), num)

	// Generate genesis and keys
	genesis, genesisValidatorKey, _ := types.GetGenesisWithPrivkey("test-chain")
	remoteSigner, err := remote_signer.NewNoopSigner(genesisValidatorKey)
	require.NoError(err)

	aggListenAddress := config.P2P.ListenAddress
	aggPeers := config.P2P.Peers
	executor, sequencer, daClient, aggPrivKey, ds, stopDAHeightTicker := createTestComponents(t, config)
	if d, ok := daClient.(*testda.DummyDA); ok {
		d.Reset()
	}
	aggPeerID, err := peer.IDFromPrivateKey(aggPrivKey)
	require.NoError(err)

	logger := zerolog.Nop()
	if testing.Verbose() {
		logger = zerolog.New(zerolog.NewTestWriter(t))
	}

	aggP2PClient, err := newTestP2PClient(config, aggPrivKey, ds, genesis.ChainID, logger)
	require.NoError(err)

	aggNode, err := NewNode(
		config,
		executor,
		sequencer,
		daClient,
		remoteSigner,
		aggP2PClient,
		genesis,
		ds,
		DefaultMetricsProvider(evconfig.DefaultInstrumentationConfig()),
		logger,
		NodeOptions{},
	)
	require.NoError(err)

	// Update cleanup to cancel the context instead of calling Stop
	cleanup := func() {
		stopDAHeightTicker()
	}

	nodes[0], cleanups[0] = aggNode.(*FullNode), cleanup
	config.Node.Aggregator = false
	peersList := []string{}
	if aggPeers != "none" {
		aggPeerAddress := fmt.Sprintf("%s/p2p/%s", aggListenAddress, aggPeerID.Loggable()["peerID"].(string))
		peersList = append(peersList, aggPeerAddress)
	}
	for i := 1; i < num; i++ {
		if aggPeers != "none" {
			config.P2P.Peers = strings.Join(peersList, ",")
		}
		config.P2P.ListenAddress = fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 40001+i)
		config.RPC.Address = fmt.Sprintf("127.0.0.1:%d", 8001+i)
		executor, sequencer, daClient, nodePrivKey, ds, stopDAHeightTicker := createTestComponents(t, config)
		stopDAHeightTicker()

		nodeP2PClient, err := newTestP2PClient(config, nodePrivKey, ds, genesis.ChainID, logger)
		require.NoError(err)

		node, err := NewNode(
			config,
			executor,
			sequencer,
			daClient,
			nil,
			nodeP2PClient,
			genesis,
			ds,
			DefaultMetricsProvider(evconfig.DefaultInstrumentationConfig()),
			logger,
			NodeOptions{},
		)
		require.NoError(err)
		cleanup := func() {
			// No-op: ticker already stopped
		}
		nodes[i], cleanups[i] = node.(*FullNode), cleanup
		nodePeerID, err := peer.IDFromPrivateKey(nodePrivKey)
		require.NoError(err)
		peersList = append(peersList, fmt.Sprintf("%s/p2p/%s", config.P2P.ListenAddress, nodePeerID.Loggable()["peerID"].(string)))
	}

	return nodes, cleanups
}

// newTestP2PClient creates a p2p.Client for testing.
func newTestP2PClient(config evconfig.Config, privKey crypto.PrivKey, ds datastore.Batching, chainID string, logger zerolog.Logger) (*p2p.Client, error) {
	return p2p.NewClient(config.P2P, privKey, ds, chainID, logger, nil)
}

// Helper to create N contexts and cancel functions
func createNodeContexts(n int) ([]context.Context, []context.CancelFunc) {
	ctxs := make([]context.Context, n)
	cancels := make([]context.CancelFunc, n)
	for i := range n {
		ctx, cancel := context.WithCancel(context.Background())
		ctxs[i] = ctx
		cancels[i] = cancel
	}
	return ctxs, cancels
}

// Helper to start a single node in a goroutine and add to wait group
func startNodeInBackground(t *testing.T, nodes []*FullNode, ctxs []context.Context, wg *sync.WaitGroup, idx int, errChan chan<- error) {
	wg.Add(1)
	go func(node *FullNode, ctx context.Context, idx int) {
		defer wg.Done()
		err := node.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			if errChan != nil {
				errChan <- err
			} else {
				t.Logf("Error running node %d: %v", idx, err)
			}
		}
	}(nodes[idx], ctxs[idx], idx)
}

// Helper to cancel all contexts and wait for goroutines with timeout
func shutdownAndWait[T ~func()](t *testing.T, cancels []T, wg *sync.WaitGroup, timeout time.Duration) {
	for _, cancel := range cancels {
		cancel()
	}
	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()
	select {
	case <-waitCh:
		// Nodes stopped successfully
	case <-time.After(timeout):
		t.Log("Warning: Not all nodes stopped gracefully within timeout")
	}
}

// Helper to check that all nodes are synced up to a given height (all block hashes match for all heights up to maxHeight)
func assertAllNodesSynced(t *testing.T, nodes []*FullNode, maxHeight uint64) {
	t.Helper()
	for height := uint64(1); height <= maxHeight; height++ {
		var refHash []byte
		for i, node := range nodes {
			header, _, err := node.Store.GetBlockData(context.Background(), height)
			require.NoError(t, err)
			if i == 0 {
				refHash = header.Hash()
			} else {
				headerHash := header.Hash()
				require.EqualValues(t, refHash, headerHash, "Block hash mismatch at height %d between node 0 and node %d", height, i)
			}
		}
	}
}

func verifyNodesSynced(node1, syncingNode Node, source Source) error {
	return testutils.Retry(300, 100*time.Millisecond, func() error {
		sequencerHeight, err := getNodeHeight(node1, source)
		if err != nil {
			return err
		}
		syncingHeight, err := getNodeHeight(syncingNode, source)
		if err != nil {
			return err
		}
		if sequencerHeight >= syncingHeight {
			return nil
		}
		return fmt.Errorf("nodes not synced: sequencer at height %v, syncing node at height %v", sequencerHeight, syncingHeight)
	})
}
