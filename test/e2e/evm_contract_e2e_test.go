//go:build evm

package e2e

import (
	"context"
	"crypto/ecdsa"
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
	workDir := t.TempDir()
	sequencerHome := filepath.Join(workDir, "evm-sequencer")

	client, _, cleanup := setupTestSequencer(t, sequencerHome)
	defer cleanup()

	ctx := t.Context()
	var globalNonce uint64 = 0

	// 1. Deploy Contract
	t.Log("Deploying Storage contract...")

	privateKey, err := crypto.HexToECDSA(TestPrivateKey)
	require.NoError(t, err)
	chainIDInt, ok := new(big.Int).SetString(DefaultChainID, 10)
	require.True(t, ok)

	contractAddress, nextNonce := deployContract(t, ctx, client, StorageContractBytecode, globalNonce, privateKey, chainIDInt)
	globalNonce = nextNonce

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
	var receipt *types.Receipt
	require.Eventually(t, func() bool {
		receipt, err = client.TransactionReceipt(ctx, storeTxHash)
		return err == nil && receipt != nil
	}, 15*time.Second, 500*time.Millisecond, "Store transaction should be included")
	require.Equal(t, uint64(1), receipt.Status, "Store tx failed! GasUsed: %d", receipt.GasUsed)

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

// Event contract bytecode (EventEmitter)
//
// Contract Source:
//
//	contract EventEmitter {
//	    event Log1(bytes32 indexed topic, bytes32 data);
//
//	    function emitLog() public {
//	        emit Log1(
//	            0xdeadbeef00000000000000000000000000000000000000000000000000000000,
//	            0xcafe000000000000000000000000000000000000000000000000000000000000
//	        );
//	    }
//	}
const (
	EventContractBytecode = "6050600c60003960506000f360206000527fcafe0000000000000000000000000000000000000000000000000000000000006000527fdeadbeef0000000000000000000000000000000000000000000000000000000060206000a100"
)

// TestEvmContractEvents tests that EVM events (LOG opcodes) are correctly emitted and retrievable.
func TestEvmContractEvents(t *testing.T) {
	workDir := t.TempDir()
	sequencerHome := filepath.Join(workDir, "evm-sequencer-events")

	// Setup sequencer
	client, _, cleanup := setupTestSequencer(t, sequencerHome)
	defer cleanup()

	ctx := t.Context()
	var globalNonce uint64 = 0

	// 1. Deploy EventEmitter Contract
	t.Log("Deploying EventEmitter contract...")

	privateKey, err := crypto.HexToECDSA(TestPrivateKey)
	require.NoError(t, err)
	chainIDInt, ok := new(big.Int).SetString(DefaultChainID, 10)
	require.True(t, ok)

	contractAddress, nextNonce := deployContract(t, ctx, client, EventContractBytecode, globalNonce, privateKey, chainIDInt)
	globalNonce = nextNonce

	t.Logf("✅ EventEmitter contract deployed at: %s", contractAddress.Hex())

	// 2. Trigger Event
	t.Log("Triggering event...")

	txTrigger := types.NewTx(&types.LegacyTx{
		Nonce:    globalNonce,
		To:       &contractAddress,
		Value:    big.NewInt(0),
		Gas:      500000,
		GasPrice: big.NewInt(30000000000),
		Data:     []byte{}, // Any call triggers the log
	})

	signedTxTrigger, err := types.SignTx(txTrigger, types.NewEIP155Signer(chainIDInt), privateKey)
	require.NoError(t, err)

	err = client.SendTransaction(ctx, signedTxTrigger)
	require.NoError(t, err)

	triggerTxHash := signedTxTrigger.Hash()
	globalNonce++

	// Wait for receipt
	var triggerReceipt *types.Receipt
	require.Eventually(t, func() bool {
		triggerReceipt, err = client.TransactionReceipt(ctx, triggerTxHash)
		return err == nil && triggerReceipt != nil
	}, 15*time.Second, 500*time.Millisecond, "Trigger transaction should be included")

	require.Equal(t, uint64(1), triggerReceipt.Status, "Trigger tx failed! GasUsed: %d", triggerReceipt.GasUsed)

	// 3. Verify Log in Receipt
	t.Logf("Trigger Receipt: Status=%d, GasUsed=%d, Logs=%d", triggerReceipt.Status, triggerReceipt.GasUsed, len(triggerReceipt.Logs))
	require.Len(t, triggerReceipt.Logs, 1, "Should have 1 log in receipt")
	log := triggerReceipt.Logs[0]

	// Expected Log
	expectedTopic := common.HexToHash("0xdeadbeef00000000000000000000000000000000000000000000000000000000")
	expectedData := common.Hex2Bytes("cafe000000000000000000000000000000000000000000000000000000000000")

	require.Equal(t, contractAddress, log.Address, "Log address should match contract")
	require.Len(t, log.Topics, 1, "Should have 1 topic")
	require.Equal(t, expectedTopic, log.Topics[0], "Topic should match 0xdeadbeef...")
	require.Equal(t, expectedData, log.Data, "Data should match 0xcafe...")

	t.Log("✅ Log verification in receipt successful")

	// 4. Verify eth_getLogs
	t.Log("Verifying eth_getLogs...")

	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(0),
		ToBlock:   nil, // Latest
		Addresses: []common.Address{contractAddress},
		Topics:    [][]common.Hash{{expectedTopic}},
	}

	logs, err := client.FilterLogs(ctx, query)
	require.NoError(t, err)
	require.Len(t, logs, 1, "eth_getLogs should return 1 log")

	retrievedLog := logs[0]
	require.Equal(t, contractAddress, retrievedLog.Address)
	require.Equal(t, expectedTopic, retrievedLog.Topics[0])
	require.Equal(t, expectedData, retrievedLog.Data)

	t.Log("✅ eth_getLogs verification successful")
}

// setupTestSequencer sets up a single sequencer node for testing.
// Returns the ethclient, genesis hash, and a cleanup function.
func setupTestSequencer(t testing.TB, homeDir string, extraArgs ...string) (*ethclient.Client, string, func()) {
	sut := NewSystemUnderTest(t)

	genesisHash, seqEthURL := setupSequencerOnlyTest(t, sut, homeDir, extraArgs...)
	t.Logf("Sequencer started at %s (Genesis: %s)", seqEthURL, genesisHash)

	client, err := ethclient.Dial(seqEthURL)
	require.NoError(t, err)

	cleanup := func() {
		client.Close()
	}
	return client, genesisHash, cleanup
}

// deployContract helps deploy a contract and waits for its inclusion.
// Returns the deployed contract address and the next nonce.
func deployContract(t testing.TB, ctx context.Context, client *ethclient.Client, bytecodeStr string, nonce uint64, privateKey *ecdsa.PrivateKey, chainID *big.Int) (common.Address, uint64) {
	bytecode, err := hexutil.Decode("0x" + bytecodeStr)
	require.NoError(t, err)

	txDeploy := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       nil, // nil for contract creation
		Value:    big.NewInt(0),
		Gas:      3000000,
		GasPrice: big.NewInt(30000000000),
		Data:     bytecode,
	})

	signedTxDeploy, err := types.SignTx(txDeploy, types.NewEIP155Signer(chainID), privateKey)
	require.NoError(t, err)

	err = client.SendTransaction(ctx, signedTxDeploy)
	require.NoError(t, err)

	deployTxHash := signedTxDeploy.Hash()
	t.Logf("Contract deployment tx submitted: %s", deployTxHash.Hex())

	var receipt *types.Receipt
	require.Eventually(t, func() bool {
		receipt, err = client.TransactionReceipt(ctx, deployTxHash)
		return err == nil && receipt != nil
	}, 20*time.Second, 500*time.Millisecond, "Contract deployment should be included")
	require.Equal(t, uint64(1), receipt.Status, "Contract deployment tx failed! GasUsed: %d", receipt.GasUsed)
	return receipt.ContractAddress, nonce + 1
}
