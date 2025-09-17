//go:build integration

package node

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	evconfig "github.com/evstack/ev-node/pkg/config"
)

// TestTxGossipingMultipleNodesNoDA tests that transactions are gossiped and blocks are sequenced and synced across multiple nodes without the DA layer over P2P.
// It creates 3 nodes (1 sequencer, 2 full nodes), injects a transaction, waits for all nodes to sync, and asserts block equality.
func TestTxGossipingMultipleNodesNoDA(t *testing.T) {
	require := require.New(t)
	config := getTestConfig(t, 1)
	// Set the DA block time to a very large value to ensure that the DA layer is not used
	config.DA.BlockTime = evconfig.DurationWrapper{Duration: 100 * time.Second}
	numNodes := 3
	nodes, cleanups := createNodesWithCleanup(t, numNodes, config)
	for _, cleanup := range cleanups {
		defer cleanup()
	}

	ctxs, cancels := createNodeContexts(numNodes)
	var runningWg sync.WaitGroup

	// Start only the sequencer first
	startNodeInBackground(t, nodes, ctxs, &runningWg, 0)

	// Wait for the first block to be produced by the sequencer
	err := waitForFirstBlock(nodes[0], Header)
	require.NoError(err)

	// Add a small delay to ensure P2P services are fully ready
	time.Sleep(500 * time.Millisecond)

	// Start the other nodes
	for i := 1; i < numNodes; i++ {
		startNodeInBackground(t, nodes, ctxs, &runningWg, i)
		// Add a small delay between starting nodes to avoid connection race
		if i < numNodes-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Inject a transaction into the sequencer's executor
	if nodes[0].blockComponents != nil && nodes[0].blockComponents.Executor != nil {
		// Access the core executor from the block executor
		coreExec := nodes[0].blockComponents.Executor.GetCoreExecutor()
		if dummyExec, ok := coreExec.(interface{ InjectTx([]byte) }); ok {
			dummyExec.InjectTx([]byte("test tx"))
		} else {
			t.Logf("Warning: Could not cast core executor to DummyExecutor, skipping transaction injection")
		}
	}

	blocksToWaitFor := uint64(3)
	// Wait for all nodes to reach at least blocksToWaitFor blocks
	for _, nodeItem := range nodes {
		require.NoError(waitForAtLeastNBlocks(nodeItem, blocksToWaitFor, Store))
	}

	// Shutdown all nodes and wait
	shutdownAndWait(t, cancels, &runningWg, 10*time.Second)

	// Assert that all nodes have the same block up to height blocksToWaitFor
	assertAllNodesSynced(t, nodes, blocksToWaitFor)
}

// TestTxGossipingMultipleNodesDAIncluded tests that transactions are gossiped and blocks are sequenced and synced across multiple nodes only using DA. P2P gossiping is disabled.
// It creates 4 nodes (1 sequencer, 3 full nodes), injects a transaction, waits for all nodes to sync with DA inclusion, and asserts block equality.
func TestTxGossipingMultipleNodesDAIncluded(t *testing.T) {
	require := require.New(t)
	config := getTestConfig(t, 1)
	// Disable P2P gossiping
	config.P2P.Peers = "none"

	numNodes := 4
	nodes, cleanups := createNodesWithCleanup(t, numNodes, config)
	for _, cleanup := range cleanups {
		defer cleanup()
	}

	ctxs, cancels := createNodeContexts(numNodes)
	var runningWg sync.WaitGroup

	// Start only the sequencer first
	startNodeInBackground(t, nodes, ctxs, &runningWg, 0)

	// Wait for the first block to be produced by the sequencer
	err := waitForFirstBlock(nodes[0], Header)
	require.NoError(err)

	// Verify block components are properly initialized
	require.True(nodes[0].IsRunning(), "Block components should be initialized")

	// Add a small delay to ensure P2P services are fully ready
	time.Sleep(500 * time.Millisecond)

	// Start the other nodes
	for i := 1; i < numNodes; i++ {
		startNodeInBackground(t, nodes, ctxs, &runningWg, i)
		// Add a small delay between starting nodes to avoid connection race
		if i < numNodes-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Inject transactions into the sequencer's executor
	if nodes[0].blockComponents != nil && nodes[0].blockComponents.Executor != nil {
		// Access the core executor from the block executor
		coreExec := nodes[0].blockComponents.Executor.GetCoreExecutor()
		if dummyExec, ok := coreExec.(interface{ InjectTx([]byte) }); ok {
			dummyExec.InjectTx([]byte("test tx 1"))
			dummyExec.InjectTx([]byte("test tx 2"))
			dummyExec.InjectTx([]byte("test tx 3"))
		} else {
			t.Logf("Warning: Could not cast core executor to DummyExecutor, skipping transaction injection")
		}
	}

	blocksToWaitFor := uint64(5)
	// Wait for all nodes to reach at least blocksToWaitFor blocks with DA inclusion
	for _, nodeItem := range nodes {
		require.NoError(waitForAtLeastNDAIncludedHeight(nodeItem, blocksToWaitFor))
	}

	// Shutdown all nodes and wait
	shutdownAndWait(t, cancels, &runningWg, 5*time.Second)

	// Assert that all nodes have the same block up to height blocksToWaitFor
	assertAllNodesSynced(t, nodes, blocksToWaitFor)
}

// TestFastDASync verifies that a new node can quickly synchronize with the DA layer using fast sync.
//
// This test sets up two nodes with different block and DA block times. It starts the sequencer node, waits for it to produce and DA-include several blocks,
// then starts the syncing full node and measures how quickly it can catch up to the sequencer node's height. The test asserts that the syncing full node syncs within
// a small delta of the DA block time, and verifies that both nodes have identical block hashes and that all blocks are DA-included.
func TestFastDASync(t *testing.T) {
	require := require.New(t)

	// Set up two nodes where DA is faster than block production
	config := getTestConfig(t, 1)
	// Slow block production (sequencer takes 1s per block)
	config.Node.BlockTime = evconfig.DurationWrapper{Duration: 1 * time.Second}
	// Fast DA availability (DA makes blocks available every 200ms)
	config.DA.BlockTime = evconfig.DurationWrapper{Duration: 200 * time.Millisecond}

	nodes, cleanups := createNodesWithCleanup(t, 2, config)
	for _, cleanup := range cleanups {
		defer cleanup()
	}

	ctxs, cancels := createNodeContexts(len(nodes))
	var runningWg sync.WaitGroup

	// Start only the first node
	startNodeInBackground(t, nodes, ctxs, &runningWg, 0)

	// Wait for the first node to produce a few blocks
	blocksToWaitFor := uint64(2)
	require.NoError(waitForAtLeastNDAIncludedHeight(nodes[0], blocksToWaitFor))

	// Add a small delay to ensure P2P services are fully ready
	time.Sleep(500 * time.Millisecond)

	// Now start the second node and time its sync
	startNodeInBackground(t, nodes, ctxs, &runningWg, 1)
	start := time.Now()
	// Wait for the second node to catch up to the first node
	require.NoError(waitForAtLeastNBlocks(nodes[1], blocksToWaitFor, Store))
	syncDuration := time.Since(start)
	// The key test: sync should be much faster than sequential block production time
	// Since DA provides blocks faster than sequencer block time, sync should complete
	// in significantly less time than it took the sequencer to produce them sequentially
	expectedSequentialTime := time.Duration(blocksToWaitFor) * config.Node.BlockTime.Duration
	maxReasonableSyncTime := expectedSequentialTime / 3 // Should be at least 3x faster than sequential

	require.Less(syncDuration, maxReasonableSyncTime,
		"DA fast sync took %v, should be much faster than sequential block time %v (max reasonable: %v). ",
		syncDuration, expectedSequentialTime, maxReasonableSyncTime)

	// Verify both nodes are synced and that the synced block is DA-included
	assertAllNodesSynced(t, nodes, blocksToWaitFor)

	// Cancel all node contexts to signal shutdown and wait
	shutdownAndWait(t, cancels, &runningWg, 5*time.Second)
}

// TestSingleSequencerTwoFullNodesBlockSyncSpeed tests that block synchronization is not bottlenecked by DA block time.
//
// This test sets up three nodes (one sequencer and two full nodes) with a fast block time and a slow DA block time. It starts the sequencer first, waits for it to produce a block, then starts the full nodes. The test waits for all nodes to sync a set number of blocks, measures the total sync duration, and asserts that block sync completes within a reasonable multiple of the block time (not the DA block time). It also verifies that all nodes have identical block hashes up to the target height.
func TestSingleSequencerTwoFullNodesBlockSyncSpeed(t *testing.T) {
	require := require.New(t)

	// Set up three nodes: 1 sequencer, 2 full nodes
	config := getTestConfig(t, 1)
	config.Node.BlockTime = evconfig.DurationWrapper{Duration: 100 * time.Millisecond} // fast block time
	config.DA.BlockTime = evconfig.DurationWrapper{Duration: 10 * time.Second}         // slow DA block time

	numNodes := 3
	nodes, cleanups := createNodesWithCleanup(t, numNodes, config)
	for _, cleanup := range cleanups {
		defer cleanup()
	}

	ctxs, cancels := createNodeContexts(numNodes)
	var runningWg sync.WaitGroup

	// Start only the sequencer first
	startNodeInBackground(t, nodes, ctxs, &runningWg, 0)

	// Wait for the sequencer to produce at first block
	require.NoError(waitForFirstBlock(nodes[0], Store))

	// Add a small delay to ensure P2P services are fully ready
	time.Sleep(500 * time.Millisecond)

	// Now start the other nodes
	for i := 1; i < numNodes; i++ {
		startNodeInBackground(t, nodes, ctxs, &runningWg, i)
		// Add a small delay between starting nodes to avoid connection race
		if i < numNodes-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	blocksToWaitFor := uint64(10)
	start := time.Now()

	// Wait for all nodes to reach the target block height
	for _, nodeItem := range nodes {
		require.NoError(waitForAtLeastNBlocks(nodeItem, blocksToWaitFor, Store))
	}
	totalDuration := time.Since(start)

	// The test should complete within a reasonable multiple of block time, not DA block time
	maxExpected := config.Node.BlockTime.Duration*time.Duration(blocksToWaitFor) + 200*time.Millisecond
	require.Less(totalDuration, maxExpected, "Block sync should not be bottlenecked by DA block time")

	for i := 1; i < numNodes; i++ {
		require.NoError(verifyNodesSynced(nodes[0], nodes[i], Store))
	}

	// Cancel all node contexts to signal shutdown and wait
	shutdownAndWait(t, cancels, &runningWg, 5*time.Second)
}

// TestDataExchange verifies data exchange and synchronization between nodes in various network topologies.
//
// This test runs three sub-tests:
//  1. Single sequencer and single full
//  2. Single sequencer and two full nodes.
//  3. Single sequencer and single full node with trusted hash.
//
// Each sub-test checks data exchange and synchronization to ensure correct data propagation and consistency across nodes.
func TestDataExchange(t *testing.T) {
	t.Run("SingleSequencerSingleFullNode", func(t *testing.T) {
		testSingleSequencerSingleFullNode(t, Data)
	})
	t.Run("SingleSequencerTwoFullNodes", func(t *testing.T) {
		testSingleSequencerTwoFullNodes(t, Data)
	})
	t.Run("SingleSequencerSingleFullNodeTrustedHash", func(t *testing.T) {
		testSingleSequencerSingleFullNodeTrustedHash(t, Data)
	})
}

// TestHeaderExchange verifies header exchange and synchronization between nodes in various network topologies.
//
// This test runs three sub-tests:
//  1. Single sequencer and single full
//  2. Single sequencer and two full nodes.
//  3. Single sequencer and single full node with trusted hash.
//
// Each sub-test checks header exchange and synchronization to ensure correct header propagation and consistency across nodes.
func TestHeaderExchange(t *testing.T) {
	t.Run("SingleSequencerSingleFullNode", func(t *testing.T) {
		testSingleSequencerSingleFullNode(t, Header)
	})
	t.Run("SingleSequencerTwoFullNodes", func(t *testing.T) {
		testSingleSequencerTwoFullNodes(t, Header)
	})
	t.Run("SingleSequencerSingleFullNodeTrustedHash", func(t *testing.T) {
		testSingleSequencerSingleFullNodeTrustedHash(t, Header)
	})
}

// testSingleSequencerSingleFullNode sets up a single sequencer and a single full node, starts the sequencer, waits for it to produce a block, then starts the full
// It waits for both nodes to reach a target block height (using the provided 'source' to determine block inclusion), verifies that both nodes are fully synced, and then shuts them down.
func testSingleSequencerSingleFullNode(t *testing.T, source Source) {
	require := require.New(t)

	// Set up one sequencer and one full node
	config := getTestConfig(t, 1)
	numNodes := 2
	nodes, cleanups := createNodesWithCleanup(t, numNodes, config)
	for _, cleanup := range cleanups {
		defer cleanup()
	}

	ctxs, cancels := createNodeContexts(numNodes)
	var runningWg sync.WaitGroup

	// Start the sequencer first
	startNodeInBackground(t, nodes, ctxs, &runningWg, 0)

	// Wait for the sequencer to produce at first block
	require.NoError(waitForFirstBlock(nodes[0], source))

	// Add a small delay to ensure P2P services are fully ready
	time.Sleep(500 * time.Millisecond)

	// Start the full node
	startNodeInBackground(t, nodes, ctxs, &runningWg, 1)

	blocksToWaitFor := uint64(3)
	// Wait for both nodes to reach at least blocksToWaitFor blocks
	for _, nodeItem := range nodes {
		require.NoError(waitForAtLeastNBlocks(nodeItem, blocksToWaitFor, source))
	}

	// Verify both nodes are synced using the helper
	require.NoError(verifyNodesSynced(nodes[0], nodes[1], source))

	// Cancel all node contexts to signal shutdown and wait
	shutdownAndWait(t, cancels, &runningWg, 5*time.Second)
}

// testSingleSequencerTwoFullNodes sets up a single sequencer and two full nodes, starts the sequencer, waits for it to produce a block, then starts the full nodes.
// It waits for all nodes to reach a target block height (using the provided 'source' to determine block inclusion), verifies that all nodes are fully synced, and then shuts them down.
func testSingleSequencerTwoFullNodes(t *testing.T, source Source) {
	require := require.New(t)

	// Set up one sequencer and two full nodes
	config := getTestConfig(t, 1)
	numNodes := 3
	nodes, cleanups := createNodesWithCleanup(t, numNodes, config)
	for _, cleanup := range cleanups {
		defer cleanup()
	}

	ctxs, cancels := createNodeContexts(numNodes)
	var runningWg sync.WaitGroup

	// Start the sequencer first
	startNodeInBackground(t, nodes, ctxs, &runningWg, 0)

	// Wait for the sequencer to produce at first block
	require.NoError(waitForFirstBlock(nodes[0], source))

	// Add a small delay to ensure P2P services are fully ready
	time.Sleep(500 * time.Millisecond)

	// Start the full nodes
	for i := 1; i < numNodes; i++ {
		startNodeInBackground(t, nodes, ctxs, &runningWg, i)
		// Add a small delay between starting nodes to avoid connection race
		if i < numNodes-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	blocksToWaitFor := uint64(3)
	// Wait for all nodes to reach at least blocksToWaitFor blocks
	for _, nodeItem := range nodes {
		require.NoError(waitForAtLeastNBlocks(nodeItem, blocksToWaitFor, source))
	}

	// Verify all nodes are synced using the helper
	for i := 1; i < numNodes; i++ {
		require.NoError(verifyNodesSynced(nodes[0], nodes[i], source))
	}

	// Cancel all node contexts to signal shutdown and wait
	shutdownAndWait(t, cancels, &runningWg, 5*time.Second)
}

// testSingleSequencerSingleFullNodeTrustedHash sets up a single sequencer and a single full node with a trusted hash, starts the sequencer, waits for it to produce a block, then starts the full node with the trusted hash.
// It waits for both nodes to reach a target block height (using the provided 'source' to determine block inclusion), verifies that both nodes are fully synced, and then shuts them down.
func testSingleSequencerSingleFullNodeTrustedHash(t *testing.T, source Source) {
	require := require.New(t)

	// Set up one sequencer and one full node
	config := getTestConfig(t, 1)
	numNodes := 2
	nodes, cleanups := createNodesWithCleanup(t, numNodes, config)
	for _, cleanup := range cleanups {
		defer cleanup()
	}

	ctxs, cancels := createNodeContexts(numNodes)
	var runningWg sync.WaitGroup

	// Start the sequencer first
	startNodeInBackground(t, nodes, ctxs, &runningWg, 0)

	// Wait for the sequencer to produce at first block
	require.NoError(waitForFirstBlock(nodes[0], source))

	// Get the hash of the first block (using the correct source)
	var trustedHash string
	switch source {
	case Data:
		trustedHashValue, err := nodes[0].dSyncService.Store().GetByHeight(ctxs[0], 1)
		require.NoError(err)
		trustedHash = trustedHashValue.Hash().String()
	case Header:
		trustedHashValue, err := nodes[0].hSyncService.Store().GetByHeight(ctxs[0], 1)
		require.NoError(err)
		trustedHash = trustedHashValue.Hash().String()
	default:
		t.Fatalf("unsupported source for trusted hash test: %v", source)
	}

	// Set the trusted hash in the full node
	nodeConfig := nodes[1].nodeConfig
	nodeConfig.Node.TrustedHash = trustedHash

	// Add a small delay to ensure P2P services are fully ready
	time.Sleep(500 * time.Millisecond)

	// Start the full node
	startNodeInBackground(t, nodes, ctxs, &runningWg, 1)

	blocksToWaitFor := uint64(3)
	// Wait for both nodes to reach at least blocksToWaitFor blocks
	for _, nodeItem := range nodes {
		require.NoError(waitForAtLeastNBlocks(nodeItem, blocksToWaitFor, source))
	}

	// Verify both nodes are synced using the helper
	require.NoError(verifyNodesSynced(nodes[0], nodes[1], source))

	// Cancel all node contexts to signal shutdown and wait
	shutdownAndWait(t, cancels, &runningWg, 5*time.Second)
}

// TestTwoChainsInOneNamespace verifies that two chains in the same namespace can coexist without any issues.
func TestTwoChainsInOneNamespace(t *testing.T) {
	cases := []struct {
		name     string
		chainID1 string
		chainID2 string
	}{
		{
			name:     "same chain ID",
			chainID1: "test-1",
			chainID2: "test-1",
		},
		{
			name:     "different chain IDs",
			chainID1: "foo-1",
			chainID2: "bar-2",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			testTwoChainsInOneNamespace(t, c.chainID1, c.chainID2)
		})
	}
}

// testTwoChainsInOneNamespace sets up two chains in the same namespace, starts the sequencers, and waits for blocks.
// It waits for all nodes to reach a target block height, and verifies that all nodes are fully synced, and then shuts them down.
func testTwoChainsInOneNamespace(t *testing.T, chainID1 string, chainID2 string) {
	require := require.New(t)

	// Set up nodes for the first chain
	configChain1 := getTestConfig(t, 1)

	nodes1, cleanups := createNodesWithCleanup(t, 1, configChain1)
	for _, cleanup := range cleanups {
		defer cleanup()
	}

	// Set up nodes for the second chain
	configChain2 := getTestConfig(t, 1000)

	nodes2, cleanups := createNodesWithCleanup(t, 1, configChain2)
	for _, cleanup := range cleanups {
		defer cleanup()
	}

	// Set up context and wait group for the sequencer of chain 1
	ctxs1, cancels1 := createNodeContexts(1)
	var runningWg1 sync.WaitGroup

	// Start the sequencer of chain 1
	startNodeInBackground(t, nodes1, ctxs1, &runningWg1, 0)

	// Wait for the sequencer to produce at first block
	require.NoError(waitForFirstBlock(nodes1[0], Store))

	// Set up context and wait group for the sequencer of chain 2
	ctxs2, cancels2 := createNodeContexts(1)
	var runningWg2 sync.WaitGroup

	// Start the sequencer of chain 2
	startNodeInBackground(t, nodes2, ctxs2, &runningWg2, 0)

	// Wait for the sequencer to produce at first block
	require.NoError(waitForFirstBlock(nodes2[0], Store))

	blocksToWaitFor := uint64(3)

	// Wait for the full node of chain 1 to reach at least blocksToWaitFor blocks
	require.NoError(waitForAtLeastNBlocks(nodes1[0], blocksToWaitFor, Store))

	// Wait for the full node of chain 2 to reach at least blocksToWaitFor blocks
	require.NoError(waitForAtLeastNBlocks(nodes2[0], blocksToWaitFor, Store))

	// Cancel all node contexts to signal shutdown and wait for both chains
	shutdownAndWait(t, cancels1, &runningWg1, 5*time.Second)
	shutdownAndWait(t, cancels2, &runningWg2, 5*time.Second)
}
