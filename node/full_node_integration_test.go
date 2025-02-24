package node

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	mrand "math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	cmconfig "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/crypto/ed25519"
	cryptoenc "github.com/cometbft/cometbft/crypto/encoding"
	"github.com/stretchr/testify/assert"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/proxy"
	cmtypes "github.com/cometbft/cometbft/types"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	goDA "github.com/rollkit/go-da"
	damock "github.com/rollkit/go-da/mocks"
	"github.com/rollkit/rollkit/block"
	"github.com/rollkit/rollkit/config"
	"github.com/rollkit/rollkit/da"
	test "github.com/rollkit/rollkit/test/log"
	"github.com/rollkit/rollkit/test/mocks"
	"github.com/rollkit/rollkit/types"
)

const initialBlockWait = 5

func prepareProposalResponse(_ context.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	return &abci.ResponsePrepareProposal{
		Txs: req.Txs,
	}, nil
}

func TestAggregatorMode(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	app := &mocks.Application{}
	app.On("InitChain", mock.Anything, mock.Anything).Return(&abci.ResponseInitChain{}, nil)
	app.On("CheckTx", mock.Anything, mock.Anything).Return(&abci.ResponseCheckTx{}, nil)
	app.On("PrepareProposal", mock.Anything, mock.Anything).Return(prepareProposalResponse).Maybe()
	app.On("ProcessProposal", mock.Anything, mock.Anything).Return(&abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil)
	app.On("FinalizeBlock", mock.Anything, mock.Anything).Return(finalizeBlockResponse)
	app.On("Commit", mock.Anything, mock.Anything).Return(&abci.ResponseCommit{}, nil)

	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	genesisDoc, genesisValidatorKey := types.GetGenesisWithPrivkey(types.DefaultSigningKeyType, "TestAggregatorMode")
	signingKey, err := types.PrivKeyToSigningKey(genesisValidatorKey)
	require.NoError(err)
	blockManagerConfig := config.BlockManagerConfig{
		BlockTime: 1 * time.Second,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	node, err := newFullNode(ctx, config.NodeConfig{DAAddress: MockDAAddress, DANamespace: MockDANamespace, Aggregator: true, BlockManagerConfig: blockManagerConfig, SequencerAddress: MockSequencerAddress}, key, signingKey, proxy.NewLocalClientCreator(app), genesisDoc, DefaultMetricsProvider(cmconfig.DefaultInstrumentationConfig()), log.TestingLogger())
	require.NoError(err)
	require.NotNil(node)

	assert.False(node.IsRunning())

	startNodeWithCleanup(t, node)

	ctx, cancel = context.WithCancel(context.TODO())
	defer cancel()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(time.Duration(mrand.Uint32()%20) * time.Millisecond) //nolint:gosec
			}
		}
	}()
}

// TestTxGossipingAndAggregation setups a network of nodes, with single aggregator and multiple producers.
// Nodes should gossip transactions and aggregator node should produce blocks.
func TestTxGossipingAndAggregation(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	clientNodes := 4
	aggCtx := context.Background()
	ctx := context.Background()
	nodes, apps := createNodes(aggCtx, ctx, clientNodes+1, getBMConfig(), "TestTxGossipingAndAggregation", false, t)
	startNodes(nodes, apps, t)
	defer func() {
		for _, n := range nodes {
			assert.NoError(n.Stop())
		}
	}()

	// wait for nodes to start up and sync up till numBlocksToWaitFor
	// this is a guess work to make test stable and dependent on number of nodes, as each node has to send tx
	numBlocksToWaitFor := initialBlockWait + 2*len(nodes)
	for i := 1; i < len(nodes); i++ {
		require.NoError(waitForAtLeastNBlocks(nodes[i], numBlocksToWaitFor, Store))
	}

	// Cancel all the nodes before checking the calls to ABCI methods were done correctly
	// Can't stop here, because we need access to Store to test the state.
	for _, node := range nodes {
		node.Cancel()
	}

	// Now that the nodes are cancelled, it should be safe to access the mock
	// calls outside the mutex controlled methods.
	//
	// The reason we do this is because in the beginning of the test, we
	// check that we have produced at least N blocks, which means we could
	// have over produced. So when checking the calls, we also want to check
	// that we called FinalizeBlock at least N times.
	aggApp := apps[0]
	apps = apps[1:]

	checkCalls := func(app *mocks.Application, numBlocks int) error {
		calls := app.Calls
		numCalls := 0
		for _, call := range calls {
			if call.Method == "FinalizeBlock" {
				numCalls++
			}
		}
		if numBlocks > numCalls {
			return fmt.Errorf("expected at least %d calls to FinalizeBlock, got %d calls", numBlocks, numCalls)
		}
		return nil
	}

	require.NoError(checkCalls(aggApp, numBlocksToWaitFor))
	aggApp.AssertExpectations(t)

	for i, app := range apps {
		require.NoError(checkCalls(app, numBlocksToWaitFor))
		app.AssertExpectations(t)

		// assert that all blocks known to node are same as produced by aggregator
		for h := uint64(1); h <= nodes[i].Store.Height(); h++ {
			aggHeader, aggData, err := nodes[0].Store.GetBlockData(ctx, h)
			require.NoError(err)
			nodeHeader, nodeData, err := nodes[i].Store.GetBlockData(ctx, h)
			require.NoError(err)
			assert.Equal(aggHeader, nodeHeader, fmt.Sprintf("height: %d", h))
			assert.Equal(aggData, nodeData, fmt.Sprintf("height: %d", h))
		}
	}
}

func TestLazyAggregator(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	app := &mocks.Application{}
	app.On("InitChain", mock.Anything, mock.Anything).Return(&abci.ResponseInitChain{}, nil)
	app.On("CheckTx", mock.Anything, mock.Anything).Return(&abci.ResponseCheckTx{}, nil)
	app.On("PrepareProposal", mock.Anything, mock.Anything).Return(prepareProposalResponse).Maybe()
	app.On("ProcessProposal", mock.Anything, mock.Anything).Return(&abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil)
	app.On("FinalizeBlock", mock.Anything, mock.Anything).Return(finalizeBlockResponse)
	app.On("Commit", mock.Anything, mock.Anything).Return(&abci.ResponseCommit{}, nil)

	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	genesisDoc, genesisValidatorKey := types.GetGenesisWithPrivkey(types.DefaultSigningKeyType, "TestLazyAggregator")
	signingKey, err := types.PrivKeyToSigningKey(genesisValidatorKey)
	require.NoError(err)
	blockManagerConfig := config.BlockManagerConfig{
		// After the genesis header is published, the syncer is started
		// which takes little longer (due to initialization) and the syncer
		// tries to retrieve the genesis header and check that is it recent
		// (genesis header time is not older than current minus 1.5x blocktime)
		// to allow sufficient time for syncer initialization, we cannot set
		// the blocktime too short. in future, we can add a configuration
		// in go-header syncer initialization to not rely on blocktime, but the
		// config variable
		BlockTime:      1 * time.Second,
		LazyAggregator: true,
		LazyBlockTime:  5 * time.Second,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	node, err := NewNode(ctx, config.NodeConfig{
		DAAddress:          MockDAAddress,
		DANamespace:        MockDANamespace,
		Aggregator:         true,
		BlockManagerConfig: blockManagerConfig,
		SequencerAddress:   MockSequencerAddress,
	}, key, signingKey, proxy.NewLocalClientCreator(app), genesisDoc, DefaultMetricsProvider(cmconfig.DefaultInstrumentationConfig()), log.TestingLogger())
	require.NoError(err)
	assert.False(node.IsRunning())

	startNodeWithCleanup(t, node)
	require.NoError(waitForFirstBlock(node.(*FullNode), Header))

	client := node.GetClient()

	_, err = client.BroadcastTxAsync(context.Background(), []byte{0, 0, 0, 1})
	assert.NoError(err)
	require.NoError(waitForAtLeastNBlocks(node, 2, Header))

	_, err = client.BroadcastTxAsync(context.Background(), []byte{0, 0, 0, 2})
	assert.NoError(err)
	require.NoError(waitForAtLeastNBlocks(node, 3, Header))

	_, err = client.BroadcastTxAsync(context.Background(), []byte{0, 0, 0, 3})
	assert.NoError(err)

	require.NoError(waitForAtLeastNBlocks(node, 4, Header))

	// LazyBlockTime should trigger another block even without transactions
	require.NoError(waitForAtLeastNBlocks(node, 5, Header))
}

// TestFastDASync verifies that nodes can sync DA blocks faster than the DA block time
func TestFastDASync(t *testing.T) {
	// Test setup, create require and contexts for aggregator and client nodes
	require := require.New(t)
	aggCtx, aggCancel := context.WithCancel(context.Background())
	defer aggCancel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set test params
	clientNodes := 2
	bmConfig := getBMConfig()
	// Set the DABlockTime to a large value to avoid test failures due to
	// slow CI machines
	bmConfig.DABlockTime = 1 * time.Second
	// Set BlockTime to 2x DABlockTime to ensure that the aggregator node is
	// producing DA blocks faster than rollup blocks. This is to force the
	// block syncing to align with DA inclusions.
	bmConfig.BlockTime = 2 * bmConfig.DABlockTime
	const numberOfBlocksToSyncTill = 5

	// Create the 2 nodes
	nodes, _ := createNodes(aggCtx, ctx, clientNodes, bmConfig, "TestFastDASync", false, t)

	node1 := nodes[0]
	node2 := nodes[1]

	// Start node 1
	startNodeWithCleanup(t, node1)

	// Wait for node 1 to sync the first numberOfBlocksToSyncTill
	require.NoError(waitForAtLeastNBlocks(node1, numberOfBlocksToSyncTill, Store))

	// Now that node 1 has already synced, start the second node
	startNodeWithCleanup(t, node2)

	// Start and launch the timer in a go routine to ensure that the test
	// fails if the nodes do not sync before the timer expires
	ch := make(chan struct{})
	defer safeClose(ch)
	// After the first DA block time passes, the node should signal RetrieveLoop once, and it
	// should catch up to the latest block height pretty soon after.
	timer := time.NewTimer(1*bmConfig.DABlockTime + 500*time.Millisecond)
	go func() {
		select {
		case <-ch:
			// Channel closed before timer expired.
			return
		case <-timer.C:
			// Timer expired before channel closed.
			safeClose(ch)
			require.FailNow("nodes did not sync before DA Block time")
			return
		}
	}()

	// Check that the nodes are synced in a loop. We don't use the helper
	// function here so that we can catch if the channel is closed to exit
	// the test quickly.
	require.NoError(Retry(300, 100*time.Millisecond, func() error {
		select {
		case <-ch:
			require.FailNow("channel closed")
		default:
		}
		nHeight, err := getNodeHeight(node2, Store)
		if err != nil {
			return err
		}
		if nHeight >= uint64(numberOfBlocksToSyncTill) {
			return nil
		}
		return fmt.Errorf("expected height > %v, got %v", numberOfBlocksToSyncTill, nHeight)
	}))

	// Verify the nodes are synced
	require.NoError(verifyNodesSynced(node1, node2, Store))

	// Verify that the block we synced to is DA included. This is to
	// ensure that the test is passing due to the DA syncing, since the P2P
	// block sync will sync quickly but the block won't be DA included.
	header, _, err := node2.Store.GetBlockData(ctx, numberOfBlocksToSyncTill)
	require.NoError(err)
	require.True(node2.blockManager.IsDAIncluded(header.Hash()))
}

// TestChangeValSet tests the scenario where the sequencer changes and the chain is able to provide blocks by new sequencer
func TestChangeValSet(t *testing.T) {
	// clean up node data
	defer func() {
		if err := os.RemoveAll("valset_change"); err != nil {
			t.Logf("Failed to remove directory: %v", err)
		}
	}()

	assert := assert.New(t)
	require := require.New(t)

	app := &mocks.Application{}
	app.On("InitChain", mock.Anything, mock.Anything).Return(&abci.ResponseInitChain{}, nil)
	app.On("CheckTx", mock.Anything, mock.Anything).Return(func() (*abci.ResponseCheckTx, error) {
		return &abci.ResponseCheckTx{}, nil
	})
	app.On("PrepareProposal", mock.Anything, mock.Anything).Return(prepareProposalResponse).Maybe()
	app.On("ProcessProposal", mock.Anything, mock.Anything).Return(&abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil)
	app.On("Commit", mock.Anything, mock.Anything).Return(&abci.ResponseCommit{}, nil)

	// tmpubKey1
	key, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	genesisDoc, genesisValidatorKey := types.GetGenesisWithPrivkey(types.DefaultSigningKeyType, "TestChangeValSet")
	signingKey, err := types.PrivKeyToSigningKey(genesisValidatorKey)
	require.NoError(err)
	tmPubKey1, err := cryptoenc.PubKeyToProto(genesisDoc.Validators[0].PubKey)
	require.NoError(err)

	// tmpubKey2
	key2 := ed25519.GenPrivKey()
	tmPubKey2, _ := cryptoenc.PubKeyToProto(key2.PubKey())

	// update valset in height 10
	app.On("FinalizeBlock", mock.Anything, mock.Anything).Return(func(ctx context.Context, req *abci.RequestFinalizeBlock) (resp *abci.ResponseFinalizeBlock, err error) {
		if req.Height == 10 {
			return &abci.ResponseFinalizeBlock{
				ValidatorUpdates: []abci.ValidatorUpdate{
					{
						PubKey: tmPubKey1,
						Power:  0,
					},
					{
						PubKey: tmPubKey2,
						Power:  1,
					},
				},
			}, nil
		}
		return finalizeBlockResponse(ctx, req)

	})
	require.NoError(err)
	blockManagerConfig := config.BlockManagerConfig{
		// After the genesis header is published, the syncer is started
		// which takes little longer (due to initialization) and the syncer
		// tries to retrieve the genesis header and check that is it recent
		// (genesis header time is not older than current minus 1.5x blocktime)
		// to allow sufficient time for syncer initialization, we cannot set
		// the blocktime too short. in future, we can add a configuration
		// in go-header syncer initialization to not rely on blocktime, but the
		// config variable
		BlockTime:     1 * time.Second,
		LazyBlockTime: 5 * time.Second,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node1, err := NewNode(ctx, config.NodeConfig{
		DAAddress:          MockDAAddress,
		DANamespace:        MockDANamespace,
		Aggregator:         true,
		BlockManagerConfig: blockManagerConfig,
		RootDir:            "valset_change",
		SequencerAddress:   MockSequencerAddress,
	}, key, signingKey, proxy.NewLocalClientCreator(app), genesisDoc, DefaultMetricsProvider(cmconfig.DefaultInstrumentationConfig()), log.TestingLogger())
	assert.False(node1.IsRunning())
	assert.NoError(err)

	// start node 1
	require.NoError(node1.Start())
	// node will be stopped at block 10 because of the change in valset
	require.NoError(waitForAtLeastNBlocks(node1, 10, Store))

	// stop node 1
	require.NoError(node1.Stop())

	signingKey2, err := types.PrivKeyToSigningKey(key2)
	assert.NoError(err)
	node2, err := NewNode(ctx, config.NodeConfig{
		DAAddress:          MockDAAddress,
		DANamespace:        MockDANamespace,
		Aggregator:         true,
		BlockManagerConfig: blockManagerConfig,
		RootDir:            "valset_change",
		SequencerAddress:   MockSequencerAddress,
	},
		key, signingKey2, proxy.NewLocalClientCreator(app), genesisDoc, DefaultMetricsProvider(cmconfig.DefaultInstrumentationConfig()), log.TestingLogger())

	assert.False(node2.IsRunning())
	assert.NoError(err)
	// start node normally
	require.NoError(node2.Start())

	// run 10 block with the new sequencer
	require.NoError(waitForAtLeastNBlocks(node2, 10, Store))
}

// TestSingleAggregatorTwoFullNodesBlockSyncSpeed tests the scenario where the chain's block time is much faster than the DA's block time. In this case, the full nodes should be able to use block sync to sync blocks much faster than syncing from the DA layer, and the test should conclude within block time
func TestSingleAggregatorTwoFullNodesBlockSyncSpeed(t *testing.T) {
	require := require.New(t)

	aggCtx, aggCancel := context.WithCancel(context.Background())
	defer aggCancel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clientNodes := 3
	bmConfig := getBMConfig()
	bmConfig.BlockTime = 1 * time.Second
	bmConfig.DABlockTime = 25 * time.Second // takes longer to sync due to communication with sequencer
	const numberOfBlocksTSyncTill = 5

	ch := make(chan struct{})
	defer safeClose(ch)
	timer := time.NewTimer(bmConfig.DABlockTime)

	go func() {
		select {
		case <-ch:
			// Channel closed before timer expired.
			return
		case <-timer.C:
			// Timer expired before channel closed.
			t.Error("nodes did not sync before DA Block time")
			return
		}
	}()
	nodes, _ := createNodes(aggCtx, ctx, clientNodes, bmConfig, "TestSingleAggregatorTwoFullNodesBlockSyncSpeed", false, t)

	node1 := nodes[0]
	node2 := nodes[1]
	node3 := nodes[2]

	startNodeWithCleanup(t, node1)
	require.NoError(waitForFirstBlock(node1, Store))

	startNodeWithCleanup(t, node2)
	startNodeWithCleanup(t, node3)

	require.NoError(waitForAtLeastNBlocks(node2, numberOfBlocksTSyncTill, Store))
	require.NoError(waitForAtLeastNBlocks(node3, numberOfBlocksTSyncTill, Store))

	require.NoError(verifyNodesSynced(node1, node2, Store))
	require.NoError(verifyNodesSynced(node1, node3, Store))
}

func TestBlockExchange(t *testing.T) {
	t.Run("SingleAggregatorSingleFullNode", func(t *testing.T) {
		testSingleAggregatorSingleFullNode(t, Block)
	})
	t.Run("SingleAggregatorTwoFullNode", func(t *testing.T) {
		testSingleAggregatorTwoFullNode(t, Block)
	})
	t.Run("SingleAggregatorSingleFullNodeTrustedHash", func(t *testing.T) {
		testSingleAggregatorSingleFullNodeTrustedHash(t, Block)
	})
}

func TestHeaderExchange(t *testing.T) {
	t.Run("SingleAggregatorSingleFullNode", func(t *testing.T) {
		testSingleAggregatorSingleFullNode(t, Header)
	})
	t.Run("SingleAggregatorTwoFullNode", func(t *testing.T) {
		testSingleAggregatorTwoFullNode(t, Header)
	})
	t.Run("SingleAggregatorSingleFullNodeTrustedHash", func(t *testing.T) {
		testSingleAggregatorSingleFullNodeTrustedHash(t, Header)
	})
	t.Run("SingleAggregatorSingleFullNodeSingleLightNode", testSingleAggregatorSingleFullNodeSingleLightNode)
}

func TestSubmitBlocksToDA(t *testing.T) {
	require := require.New(t)

	clientNodes := 1
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nodes, _ := createNodes(ctx, context.Background(), clientNodes, config.BlockManagerConfig{
		DABlockTime: 20 * time.Millisecond,
		BlockTime:   10 * time.Millisecond,
	}, "TestSubmitBlocksToDA", false, t)
	seq := nodes[0]
	startNodeWithCleanup(t, seq)

	numberOfBlocksToSyncTill := seq.Store.Height()

	//Make sure all produced blocks made it to DA
	for i := uint64(1); i <= numberOfBlocksToSyncTill; i++ {
		require.NoError(Retry(300, 100*time.Millisecond, func() error {
			header, _, err := seq.Store.GetBlockData(ctx, i)
			if err != nil {
				return err
			}
			if !seq.blockManager.IsDAIncluded(header.Hash()) {
				return fmt.Errorf("block %d not DA included", header.Height())
			}
			return nil
		}))
	}
}

func TestTwoRollupsInOneNamespace(t *testing.T) {
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

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doTestTwoRollupsInOneNamespace(t, tc.chainID1, tc.chainID2)
		})
	}
}

func doTestTwoRollupsInOneNamespace(t *testing.T, chainID1, chainID2 string) {
	require := require.New(t)

	const (
		n             = 2
		daStartHeight = 1
		daMempoolTTL  = 5
		blockTime1    = 100 * time.Millisecond
		blockTime2    = 50 * time.Millisecond
	)

	mainCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	agg1Ctx := context.WithoutCancel(mainCtx)
	nodes1Ctx := context.WithoutCancel(mainCtx)

	rollupNetwork1, apps1 := createNodes(agg1Ctx, nodes1Ctx, n, config.BlockManagerConfig{
		BlockTime:     blockTime1,
		DABlockTime:   blockTime1,
		DAStartHeight: daStartHeight,
		DAMempoolTTL:  daMempoolTTL,
	}, chainID1, true, t)

	require.Len(rollupNetwork1, n)
	require.Len(apps1, n)

	agg2Ctx := context.WithoutCancel(mainCtx)
	nodes2Ctx := context.WithoutCancel(mainCtx)

	rollupNetwork2, apps2 := createNodes(agg2Ctx, nodes2Ctx, n, config.BlockManagerConfig{
		BlockTime:     blockTime2,
		DABlockTime:   blockTime2,
		DAStartHeight: daStartHeight,
		DAMempoolTTL:  daMempoolTTL,
	}, chainID2, true, t)

	require.Len(rollupNetwork2, n)
	require.Len(apps2, n)

	// same mock DA has to be used by all nodes to simulate posting to/retrieving from same namespace
	dalc := getMockDA(t)
	for _, node := range append(rollupNetwork1, rollupNetwork2...) {
		node.dalc = dalc
		node.blockManager.SetDALC(dalc)
	}

	agg1 := rollupNetwork1[0]
	agg2 := rollupNetwork2[0]

	node1 := rollupNetwork1[1]
	node2 := rollupNetwork2[1]

	// start both aggregators and wait for 10 blocks
	require.NoError(agg1.Start())
	require.NoError(agg2.Start())

	require.NoError(waitForAtLeastNBlocks(agg1, 10, Store))
	require.NoError(waitForAtLeastNBlocks(agg2, 10, Store))

	// get the number of submitted blocks from aggregators before stopping (as it closes the store)
	lastSubmittedHeight1 := getLastSubmittedHeight(agg1Ctx, agg1, t)
	lastSubmittedHeight2 := getLastSubmittedHeight(agg2Ctx, agg2, t)

	// make sure that there are any blocks for syncing
	require.Greater(lastSubmittedHeight1, 1)
	require.Greater(lastSubmittedHeight2, 1)

	// now stop the aggregators and run the full nodes to ensure sync from D
	require.NoError(agg1.Stop())
	require.NoError(agg2.Stop())

	startNodeWithCleanup(t, node1)
	startNodeWithCleanup(t, node2)

	// check that full nodes are able to sync blocks from DA
	require.NoError(waitForAtLeastNBlocks(node1, lastSubmittedHeight1, Store))
	require.NoError(waitForAtLeastNBlocks(node2, lastSubmittedHeight2, Store))
}

func getLastSubmittedHeight(ctx context.Context, node *FullNode, t *testing.T) int {
	raw, err := node.Store.GetMetadata(ctx, block.LastSubmittedHeightKey)
	require.NoError(t, err)
	require.NotEmpty(t, raw)

	val, err := strconv.ParseUint(string(raw), 10, 64)
	require.NoError(t, err)
	return int(val)
}

func TestMaxPending(t *testing.T) {
	cases := []struct {
		name       string
		maxPending uint64
	}{
		{
			name:       "no limit",
			maxPending: 0,
		},
		{
			name:       "10 pending blocks limit",
			maxPending: 10,
		},
		{
			name:       "50 pending blocks limit",
			maxPending: 50,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doTestMaxPending(tc.maxPending, t)
		})
	}
}

func doTestMaxPending(maxPending uint64, t *testing.T) {
	require := require.New(t)

	clientNodes := 1
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	nodes, _ := createNodes(ctx, context.Background(), clientNodes, config.BlockManagerConfig{
		DABlockTime:      20 * time.Millisecond,
		BlockTime:        10 * time.Millisecond,
		MaxPendingBlocks: maxPending,
	}, "TestMaxPending", false, t)
	seq := nodes[0]
	mockDA := &damock.MockDA{}

	// make sure mock DA is not accepting any submissions
	mockDA.On("MaxBlobSize", mock.Anything).Return(uint64(123456789), nil)
	mockDA.On("Submit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("DA not available"))

	dalc := da.NewDAClient(mockDA, 1234, 5678, goDA.Namespace(MockDANamespace), nil, log.NewNopLogger())
	require.NotNil(dalc)
	seq.dalc = dalc
	seq.blockManager.SetDALC(dalc)

	startNodeWithCleanup(t, seq)

	if maxPending == 0 { // if there is no limit, sequencer should produce blocks even DA is unavailable
		require.NoError(waitForAtLeastNBlocks(seq, 3, Store))
		return
	} else { // if there is a limit, sequencer should produce exactly maxPending blocks and pause
		require.NoError(waitForAtLeastNBlocks(seq, int(maxPending), Store)) //nolint:gosec
		// wait few block times and ensure that new blocks are not produced
		time.Sleep(3 * seq.nodeConfig.BlockTime)
		require.EqualValues(maxPending, seq.Store.Height())
	}

	// change mock function to start "accepting" blobs
	mockDA.On("Submit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Unset()
	mockDA.On("Submit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, blobs [][]byte, gasPrice float64, namespace []byte) ([][]byte, error) {
			hashes := make([][]byte, len(blobs))
			for i, blob := range blobs {
				sha := sha256.Sum256(blob)
				hashes[i] = sha[:]
			}
			return hashes, nil
		})

	// wait for next block to ensure that sequencer is producing blocks again
	require.NoError(waitForAtLeastNBlocks(seq, int(maxPending+1), Store)) //nolint:gosec
}

// TODO: https://github.com/rollkit/rollkit/issues/1811
// func TestCatchUp(t *testing.T) {
// 	cases := []struct {
// 		name string
// 		n    int
// 	}{
// 		{
// 			name: "small catch-up",
// 			n:    200,
// 		},
// 		{
// 			name: "large catch-up",
// 			n:    1000,
// 		},
// 	}

// 	for _, tc := range cases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			doTestCatchup(t, tc.n)
// 		})
// 	}
// }

// func doTestCatchup(t *testing.T, n int) {
// 	//t.Parallel()
// 	require := require.New(t)

// 	const (
// 		timeout     = 300 * time.Second
// 		clientNodes = 2
// 		blockTime   = 10 * time.Millisecond
// 		daBlockTime = 20 * time.Millisecond
// 	)
// 	ctx, cancel := context.WithTimeout(context.Background(), timeout)
// 	defer cancel()
// 	nodes, _ := createNodes(
// 		ctx,
// 		context.Background(),
// 		clientNodes,
// 		config.BlockManagerConfig{
// 			DABlockTime: daBlockTime,
// 			BlockTime:   blockTime,
// 		},
// 		types.TestChainID,
// 		false,
// 		t,
// 	)
// 	seq := nodes[0]
// 	full := nodes[1]
// 	mockDA := &mocks.DA{}

// 	// make sure mock DA is not accepting any submissions to force P2P only sync
// 	mockDA.On("MaxBlobSize", mock.Anything).Return(uint64(123456789), nil)
// 	daError := errors.New("DA not available")
// 	mockDA.On("Submit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, daError)
// 	mockDA.On("GetIDs", mock.Anything, mock.Anything, mock.Anything).Return(nil, daError)

// 	dalc := da.NewDAClient(mockDA, 1234, 5678, goDA.Namespace(MockDANamespace), log.NewNopLogger())
// 	require.NotNil(dalc)

// 	for _, node := range nodes {
// 		node.dalc = dalc
// 		node.blockManager.SetDALC(dalc)
// 	}

// 	// start sequencer
// 	startNodeWithCleanup(t, seq)

// 	// submit random transactions
// 	ch := make(chan *struct{})

// 	// spam transactions until n blocks are produced
// 	txCtx := context.WithoutCancel(ctx)
// 	go func() {
// 		for {
// 			select {
// 			case <-ch:
// 				return
// 			default:
// 				for i := uint(1); i < 10; i++ {
// 					_, _ = seq.client.BroadcastTxAsync(txCtx, types.GetRandomBytes(10*i))
// 				}
// 				_, _ = seq.client.BroadcastTxCommit(txCtx, types.GetRandomBytes(1000))
// 			}
// 		}
// 	}()

// 	require.NoError(waitForAtLeastNBlocks(seq, n, Store))
// 	ch <- nil

// 	startNodeWithCleanup(t, full)
// 	require.NoError(waitForAtLeastNBlocks(full, n, Store))
// }

func testSingleAggregatorSingleFullNode(t *testing.T, source Source) {
	require := require.New(t)

	aggCtx, aggCancel := context.WithCancel(context.Background())
	defer aggCancel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clientNodes := 2
	nodes, _ := createNodes(aggCtx, ctx, clientNodes, getBMConfig(), "testSingleAggregatorSingleFullNode", false, t)

	node1 := nodes[0]
	node2 := nodes[1]

	startNodeWithCleanup(t, node1)
	require.NoError(waitForFirstBlock(node1, source))

	startNodeWithCleanup(t, node2)
	require.NoError(waitForAtLeastNBlocks(node2, 2, source))

	require.NoError(verifyNodesSynced(node1, node2, source))
}

func testSingleAggregatorTwoFullNode(t *testing.T, source Source) {
	require := require.New(t)

	aggCtx, aggCancel := context.WithCancel(context.Background())
	defer aggCancel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clientNodes := 3
	nodes, _ := createNodes(aggCtx, ctx, clientNodes, getBMConfig(), "testSingleAggregatorTwoFullNode", false, t)

	node1 := nodes[0]
	node2 := nodes[1]
	node3 := nodes[2]

	startNodeWithCleanup(t, node1)
	require.NoError(waitForFirstBlock(node1, source))

	startNodeWithCleanup(t, node2)
	startNodeWithCleanup(t, node3)

	require.NoError(waitForAtLeastNBlocks(node2, 2, source))
	require.NoError(waitForAtLeastNBlocks(node3, 2, source))

	require.NoError(verifyNodesSynced(node1, node2, source))
	require.NoError(verifyNodesSynced(node1, node3, source))
}

func testSingleAggregatorSingleFullNodeTrustedHash(t *testing.T, source Source) {
	require := require.New(t)

	aggCtx, aggCancel := context.WithCancel(context.Background())
	defer aggCancel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clientNodes := 2
	nodes, _ := createNodes(aggCtx, ctx, clientNodes, getBMConfig(), "testSingleAggregatorSingleFullNodeTrustedHash", false, t)

	node1 := nodes[0]
	node2 := nodes[1]

	startNodeWithCleanup(t, node1)
	require.NoError(waitForFirstBlock(node1, source))

	// Get the trusted hash from node1 and pass it to node2 config
	trustedHash, err := node1.hSyncService.Store().GetByHeight(aggCtx, 1)
	require.NoError(err)

	node2.nodeConfig.TrustedHash = trustedHash.Hash().String()
	startNodeWithCleanup(t, node2)

	require.NoError(waitForAtLeastNBlocks(node1, 2, source))
	require.NoError(verifyNodesSynced(node1, node2, source))
}

func testSingleAggregatorSingleFullNodeSingleLightNode(t *testing.T) {
	require := require.New(t)

	aggCtx, aggCancel := context.WithCancel(context.Background())
	defer aggCancel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	num := 3
	keys := make([]crypto.PrivKey, num)
	for i := 0; i < num; i++ {
		keys[i], _, _ = crypto.GenerateEd25519Key(rand.Reader)
	}
	dalc := getMockDA(t)
	bmConfig := getBMConfig()
	chainID := "testSingleAggregatorSingleFullNodeSingleLightNode"
	sequencer, _ := createAndConfigureNode(aggCtx, 0, true, false, chainID, keys, bmConfig, dalc, t)
	fullNode, _ := createAndConfigureNode(ctx, 1, false, false, chainID, keys, bmConfig, dalc, t)
	lightNode, _ := createNode(ctx, 2, false, true, keys, bmConfig, chainID, false, t)

	startNodeWithCleanup(t, sequencer)
	require.NoError(waitForFirstBlock(sequencer, Header))
	startNodeWithCleanup(t, fullNode)
	startNodeWithCleanup(t, lightNode)

	require.NoError(waitForAtLeastNBlocks(sequencer.(*FullNode), 2, Header))
	require.NoError(verifyNodesSynced(sequencer, fullNode, Header))
	require.NoError(verifyNodesSynced(fullNode, lightNode, Header))
}

func getMockApplication() *mocks.Application {
	app := &mocks.Application{}
	app.On("InitChain", mock.Anything, mock.Anything).Return(&abci.ResponseInitChain{}, nil)
	app.On("CheckTx", mock.Anything, mock.Anything).Return(&abci.ResponseCheckTx{}, nil)
	app.On("Commit", mock.Anything, mock.Anything).Return(&abci.ResponseCommit{}, nil)
	app.On("PrepareProposal", mock.Anything, mock.Anything).Return(prepareProposalResponse).Maybe()
	app.On("ProcessProposal", mock.Anything, mock.Anything).Return(&abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil)
	app.On("FinalizeBlock", mock.Anything, mock.Anything).Return(finalizeBlockResponse)
	return app
}

// Starts the given nodes using the given wait group to synchronize them
// and wait for them to gossip transactions
func startNodes(nodes []*FullNode, apps []*mocks.Application, t *testing.T) {
	// stop the nodes in case of a failure during start procedure
	defer func() {
		if t.Failed() {
			for _, n := range nodes {
				assert.NoError(t, n.Stop())
			}
		}
	}()

	// Wait for aggregator node to publish the first block for full nodes to initialize header exchange service
	require.NoError(t, nodes[0].Start())
	require.NoError(t, waitForFirstBlock(nodes[0], Header))
	for i := 1; i < len(nodes); i++ {
		tryHardToStartNode(nodes[i], t)
	}

	// wait for nodes to start up and establish connections
	for i := 1; i < len(nodes); i++ {
		require.NoError(t, waitForAtLeastNBlocks(nodes[i], initialBlockWait, Header))
	}

	for i := 1; i < len(nodes); i++ {
		data := strconv.Itoa(i) + time.Now().String()
		require.NoError(t, nodes[i].p2pClient.GossipTx(context.TODO(), []byte(data)))
	}

	timeout := time.NewTimer(time.Second * 30)
	doneChan := make(chan struct{})
	go func() {
		defer close(doneChan)
		// create a MockTester, to catch the Failed asserts from the Mock package
		m := MockTester{t: t}
		// We don't need to check any specific arguments to FinalizeBlock
		// so just use a function that returns "true" for matching the args
		matcher := mock.MatchedBy(func(i interface{}) bool { return true })
		err := Retry(300, 100*time.Millisecond, func() error {
			for i := 0; i < len(apps); i++ {
				if !apps[i].AssertCalled(m, "FinalizeBlock", matcher, matcher) {
					return errors.New("FinalizeBlock hasn't been called yet")
				}
			}
			return nil
		})
		assert := assert.New(t)
		assert.NoError(err)
	}()
	select {
	case <-doneChan:
	case <-timeout.C:
		t.FailNow()
	}
}

// tryHardToStartNode attempts to start a FullNode up to three times.
// If the node fails to start after three attempts, the test is marked as failed.
// This method was introduced to make tests in CI less flaky.
func tryHardToStartNode(node *FullNode, t *testing.T) {
	for r := 0; r < 3; r++ {
		if node.Start() != nil {
			return
		}
	}
	t.Fatal("Failed to start node")
}

// Creates the given number of nodes the given nodes using the given wait group to synchornize them
func createNodes(aggCtx, ctx context.Context, num int, bmConfig config.BlockManagerConfig, chainID string, skipPeers bool, t *testing.T) ([]*FullNode, []*mocks.Application) {
	t.Helper()

	if aggCtx == nil {
		aggCtx = context.Background()
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// create keys first, as they are required for P2P connections
	keys := make([]crypto.PrivKey, num)
	for i := 0; i < num; i++ {
		keys[i], _, _ = crypto.GenerateEd25519Key(rand.Reader)
	}

	nodes := make([]*FullNode, num)
	apps := make([]*mocks.Application, num)
	dalc := getMockDA(t)

	node, app := createNode(aggCtx, 0, true, false, keys, bmConfig, chainID, skipPeers, t)
	apps[0] = app
	nodes[0] = node.(*FullNode)
	// use same, common DALC, so nodes can share data
	nodes[0].dalc = dalc
	nodes[0].blockManager.SetDALC(dalc)
	for i := 1; i < num; i++ {
		node, apps[i] = createNode(ctx, i, false, false, keys, bmConfig, chainID, skipPeers, t)
		nodes[i] = node.(*FullNode)
		nodes[i].dalc = dalc
		nodes[i].blockManager.SetDALC(dalc)
	}

	return nodes, apps
}

func createNode(ctx context.Context, n int, aggregator bool, isLight bool, keys []crypto.PrivKey, bmConfig config.BlockManagerConfig, chainID string, skipPeers bool, t *testing.T) (Node, *mocks.Application) {
	t.Helper()
	require := require.New(t)
	// nodes will listen on consecutive ports on local interface
	// random connections to other nodes will be added
	startPort := 10000
	p2pConfig := config.P2PConfig{
		ListenAddress: "/ip4/127.0.0.1/tcp/" + strconv.Itoa(startPort+n),
	}
	for i := 0; !skipPeers && i < len(keys); i++ {
		if i == n {
			continue
		}
		r := i
		id, err := peer.IDFromPrivateKey(keys[r])
		require.NoError(err)
		p2pConfig.Seeds += "/ip4/127.0.0.1/tcp/" + strconv.Itoa(startPort+r) + "/p2p/" + id.Loggable()["peerID"].(string) + ","
	}
	p2pConfig.Seeds = strings.TrimSuffix(p2pConfig.Seeds, ",")

	app := getMockApplication()

	if ctx == nil {
		ctx = context.Background()
	}

	pubkeyBytes, err := keys[0].GetPublic().Raw()
	require.NoError(err)
	var pubkey ed25519.PubKey = pubkeyBytes
	genesisValidators := []cmtypes.GenesisValidator{
		{
			Address: pubkey.Address(),
			PubKey:  pubkey,
			Power:   int64(1),
			Name:    "sequencer",
		},
	}

	genesis := &cmtypes.GenesisDoc{ChainID: chainID, Validators: genesisValidators}
	// TODO: need to investigate why this needs to be done for light nodes
	genesis.InitialHeight = 1
	node, err := NewNode(
		ctx,
		config.NodeConfig{
			DAAddress:          MockDAAddress,
			DANamespace:        MockDANamespace,
			P2P:                p2pConfig,
			Aggregator:         aggregator,
			BlockManagerConfig: bmConfig,
			Light:              isLight,
			SequencerAddress:   MockSequencerAddress,
		},
		keys[n],
		keys[n],
		proxy.NewLocalClientCreator(app),
		genesis,
		DefaultMetricsProvider(cmconfig.DefaultInstrumentationConfig()),
		test.NewFileLoggerCustom(t, test.TempLogFileName(t, fmt.Sprintf("node%v", n))).With("node", n))
	require.NoError(err)
	require.NotNil(node)

	return node, app
}

func createAndConfigureNode(ctx context.Context, n int, aggregator bool, isLight bool, chainID string, keys []crypto.PrivKey, bmConfig config.BlockManagerConfig, dalc *da.DAClient, t *testing.T) (Node, *mocks.Application) {
	t.Helper()
	node, app := createNode(ctx, n, aggregator, isLight, keys, bmConfig, chainID, false, t)
	node.(*FullNode).dalc = dalc
	node.(*FullNode).blockManager.SetDALC(dalc)

	return node, app
}
