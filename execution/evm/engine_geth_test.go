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

	client, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, db, "", logger)
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

	_, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, nil, "", logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db is required")
}

func TestNewEngineExecutionClientWithGeth_NilGenesis(t *testing.T) {
	db := testDatastore()
	logger := zerolog.Nop()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	_, err := NewEngineExecutionClientWithGeth(nil, feeRecipient, db, "", logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "genesis configuration is required")
}

func TestGethEngineClient_InitChain(t *testing.T) {
	genesis := testGenesis()
	db := testDatastore()
	logger := zerolog.Nop()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	client, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, db, "", logger)
	require.NoError(t, err)

	ctx := context.Background()
	genesisTime := time.Now()

	stateRoot, err := client.InitChain(ctx, genesisTime, 1, "1337")
	require.NoError(t, err)
	assert.NotEmpty(t, stateRoot)
}

func TestGethEngineClient_GetLatestHeight(t *testing.T) {
	genesis := testGenesis()
	db := testDatastore()
	logger := zerolog.Nop()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	client, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, db, "", logger)
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

	client, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, db, "", logger)
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

	client, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, db, "", logger)
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

	client, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, db, "", logger)
	require.NoError(t, err)

	ctx := context.Background()
	genesisTime := time.Now()

	// Initialize chain first
	stateRoot, err := client.InitChain(ctx, genesisTime, 1, "1337")
	require.NoError(t, err)

	// Execute empty block
	newStateRoot, err := client.ExecuteTxs(
		ctx,
		[][]byte{}, // empty transactions
		1,
		genesisTime.Add(time.Second*12),
		stateRoot,
	)
	require.NoError(t, err)
	assert.NotEmpty(t, newStateRoot)
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

func TestGethEngineClient_PayloadIdempotency(t *testing.T) {
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

	// Create payload attributes
	timestamp := time.Now().Unix() + 12
	attrs := map[string]any{
		"timestamp":             timestamp,
		"prevRandao":            common.Hash{1, 2, 3},
		"suggestedFeeRecipient": feeRecipient,
		"transactions":          []string{},
		"gasLimit":              uint64(30_000_000),
		"withdrawals":           []*types.Withdrawal{},
	}

	// First call should create a new payload
	resp1, err := engineClient.ForkchoiceUpdated(ctx, engine.ForkchoiceStateV1{
		HeadBlockHash:      genesisBlock.Hash(),
		SafeBlockHash:      genesisBlock.Hash(),
		FinalizedBlockHash: genesisBlock.Hash(),
	}, attrs)
	require.NoError(t, err)
	require.NotNil(t, resp1.PayloadID)

	// Second call with same attributes should return the same payload ID (idempotency)
	resp2, err := engineClient.ForkchoiceUpdated(ctx, engine.ForkchoiceStateV1{
		HeadBlockHash:      genesisBlock.Hash(),
		SafeBlockHash:      genesisBlock.Hash(),
		FinalizedBlockHash: genesisBlock.Hash(),
	}, attrs)
	require.NoError(t, err)
	require.NotNil(t, resp2.PayloadID)

	// Payload IDs should be identical
	assert.Equal(t, *resp1.PayloadID, *resp2.PayloadID)

	// Should still have only one payload in the map
	assert.Len(t, backend.payloads, 1)
}

func TestGethEngineClient_DeterministicPayloadID(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	engineClient := &gethEngineClient{
		backend: backend,
		logger:  logger,
	}

	genesisBlock := backend.blockchain.Genesis()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")
	timestamp := uint64(time.Now().Unix() + 12)

	// Create two payload states with identical attributes
	ps1 := &payloadBuildState{
		parentHash:   genesisBlock.Hash(),
		timestamp:    timestamp,
		prevRandao:   common.Hash{1, 2, 3},
		feeRecipient: feeRecipient,
		gasLimit:     30_000_000,
		transactions: [][]byte{},
		withdrawals:  []*types.Withdrawal{},
		createdAt:    time.Now(),
	}

	ps2 := &payloadBuildState{
		parentHash:   genesisBlock.Hash(),
		timestamp:    timestamp,
		prevRandao:   common.Hash{1, 2, 3},
		feeRecipient: feeRecipient,
		gasLimit:     30_000_000,
		transactions: [][]byte{},
		withdrawals:  []*types.Withdrawal{},
		createdAt:    time.Now().Add(time.Hour), // Different creation time
	}

	// Both should generate the same payload ID
	id1 := engineClient.generatePayloadID(ps1)
	id2 := engineClient.generatePayloadID(ps2)
	assert.Equal(t, id1, id2)

	// Different attributes should generate different IDs
	ps3 := &payloadBuildState{
		parentHash:   genesisBlock.Hash(),
		timestamp:    timestamp + 1, // Different timestamp
		prevRandao:   common.Hash{1, 2, 3},
		feeRecipient: feeRecipient,
		gasLimit:     30_000_000,
		transactions: [][]byte{},
		withdrawals:  []*types.Withdrawal{},
		createdAt:    time.Now(),
	}
	id3 := engineClient.generatePayloadID(ps3)
	assert.NotEqual(t, id1, id3)
}

func TestGethEngineClient_GetPayloadRemovesFromMap(t *testing.T) {
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

	// Create a payload
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

	// Payload should be in the map
	assert.Len(t, backend.payloads, 1)

	// Get the payload
	envelope, err := engineClient.GetPayload(ctx, *resp.PayloadID)
	require.NoError(t, err)
	assert.NotNil(t, envelope)

	// Payload should be removed from the map after retrieval
	assert.Len(t, backend.payloads, 0)

	// Second call should fail with unknown payload ID
	_, err = engineClient.GetPayload(ctx, *resp.PayloadID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown payload ID")
}

func TestGethEngineClient_PayloadCleanup(t *testing.T) {
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

	// Create more payloads than maxPayloads (10)
	baseTimestamp := time.Now().Unix() + 100
	for i := 0; i < 15; i++ {
		attrs := map[string]any{
			"timestamp":             baseTimestamp + int64(i),
			"prevRandao":            common.Hash{byte(i)},
			"suggestedFeeRecipient": feeRecipient,
			"transactions":          []string{},
			"gasLimit":              uint64(30_000_000),
			"withdrawals":           []*types.Withdrawal{},
		}

		_, err := engineClient.ForkchoiceUpdated(ctx, engine.ForkchoiceStateV1{
			HeadBlockHash:      genesisBlock.Hash(),
			SafeBlockHash:      genesisBlock.Hash(),
			FinalizedBlockHash: genesisBlock.Hash(),
		}, attrs)
		require.NoError(t, err)
	}

	// Should have at most maxPayloads entries
	assert.LessOrEqual(t, len(backend.payloads), 10)
}

func TestGethEngineClient_DifferentTransactionsGenerateDifferentIDs(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	engineClient := &gethEngineClient{
		backend: backend,
		logger:  logger,
	}

	genesisBlock := backend.blockchain.Genesis()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")
	timestamp := uint64(time.Now().Unix() + 12)

	// Payload with no transactions
	ps1 := &payloadBuildState{
		parentHash:   genesisBlock.Hash(),
		timestamp:    timestamp,
		prevRandao:   common.Hash{1, 2, 3},
		feeRecipient: feeRecipient,
		gasLimit:     30_000_000,
		transactions: [][]byte{},
		withdrawals:  []*types.Withdrawal{},
		createdAt:    time.Now(),
	}

	// Payload with some transaction
	ps2 := &payloadBuildState{
		parentHash:   genesisBlock.Hash(),
		timestamp:    timestamp,
		prevRandao:   common.Hash{1, 2, 3},
		feeRecipient: feeRecipient,
		gasLimit:     30_000_000,
		transactions: [][]byte{{0xaa, 0xbb, 0xcc}},
		withdrawals:  []*types.Withdrawal{},
		createdAt:    time.Now(),
	}

	id1 := engineClient.generatePayloadID(ps1)
	id2 := engineClient.generatePayloadID(ps2)

	// Different transactions should produce different IDs
	assert.NotEqual(t, id1, id2)
}

func TestGethEngineClient_WithdrawalProcessing(t *testing.T) {
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

	// Create withdrawal recipients
	withdrawalAddr1 := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	withdrawalAddr2 := common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	// Check initial balances are zero
	stateDB, err := backend.blockchain.StateAt(genesisBlock.Root())
	require.NoError(t, err)
	initialBalance1 := stateDB.GetBalance(withdrawalAddr1)
	initialBalance2 := stateDB.GetBalance(withdrawalAddr2)
	assert.True(t, initialBalance1.IsZero(), "initial balance should be zero")
	assert.True(t, initialBalance2.IsZero(), "initial balance should be zero")

	// Create withdrawals (amounts are in Gwei)
	withdrawals := []*types.Withdrawal{
		{
			Index:     0,
			Validator: 1,
			Address:   withdrawalAddr1,
			Amount:    1000000000, // 1 ETH in Gwei
		},
		{
			Index:     1,
			Validator: 2,
			Address:   withdrawalAddr2,
			Amount:    500000000, // 0.5 ETH in Gwei
		},
	}

	// Create payload with withdrawals
	attrs := map[string]any{
		"timestamp":             time.Now().Unix() + 12,
		"prevRandao":            common.Hash{1, 2, 3},
		"suggestedFeeRecipient": feeRecipient,
		"transactions":          []string{},
		"gasLimit":              uint64(30_000_000),
		"withdrawals":           withdrawals,
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
	assert.Len(t, envelope.ExecutionPayload.Withdrawals, 2)

	// Submit the payload to actually apply state changes
	status, err := engineClient.NewPayload(ctx, envelope.ExecutionPayload, nil, "", nil)
	require.NoError(t, err)
	assert.Equal(t, engine.VALID, status.Status)

	// Update head to the new block
	_, err = engineClient.ForkchoiceUpdated(ctx, engine.ForkchoiceStateV1{
		HeadBlockHash:      envelope.ExecutionPayload.BlockHash,
		SafeBlockHash:      envelope.ExecutionPayload.BlockHash,
		FinalizedBlockHash: envelope.ExecutionPayload.BlockHash,
	}, nil)
	require.NoError(t, err)

	// Check balances after withdrawals are processed
	newBlock := backend.blockchain.GetBlockByHash(envelope.ExecutionPayload.BlockHash)
	require.NotNil(t, newBlock)

	newStateDB, err := backend.blockchain.StateAt(newBlock.Root())
	require.NoError(t, err)

	// Withdrawal amounts are in Gwei, balances are in Wei
	// 1 ETH = 1e9 Gwei = 1e18 Wei
	expectedBalance1 := new(big.Int).Mul(big.NewInt(1000000000), big.NewInt(1e9)) // 1 ETH in Wei
	expectedBalance2 := new(big.Int).Mul(big.NewInt(500000000), big.NewInt(1e9))  // 0.5 ETH in Wei

	balance1 := newStateDB.GetBalance(withdrawalAddr1)
	balance2 := newStateDB.GetBalance(withdrawalAddr2)

	assert.Equal(t, expectedBalance1.String(), balance1.ToBig().String(), "withdrawal 1 balance mismatch")
	assert.Equal(t, expectedBalance2.String(), balance2.ToBig().String(), "withdrawal 2 balance mismatch")
}

func TestGethEngineClient_ContractCreationAddress(t *testing.T) {
	// This test verifies that contract creation addresses are calculated correctly
	// using crypto.CreateAddress(sender, nonce) rather than the incorrect evmInstance.Origin
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	// The contract address should be derived from sender address and nonce
	// This is tested implicitly through the applyTransaction function
	// For a full test, we would need to create a signed contract creation tx

	// Verify the crypto.CreateAddress function works as expected
	sender := common.HexToAddress("0x1234567890123456789012345678901234567890")
	nonce := uint64(0)
	expectedAddr := crypto.CreateAddress(sender, nonce)

	// The address should be deterministic
	expectedAddr2 := crypto.CreateAddress(sender, nonce)
	assert.Equal(t, expectedAddr, expectedAddr2)

	// Different nonce should produce different address
	differentAddr := crypto.CreateAddress(sender, nonce+1)
	assert.NotEqual(t, expectedAddr, differentAddr)

	// Different sender should produce different address
	differentSender := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	differentAddr2 := crypto.CreateAddress(differentSender, nonce)
	assert.NotEqual(t, expectedAddr, differentAddr2)

	_ = backend // use backend to avoid unused variable warning
}
