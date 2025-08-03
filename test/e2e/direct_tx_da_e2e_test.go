//go:build evm
// +build evm

package e2e

import (
	"context"
	"flag"
	"github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/da/jsonrpc"
	logging "github.com/ipfs/go-log/v2"
	"path/filepath"
	"testing"
	"time"

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
			ChainID: "rollkit-test", // sequencer chain-id
		},
		Txs: []types.Tx{txBytes},
	}

	// Marshal the Data struct to bytes
	dataBytes, err := proto.Marshal(data.ToProto())
	require.NoError(t, err, "Should be able to marshal Data struct")
	// Marshal the request to JSON
	// Send the request to the DA layer
	logger := logging.Logger("test")
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

	// Wait for the transaction to be included in a block
	// The DirectTxReaper should pick up the transaction from the DA layer and submit it to the sequencer
	t.Log("Waiting for transaction to be included in a block...")
	require.Eventually(t, func() bool {
		if evm.CheckTxIncluded(t, tx.Hash()) {
			return true
		}
		//go func() { // submit any TX to trigger block creation
		//	tx := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &globalNonce)
		//	evm.SubmitTransaction(t, tx)
		//}()
		return false
	}, 1*time.Second, 500*time.Millisecond, "Transaction should be included in a block")

	// Verify transaction details
	receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
	require.NoError(t, err, "Should be able to get transaction receipt")
	require.Equal(t, uint64(1), receipt.Status, "Transaction should be successful")

	t.Logf("✅ Direct transaction to DA force inclusion test passed. Transaction included in block %d", receipt.BlockNumber.Uint64())
}
