package node

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/log"
	testutils "github.com/celestiaorg/utils/test"
	cmcfg "github.com/cometbft/cometbft/config"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"

	"github.com/rollkit/rollkit/config"
	"github.com/rollkit/rollkit/types"
)

// NodeIntegrationTestSuite is a test suite for node integration tests
type NodeIntegrationTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
	node   Node
	seqSrv *grpc.Server
}

// SetupTest is called before each test
func (s *NodeIntegrationTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	// Setup node with proper configuration
	config := getTestConfig(1)
	config.BlockTime = 100 * time.Millisecond        // Faster block production for tests
	config.DABlockTime = 200 * time.Millisecond      // Faster DA submission for tests
	config.BlockManagerConfig.MaxPendingBlocks = 100 // Allow more pending blocks

	genesis, genesisValidatorKey := types.GetGenesisWithPrivkey(types.DefaultSigningKeyType, "test-chain")
	signingKey, err := types.PrivKeyToSigningKey(genesisValidatorKey)
	require.NoError(s.T(), err)

	s.seqSrv = startMockSequencerServerGRPC(MockSequencerAddress)
	require.NotNil(s.T(), s.seqSrv)

	p2pKey := generateSingleKey()

	node, err := NewNode(
		s.ctx,
		config,
		p2pKey,
		signingKey,
		genesis,
		DefaultMetricsProvider(cmcfg.DefaultInstrumentationConfig()),
		log.NewTestLogger(s.T()),
	)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), node)

	fn, ok := node.(*FullNode)
	require.True(s.T(), ok)

	err = fn.Start(s.ctx)
	require.NoError(s.T(), err)

	s.node = fn

	// Wait for node initialization with retry
	err = testutils.Retry(60, 100*time.Millisecond, func() error {
		height, err := getNodeHeight(s.node, Header)
		if err != nil {
			return err
		}
		if height == 0 {
			return fmt.Errorf("waiting for first block")
		}
		return nil
	})
	require.NoError(s.T(), err, "Node failed to produce first block")

	// Wait for DA inclusion with longer timeout
	err = testutils.Retry(100, 100*time.Millisecond, func() error {
		daHeight := s.node.(*FullNode).blockManager.GetDAIncludedHeight()
		if daHeight == 0 {
			return fmt.Errorf("waiting for DA inclusion")
		}
		return nil
	})
	require.NoError(s.T(), err, "Failed to get DA inclusion")
}

// TearDownTest is called after each test
func (s *NodeIntegrationTestSuite) TearDownTest() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.node != nil {
		_ = s.node.Stop(s.ctx)
	}
	if s.seqSrv != nil {
		s.seqSrv.GracefulStop()
	}
}

// TestNodeIntegrationTestSuite runs the test suite
func TestNodeIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(NodeIntegrationTestSuite))
}

func (s *NodeIntegrationTestSuite) waitForHeight(targetHeight uint64) error {
	return waitForAtLeastNBlocks(s.node, int(targetHeight), Store)
}

func (s *NodeIntegrationTestSuite) TestBlockProduction() {
	// Wait for at least one block to be produced and transactions to be included
	time.Sleep(5 * time.Second) // Give more time for the full flow

	// Get transactions from executor to verify they are being injected
	execTxs, err := s.node.(*FullNode).blockManager.GetExecutor().GetTxs(s.ctx)
	s.NoError(err)
	s.T().Logf("Number of transactions from executor: %d", len(execTxs))

	// Wait for at least one block to be produced
	err = s.waitForHeight(1)
	s.NoError(err, "Failed to produce first block")

	// Get the current height
	height := s.node.(*FullNode).Store.Height()
	s.GreaterOrEqual(height, uint64(1), "Expected block height >= 1")

	// Get all blocks and log their contents
	for h := uint64(1); h <= height; h++ {
		header, data, err := s.node.(*FullNode).Store.GetBlockData(s.ctx, h)
		s.NoError(err)
		s.NotNil(header)
		s.NotNil(data)
		s.T().Logf("Block height: %d, Time: %s, Number of transactions: %d", h, header.Time(), len(data.Txs))
	}

	// Get the latest block
	header, data, err := s.node.(*FullNode).Store.GetBlockData(s.ctx, height)
	s.NoError(err)
	s.NotNil(header)
	s.NotNil(data)

	// Log block details
	s.T().Logf("Latest block height: %d, Time: %s, Number of transactions: %d", height, header.Time(), len(data.Txs))

	// Verify chain state
	state, err := s.node.(*FullNode).Store.GetState(s.ctx)
	s.NoError(err)
	s.Equal(height, state.LastBlockHeight)

	// Verify block content
	s.NotEmpty(data.Txs, "Expected block to contain transactions")
}

// nolint:unused
func (s *NodeIntegrationTestSuite) setupNodeWithConfig(conf config.NodeConfig) Node {
	genesis, signingKey := types.GetGenesisWithPrivkey(types.DefaultSigningKeyType, "test-chain")
	key, err := types.PrivKeyToSigningKey(signingKey)
	require.NoError(s.T(), err)

	p2pKey := generateSingleKey()

	node, err := NewNode(
		s.ctx,
		conf,
		p2pKey,
		key,
		genesis,
		DefaultMetricsProvider(cmcfg.DefaultInstrumentationConfig()),
		log.NewTestLogger(s.T()),
	)
	require.NoError(s.T(), err)

	err = node.Start(s.ctx)
	require.NoError(s.T(), err)

	return node
}
