//go:build evm

package e2e

import (
	"context"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

// Storage contract bytecode (Simple storage: store & retrieve)
//
// Contract Source:
//
//	contract Storage {
//	    uint256 number;
//	    function store(uint256 num) public {
//	        number = num;
//	    }
//	    function retrieve() public view returns (uint256) {
//	        return number;
//	    }
//	}
const (
	StorageContractBytecode = "6018600c60003960186000f33615600c57600035600055005b60005460005260206000f3"
)

// TestEvmContractDeploymentAndInteraction tests deploying a smart contract and interacting with it.
//
// Test Flow:
// 1. Setup a sequencer node
// 2. Deploy the Storage contract using the pre-compiled bytecode
// 3. Wait for deployment to be included in a block
// 4. Send a transaction to call store(42)
// 5. Wait for the transaction to be included
// 6. Call retrieve() via eth_call and verify the return value is 42
func TestEvmContractDeploymentAndInteraction(t *testing.T) {
	sut := NewSystemUnderTest(t)
	workDir := t.TempDir()
	sequencerHome := filepath.Join(workDir, "evm-sequencer")

	genesisHash, seqEthURL := setupSequencerOnlyTest(t, sut, sequencerHome)
	t.Logf("Sequencer started at %s (Genesis: %s)", seqEthURL, genesisHash)

	client, err := ethclient.Dial(seqEthURL)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	var globalNonce uint64 = 0

	// 1. Deploy Contract
	t.Log("Deploying Storage contract...")
	bytecode, err := hexutil.Decode("0x" + StorageContractBytecode)
	require.NoError(t, err)

	// Create contract creation transaction manually
	privateKey, err := crypto.HexToECDSA(TestPrivateKey)
	require.NoError(t, err)

	chainIDInt, ok := new(big.Int).SetString(DefaultChainID, 10)
	require.True(t, ok)

	txDeploy := types.NewTx(&types.LegacyTx{
		Nonce:    globalNonce,
		To:       nil, // nil for contract creation
		Value:    big.NewInt(0),
		Gas:      3000000,
		GasPrice: big.NewInt(30000000000),
		Data:     bytecode,
	})

	signedTxDeploy, err := types.SignTx(txDeploy, types.NewEIP155Signer(chainIDInt), privateKey)
	require.NoError(t, err)

	err = client.SendTransaction(ctx, signedTxDeploy)
	require.NoError(t, err)

	deployTxHash := signedTxDeploy.Hash()
	t.Logf("Contract deployment tx submitted: %s", deployTxHash.Hex())
	globalNonce++

	// Wait for deployment inclusion
	var contractAddress common.Address
	require.Eventually(t, func() bool {
		receipt, err := client.TransactionReceipt(ctx, deployTxHash)
		if err == nil && receipt != nil && receipt.Status == 1 {
			contractAddress = receipt.ContractAddress
			return true
		}
		return false
	}, 20*time.Second, 500*time.Millisecond, "Contract deployment should be included")

	t.Logf("✅ Contract deployed at: %s", contractAddress.Hex())

	// 2. Call store(42) -> 42 is 0x2a
	t.Log("Calling set(42)...")

	// Data: 32 bytes representing 42
	storeData, err := hexutil.Decode("0x000000000000000000000000000000000000000000000000000000000000002a")
	require.NoError(t, err)

	txStore := types.NewTx(&types.LegacyTx{
		Nonce:    globalNonce,
		To:       &contractAddress,
		Value:    big.NewInt(0),
		Gas:      500000, // Should be plenty for simple SSTORE
		GasPrice: big.NewInt(30000000000),
		Data:     storeData,
	})

	signedTxStore, err := types.SignTx(txStore, types.NewEIP155Signer(chainIDInt), privateKey)
	require.NoError(t, err)

	err = client.SendTransaction(ctx, signedTxStore)
	require.NoError(t, err)

	storeTxHash := signedTxStore.Hash()
	t.Logf("Store tx submitted: %s", storeTxHash.Hex())
	globalNonce++

	// Wait for store tx inclusion with debugging
	require.Eventually(t, func() bool {
		receipt, err := client.TransactionReceipt(ctx, storeTxHash)
		if err != nil {
			return false
		}
		if receipt != nil {
			if receipt.Status == 1 {
				return true
			}
			t.Logf("Store tx failed! Status: %d, GasUsed: %d", receipt.Status, receipt.GasUsed)
			return false
		}
		return false
	}, 15*time.Second, 500*time.Millisecond, "Store transaction should be included")

	t.Log("✅ Store transaction confirmed")

	// 3. Call retrieve() and verify result
	t.Log("Calling get() to verify state...")

	// Data: empty (to trigger get path)
	retrieveData := []byte{}

	callMsg := ethereum.CallMsg{
		To:   &contractAddress,
		Data: retrieveData,
	}

	result, err := client.CallContract(ctx, callMsg, nil)
	require.NoError(t, err)

	t.Logf("Retrieve result: %s", hexutil.Encode(result))

	// Expected result: 32 bytes representing 42 (0x2a)
	expected := common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000002a").Bytes()
	require.Equal(t, expected, result, "Retrieve should return 42")

	t.Log("✅ State verification successful: retrieve() returned 42")
}
