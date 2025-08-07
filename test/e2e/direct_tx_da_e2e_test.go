//go:build evm
// +build evm

package e2e

import (
	"context"
	"flag"
	"path/filepath"
	"testing"
	"time"

	"github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/da/jsonrpc"
	"github.com/rs/zerolog"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/evstack/ev-node/execution/evm"
	"github.com/evstack/ev-node/types"
)

// TestDirectTxToDAForceInclusion tests the direct transaction to DA force inclusion functionality.
// It creates and submits a transaction directly to the DA layer and verifies that the transaction
// is included in a block via the DirectTxReaper.
func TestDirectTxToDAForceInclusion(t *testing.T) {
	flag.Parse()
	workDir := t.TempDir()
	nodeHome := filepath.Join(workDir, "evm-agg")
	sut := NewSystemUnderTest(t)

	// Setup sequencer with direct transaction support
	genesisHash := setupSequencerOnlyTest(t, sut, nodeHome)
	t.Logf("Genesis hash: %s", genesisHash)

	// Connect to EVM
	client, err := ethclient.Dial(SequencerEthURL)
	require.NoError(t, err, "Should be able to connect to EVM")
	defer client.Close()

	// Create a transaction with a specific nonce
	var nonce uint64 = 0
	tx := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &nonce)
	t.Logf("Created transaction with hash: %s and nonce: %d", tx.Hash().Hex(), nonce)

	// Get the raw transaction bytes
	txBytes, err := tx.MarshalBinary()
	require.NoError(t, err, "Should be able to marshal transaction")

	// Create a Data struct with the transaction
	data := types.Data{
		Metadata: &types.Metadata{
			ChainID: "evolve-test", // sequencer chain-id
		},
		Txs: []types.Tx{txBytes},
	}

	// Marshal the Data struct to bytes
	dataBytes, err := proto.Marshal(data.ToProto())
	require.NoError(t, err, "Should be able to marshal Data struct")
	// Marshal the request to JSON
	// Send the request to the DA layer
	logger := zerolog.New(zerolog.NewTestWriter(t)).Level(zerolog.WarnLevel).With().Str("module", "test").Logger()
	const defaultNamespace = ""
	daClient, err := jsonrpc.NewClient(t.Context(), logger, DAAddress, "", defaultNamespace)
	require.NoError(t, err)
	ids, err := daClient.DA.Submit(t.Context(), []da.Blob{dataBytes}, 1, nil)
	require.NoError(t, err)
	t.Logf("Submitted transaction directly to DA layer...: %X\n", ids)

	blobs, err := daClient.DA.Get(t.Context(), ids, nil)
	require.NoError(t, err)
	require.NotEmpty(t, blobs)
	t.Logf("Transaction landed on DA layer...: %X\n", ids)
	t.Cleanup(func() {
		if t.Failed() {
			sut.PrintBuffer()
		}
	})
	// Wait for the transaction to be included in a block
	// The DirectTxReaper should pick up the transaction from the DA layer and submit it to the sequencer
	t.Log("Waiting for transaction to be included in a block...")
	require.Eventually(t, func() bool {
		if evm.CheckTxIncluded(t, tx.Hash()) {
			return true
		}
		return false
	}, 2*time.Second, 300*time.Millisecond, "Transaction should be included in a block")

	// Verify transaction details
	receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
	require.NoError(t, err, "Should be able to get transaction receipt")
	require.Equal(t, uint64(1), receipt.Status, "Transaction should be successful")

	t.Logf("✅ Direct transaction to DA force inclusion test passed. Transaction included in block %d", receipt.BlockNumber.Uint64())
}

// TestDirectTxToDAFullNodeFallback tests the direct transaction to DA functionality
// for a fullnode in fallback mode when the sequencer is unavailable.
// It creates a sequencer and fullnode, shuts down the sequencer to trigger fallback mode,
// then submits a transaction directly to the DA layer and verifies that the fullnode
// processes it correctly in fallback mode.
func TestDirectTxToDAFullNodeFallback(t *testing.T) {
	t.Skip("The logic for full node state updates is not fully implemented yet. Skipping this test for now.")
	flag.Parse()
	workDir := t.TempDir()
	sequencerHome := filepath.Join(workDir, "evm-sequencer")
	fullNodeHome := filepath.Join(workDir, "evm-full-node")
	sut := NewSystemUnderTest(t)

	// Setup sequencer and fullnode
	jwtSecret, fullNodeJwtSecret, genesisHash := setupCommonEVMTest(t, sut, true)
	t.Logf("Genesis hash: %s", genesisHash)

	// Initialize and start sequencer node
	setupSequencerNode(t, sut, sequencerHome, jwtSecret, genesisHash)
	t.Log("Sequencer node is up")

	// Get sequencer P2P address for fullnode connection
	sequencerP2PAddress := "/ip4/127.0.0.1/tcp/" + RollkitP2PPort + "/p2p/" + NodeID(t, sequencerHome).String()

	// Setup fullnode connected to sequencer
	setupFullNode(t, sut, fullNodeHome, sequencerHome, fullNodeJwtSecret, genesisHash, sequencerP2PAddress)
	t.Log("Full node is up and connected to sequencer")

	// Connect to fullnode EVM
	fullNodeClient, err := ethclient.Dial(FullNodeEthURL)
	require.NoError(t, err, "Should be able to connect to fullnode EVM")
	defer fullNodeClient.Close()

	// Wait for fullnode to sync with sequencer
	time.Sleep(2 * time.Second)

	// Shut down sequencer to trigger fallback mode
	sut.ShutdownByCmd("evm-single")
	t.Log("Sequencer shut down - fullnode should enter fallback mode")

	// Wait a moment for the shutdown to take effect
	time.Sleep(1 * time.Second)

	// Create a transaction with a specific nonce
	var nonce uint64 = 0
	tx := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &nonce)
	t.Logf("Created transaction with hash: %s and nonce: %d", tx.Hash().Hex(), nonce)

	// Get the raw transaction bytes
	txBytes, err := tx.MarshalBinary()
	require.NoError(t, err, "Should be able to marshal transaction")

	// Create a Data struct with the transaction
	data := types.Data{
		Metadata: &types.Metadata{
			ChainID: "evolve-test", // sequencer chain-id
		},
		Txs: []types.Tx{txBytes},
	}

	// Marshal the Data struct to bytes
	dataBytes, err := proto.Marshal(data.ToProto())
	require.NoError(t, err, "Should be able to marshal Data struct")

	// Send the request to the DA layer
	logger := zerolog.New(zerolog.NewTestWriter(t)).Level(zerolog.WarnLevel).With().Str("module", "test").Logger()
	const defaultNamespace = ""
	daClient, err := jsonrpc.NewClient(t.Context(), logger, DAAddress, "", defaultNamespace)
	require.NoError(t, err)
	ids, err := daClient.DA.Submit(t.Context(), []da.Blob{dataBytes}, 1, nil)
	require.NoError(t, err)
	t.Logf("Submitted transaction directly to DA layer: %X", ids)

	blobs, err := daClient.DA.Get(t.Context(), ids, nil)
	require.NoError(t, err)
	require.NotEmpty(t, blobs)
	t.Logf("Transaction landed on DA layer: %X", ids)

	t.Cleanup(func() {
		if t.Failed() {
			sut.PrintBuffer()
		}
	})

	// Wait for the transaction to be included in a block by the fullnode in fallback mode
	t.Log("Waiting for fullnode to process transaction in fallback mode...")

	require.Eventually(t, func() bool {
		receipt, err := fullNodeClient.TransactionReceipt(context.Background(), tx.Hash())
		if err == nil && receipt != nil && receipt.Status == 1 {
			t.Logf("Transaction included in block %d by fullnode in fallback mode", receipt.BlockNumber.Uint64())
			return true
		}
		return false
	}, 10*time.Second, 500*time.Millisecond, "Transaction should be included in a block by fullnode in fallback mode")

	// assert direct tx included in a block
	require.Eventually(t, func() bool {
		if evm.CheckTxIncluded(t, tx.Hash()) {
			return true
		}
		return false
	}, 2*time.Second, 300*time.Millisecond, "Transaction should be included in a block")

	// Verify transaction details
	receipt, err := fullNodeClient.TransactionReceipt(context.Background(), tx.Hash())
	require.NoError(t, err, "Should be able to get transaction receipt")
	require.Equal(t, uint64(1), receipt.Status, "Transaction should be successful")

	t.Logf("✅ Direct transaction to DA fullnode fallback test passed. Transaction included in block %d", receipt.BlockNumber.Uint64())
}
