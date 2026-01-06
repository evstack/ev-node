//go:build integration

package node

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	coreexecutor "github.com/evstack/ev-node/core/execution"
	evconfig "github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/test/testda"
)

// FullNodeTestSuite is a test suite for full node integration tests
type FullNodeTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
	node   *FullNode

	errCh     chan error
	runningWg sync.WaitGroup
}

// startNodeInBackground starts the given node in a background goroutine
// and adds to the wait group for proper cleanup
func (s *FullNodeTestSuite) startNodeInBackground(node *FullNode) {
	s.runningWg.Add(1)
	go func() {
		defer s.runningWg.Done()
		err := node.Run(s.ctx)
		select {
		case s.errCh <- err:
		default:
			s.T().Logf("Error channel full, discarding error: %v", err)
		}
	}()
}

// SetupTest initializes the test context, creates a test node, and starts it in the background.
// It also verifies that the node is running, producing blocks, and properly initialized.
func (s *FullNodeTestSuite) SetupTest() {
	require := require.New(s.T())
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.errCh = make(chan error, 1)

	// Setup a test node
	config := getTestConfig(s.T(), 1)

	// Add debug logging for configuration
	s.T().Logf("Test configuration: BlockTime=%v, DABlockTime=%v, MaxPendingHeadersAndData=%d",
		config.Node.BlockTime.Duration, config.DA.BlockTime.Duration, config.Node.MaxPendingHeadersAndData)

	node, cleanup := createNodeWithCleanup(s.T(), config)
	s.T().Cleanup(func() {
		cleanup()
	})

	s.node = node

	// Start the node in a goroutine using Run instead of Start
	s.startNodeInBackground(s.node)
	s.T().Cleanup(func() { shutdownAndWait(s.T(), []context.CancelFunc{s.cancel}, &s.runningWg, 10*time.Second) })

	// Verify that the node is running and producing blocks
	err := waitForFirstBlock(s.node, Header)
	require.NoError(err, "Failed to get node height")

	// Wait for the first block to be DA included
	err = waitForFirstBlockToBeDAIncluded(s.node)
	require.NoError(err, "Failed to get DA inclusion")

	// Verify block components are properly initialized
	require.NotNil(castState(s.T(), s.node).bc, "Block components should be initialized")
}

// TearDownTest cancels the test context and waits for the node to stop, ensuring proper cleanup after each test.
// It also checks for any errors that occurred during node shutdown.
func (s *FullNodeTestSuite) TearDownTest() {
	if s.cancel != nil {
		s.cancel() // Cancel context to stop the node

		// Wait for the node to stop with a timeout
		waitCh := make(chan struct{})
		go func() {
			s.runningWg.Wait()
			close(waitCh)
		}()

		select {
		case <-waitCh:
			// Node stopped successfully
		case <-time.After(10 * time.Second):
			s.T().Log("Warning: Node did not stop gracefully within timeout")
		}

		// Check for any errors
		select {
		case err := <-s.errCh:
			if err != nil && !errors.Is(err, context.Canceled) {
				s.T().Logf("Error stopping node in teardown: %v", err)
			}
		default:
			// No error
		}
	}
}

// TestFullNodeTestSuite runs the FullNodeTestSuite using testify's suite runner.
// This is the entry point for running all integration tests in this suite.
func TestFullNodeTestSuite(t *testing.T) {
	suite.Run(t, new(FullNodeTestSuite))
}

// TestBlockProduction verifies block production and state after injecting a transaction.
// It checks that blocks are produced, state is updated, and the injected transaction is included in one of the blocks.
func (s *FullNodeTestSuite) TestBlockProduction() {
	testTx := []byte("test transaction")

	// Inject transaction through the node's block components (same as integration tests)
	if state := castState(s.T(), s.node); state.bc != nil && state.bc.Executor != nil {
		// Access the core executor from the block executor
		coreExec := state.bc.Executor.GetCoreExecutor()
		if dummyExec, ok := coreExec.(interface{ InjectTx([]byte) }); ok {
			dummyExec.InjectTx(testTx)
			// Notify the executor about new transactions
			state.bc.Executor.NotifyNewTransactions()
		} else {
			s.T().Fatalf("Could not cast core executor to DummyExecutor")
		}
	} else {
		s.T().Fatalf("Block components or executor not available")
	}

	// Wait for reaper to process transactions (reaper runs every 1 second by default)
	time.Sleep(1200 * time.Millisecond)

	err := waitForAtLeastNBlocks(s.node, 5, Store)
	s.NoError(err, "Failed to produce more than 5 blocks")

	// Get the current height
	height, err := s.node.Store.Height(s.ctx)
	require.NoError(s.T(), err)
	s.GreaterOrEqual(height, uint64(5), "Expected block height >= 5")

	// Verify chain state
	state, err := s.node.Store.GetState(s.ctx)
	s.NoError(err)
	s.GreaterOrEqual(height, state.LastBlockHeight)

	foundTx := false
	for h := uint64(1); h <= height; h++ {
		// Get the block data for each height
		header, data, err := s.node.Store.GetBlockData(s.ctx, h)
		s.NoError(err)
		s.NotNil(header)
		s.NotNil(data)

		// Log block details
		s.T().Logf("Block height: %d, Time: %s, Number of transactions: %d", h, header.Time(), len(data.Txs))

		// Check if testTx is in this block
		for _, tx := range data.Txs {
			if string(tx) == string(testTx) {
				foundTx = true
				break
			}
		}
	}

	// Verify at least one block contains the test transaction
	s.True(foundTx, "Expected at least one block to contain the test transaction")
}

// TestSubmitBlocksToDA verifies that blocks produced by the node are properly submitted to the Data Availability (DA) layer.
// It injects a transaction, waits for several blocks to be produced and DA-included, and asserts that all blocks are DA included.
func (s *FullNodeTestSuite) TestSubmitBlocksToDA() {
	// Inject transaction through the node's block components
	if state := castState(s.T(), s.node); state.bc != nil && state.bc.Executor != nil {
		coreExec := state.bc.Executor.GetCoreExecutor()
		if dummyExec, ok := coreExec.(interface{ InjectTx([]byte) }); ok {
			dummyExec.InjectTx([]byte("test transaction"))
			// Notify the executor about new transactions
			state.bc.Executor.NotifyNewTransactions()
		} else {
			s.T().Fatalf("Could not cast core executor to DummyExecutor")
		}
	} else {
		s.T().Fatalf("Block components or executor not available")
	}

	// Wait for reaper to process transactions (reaper runs every 1 second by default)
	time.Sleep(1200 * time.Millisecond)
	n := uint64(5)
	err := waitForAtLeastNBlocks(s.node, n, Store)
	s.NoError(err, "Failed to produce second block")
	err = waitForAtLeastNDAIncludedHeight(s.node, n)
	s.NoError(err, "Failed to get DA inclusion")
	for height := uint64(1); height <= n; height++ {
		header, data, err := s.node.Store.GetBlockData(s.ctx, height)
		require.NoError(s.T(), err)

		ok, err := castState(s.T(), s.node).bc.Submitter.IsHeightDAIncluded(height, header, data)
		require.NoError(s.T(), err)
		require.True(s.T(), ok, "Block at height %d is not DA included", height)
	}
}

// TestGenesisInitialization checks that the node's state is correctly initialized from the genesis document.
// It asserts that the initial height and chain ID in the state match those in the genesis.
func (s *FullNodeTestSuite) TestGenesisInitialization() {
	require := require.New(s.T())

	// Verify genesis state
	state := castState(s.T(), s.node).bc.GetLastState()
	require.Equal(s.node.genesis.InitialHeight, state.InitialHeight)
	require.Equal(s.node.genesis.ChainID, state.ChainID)
}

// TestStateRecovery verifies that the node can recover its state after a restart.
// It would check that the block height after restart is at least as high as before.
func TestStateRecovery(t *testing.T) {
	require := require.New(t)

	// Set up one sequencer
	config := getTestConfig(t, 1)
	executor, sequencer, dac, ds, nodeKey, stopDAHeightTicker := createTestComponents(t, config)
	node, cleanup := createNodeWithCustomComponents(t, config, executor, sequencer, dac, nodeKey, ds, stopDAHeightTicker)
	defer cleanup()

	var runningWg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the sequencer first
	startNodeInBackground(t, []*FullNode{node}, []context.Context{ctx}, &runningWg, 0, nil)
	t.Cleanup(func() { shutdownAndWait(t, []context.CancelFunc{cancel}, &runningWg, 10*time.Second) })

	blocksToWaitFor := uint64(20)
	// Wait for the sequencer to produce at first block
	require.NoError(waitForAtLeastNBlocks(node, blocksToWaitFor, Store))

	// Get current state
	originalHeight, err := getNodeHeight(node, Store)
	require.NoError(err)
	require.GreaterOrEqual(originalHeight, blocksToWaitFor)

	// Stop the current node and wait for shutdown with a timeout
	shutdownAndWait(t, []context.CancelFunc{cancel}, &runningWg, 60*time.Second)

	// Create a new node instance using the same components
	executor, sequencer, dac, _, nodeKey, stopDAHeightTicker = createTestComponents(t, config)
	node, cleanup = createNodeWithCustomComponents(t, config, executor, sequencer, dac, nodeKey, ds, stopDAHeightTicker)
	defer cleanup()

	// Verify state persistence
	recoveredHeight, err := getNodeHeight(node, Store)
	require.NoError(err)
	require.GreaterOrEqual(recoveredHeight, originalHeight, "recovered height should be greater than or equal to original height")
}

// TestMaxPendingHeadersAndData verifies that the sequencer will stop producing blocks when the maximum number of pending headers or data is reached.
// It reconfigures the node with a low max pending value, waits for block production, and checks the pending block count.
func TestMaxPendingHeadersAndData(t *testing.T) {
	require := require.New(t)
	// Reconfigure node with low max pending
	config := getTestConfig(t, 1)
	config.Node.MaxPendingHeadersAndData = 2

	// Set DA block time large enough to avoid header submission to DA layer
	config.DA.BlockTime = evconfig.DurationWrapper{Duration: 100 * time.Second}

	node, cleanup := createNodeWithCleanup(t, config)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var runningWg sync.WaitGroup
	startNodeInBackground(t, []*FullNode{node}, []context.Context{ctx}, &runningWg, 0, nil)
	t.Cleanup(func() { shutdownAndWait(t, []context.CancelFunc{cancel}, &runningWg, 10*time.Second) })

	// Wait blocks to be produced up to max pending
	numExtraBlocks := uint64(5)
	time.Sleep(time.Duration(config.Node.MaxPendingHeadersAndData+numExtraBlocks) * config.Node.BlockTime.Duration)

	// Verify that the node is not producing blocks beyond the max pending limit
	height, err := getNodeHeight(node, Store)
	require.NoError(err)
	require.LessOrEqual(height, config.Node.MaxPendingHeadersAndData)
}

// TestBatchQueueThrottlingWithDAFailure tests that when DA layer fails and MaxPendingHeadersAndData
// is reached, the system behaves correctly and doesn't run into resource exhaustion.
// This test uses the dummy sequencer but demonstrates the scenario that would occur
// with a real single sequencer having queue limits.
func TestBatchQueueThrottlingWithDAFailure(t *testing.T) {
	require := require.New(t)

	// Set up configuration with low limits to trigger throttling quickly
	config := getTestConfig(t, 1)
	config.Node.MaxPendingHeadersAndData = 3 // Low limit to quickly reach pending limit after DA failure
	config.Node.BlockTime = evconfig.DurationWrapper{Duration: 25 * time.Millisecond}
	config.DA.BlockTime = evconfig.DurationWrapper{Duration: 100 * time.Millisecond} // Longer DA time to ensure blocks are produced first

	// Create test components
	executor, sequencer, dummyDA, ds, nodeKey, stopDAHeightTicker := createTestComponents(t, config)
	defer stopDAHeightTicker()

	// Cast executor to DummyExecutor so we can inject transactions
	dummyExecutor, ok := executor.(*coreexecutor.DummyExecutor)
	require.True(ok, "Expected DummyExecutor implementation")

	// Cast dummyDA to our test double so we can simulate failures
	dummyDAImpl, ok := dummyDA.(*testda.DummyDA)
	require.True(ok, "Expected testda.DummyDA implementation")

	// Create node with components
	node, cleanup := createNodeWithCustomComponents(t, config, executor, sequencer, dummyDAImpl, nodeKey, ds, func() {})
	defer cleanup()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var runningWg sync.WaitGroup
	errChan := make(chan error, 1)
	startNodeInBackground(t, []*FullNode{node}, []context.Context{ctx}, &runningWg, 0, errChan)
	t.Cleanup(func() { shutdownAndWait(t, []context.CancelFunc{cancel}, &runningWg, 10*time.Second) })
	require.Len(errChan, 0, "Expected no errors when starting node")

	// Wait for the node to start producing blocks
	waitForBlockN(t, 1, node, config.Node.BlockTime.Duration)

	// Inject some initial transactions to get the system working
	for i := 0; i < 5; i++ {
		dummyExecutor.InjectTx([]byte(fmt.Sprintf("initial-tx-%d", i)))
	}

	waitForBlockN(t, 2, node, config.Node.BlockTime.Duration)
	t.Log("Initial blocks produced successfully")

	// Get the current height before DA failure
	initialHeight, err := getNodeHeight(node, Store)
	require.NoError(err)
	t.Logf("Height before DA failure: %d", initialHeight)

	// Simulate DA layer going down
	t.Log("Simulating DA layer failure")
	dummyDAImpl.SetSubmitFailure(true)

	// Continue injecting transactions - this tests the behavior when:
	// 1. DA layer is down (can't submit blocks to DA)
	// 2. MaxPendingHeadersAndData limit is reached (stops block production)
	// 3. Reaper continues trying to submit transactions
	// In a real single sequencer, this would fill the batch queue and eventually return ErrQueueFull
	go func() {
		for i := 0; i < 100; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				dummyExecutor.InjectTx([]byte(fmt.Sprintf("tx-after-da-failure-%d", i)))
				time.Sleep(config.Node.BlockTime.Duration / 2) // Inject faster than block time
			}
		}
	}()

	// Wait for the pending headers/data to reach the MaxPendingHeadersAndData limit
	// This should cause block production to stop
	time.Sleep(3 * config.Node.BlockTime.Duration)

	// Verify that block production has stopped due to MaxPendingHeadersAndData
	heightAfterDAFailure, err := getNodeHeight(node, Store)
	require.NoError(err)
	t.Logf("Height after DA failure: %d", heightAfterDAFailure)

	// Wait a bit more and verify height didn't increase significantly
	time.Sleep(5 * config.Node.BlockTime.Duration)
	finalHeight, err := getNodeHeight(node, Store)
	require.NoError(err)
	t.Logf("Final height: %d", finalHeight)
	cancel() // stop the node
	// The height should not have increased much due to MaxPendingHeadersAndData limit
	// Allow at most 3 additional blocks due to timing and pending blocks in queue
	heightIncrease := finalHeight - heightAfterDAFailure
	require.LessOrEqual(heightIncrease, uint64(3),
		"Height should not increase significantly when DA is down and MaxPendingHeadersAndData limit is reached")

	t.Logf("Successfully demonstrated that MaxPendingHeadersAndData prevents runaway block production when DA fails")
	t.Logf("Height progression: initial=%d, after_DA_failure=%d, final=%d",
		initialHeight, heightAfterDAFailure, finalHeight)

	// This test demonstrates the scenario described in the PR:
	// - DA layer goes down (SetSubmitFailure(true))
	// - Block production stops when MaxPendingHeadersAndData limit is reached
	// - Reaper continues injecting transactions (would fill batch queue in real single sequencer)
	// - In a real single sequencer with queue limits, this would eventually return ErrQueueFull
	//   preventing unbounded resource consumption

	t.Log("NOTE: This test uses DummySequencer. In a real deployment with SingleSequencer,")
	t.Log("the batch queue would fill up and return ErrQueueFull, providing backpressure.")
}

// waitForBlockN waits for the node to produce a block with height >= n.
func waitForBlockN(t *testing.T, n uint64, node *FullNode, blockInterval time.Duration, timeout ...time.Duration) {
	t.Helper()
	if len(timeout) == 0 {
		timeout = []time.Duration{time.Duration(n+1)*blockInterval + time.Second/2}
	}
	require.Eventually(t, func() bool {
		got, err := getNodeHeight(node, Store)
		require.NoError(t, err)
		return got >= n
	}, timeout[0], blockInterval/2)
}

func TestReadinessEndpointWhenBlockProductionStops(t *testing.T) {
	require := require.New(t)

	httpClient := &http.Client{Timeout: 1 * time.Second}

	config := getTestConfig(t, 1)
	config.Node.Aggregator = true
	config.Node.BlockTime = evconfig.DurationWrapper{Duration: 500 * time.Millisecond}
	config.Node.MaxPendingHeadersAndData = 2
	config.DA.BlockTime = evconfig.DurationWrapper{Duration: 100 * time.Second}

	node, cleanup := createNodeWithCleanup(t, config)
	defer cleanup()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var runningWg sync.WaitGroup
	errChan := make(chan error, 1)
	startNodeInBackground(t, []*FullNode{node}, []context.Context{ctx}, &runningWg, 0, errChan)
	t.Cleanup(func() { shutdownAndWait(t, []context.CancelFunc{cancel}, &runningWg, 10*time.Second) })
	require.Len(errChan, 0, "Expected no errors when starting node")

	waitForBlockN(t, 2, node, config.Node.BlockTime.Duration)
	require.Len(errChan, 0, "Expected no errors when starting node")

	require.Eventually(func() bool {
		resp, err := httpClient.Get("http://" + config.RPC.Address + "/health/ready")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 100*time.Millisecond, "Readiness should be READY while producing blocks")

	time.Sleep(time.Duration(config.Node.MaxPendingHeadersAndData+2) * config.Node.BlockTime.Duration)

	height, err := getNodeHeight(node, Store)
	require.NoError(err)
	require.LessOrEqual(height, config.Node.MaxPendingHeadersAndData)

	require.Eventually(func() bool {
		resp, err := httpClient.Get("http://" + config.RPC.Address + "/health/ready")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusServiceUnavailable
	}, 10*time.Second, 100*time.Millisecond, "Readiness should be UNREADY after aggregator stops producing blocks (5x block time)")
}
