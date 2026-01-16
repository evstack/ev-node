package evm

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testGenesis creates a genesis configuration for testing.
func testGenesis() *core.Genesis {
	// Generate a test account with some balance
	testKey, _ := crypto.GenerateKey()
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)

	return &core.Genesis{
		Config:     params.AllDevChainProtocolChanges,
		Difficulty: big.NewInt(0),
		GasLimit:   30_000_000,
		Alloc: types.GenesisAlloc{
			testAddr: {Balance: new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18))},
		},
		Timestamp: uint64(time.Now().Unix()),
	}
}

// testDatastore creates an in-memory datastore for testing.
func testDatastore() ds.Batching {
	return sync.MutexWrap(ds.NewMapDatastore())
}

func TestNewEngineExecutionClientWithGeth(t *testing.T) {
	genesis := testGenesis()
	db := testDatastore()
	logger := zerolog.Nop()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	client, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, db, logger)
	require.NoError(t, err)
	require.NotNil(t, client)

	// Verify genesis hash is set
	assert.NotEqual(t, common.Hash{}, client.genesisHash)

	// Verify fee recipient is set
	assert.Equal(t, feeRecipient, client.feeRecipient)
}

func TestNewEngineExecutionClientWithGeth_NilDB(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	_, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, nil, logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db is required")
}

func TestNewEngineExecutionClientWithGeth_NilGenesis(t *testing.T) {
	db := testDatastore()
	logger := zerolog.Nop()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	_, err := NewEngineExecutionClientWithGeth(nil, feeRecipient, db, logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "genesis configuration is required")
}

func TestGethEngineClient_InitChain(t *testing.T) {
	genesis := testGenesis()
	db := testDatastore()
	logger := zerolog.Nop()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	client, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, db, logger)
	require.NoError(t, err)

	ctx := context.Background()
	genesisTime := time.Now()

	stateRoot, maxBytes, err := client.InitChain(ctx, genesisTime, 1, "1337")
	require.NoError(t, err)
	assert.NotEmpty(t, stateRoot)
	// maxBytes is the gas limit from the genesis block
	assert.Greater(t, maxBytes, uint64(0))
}

func TestGethEngineClient_GetLatestHeight(t *testing.T) {
	genesis := testGenesis()
	db := testDatastore()
	logger := zerolog.Nop()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	client, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, db, logger)
	require.NoError(t, err)

	ctx := context.Background()
	height, err := client.GetLatestHeight(ctx)
	require.NoError(t, err)
	// At genesis, height should be 0
	assert.Equal(t, uint64(0), height)
}

func TestGethEthClient_HeaderByNumber(t *testing.T) {
	genesis := testGenesis()
	db := testDatastore()
	logger := zerolog.Nop()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	client, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, db, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Get genesis block header (block 0)
	header, err := client.ethClient.HeaderByNumber(ctx, big.NewInt(0))
	require.NoError(t, err)
	assert.NotNil(t, header)
	assert.Equal(t, uint64(0), header.Number.Uint64())

	// Get latest header
	latestHeader, err := client.ethClient.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	assert.NotNil(t, latestHeader)
}

func TestGethEthClient_GetTxs_EmptyPool(t *testing.T) {
	genesis := testGenesis()
	db := testDatastore()
	logger := zerolog.Nop()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	client, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, db, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Empty mempool should return empty list
	txs, err := client.GetTxs(ctx)
	require.NoError(t, err)
	assert.Empty(t, txs)
}

func TestGethEngineClient_ExecuteTxs_EmptyBlock(t *testing.T) {
	genesis := testGenesis()
	db := testDatastore()
	logger := zerolog.Nop()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	client, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, db, logger)
	require.NoError(t, err)

	ctx := context.Background()
	genesisTime := time.Now()

	// Initialize chain first
	stateRoot, _, err := client.InitChain(ctx, genesisTime, 1, "1337")
	require.NoError(t, err)

	// Execute empty block
	newStateRoot, gasUsed, err := client.ExecuteTxs(
		ctx,
		[][]byte{}, // empty transactions
		1,
		genesisTime.Add(time.Second*12),
		stateRoot,
	)
	require.NoError(t, err)
	assert.NotEmpty(t, newStateRoot)
	assert.Equal(t, uint64(0), gasUsed) // No transactions, no gas used
}

func TestGethEngineClient_ForkchoiceUpdated(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	engineClient := &gethEngineClient{
		backend: backend,
		logger:  logger,
	}

	ctx := context.Background()
	genesisBlock := backend.blockchain.Genesis()

	// Test forkchoice update without payload attributes
	resp, err := engineClient.ForkchoiceUpdated(ctx, engine.ForkchoiceStateV1{
		HeadBlockHash:      genesisBlock.Hash(),
		SafeBlockHash:      genesisBlock.Hash(),
		FinalizedBlockHash: genesisBlock.Hash(),
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, engine.VALID, resp.PayloadStatus.Status)
	assert.Nil(t, resp.PayloadID)
}

func TestGethEngineClient_ForkchoiceUpdated_WithPayloadAttributes(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	engineClient := &gethEngineClient{
		backend: backend,
		logger:  logger,
	}

	ctx := context.Background()
	genesisBlock := backend.blockchain.Genesis()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	// Test forkchoice update with payload attributes
	attrs := map[string]any{
		"timestamp":             time.Now().Unix() + 12,
		"prevRandao":            common.Hash{1, 2, 3},
		"suggestedFeeRecipient": feeRecipient,
		"transactions":          []string{},
		"gasLimit":              uint64(30_000_000),
		"withdrawals":           []*types.Withdrawal{},
	}

	resp, err := engineClient.ForkchoiceUpdated(ctx, engine.ForkchoiceStateV1{
		HeadBlockHash:      genesisBlock.Hash(),
		SafeBlockHash:      genesisBlock.Hash(),
		FinalizedBlockHash: genesisBlock.Hash(),
	}, attrs)
	require.NoError(t, err)
	assert.Equal(t, engine.VALID, resp.PayloadStatus.Status)
	assert.NotNil(t, resp.PayloadID)
}

func TestGethEngineClient_GetPayload(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	engineClient := &gethEngineClient{
		backend: backend,
		logger:  logger,
	}

	ctx := context.Background()
	genesisBlock := backend.blockchain.Genesis()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	// First, create a payload
	attrs := map[string]any{
		"timestamp":             time.Now().Unix() + 12,
		"prevRandao":            common.Hash{1, 2, 3},
		"suggestedFeeRecipient": feeRecipient,
		"transactions":          []string{},
		"gasLimit":              uint64(30_000_000),
		"withdrawals":           []*types.Withdrawal{},
	}

	resp, err := engineClient.ForkchoiceUpdated(ctx, engine.ForkchoiceStateV1{
		HeadBlockHash:      genesisBlock.Hash(),
		SafeBlockHash:      genesisBlock.Hash(),
		FinalizedBlockHash: genesisBlock.Hash(),
	}, attrs)
	require.NoError(t, err)
	require.NotNil(t, resp.PayloadID)

	// Get the payload
	envelope, err := engineClient.GetPayload(ctx, *resp.PayloadID)
	require.NoError(t, err)
	assert.NotNil(t, envelope)
	assert.NotNil(t, envelope.ExecutionPayload)
	assert.Equal(t, uint64(1), envelope.ExecutionPayload.Number)
	assert.Equal(t, feeRecipient, envelope.ExecutionPayload.FeeRecipient)
}

func TestGethEngineClient_NewPayload(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	engineClient := &gethEngineClient{
		backend: backend,
		logger:  logger,
	}

	ctx := context.Background()
	genesisBlock := backend.blockchain.Genesis()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	// Create and get a payload
	attrs := map[string]any{
		"timestamp":             time.Now().Unix() + 12,
		"prevRandao":            common.Hash{1, 2, 3},
		"suggestedFeeRecipient": feeRecipient,
		"transactions":          []string{},
		"gasLimit":              uint64(30_000_000),
		"withdrawals":           []*types.Withdrawal{},
	}

	resp, err := engineClient.ForkchoiceUpdated(ctx, engine.ForkchoiceStateV1{
		HeadBlockHash:      genesisBlock.Hash(),
		SafeBlockHash:      genesisBlock.Hash(),
		FinalizedBlockHash: genesisBlock.Hash(),
	}, attrs)
	require.NoError(t, err)
	require.NotNil(t, resp.PayloadID)

	envelope, err := engineClient.GetPayload(ctx, *resp.PayloadID)
	require.NoError(t, err)

	// Submit the payload
	status, err := engineClient.NewPayload(ctx, envelope.ExecutionPayload, nil, "", nil)
	require.NoError(t, err)
	assert.Equal(t, engine.VALID, status.Status)
}

func TestGethEngineClient_ForkchoiceUpdated_UnknownHead(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	engineClient := &gethEngineClient{
		backend: backend,
		logger:  logger,
	}

	ctx := context.Background()

	// Try to set head to unknown block
	unknownHash := common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234")
	resp, err := engineClient.ForkchoiceUpdated(ctx, engine.ForkchoiceStateV1{
		HeadBlockHash:      unknownHash,
		SafeBlockHash:      unknownHash,
		FinalizedBlockHash: unknownHash,
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, engine.SYNCING, resp.PayloadStatus.Status)
}

func TestGethBackend_Close(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)

	// Close should not error
	err = backend.Close()
	require.NoError(t, err)
}

func TestCalcBaseFee(t *testing.T) {
	config := params.AllDevChainProtocolChanges

	// Create a mock parent header
	parent := &types.Header{
		Number:   big.NewInt(100),
		GasLimit: 30_000_000,
		GasUsed:  15_000_000, // 50% full
		BaseFee:  big.NewInt(1_000_000_000),
	}

	baseFee := calcBaseFee(config, parent)
	require.NotNil(t, baseFee)

	// When block is exactly 50% full, base fee should remain the same
	assert.Equal(t, parent.BaseFee, baseFee)
}

func TestCalcBaseFee_OverTarget(t *testing.T) {
	config := params.AllDevChainProtocolChanges

	// Create a mock parent header that was more than 50% full
	parent := &types.Header{
		Number:   big.NewInt(100),
		GasLimit: 30_000_000,
		GasUsed:  20_000_000, // ~67% full
		BaseFee:  big.NewInt(1_000_000_000),
	}

	baseFee := calcBaseFee(config, parent)
	require.NotNil(t, baseFee)

	// Base fee should increase when block is more than 50% full
	assert.Greater(t, baseFee.Int64(), parent.BaseFee.Int64())
}

func TestCalcBaseFee_UnderTarget(t *testing.T) {
	config := params.AllDevChainProtocolChanges

	// Create a mock parent header that was less than 50% full
	parent := &types.Header{
		Number:   big.NewInt(100),
		GasLimit: 30_000_000,
		GasUsed:  5_000_000, // ~17% full
		BaseFee:  big.NewInt(1_000_000_000),
	}

	baseFee := calcBaseFee(config, parent)
	require.NotNil(t, baseFee)

	// Base fee should decrease when block is less than 50% full
	assert.Less(t, baseFee.Int64(), parent.BaseFee.Int64())
}

func TestParsePayloadAttributes(t *testing.T) {
	logger := zerolog.Nop()
	engineClient := &gethEngineClient{logger: logger}

	parentHash := common.HexToHash("0xabcd")
	feeRecipient := common.HexToAddress("0x1234")
	timestamp := int64(1234567890)

	attrs := map[string]any{
		"timestamp":             timestamp,
		"prevRandao":            common.Hash{1, 2, 3},
		"suggestedFeeRecipient": feeRecipient,
		"transactions":          []string{"0xaabbcc", "0xddeeff"},
		"gasLimit":              uint64(30_000_000),
	}

	state, err := engineClient.parsePayloadAttributes(parentHash, attrs)
	require.NoError(t, err)
	assert.Equal(t, parentHash, state.parentHash)
	assert.Equal(t, uint64(timestamp), state.timestamp)
	assert.Equal(t, feeRecipient, state.feeRecipient)
	assert.Equal(t, uint64(30_000_000), state.gasLimit)
	assert.Len(t, state.transactions, 2)
}

func TestListHasher(t *testing.T) {
	hasher := trie.NewListHasher()

	// Test Update and Hash
	err := hasher.Update([]byte("key1"), []byte("value1"))
	require.NoError(t, err)

	hash1 := hasher.Hash()
	assert.NotEqual(t, common.Hash{}, hash1)

	// Test Reset
	hasher.Reset()
	err = hasher.Update([]byte("key2"), []byte("value2"))
	require.NoError(t, err)

	hash2 := hasher.Hash()
	assert.NotEqual(t, common.Hash{}, hash2)
	assert.NotEqual(t, hash1, hash2)
}

func TestCreateBloomFromReceipts(t *testing.T) {
	// Empty receipts
	bloom := createBloomFromReceipts([]*types.Receipt{})
	assert.Equal(t, types.Bloom{}, bloom)

	// Single receipt with no logs
	receipt := &types.Receipt{
		Status: types.ReceiptStatusSuccessful,
		Logs:   []*types.Log{},
	}
	bloom = createBloomFromReceipts([]*types.Receipt{receipt})
	assert.NotNil(t, bloom)
}
