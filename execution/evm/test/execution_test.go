//go:build evm
// +build evm

package test

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/execution/evm"
)

const (
	CHAIN_ID          = "1234"
	GENESIS_HASH      = "0x2b8bbb1ea1e04f9c9809b4b278a8687806edc061a356c7dbc491930d8e922503"
	GENESIS_STATEROOT = "0x05e9954443da80d86f2104e56ffdfd98fe21988730684360104865b3dc8191b4"
	TEST_PRIVATE_KEY  = "cece4f25ac74deb1468965160c7185e07dff413f23fcadb611b05ca37ab0a52e"
	TEST_TO_ADDRESS   = "0x944fDcD1c868E3cC566C78023CcB38A32cDA836E"
)

// TestEngineExecution tests the end-to-end execution flow of the EVM engine client.
// The test has two phases:
//
// Build Chain Phase:
// - Sets up test Reth engine with JWT auth
// - Initializes chain with genesis parameters
// - For blocks 1-10:
//   - Generates and submits random transactions
//   - Block 4 has 0 transactions as edge case
//   - Executes transactions and verifies state changes
//   - Stores payloads for sync testing
//
// Sync Chain Phase:
// - Creates fresh engine instance
// - Replays stored payloads
// - Verifies execution matches original:
//   - State roots
//   - Block data
//   - Transaction counts
//
// Validates the engine can process transactions, maintain state,
// handle empty blocks, and support chain replication.
func TestEngineExecution(t *testing.T) {
	allPayloads := make([][][]byte, 0, 10)        // Slice to store payloads from build to sync phase
	allTimestamps := make([]time.Time, 0, 10)     // Slice to store timestamps from build phase
	buildPhaseStateRoots := make([][]byte, 0, 10) // Slice to store state roots from build phase

	initialHeight := uint64(1)
	genesisHash := common.HexToHash(GENESIS_HASH)
	genesisTime := time.Now().UTC().Truncate(time.Second)
	genesisStateRoot := common.HexToHash(GENESIS_STATEROOT)
	GenesisStateRoot := genesisStateRoot[:]

	t.Run("Build chain", func(tt *testing.T) {
		rethNode := SetupTestRethNode(t)

		ni, err := rethNode.GetNetworkInfo(context.TODO())
		require.NoError(tt, err)
		ethURL := "http://127.0.0.1:" + ni.External.Ports.RPC
		engineURL := "http://127.0.0.1:" + ni.External.Ports.Engine

		executionClient, err := evm.NewEngineExecutionClient(
			ethURL,
			engineURL,
			rethNode.JWTSecretHex(),
			genesisHash,
			common.Address{},
		)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
		defer cancel()
		stateRoot, gasLimit, err := executionClient.InitChain(ctx, genesisTime, initialHeight, CHAIN_ID)
		require.NoError(t, err)
		require.Equal(t, GenesisStateRoot, stateRoot)
		require.NotZero(t, gasLimit)

		prevStateRoot := GenesisStateRoot
		lastHeight, lastHash, lastTxs := checkLatestBlock(tt, ctx, ethURL)
		lastNonce := uint64(0)

		// Use a base timestamp and increment for each block to ensure proper ordering
		baseTimestamp := time.Now()

		for blockHeight := initialHeight; blockHeight <= 10; blockHeight++ {
			nTxs := int(blockHeight) + 10
			// randomly use no transactions
			if blockHeight == 4 {
				nTxs = 0
			}

			txs := make([]*ethTypes.Transaction, nTxs)

			ethClient := createEthClient(t, ethURL)
			defer ethClient.Close()
			for i := range txs {
				txs[i] = evm.GetRandomTransaction(t, TEST_PRIVATE_KEY, TEST_TO_ADDRESS, CHAIN_ID, 22000, &lastNonce)
				require.NoError(tt, ethClient.SendTransaction(context.Background(), txs[i]))
			}

			payload, err := executionClient.GetTxs(ctx)
			require.NoError(tt, err)
			require.Lenf(tt, payload, nTxs, "expected %d transactions, got %d", nTxs, len(payload))

			allPayloads = append(allPayloads, payload)

			// Check latest block before execution
			beforeHeight, beforeHash, beforeTxs := checkLatestBlock(tt, ctx, ethURL)
			require.Equal(tt, lastHeight, beforeHeight, "Latest block height should match")
			require.Equal(tt, lastHash.Hex(), beforeHash.Hex(), "Latest block hash should match")
			require.Equal(tt, lastTxs, beforeTxs, "Number of transactions should match")

			// Use incremented timestamp for each block to ensure proper ordering
			blockTimestamp := baseTimestamp.Add(time.Duration(blockHeight-initialHeight) * time.Second)
			allTimestamps = append(allTimestamps, blockTimestamp)

			// Execute transactions and get the new state root
			newStateRoot, maxBytes, err := executionClient.ExecuteTxs(ctx, payload, blockHeight, blockTimestamp, prevStateRoot)
			require.NoError(tt, err)
			if nTxs > 0 {
				require.NotZero(tt, maxBytes)
			}

			err = executionClient.SetFinal(ctx, blockHeight)
			require.NoError(tt, err)

			// Check latest block after execution
			lastHeight, lastHash, lastTxs = checkLatestBlock(tt, ctx, ethURL)
			require.Equal(tt, blockHeight, lastHeight, "Latest block height should match")
			require.NotEmpty(tt, lastHash.Hex(), "Latest block hash should not be empty")
			require.Equal(tt, lastTxs, nTxs, "Number of transactions should be equal")

			if nTxs == 0 {
				require.Equal(tt, prevStateRoot, newStateRoot)
			} else {
				require.NotEqual(tt, prevStateRoot, newStateRoot)
			}

			// Store the state root from build phase for later comparison in sync phase
			buildPhaseStateRoots = append(buildPhaseStateRoots, newStateRoot)
			tt.Logf("Build phase block %d: stored state root %x", blockHeight, newStateRoot)

			prevStateRoot = newStateRoot
		}
	})

	if t.Failed() {
		return
	}

	// start new container and try to sync
	t.Run("Sync chain", func(tt *testing.T) {
		rethNode := SetupTestRethNode(t)

		ni, err := rethNode.GetNetworkInfo(context.TODO())
		require.NoError(tt, err)
		ethURL := "http://127.0.0.1:" + ni.External.Ports.RPC
		engineURL := "http://127.0.0.1:" + ni.External.Ports.Engine

		executionClient, err := evm.NewEngineExecutionClient(
			ethURL,
			engineURL,
			rethNode.JWTSecretHex(),
			genesisHash,
			common.Address{},
		)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
		defer cancel()
		stateRoot, gasLimit, err := executionClient.InitChain(ctx, genesisTime, initialHeight, CHAIN_ID)
		require.NoError(t, err)
		require.Equal(t, GenesisStateRoot, stateRoot)
		require.NotZero(t, gasLimit)

		prevStateRoot := GenesisStateRoot
		lastHeight, lastHash, lastTxs := checkLatestBlock(tt, ctx, ethURL)

		for blockHeight := initialHeight; blockHeight <= 10; blockHeight++ {
			payload := allPayloads[blockHeight-1]

			// Check latest block before execution
			beforeHeight, beforeHash, beforeTxs := checkLatestBlock(tt, ctx, ethURL)
			require.Equal(tt, lastHeight, beforeHeight, "Latest block height should match")
			require.Equal(tt, lastHash.Hex(), beforeHash.Hex(), "Latest block hash should match")
			require.Equal(tt, lastTxs, beforeTxs, "Number of transactions should match")

			// Use timestamp from build phase for each block to ensure proper ordering
			blockTimestamp := allTimestamps[blockHeight-1]
			newStateRoot, maxBytes, err := executionClient.ExecuteTxs(ctx, payload, blockHeight, blockTimestamp, prevStateRoot)
			require.NoError(t, err)
			if len(payload) > 0 {
				require.NotZero(tt, maxBytes)
			}
			if len(payload) == 0 {
				require.Equal(tt, prevStateRoot, newStateRoot)
			} else {
				require.NotEqual(tt, prevStateRoot, newStateRoot)
			}

			// Verify that the sync phase state root matches the build phase state root
			expectedStateRoot := buildPhaseStateRoots[blockHeight-1]
			require.Equal(tt, expectedStateRoot, newStateRoot,
				"Sync phase state root for block %d should match build phase state root. Expected: %x, Got: %x",
				blockHeight, expectedStateRoot, newStateRoot)
			tt.Logf("Sync phase block %d: state root %x matches build phase ✓", blockHeight, newStateRoot)

			err = executionClient.SetFinal(ctx, blockHeight)
			require.NoError(tt, err)

			// Check latest block after execution
			lastHeight, lastHash, lastTxs = checkLatestBlock(tt, ctx, ethURL)
			require.Equal(tt, blockHeight, lastHeight, "Latest block height should match")
			require.NotEmpty(tt, lastHash.Hex(), "Latest block hash should not be empty")
			require.GreaterOrEqual(tt, lastTxs, 0, "Number of transactions should be non-negative")

			prevStateRoot = newStateRoot
		}
	})
}

// createEthClient creates an Ethereum client for checking block information
func createEthClient(t *testing.T, ethURL string) *ethclient.Client {
	t.Helper()

	// Use the same ETH URL as in the tests
	ethClient, err := ethclient.Dial(ethURL)
	require.NoError(t, err, "Failed to create Ethereum client")

	return ethClient
}

// checkLatestBlock retrieves and returns the latest block height, hash, and transaction count using Ethereum API
func checkLatestBlock(t *testing.T, ctx context.Context, ethURL string) (uint64, common.Hash, int) {
	t.Helper()

	// Create an Ethereum client
	ethClient := createEthClient(t, ethURL)
	defer ethClient.Close()

	// Get the latest block header
	header, err := ethClient.HeaderByNumber(ctx, nil) // nil means latest block
	if err != nil {
		t.Logf("Warning: Failed to get latest block header: %v", err)
		return 0, common.Hash{}, 0
	}

	blockNumber := header.Number.Uint64()
	blockHash := header.Hash()

	// Get the full block to count transactions
	block, err := ethClient.BlockByNumber(ctx, header.Number)
	if err != nil {
		t.Logf("Warning: Failed to get full block: %v", err)
		t.Logf("Latest block: height=%d, hash=%s, txs=unknown", blockNumber, blockHash.Hex())
		return blockNumber, blockHash, 0
	}

	txCount := len(block.Transactions())

	//t.Logf("Latest block: height=%d, hash=%s, txs=%d", blockNumber, blockHash.Hex(), txCount)
	return blockNumber, blockHash, txCount
}
