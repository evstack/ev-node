package single

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/evstack/ev-node/core/execution"
	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/test/testda"
)

// MockFullDAClient combines MockClient and MockVerifier to implement FullDAClient
type MockFullDAClient struct {
	*mocks.MockClient
	*mocks.MockVerifier
}

func newMockFullDAClient(t *testing.T) *MockFullDAClient {
	return &MockFullDAClient{
		MockClient:   mocks.NewMockClient(t),
		MockVerifier: mocks.NewMockVerifier(t),
	}
}

func newDummyDA(maxBlobSize uint64) *testda.DummyDA {
	return testda.New(testda.WithMaxBlobSize(maxBlobSize))
}

// newTestSequencer creates a sequencer for tests that don't need full initialization
func newTestSequencer(t *testing.T, db ds.Batching, daClient block.FullDAClient) *Sequencer {
	logger := zerolog.Nop()

	gen := genesis.Genesis{
		ChainID:       "test",
		DAStartHeight: 100,
	}

	seq, err := NewSequencer(
		logger,
		db,
		daClient,
		config.DefaultConfig(),
		[]byte("test"),
		0, // unlimited queue
		gen,
		&mockExecutor{},
	)
	require.NoError(t, err)
	return seq
}

func TestSequencer_SubmitBatchTxs(t *testing.T) {
	dummyDA := newDummyDA(100_000_000)
	db := ds.NewMapDatastore()
	Id := []byte("test1")
	logger := zerolog.Nop()
	seq, err := NewSequencer(logger, db, dummyDA, config.DefaultConfig(), Id, 1000, genesis.Genesis{}, &mockExecutor{})
	if err != nil {
		t.Fatalf("Failed to create sequencer: %v", err)
	}
	defer func() {
		err := db.Close()
		if err != nil {
			t.Fatalf("Failed to close sequencer: %v", err)
		}
	}()

	// Test with initial ID
	tx := []byte("transaction1")

	res, err := seq.SubmitBatchTxs(context.Background(), coresequencer.SubmitBatchTxsRequest{Id: Id, Batch: &coresequencer.Batch{Transactions: [][]byte{tx}}})
	if err != nil {
		t.Fatalf("Failed to submit transaction: %v", err)
	}
	if res == nil {
		t.Fatal("Expected response to not be nil")
	}

	// Verify the transaction was added
	nextBatchresp, err := seq.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{Id: Id})
	if err != nil {
		t.Fatalf("Failed to get next batch: %v", err)
	}
	if len(nextBatchresp.Batch.Transactions) != 1 {
		t.Fatalf("Expected 1 transaction, got %d", len(nextBatchresp.Batch.Transactions))
	}

	// Test with a different ID (expecting an error due to mismatch)
	res, err = seq.SubmitBatchTxs(context.Background(), coresequencer.SubmitBatchTxsRequest{Id: []byte("test2"), Batch: &coresequencer.Batch{Transactions: [][]byte{tx}}})
	if err == nil {
		t.Fatal("Expected error for invalid ID, got nil")
	}
	if !errors.Is(err, ErrInvalidId) {
		t.Fatalf("Expected ErrInvalidId, got %v", err)
	}
	if res != nil {
		t.Fatal("Expected nil response for error case")
	}
}

func TestSequencer_SubmitBatchTxs_EmptyBatch(t *testing.T) {
	dummyDA := newDummyDA(100_000_000)
	db := ds.NewMapDatastore()
	Id := []byte("test1")
	logger := zerolog.Nop()
	seq, err := NewSequencer(logger, db, dummyDA, config.DefaultConfig(), Id, 1000, genesis.Genesis{}, &mockExecutor{})
	require.NoError(t, err, "Failed to create sequencer")
	defer func() {
		err := db.Close()
		require.NoError(t, err, "Failed to close sequencer")
	}()

	// Test submitting an empty batch
	res, err := seq.SubmitBatchTxs(context.Background(), coresequencer.SubmitBatchTxsRequest{
		Id:    Id,
		Batch: &coresequencer.Batch{Transactions: [][]byte{}}, // Empty transactions
	})
	require.NoError(t, err, "Submitting empty batch should not return an error")
	require.NotNil(t, res, "Response should not be nil for empty batch submission")

	// Verify that no batch was added to the queue
	nextBatchResp, err := seq.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{Id: Id})
	require.NoError(t, err, "Getting next batch after empty submission failed")
	require.NotNil(t, nextBatchResp, "GetNextBatch response should not be nil")
	require.Empty(t, nextBatchResp.Batch.Transactions, "Queue should be empty after submitting an empty batch")

	// Test submitting a nil batch
	res, err = seq.SubmitBatchTxs(context.Background(), coresequencer.SubmitBatchTxsRequest{
		Id:    Id,
		Batch: nil, // Nil batch
	})
	require.NoError(t, err, "Submitting nil batch should not return an error")
	require.NotNil(t, res, "Response should not be nil for nil batch submission")

	// Verify again that no batch was added to the queue
	nextBatchResp, err = seq.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{Id: Id})
	require.NoError(t, err, "Getting next batch after nil submission failed")
	require.NotNil(t, nextBatchResp, "GetNextBatch response should not be nil")
	require.Empty(t, nextBatchResp.Batch.Transactions, "Queue should still be empty after submitting a nil batch")
}

func TestSequencer_GetNextBatch_NoLastBatch(t *testing.T) {
	db := ds.NewMapDatastore()
	ctx := context.Background()
	logger := zerolog.Nop()
	dummyDA := newDummyDA(100_000_000)

	gen := genesis.Genesis{
		ChainID:       "test",
		DAStartHeight: 100,
	}

	seq, err := NewSequencer(
		logger,
		db,
		dummyDA,
		config.DefaultConfig(),
		[]byte("test"),
		0, // unlimited queue
		gen,
		&mockExecutor{},
	)
	require.NoError(t, err)

	defer func() {
		err := db.Close()
		if err != nil {
			t.Fatalf("Failed to close sequencer: %v", err)
		}
	}()

	// Test case where lastBatchHash and seq.lastBatchHash are both nil
	res, err := seq.GetNextBatch(ctx, coresequencer.GetNextBatchRequest{Id: seq.Id})
	if err != nil {
		t.Fatalf("Failed to get next batch: %v", err)
	}

	// Ensure the time is approximately the same
	if res.Timestamp.Day() != time.Now().Day() {
		t.Fatalf("Expected timestamp day to be %d, got %d", time.Now().Day(), res.Timestamp.Day())
	}

	// Should return an empty batch
	if len(res.Batch.Transactions) != 0 {
		t.Fatalf("Expected empty batch, got %d transactions", len(res.Batch.Transactions))
	}
}

func TestSequencer_GetNextBatch_Success(t *testing.T) {
	// Initialize a new sequencer with a mock batch
	mockBatch := &coresequencer.Batch{Transactions: [][]byte{[]byte("tx1"), []byte("tx2")}}

	db := ds.NewMapDatastore()

	dummyDA := newDummyDA(100_000_000)
	seq := newTestSequencer(t, db, dummyDA)
	defer func() {
		err := db.Close()
		if err != nil {
			t.Fatalf("Failed to close sequencer: %v", err)
		}
	}()

	// Add mock batch to the BatchQueue
	err := seq.queue.AddBatch(context.Background(), *mockBatch)
	if err != nil {
		t.Fatalf("Failed to add batch: %v", err)
	}

	// Test success case with no previous lastBatchHash
	res, err := seq.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{Id: seq.Id})
	if err != nil {
		t.Fatalf("Failed to get next batch: %v", err)
	}

	// Ensure the time is approximately the same
	if res.Timestamp.Day() != time.Now().Day() {
		t.Fatalf("Expected timestamp day to be %d, got %d", time.Now().Day(), res.Timestamp.Day())
	}

	// Ensure that the transactions are present
	if len(res.Batch.Transactions) != 2 {
		t.Fatalf("Expected 2 transactions, got %d", len(res.Batch.Transactions))
	}

	batchHash, err := mockBatch.Hash()
	if err != nil {
		t.Fatalf("Failed to get batch hash: %v", err)
	}
	if len(batchHash) == 0 {
		t.Fatal("Expected batch hash to not be empty")
	}
}

func TestSequencer_VerifyBatch(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	db := ds.NewMapDatastore()
	defer func() {
		err := db.Close()
		require.NoError(err, "Failed to close datastore")
	}()

	batchData := [][]byte{[]byte("batch1"), []byte("batch2")}

	t.Run("Valid Batch", func(t *testing.T) {
		dummyDA := newDummyDA(100_000_000)
		db := ds.NewMapDatastore()
		seq := newTestSequencer(t, db, dummyDA)
		defer db.Close()

		// DummyDA always validates successfully
		res, err := seq.VerifyBatch(context.Background(), coresequencer.VerifyBatchRequest{Id: seq.Id, BatchData: batchData})
		assert.NoError(err)
		assert.NotNil(res)
		assert.True(res.Status, "Expected status to be true for valid batch")
	})

	t.Run("Invalid ID", func(t *testing.T) {
		dummyDA := newDummyDA(100_000_000)
		db := ds.NewMapDatastore()
		seq := newTestSequencer(t, db, dummyDA)
		defer db.Close()

		invalidId := []byte("invalid")
		res, err := seq.VerifyBatch(context.Background(), coresequencer.VerifyBatchRequest{Id: invalidId, BatchData: batchData})
		assert.Error(err)
		assert.Nil(res)
		assert.ErrorIs(err, ErrInvalidId)
	})
}

func TestSequencer_GetNextBatch_BeforeDASubmission(t *testing.T) {
	// Initialize a new sequencer with dummy DA
	dummyDA := newDummyDA(100_000_000)
	db := ds.NewMapDatastore()
	logger := zerolog.Nop()
	Id := []byte("test1")
	seq, err := NewSequencer(logger, db, dummyDA, config.DefaultConfig(), Id, 1000, genesis.Genesis{}, &mockExecutor{})
	if err != nil {
		t.Fatalf("Failed to create sequencer: %v", err)
	}
	defer func() {
		err := db.Close()
		if err != nil {
			t.Fatalf("Failed to close sequencer: %v", err)
		}
	}()

	// Submit a batch
	tx := []byte("transaction1")
	res, err := seq.SubmitBatchTxs(context.Background(), coresequencer.SubmitBatchTxsRequest{
		Id:    Id,
		Batch: &coresequencer.Batch{Transactions: [][]byte{tx}},
	})
	if err != nil {
		t.Fatalf("Failed to submit transaction: %v", err)
	}
	if res == nil {
		t.Fatal("Expected response to not be nil")
	}
	time.Sleep(100 * time.Millisecond)

	// Try to get the batch before DA submission
	nextBatchResp, err := seq.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{Id: Id})
	if err != nil {
		t.Fatalf("Failed to get next batch: %v", err)
	}
	if len(nextBatchResp.Batch.Transactions) != 1 {
		t.Fatalf("Expected 1 transaction, got %d", len(nextBatchResp.Batch.Transactions))
	}
	if !bytes.Equal(nextBatchResp.Batch.Transactions[0], tx) {
		t.Fatal("Expected transaction to match submitted transaction")
	}

}

func TestSequencer_GetNextBatch_ForcedInclusionAndBatch_MaxBytes(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(zerolog.NewConsoleWriter())

	// Create in-memory datastore
	db := ds.NewMapDatastore()
	defer db.Close()

	// Create mock DA client with forced inclusion namespace
	mockDA := newMockFullDAClient(t)
	forcedInclusionNS := []byte("forced-inclusion")

	// Setup namespace methods
	mockDA.MockClient.On("GetHeaderNamespace").Return([]byte("header")).Maybe()
	mockDA.MockClient.On("GetDataNamespace").Return([]byte("data")).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return(forcedInclusionNS).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	// Create forced inclusion txs that are 50 and 60 bytes
	forcedTx1 := make([]byte, 50)
	forcedTx2 := make([]byte, 60)

	// Mock Retrieve to return forced inclusion txs at DA height 100
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(100), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess},
		Data:       [][]byte{forcedTx1, forcedTx2},
	}).Once()

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq, err := NewSequencer(
		logger,
		db,
		mockDA,
		config.DefaultConfig(),
		[]byte("test-chain"),
		1000,
		gen,
		&mockExecutor{},
	)
	require.NoError(t, err)

	// Submit batch txs that are 40 bytes each
	batchTx1 := make([]byte, 40)
	batchTx2 := make([]byte, 40)
	batchTx3 := make([]byte, 40)

	submitReq := coresequencer.SubmitBatchTxsRequest{
		Id: []byte("test-chain"),
		Batch: &coresequencer.Batch{
			Transactions: [][]byte{batchTx1, batchTx2, batchTx3}, // Total 120 bytes
		},
	}

	_, err = seq.SubmitBatchTxs(ctx, submitReq)
	require.NoError(t, err)

	// Request batch with maxBytes = 150
	// Forced inclusion: 110 bytes (50 + 60)
	// Batch txs: 120 bytes (40 + 40 + 40)
	// Combined would be 230 bytes, exceeds 150
	// Should return forced txs + only 1 batch tx (110 + 40 = 150)
	getReq := coresequencer.GetNextBatchRequest{
		Id:            []byte("test-chain"),
		MaxBytes:      150,
		LastBatchData: nil,
	}

	resp, err := seq.GetNextBatch(ctx, getReq)
	require.NoError(t, err)
	require.NotNil(t, resp.Batch)

	// Should have forced txs (2) + partial batch txs
	// Total size should not exceed 150 bytes
	totalSize := 0
	for _, tx := range resp.Batch.Transactions {
		totalSize += len(tx)
	}
	assert.LessOrEqual(t, totalSize, 150, "Total batch size should not exceed maxBytes")

	// First 2 txs should be forced inclusion txs
	assert.GreaterOrEqual(t, len(resp.Batch.Transactions), 2, "Should have at least forced inclusion txs")
	assert.Equal(t, forcedTx1, resp.Batch.Transactions[0])
	assert.Equal(t, forcedTx2, resp.Batch.Transactions[1])
}

func TestSequencer_GetNextBatch_ForcedInclusion_ExceedsMaxBytes(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(zerolog.NewConsoleWriter())

	db := ds.NewMapDatastore()
	defer db.Close()

	// Create mock DA client
	mockDA := newMockFullDAClient(t)
	forcedInclusionNS := []byte("forced-inclusion")

	mockDA.MockClient.On("GetHeaderNamespace").Return([]byte("header")).Maybe()
	mockDA.MockClient.On("GetDataNamespace").Return([]byte("data")).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return(forcedInclusionNS).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	// Create forced inclusion txs where combined they exceed maxBytes
	forcedTx1 := make([]byte, 100)
	forcedTx2 := make([]byte, 80) // This would be deferred

	// First DA retrieve at height 100
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(100), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess},
		Data:       [][]byte{forcedTx1, forcedTx2},
	}).Once()

	// Second DA retrieve at height 101 (empty)
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(101), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess},
		Data:       [][]byte{},
	}).Maybe()

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq, err := NewSequencer(
		logger,
		db,
		mockDA,
		config.DefaultConfig(),
		[]byte("test-chain"),
		1000,
		gen,
		&mockExecutor{},
	)
	require.NoError(t, err)

	// Request batch with maxBytes = 120
	getReq := coresequencer.GetNextBatchRequest{
		Id:            []byte("test-chain"),
		MaxBytes:      120,
		LastBatchData: nil,
	}

	// First call - should get only first forced tx (100 bytes)
	resp, err := seq.GetNextBatch(ctx, getReq)
	require.NoError(t, err)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 1, len(resp.Batch.Transactions), "Should only include first forced tx")
	assert.Equal(t, 100, len(resp.Batch.Transactions[0]))

	// Verify checkpoint reflects that we've consumed one tx
	assert.Equal(t, uint64(1), seq.checkpoint.TxIndex, "Should have consumed one tx from cache")

	// Second call - should get the pending forced tx
	resp2, err := seq.GetNextBatch(ctx, getReq)
	require.NoError(t, err)
	require.NotNil(t, resp2.Batch)
	assert.Equal(t, 1, len(resp2.Batch.Transactions), "Should include pending forced tx")
	assert.Equal(t, 80, len(resp2.Batch.Transactions[0]))

	// Checkpoint should have moved to next DA height after consuming all cached txs
	assert.Equal(t, uint64(101), seq.checkpoint.DAHeight, "Should have moved to next DA height")
	assert.Equal(t, uint64(0), seq.checkpoint.TxIndex, "TxIndex should be reset")
}

func TestSequencer_GetNextBatch_AlwaysCheckPendingForcedInclusion(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(zerolog.NewConsoleWriter())

	db := ds.NewMapDatastore()
	defer db.Close()

	// Create mock DA client
	mockDA := newMockFullDAClient(t)
	forcedInclusionNS := []byte("forced-inclusion")

	mockDA.MockClient.On("GetHeaderNamespace").Return([]byte("header")).Maybe()
	mockDA.MockClient.On("GetDataNamespace").Return([]byte("data")).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return(forcedInclusionNS).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	// First call returns large forced txs
	largeForcedTx1, largeForcedTx2 := make([]byte, 75), make([]byte, 75)
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(100), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess},
		Data:       [][]byte{largeForcedTx1, largeForcedTx2},
	}).Once()

	// Second call (height 101) returns empty
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(101), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess},
		Data:       [][]byte{},
	}).Maybe()

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq, err := NewSequencer(
		logger,
		db,
		mockDA,
		config.DefaultConfig(),
		[]byte("test-chain"),
		1000,
		gen,
		&mockExecutor{},
	)
	require.NoError(t, err)

	// Submit a batch tx
	batchTx := make([]byte, 50)
	submitReq := coresequencer.SubmitBatchTxsRequest{
		Id: []byte("test-chain"),
		Batch: &coresequencer.Batch{
			Transactions: [][]byte{batchTx},
		},
	}
	_, err = seq.SubmitBatchTxs(ctx, submitReq)
	require.NoError(t, err)

	// First call with maxBytes = 125
	getReq := coresequencer.GetNextBatchRequest{
		Id:            []byte("test-chain"),
		MaxBytes:      125,
		LastBatchData: nil,
	}

	resp, err := seq.GetNextBatch(ctx, getReq)
	require.NoError(t, err)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 2, len(resp.Batch.Transactions), "Should have 1 forced tx + 1 batch tx")
	assert.Equal(t, 75, len(resp.Batch.Transactions[0])) // forced tx is 75 bytes
	assert.Equal(t, 50, len(resp.Batch.Transactions[1])) // batch tx is 50 bytes

	// Verify checkpoint shows one forced tx was consumed
	assert.Equal(t, uint64(1), seq.checkpoint.TxIndex, "Only one forced tx should be consumed")
	assert.Greater(t, len(seq.cachedForcedInclusionTxs), 1, "Remaining forced tx should still be cached")

	// Second call with larger maxBytes = 200
	// Should process pending forced tx first
	getReq2 := coresequencer.GetNextBatchRequest{
		Id:            []byte("test-chain"),
		MaxBytes:      200,
		LastBatchData: nil,
	}

	resp2, err := seq.GetNextBatch(ctx, getReq2)
	require.NoError(t, err)
	require.NotNil(t, resp2.Batch)
	assert.Equal(t, 1, len(resp2.Batch.Transactions), "Should include pending forced tx")
	assert.Equal(t, 75, len(resp2.Batch.Transactions[0]))

	// Checkpoint should reflect that forced tx was consumed
	assert.Equal(t, uint64(101), seq.checkpoint.DAHeight, "Should have moved to next DA height")
	assert.Equal(t, uint64(0), seq.checkpoint.TxIndex, "TxIndex should be reset after consuming all")
}

func TestSequencer_QueueLimit_Integration(t *testing.T) {
	// Test integration between sequencer and queue limits to demonstrate backpressure
	ctx := context.Background()
	logger := zerolog.Nop()

	gen := genesis.Genesis{
		ChainID:       "test",
		DAStartHeight: 100,
	}

	db := ds.NewMapDatastore()
	defer db.Close()

	dummyDA := newDummyDA(100_000_000)

	seq, err := NewSequencer(
		logger,
		db,
		dummyDA,
		config.DefaultConfig(),
		[]byte("test"),
		2, // Very small limit for testing
		gen,
		&mockExecutor{},
	)
	require.NoError(t, err)

	// Test successful batch submission within limit
	batch1 := createTestBatch(t, 3)
	req1 := coresequencer.SubmitBatchTxsRequest{
		Id:    seq.Id,
		Batch: &batch1,
	}

	resp1, err := seq.SubmitBatchTxs(ctx, req1)
	if err != nil {
		t.Fatalf("unexpected error submitting first batch: %v", err)
	}
	if resp1 == nil {
		t.Fatal("expected non-nil response")
	}

	// Test second successful batch submission at limit
	batch2 := createTestBatch(t, 4)
	req2 := coresequencer.SubmitBatchTxsRequest{
		Id:    seq.Id,
		Batch: &batch2,
	}

	resp2, err := seq.SubmitBatchTxs(ctx, req2)
	if err != nil {
		t.Fatalf("unexpected error submitting second batch: %v", err)
	}
	if resp2 == nil {
		t.Fatal("expected non-nil response")
	}

	// Test third batch submission should fail due to queue being full
	batch3 := createTestBatch(t, 5)
	req3 := coresequencer.SubmitBatchTxsRequest{
		Id:    seq.Id,
		Batch: &batch3,
	}

	resp3, err := seq.SubmitBatchTxs(ctx, req3)
	if err == nil {
		t.Error("expected error when queue is full, but got none")
	}
	if !errors.Is(err, ErrQueueFull) {
		t.Errorf("expected ErrQueueFull, got %v", err)
	}
	if resp3 != nil {
		t.Error("expected nil response when submission fails")
	}

	// Test that getting a batch frees up space
	nextResp, err := seq.GetNextBatch(ctx, coresequencer.GetNextBatchRequest{Id: seq.Id})
	if err != nil {
		t.Fatalf("unexpected error getting next batch: %v", err)
	}
	if nextResp == nil || nextResp.Batch == nil {
		t.Fatal("expected non-nil batch response")
	}

	// Now the third batch should succeed
	resp3_retry, err := seq.SubmitBatchTxs(ctx, req3)
	if err != nil {
		t.Errorf("unexpected error submitting batch after freeing space: %v", err)
	}
	if resp3_retry == nil {
		t.Error("expected non-nil response after retry")
	}

	// Test empty batch handling - should not be affected by limits
	emptyBatch := coresequencer.Batch{Transactions: nil}
	reqEmpty := coresequencer.SubmitBatchTxsRequest{
		Id:    seq.Id,
		Batch: &emptyBatch,
	}

	respEmpty, err := seq.SubmitBatchTxs(ctx, reqEmpty)
	if err != nil {
		t.Errorf("unexpected error submitting empty batch: %v", err)
	}
	if respEmpty == nil {
		t.Error("expected non-nil response for empty batch")
	}
}

// TestSequencer_DAFailureAndQueueThrottling_Integration tests the integration scenario
// where DA layer fails and the batch queue fills up, demonstrating the throttling behavior
// that prevents resource exhaustion.
func TestSequencer_DAFailureAndQueueThrottling_Integration(t *testing.T) {
	// This test simulates the scenario described in the PR:
	// 1. Start sequencer with dummy DA
	// 2. Send transactions (simulate reaper behavior)
	// 3. Make DA layer go down
	// 4. Continue sending transactions
	// 5. Eventually batch queue fills up and returns ErrQueueFull

	db := ds.NewMapDatastore()
	defer db.Close()

	// Create a dummy DA that we can make fail
	dummyDA := newDummyDA(100_000)
	stopTicker := dummyDA.StartHeightTicker(100 * time.Millisecond)
	defer stopTicker()

	// Create sequencer with small queue size to trigger throttling quickly
	queueSize := 3 // Small for testing
	logger := zerolog.Nop()
	seq, err := NewSequencer(
		logger,
		db,
		dummyDA,
		config.DefaultConfig(),
		[]byte("test-chain"),
		queueSize,
		genesis.Genesis{},
		&mockExecutor{},
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Phase 1: Normal operation - send some batches successfully
	t.Log("Phase 1: Normal operation")
	for i := 0; i < queueSize; i++ {
		batch := createTestBatch(t, i+1)
		req := coresequencer.SubmitBatchTxsRequest{
			Id:    []byte("test-chain"),
			Batch: &batch,
		}

		resp, err := seq.SubmitBatchTxs(ctx, req)
		require.NoError(t, err, "Expected successful batch submission during normal operation")
		require.NotNil(t, resp)
	}

	// At this point the queue should be full (queueSize batches)
	t.Log("Phase 2: Queue should now be full")

	// Try to add one more batch - should fail with ErrQueueFull
	overflowBatch := createTestBatch(t, queueSize+1)
	overflowReq := coresequencer.SubmitBatchTxsRequest{
		Id:    []byte("test-chain"),
		Batch: &overflowBatch,
	}

	resp, err := seq.SubmitBatchTxs(ctx, overflowReq)
	require.Error(t, err, "Expected error when queue is full")
	require.True(t, errors.Is(err, ErrQueueFull), "Expected ErrQueueFull, got %v", err)
	require.Nil(t, resp, "Expected nil response when queue is full")

	t.Log("✅ Successfully demonstrated ErrQueueFull when queue reaches limit")

	// Phase 3: Simulate DA layer going down (this would be used in block manager)
	t.Log("Phase 3: Simulating DA layer failure")
	dummyDA.SetSubmitFailure(true)

	// Phase 4: Process one batch to free up space, simulating block manager getting batches
	t.Log("Phase 4: Process one batch to free up space")
	nextResp, err := seq.GetNextBatch(ctx, coresequencer.GetNextBatchRequest{Id: []byte("test-chain")})
	require.NoError(t, err)
	require.NotNil(t, nextResp)
	require.NotNil(t, nextResp.Batch)

	// Now we should be able to add the overflow batch
	resp, err = seq.SubmitBatchTxs(ctx, overflowReq)
	require.NoError(t, err, "Expected successful submission after freeing space")
	require.NotNil(t, resp)

	// Phase 5: Continue adding batches until queue is full again
	t.Log("Phase 5: Fill queue again to demonstrate continued throttling")

	// Add batches until queue is full again
	batchesAdded := 0
	for i := 0; i < 10; i++ { // Try to add many batches
		batch := createTestBatch(t, 100+i)
		req := coresequencer.SubmitBatchTxsRequest{
			Id:    []byte("test-chain"),
			Batch: &batch,
		}

		resp, err := seq.SubmitBatchTxs(ctx, req)
		if err != nil {
			if errors.Is(err, ErrQueueFull) {
				t.Logf("✅ Queue full again after adding %d more batches", batchesAdded)
				break
			} else {
				t.Fatalf("Unexpected error: %v", err)
			}
		}
		require.NotNil(t, resp, "Expected non-nil response for successful submission")
		batchesAdded++
	}

	// The queue is already full from the overflow batch we added, so we expect 0 additional batches
	t.Log("✅ Successfully demonstrated that queue throttling prevents unbounded resource consumption")
	t.Logf("📊 Queue size limit: %d, Additional batches attempted: %d", queueSize, batchesAdded)

	// Final verification: try one more batch to confirm queue is still full
	finalBatch := createTestBatch(t, 999)
	finalReq := coresequencer.SubmitBatchTxsRequest{
		Id:    []byte("test-chain"),
		Batch: &finalBatch,
	}

	resp, err = seq.SubmitBatchTxs(ctx, finalReq)
	require.Error(t, err, "Expected final batch to fail due to full queue")
	require.True(t, errors.Is(err, ErrQueueFull), "Expected final ErrQueueFull")
	require.Nil(t, resp)

	t.Log("✅ Final verification: Queue throttling still active")

	// This test demonstrates the complete integration scenario:
	// 1. ✅ Sequencer accepts batches normally when queue has space
	// 2. ✅ Returns ErrQueueFull when queue reaches its limit
	// 3. ✅ Allows new batches when space is freed (GetNextBatch)
	// 4. ✅ Continues to throttle when queue fills up again
	// 5. ✅ Provides backpressure to prevent resource exhaustion
}

func TestSequencer_CheckpointPersistence_CrashRecovery(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(zerolog.NewConsoleWriter())

	db := ds.NewMapDatastore()
	defer db.Close()

	// Create mock DA client
	mockDA := newMockFullDAClient(t)
	forcedInclusionNS := []byte("forced-inclusion")

	mockDA.MockClient.On("GetHeaderNamespace").Return([]byte("header")).Maybe()
	mockDA.MockClient.On("GetDataNamespace").Return([]byte("data")).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return(forcedInclusionNS).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	// Create forced inclusion txs at DA height 100
	forcedTx1 := make([]byte, 100)
	forcedTx2 := make([]byte, 80)
	forcedTx3 := make([]byte, 90)

	// Will be called twice - once for seq1, once for seq2
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(100), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess},
		Data:       [][]byte{forcedTx1, forcedTx2, forcedTx3},
	}).Twice()

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	// Create first sequencer instance
	seq1, err := NewSequencer(
		logger,
		db,
		mockDA,
		config.DefaultConfig(),
		[]byte("test-chain"),
		100,
		gen,
		&mockExecutor{},
	)
	require.NoError(t, err)

	// First call - get first forced tx
	getReq := coresequencer.GetNextBatchRequest{
		Id:            []byte("test-chain"),
		MaxBytes:      120,
		LastBatchData: nil,
	}

	resp1, err := seq1.GetNextBatch(ctx, getReq)
	require.NoError(t, err)
	require.NotNil(t, resp1.Batch)
	assert.Equal(t, 1, len(resp1.Batch.Transactions), "Should get first forced tx")
	assert.Equal(t, 100, len(resp1.Batch.Transactions[0]))

	// Verify checkpoint is persisted
	assert.Equal(t, uint64(1), seq1.checkpoint.TxIndex, "Checkpoint should show 1 tx consumed")
	assert.Equal(t, uint64(100), seq1.checkpoint.DAHeight, "Checkpoint should be at DA height 100")

	// Second call - get second forced tx
	resp2, err := seq1.GetNextBatch(ctx, getReq)
	require.NoError(t, err)
	require.NotNil(t, resp2.Batch)
	assert.Equal(t, 1, len(resp2.Batch.Transactions), "Should get second forced tx")
	assert.Equal(t, 80, len(resp2.Batch.Transactions[0]))

	// Verify checkpoint updated
	assert.Equal(t, uint64(2), seq1.checkpoint.TxIndex, "Checkpoint should show 2 txs consumed")

	// SIMULATE CRASH: Create new sequencer instance with same DB
	// This simulates a node restart/crash
	seq2, err := NewSequencer(
		logger,
		db,
		mockDA,
		config.DefaultConfig(),
		[]byte("test-chain"),
		100,
		gen,
		&mockExecutor{},
	)
	require.NoError(t, err)

	// Verify checkpoint was loaded from disk
	assert.Equal(t, uint64(2), seq2.checkpoint.TxIndex, "Checkpoint should be loaded from disk")
	assert.Equal(t, uint64(100), seq2.checkpoint.DAHeight, "DA height should be loaded from disk")

	// Third call on new sequencer instance - should get third forced tx (NOT re-execute first two)
	resp3, err := seq2.GetNextBatch(ctx, getReq)
	require.NoError(t, err)
	require.NotNil(t, resp3.Batch)
	assert.Equal(t, 1, len(resp3.Batch.Transactions), "Should get third forced tx (resume from checkpoint)")
	assert.Equal(t, 90, len(resp3.Batch.Transactions[0]), "Should be third tx, not first")

	// Verify checkpoint moved to next DA height after consuming all
	assert.Equal(t, uint64(101), seq2.checkpoint.DAHeight, "Should have moved to next DA height")
	assert.Equal(t, uint64(0), seq2.checkpoint.TxIndex, "TxIndex should be reset")

	t.Log("✅ Checkpoint system successfully prevented re-execution of DA transactions after crash")
}

func TestSequencer_GetNextBatch_EmptyDABatch_IncreasesDAHeight(t *testing.T) {
	db := ds.NewMapDatastore()
	defer db.Close()
	ctx := context.Background()

	// Create mock DA client
	mockDA := newMockFullDAClient(t)
	forcedInclusionNS := []byte("forced-inclusion")

	mockDA.MockClient.On("GetHeaderNamespace").Return([]byte("header")).Maybe()
	mockDA.MockClient.On("GetDataNamespace").Return([]byte("data")).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return(forcedInclusionNS).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	// First DA epoch returns empty transactions
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(100), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess},
		Data:       [][]byte{},
	}).Once()

	// Second DA epoch also returns empty transactions
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(101), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess},
		Data:       [][]byte{},
	}).Once()

	gen := genesis.Genesis{
		ChainID:                "test",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq, err := NewSequencer(
		zerolog.Nop(),
		db,
		mockDA,
		config.DefaultConfig(),
		[]byte("test"),
		1000,
		gen,
		&mockExecutor{},
	)
	require.NoError(t, err)

	req := coresequencer.GetNextBatchRequest{
		Id:            seq.Id,
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	// Initial DA height should be 100
	assert.Equal(t, uint64(100), seq.GetDAHeight())
	assert.Equal(t, uint64(100), seq.checkpoint.DAHeight)

	// First batch - empty DA block at height 100
	resp, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 0, len(resp.Batch.Transactions))

	// DA height should have increased to 101 even though no transactions were processed
	assert.Equal(t, uint64(101), seq.GetDAHeight())
	assert.Equal(t, uint64(101), seq.checkpoint.DAHeight)
	assert.Equal(t, uint64(0), seq.checkpoint.TxIndex)

	// Second batch - empty DA block at height 101
	resp, err = seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 0, len(resp.Batch.Transactions))

	// DA height should have increased to 102
	assert.Equal(t, uint64(102), seq.GetDAHeight())
	assert.Equal(t, uint64(102), seq.checkpoint.DAHeight)
	assert.Equal(t, uint64(0), seq.checkpoint.TxIndex)
}

// mockExecutor is a mock implementation of execution.Executor for testing gas filtering
type mockExecutor struct {
	maxGas          uint64
	getInfoErr      error
	filterFunc      func(ctx context.Context, txs [][]byte, forceIncludedMask []bool) (*execution.FilterTxsResult, error)
	filterCallCount int
}

func (m *mockExecutor) InitChain(ctx context.Context, genesisTime time.Time, initialHeight uint64, chainID string) ([]byte, error) {
	return []byte("state-root"), nil
}

func (m *mockExecutor) GetTxs(ctx context.Context) ([][]byte, error) {
	return nil, nil
}

func (m *mockExecutor) ExecuteTxs(ctx context.Context, txs [][]byte, blockHeight uint64, timestamp time.Time, prevStateRoot []byte) ([]byte, error) {
	return []byte("new-state-root"), nil
}

func (m *mockExecutor) SetFinal(ctx context.Context, blockHeight uint64) error {
	return nil
}

func (m *mockExecutor) GetExecutionInfo(ctx context.Context, height uint64) (execution.ExecutionInfo, error) {
	return execution.ExecutionInfo{MaxGas: m.maxGas}, m.getInfoErr
}

func (m *mockExecutor) FilterTxs(ctx context.Context, txs [][]byte, forceIncludedMask []bool) (*execution.FilterTxsResult, error) {
	m.filterCallCount++
	if m.filterFunc != nil {
		return m.filterFunc(ctx, txs, forceIncludedMask)
	}
	// Default: return all txs as valid, no gas info
	return &execution.FilterTxsResult{
		ValidTxs:          txs,
		ForceIncludedMask: forceIncludedMask,
		GasPerTx:          nil,
	}, nil
}

func TestSequencer_GetNextBatch_WithGasFiltering(t *testing.T) {
	db := ds.NewMapDatastore()
	logger := zerolog.Nop()

	gen := genesis.Genesis{
		ChainID:                "test-gas-filter",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	mockDA := newMockFullDAClient(t)

	// Setup DA to return forced inclusion transactions
	forcedTxs := [][]byte{
		[]byte("tx1-low-gas"),  // Will fit
		[]byte("tx2-low-gas"),  // Will fit
		[]byte("tx3-high-gas"), // Won't fit due to gas
	}

	mockDA.MockClient.On("GetBlobs", mock.Anything, mock.Anything, mock.Anything).
		Return(forcedTxs, nil).Maybe()
	mockDA.MockClient.On("GetBlobsAtHeight", mock.Anything, mock.Anything, mock.Anything).
		Return(forcedTxs, nil).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return([]byte("forced")).Maybe()
	mockDA.MockClient.On("MaxBlobSize", mock.Anything).Return(uint64(1000000), nil).Maybe()

	// Configure the executor mock
	mockExec := &mockExecutor{
		maxGas: 1000000, // 1M gas limit
		filterFunc: func(ctx context.Context, txs [][]byte, forceIncludedMask []bool) (*execution.FilterTxsResult, error) {
			// Return all txs as valid with gas per tx
			// First 2 txs use 250k gas each (total 500k), third uses 600k (would exceed 1M limit)
			gasPerTx := make([]uint64, len(txs))
			for i := range txs {
				if i < 2 {
					gasPerTx[i] = 250000
				} else {
					gasPerTx[i] = 600000 // This one won't fit
				}
			}
			return &execution.FilterTxsResult{
				ValidTxs:          txs,
				ForceIncludedMask: forceIncludedMask,
				GasPerTx:          gasPerTx,
			}, nil
		},
	}

	seq, err := NewSequencer(
		logger,
		db,
		mockDA,
		config.DefaultConfig(),
		[]byte("test-gas-filter"),
		1000,
		gen,
		mockExec,
	)
	require.NoError(t, err)

	// Manually set up cached forced txs to simulate DA fetch
	seq.cachedForcedInclusionTxs = forcedTxs
	seq.checkpoint.DAHeight = 100
	seq.checkpoint.TxIndex = 0
	seq.SetDAHeight(100)

	ctx := context.Background()
	req := coresequencer.GetNextBatchRequest{
		Id:       []byte("test-gas-filter"),
		MaxBytes: 1000000,
	}

	// First call should return filtered txs (only first 2)
	resp, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)

	// Should have 2 forced txs (the ones that fit within gas limit)
	assert.Equal(t, 2, len(resp.Batch.Transactions))
	assert.True(t, resp.Batch.ForceIncludedMask[0])
	assert.True(t, resp.Batch.ForceIncludedMask[1])

	// The remaining tx should be cached for next block
	assert.Equal(t, 1, len(seq.cachedForcedInclusionTxs))
	assert.Equal(t, uint64(0), seq.checkpoint.TxIndex) // Reset because we replaced the cache

	// Filter should have been called
	assert.Equal(t, 1, mockExec.filterCallCount)

	// Second call should return the remaining tx
	mockExec.filterFunc = func(ctx context.Context, txs [][]byte, forceIncludedMask []bool) (*execution.FilterTxsResult, error) {
		// Now all remaining txs fit (return with gas that fits)
		gasPerTx := make([]uint64, len(txs))
		for i := range txs {
			gasPerTx[i] = 100000 // Low gas, will fit
		}
		return &execution.FilterTxsResult{
			ValidTxs:          txs,
			ForceIncludedMask: forceIncludedMask,
			GasPerTx:          gasPerTx,
		}, nil
	}

	resp, err = seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)

	// Should have the remaining forced tx
	assert.Equal(t, 1, len(resp.Batch.Transactions))
	assert.True(t, resp.Batch.ForceIncludedMask[0])

	// Cache should now be empty, DA height should advance
	assert.Nil(t, seq.cachedForcedInclusionTxs)
	assert.Equal(t, uint64(101), seq.checkpoint.DAHeight)
}

func TestSequencer_GetNextBatch_GasFilterError(t *testing.T) {
	db := ds.NewMapDatastore()
	logger := zerolog.Nop()

	gen := genesis.Genesis{
		ChainID:                "test-gas-error",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	mockDA := newMockFullDAClient(t)
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return([]byte("forced")).Maybe()
	mockDA.MockClient.On("MaxBlobSize", mock.Anything).Return(uint64(1000000), nil).Maybe()

	// Configure executor that returns filter error
	mockExec := &mockExecutor{
		maxGas: 1000000,
		filterFunc: func(ctx context.Context, txs [][]byte, forceIncludedMask []bool) (*execution.FilterTxsResult, error) {
			return nil, errors.New("filter error")
		},
	}

	seq, err := NewSequencer(
		logger,
		db,
		mockDA,
		config.DefaultConfig(),
		[]byte("test-gas-error"),
		1000,
		gen,
		mockExec,
	)
	require.NoError(t, err)

	// Set up cached txs
	forcedTxs := [][]byte{[]byte("tx1"), []byte("tx2")}
	seq.cachedForcedInclusionTxs = forcedTxs
	seq.checkpoint.DAHeight = 100
	seq.checkpoint.TxIndex = 0
	seq.SetDAHeight(100)

	ctx := context.Background()
	req := coresequencer.GetNextBatchRequest{
		Id:       []byte("test-gas-error"),
		MaxBytes: 1000000,
	}

	// Should succeed but fall back to unfiltered txs on error
	resp, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Should have all txs (filter error means no filtering)
	assert.Equal(t, 2, len(resp.Batch.Transactions))
}
