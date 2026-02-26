package single

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block"
	"github.com/evstack/ev-node/core/execution"
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

// createDefaultMockExecutor creates a MockExecutor with default passthrough behavior for FilterTxs and GetExecutionInfo
// It implements proper size filtering based on maxBytes parameter.
func createDefaultMockExecutor(t *testing.T) *mocks.MockExecutor {
	mockExec := mocks.NewMockExecutor(t)
	mockExec.On("GetExecutionInfo", mock.Anything).Return(execution.ExecutionInfo{MaxGas: 1000000}, nil).Maybe()
	mockExec.On("FilterTxs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, txs [][]byte, maxBytes, maxGas uint64, hasForceIncludedTransaction bool) []execution.FilterStatus {
			result := make([]execution.FilterStatus, len(txs))
			var cumulativeBytes uint64

			for i, tx := range txs {
				txBytes := uint64(len(tx))

				// Check if tx would exceed size limit
				if maxBytes > 0 && cumulativeBytes+txBytes > maxBytes {
					result[i] = execution.FilterPostpone
					continue
				}

				cumulativeBytes += txBytes
				result[i] = execution.FilterOK
			}
			return result
		},
		nil,
	).Maybe()
	return mockExec
}

// newTestSequencer creates a sequencer for tests that don't need full initialization
func newTestSequencer(t *testing.T, db ds.Batching, daClient block.FullDAClient) *Sequencer {
	logger := zerolog.Nop()

	gen := genesis.Genesis{
		ChainID:       "test",
		DAStartHeight: 100,
	}

	mockExec := createDefaultMockExecutor(t)

	seq, err := NewSequencer(
		logger,
		db,
		daClient,
		config.DefaultConfig(),
		[]byte("test"),
		0, // unlimited queue
		gen,
		mockExec,
	)
	require.NoError(t, err)
	return seq
}

func TestSequencer_SubmitBatchTxs(t *testing.T) {
	dummyDA := newDummyDA(100_000_000)
	db := ds.NewMapDatastore()
	Id := []byte("test1")
	logger := zerolog.Nop()
	mockExec := createDefaultMockExecutor(t)
	seq, err := NewSequencer(logger, db, dummyDA, config.DefaultConfig(), Id, 1000, genesis.Genesis{}, mockExec)
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
	mockExec := createDefaultMockExecutor(t)
	seq, err := NewSequencer(logger, db, dummyDA, config.DefaultConfig(), Id, 1000, genesis.Genesis{}, mockExec)
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
		createDefaultMockExecutor(t),
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
	mockExec := createDefaultMockExecutor(t)
	seq, err := NewSequencer(logger, db, dummyDA, config.DefaultConfig(), Id, 1000, genesis.Genesis{}, mockExec)
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
	logger := zerolog.New(zerolog.NewTestWriter(t))

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

	// DA head is at 100 — same as sequencer start, no catch-up needed
	mockDA.MockClient.On("GetLatestDAHeight", mock.Anything).Return(uint64(100), nil).Maybe()

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
		createDefaultMockExecutor(t),
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
	logger := zerolog.New(zerolog.NewTestWriter(t))

	db := ds.NewMapDatastore()
	defer db.Close()

	// Create mock DA client
	mockDA := newMockFullDAClient(t)
	forcedInclusionNS := []byte("forced-inclusion")

	mockDA.MockClient.On("GetHeaderNamespace").Return([]byte("header")).Maybe()
	mockDA.MockClient.On("GetDataNamespace").Return([]byte("data")).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return(forcedInclusionNS).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	// DA head is at 100 — same as sequencer start, no catch-up needed
	mockDA.MockClient.On("GetLatestDAHeight", mock.Anything).Return(uint64(100), nil).Maybe()

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
		createDefaultMockExecutor(t),
	)
	require.NoError(t, err)

	// Request batch with maxBytes = 120
	getReq := coresequencer.GetNextBatchRequest{
		Id:            []byte("test-chain"),
		MaxBytes:      120,
		LastBatchData: nil,
	}

	// First call - should get only first forced tx (100 bytes) as second (80 bytes) would exceed maxBytes=120
	resp, err := seq.GetNextBatch(ctx, getReq)
	require.NoError(t, err)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 1, len(resp.Batch.Transactions), "Should only include first forced tx")
	assert.Equal(t, 100, len(resp.Batch.Transactions[0]))

	// TxIndex tracks consumed txs from the original epoch for crash recovery
	assert.Equal(t, uint64(1), seq.checkpoint.TxIndex, "TxIndex should be 1 (consumed first forced tx)")
	assert.Equal(t, 2, len(seq.cachedForcedInclusionTxs), "Cache should still contain all original txs")

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
	logger := zerolog.New(zerolog.NewTestWriter(t))

	db := ds.NewMapDatastore()
	defer db.Close()

	// Create mock DA client
	mockDA := newMockFullDAClient(t)
	forcedInclusionNS := []byte("forced-inclusion")

	mockDA.MockClient.On("GetHeaderNamespace").Return([]byte("header")).Maybe()
	mockDA.MockClient.On("GetDataNamespace").Return([]byte("data")).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return(forcedInclusionNS).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	// DA head is at 100 — same as sequencer start, no catch-up needed
	mockDA.MockClient.On("GetLatestDAHeight", mock.Anything).Return(uint64(100), nil).Maybe()

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
		createDefaultMockExecutor(t),
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

	// TxIndex tracks consumed txs from the original epoch for crash recovery
	assert.Equal(t, uint64(1), seq.checkpoint.TxIndex, "TxIndex should be 1 (consumed first forced tx)")
	assert.Equal(t, 2, len(seq.cachedForcedInclusionTxs), "Cache should still contain all original txs")

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
		createDefaultMockExecutor(t),
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
		createDefaultMockExecutor(t),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Phase 1: Normal operation - send some batches successfully
	t.Log("Phase 1: Normal operation")
	for i := range queueSize {
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
	for i := range 10 { // Try to add many batches
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
	logger := zerolog.New(zerolog.NewTestWriter(t))

	db := ds.NewMapDatastore()
	defer db.Close()

	// Create mock DA client
	mockDA := newMockFullDAClient(t)
	forcedInclusionNS := []byte("forced-inclusion")

	mockDA.MockClient.On("GetHeaderNamespace").Return([]byte("header")).Maybe()
	mockDA.MockClient.On("GetDataNamespace").Return([]byte("data")).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return(forcedInclusionNS).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	// DA head is at 101 — close to sequencer start (100), no catch-up needed.
	// Use Maybe() since two sequencer instances share this mock.
	mockDA.MockClient.On("GetLatestDAHeight", mock.Anything).Return(uint64(101), nil).Maybe()

	// Create forced inclusion txs at DA height 100
	// Use sizes that all fit in one batch to test checkpoint advancing
	forcedTx1 := make([]byte, 50)
	forcedTx2 := make([]byte, 40)
	forcedTx3 := make([]byte, 45)

	// Will be called twice - once for seq1, once for seq2
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(100), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess},
		Data:       [][]byte{forcedTx1, forcedTx2, forcedTx3},
	}).Once()

	// Empty DA at height 101
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(101), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess},
		Data:       [][]byte{},
	}).Maybe()

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
		createDefaultMockExecutor(t),
	)
	require.NoError(t, err)

	// First call - get all forced txs (they fit in maxBytes=200)
	getReq := coresequencer.GetNextBatchRequest{
		Id:            []byte("test-chain"),
		MaxBytes:      200,
		LastBatchData: nil,
	}

	resp1, err := seq1.GetNextBatch(ctx, getReq)
	require.NoError(t, err)
	require.NotNil(t, resp1.Batch)
	assert.Equal(t, 3, len(resp1.Batch.Transactions), "Should get all 3 forced txs")

	// Verify checkpoint advanced to next DA height (all txs consumed)
	assert.Equal(t, uint64(0), seq1.checkpoint.TxIndex, "TxIndex should be reset after consuming all")
	assert.Equal(t, uint64(101), seq1.checkpoint.DAHeight, "Checkpoint should advance to next DA height")

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
		createDefaultMockExecutor(t),
	)
	require.NoError(t, err)

	// Verify checkpoint was loaded from disk
	assert.Equal(t, uint64(0), seq2.checkpoint.TxIndex, "TxIndex should be loaded from disk")
	assert.Equal(t, uint64(101), seq2.checkpoint.DAHeight, "DA height should be loaded from disk")

	// Next call on new sequencer - should NOT re-execute DA height 100 txs
	resp2, err := seq2.GetNextBatch(ctx, getReq)
	require.NoError(t, err)
	require.NotNil(t, resp2.Batch)
	// Should be empty since DA height 101 has no txs
	assert.Equal(t, 0, len(resp2.Batch.Transactions), "Should have no txs from DA height 101")

	// Verify checkpoint moved to next DA height
	assert.Equal(t, uint64(102), seq2.checkpoint.DAHeight, "Should have moved to next DA height")
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

	// DA head is at 100 — same as sequencer start, no catch-up needed
	mockDA.MockClient.On("GetLatestDAHeight", mock.Anything).Return(uint64(100), nil).Maybe()

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
		createDefaultMockExecutor(t),
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
	filterCallCount := 0
	mockExec := mocks.NewMockExecutor(t)
	mockExec.On("GetExecutionInfo", mock.Anything).Return(execution.ExecutionInfo{MaxGas: 1000000}, nil).Maybe()
	mockExec.On("FilterTxs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, txs [][]byte, maxBytes, maxGas uint64, hasForceIncludedTransaction bool) []execution.FilterStatus {
			filterCallCount++
			// Simulate gas filtering: first 2 forced txs fit, third one doesn't
			result := make([]execution.FilterStatus, len(txs))
			forcedCount := 0
			for i := range txs {
				// Count forced txs (they come first in based sequencer)
				if hasForceIncludedTransaction && i < 3 { // assuming 3 forced txs
					if forcedCount < 2 {
						result[i] = execution.FilterOK
					} else {
						result[i] = execution.FilterPostpone
					}
					forcedCount++
				} else {
					result[i] = execution.FilterOK
				}
			}
			return result
		},
		nil,
	).Maybe()

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

	// TxIndex tracks consumed txs, cache still contains all original txs
	assert.Equal(t, 3, len(seq.cachedForcedInclusionTxs), "Cache should still contain all original txs")
	assert.Equal(t, uint64(2), seq.checkpoint.TxIndex, "TxIndex should be 2 (consumed 2 txs)")

	// Filter should have been called
	assert.Equal(t, 1, filterCallCount)

	// Second call will use the same mock which now returns all txs as valid

	resp, err = seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)

	// Should have the remaining forced tx
	assert.Equal(t, 1, len(resp.Batch.Transactions))

	// All txs consumed, DA height should advance and cache cleared
	assert.Nil(t, seq.cachedForcedInclusionTxs)
	assert.Equal(t, uint64(0), seq.checkpoint.TxIndex, "TxIndex should be reset after advancing DA height")
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
	mockExec := mocks.NewMockExecutor(t)
	mockExec.On("GetExecutionInfo", mock.Anything).Return(execution.ExecutionInfo{MaxGas: 1000000}, nil).Maybe()
	mockExec.On("FilterTxs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		([]execution.FilterStatus)(nil),
		errors.New("filter error"),
	).Maybe()

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

// TestSequencer_GetNextBatch_GasFilteringPreservesUnprocessedTxs tests that when FilterTxs
// returns RemainingTxs (valid DA txs that didn't fit due to gas limits), the sequencer correctly
// preserves any transactions that weren't even processed yet due to maxBytes limits.
//
// This test uses maxBytes to limit how many txs are fetched, triggering the unprocessed txs scenario.
func TestSequencer_CatchUp_DetectsOldEpoch(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(zerolog.NewTestWriter(t))

	db := ds.NewMapDatastore()
	defer db.Close()

	mockDA := newMockFullDAClient(t)
	forcedInclusionNS := []byte("forced-inclusion")

	mockDA.MockClient.On("GetHeaderNamespace").Return([]byte("header")).Maybe()
	mockDA.MockClient.On("GetDataNamespace").Return([]byte("data")).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return(forcedInclusionNS).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	// DA head is at height 105 — sequencer starts at 100 with epoch size 1,
	// so it has missed 5 epochs (>1), triggering catch-up.
	mockDA.MockClient.On("GetLatestDAHeight", mock.Anything).Return(uint64(105), nil).Once()

	// DA epoch at height 100
	oldTimestamp := time.Now().Add(-10 * time.Minute)
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(100), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: oldTimestamp},
		Data:       [][]byte{[]byte("forced-tx-1")},
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
		createDefaultMockExecutor(t),
	)
	require.NoError(t, err)

	// Submit a mempool transaction
	_, err = seq.SubmitBatchTxs(ctx, coresequencer.SubmitBatchTxsRequest{
		Id:    []byte("test-chain"),
		Batch: &coresequencer.Batch{Transactions: [][]byte{[]byte("mempool-tx-1")}},
	})
	require.NoError(t, err)

	assert.False(t, seq.isCatchingUp(), "should not be catching up initially")

	// First GetNextBatch — DA head is far ahead, should enter catch-up
	req := coresequencer.GetNextBatchRequest{
		Id:            []byte("test-chain"),
		MaxBytes:      1000000,
		LastBatchData: nil,
	}
	resp, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp.Batch)

	assert.True(t, seq.isCatchingUp(), "should be catching up after detecting epoch gap")

	// During catch-up, batch should contain only forced inclusion tx, no mempool tx
	assert.Equal(t, 1, len(resp.Batch.Transactions), "should have only forced inclusion tx during catch-up")
	assert.Equal(t, []byte("forced-tx-1"), resp.Batch.Transactions[0])
}

func TestSequencer_CatchUp_SkipsMempoolDuringCatchUp(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(zerolog.NewTestWriter(t))

	db := ds.NewMapDatastore()
	defer db.Close()

	mockDA := newMockFullDAClient(t)
	forcedInclusionNS := []byte("forced-inclusion")

	mockDA.MockClient.On("GetHeaderNamespace").Return([]byte("header")).Maybe()
	mockDA.MockClient.On("GetDataNamespace").Return([]byte("data")).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return(forcedInclusionNS).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	// DA head is at 105 — sequencer starts at 100 with epoch size 1,
	// so it has missed multiple epochs, triggering catch-up.
	// Called once on first fetchNextDAEpoch; subsequent fetches skip the check.
	mockDA.MockClient.On("GetLatestDAHeight", mock.Anything).Return(uint64(105), nil).Once()

	// Epoch at height 100: two forced txs
	oldTimestamp := time.Now().Add(-5 * time.Minute)
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(100), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: oldTimestamp},
		Data:       [][]byte{[]byte("forced-1"), []byte("forced-2")},
	}).Once()

	// Epoch at height 101: one forced tx
	oldTimestamp2 := time.Now().Add(-4 * time.Minute)
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(101), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: oldTimestamp2},
		Data:       [][]byte{[]byte("forced-3")},
	}).Once()

	// Epoch at height 102: from the future (head reached during replay)
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(102), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusHeightFromFuture},
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
		createDefaultMockExecutor(t),
	)
	require.NoError(t, err)

	// Submit several mempool transactions
	for i := 0; i < 5; i++ {
		_, err = seq.SubmitBatchTxs(ctx, coresequencer.SubmitBatchTxsRequest{
			Id:    []byte("test-chain"),
			Batch: &coresequencer.Batch{Transactions: [][]byte{[]byte("mempool-tx")}},
		})
		require.NoError(t, err)
	}

	req := coresequencer.GetNextBatchRequest{
		Id:            []byte("test-chain"),
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	// First batch (epoch 100): only forced txs
	resp1, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	assert.True(t, seq.isCatchingUp())

	for _, tx := range resp1.Batch.Transactions {
		assert.NotEqual(t, []byte("mempool-tx"), tx, "mempool tx should not appear during catch-up")
	}
	assert.Equal(t, 2, len(resp1.Batch.Transactions), "should have 2 forced txs from epoch 100")

	// Second batch (epoch 101): only forced txs
	resp2, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	assert.True(t, seq.isCatchingUp())

	for _, tx := range resp2.Batch.Transactions {
		assert.NotEqual(t, []byte("mempool-tx"), tx, "mempool tx should not appear during catch-up")
	}
	assert.Equal(t, 1, len(resp2.Batch.Transactions), "should have 1 forced tx from epoch 101")
}

func TestSequencer_CatchUp_UsesDATimestamp(t *testing.T) {
	ctx := context.Background()

	db := ds.NewMapDatastore()
	defer db.Close()

	mockDA := newMockFullDAClient(t)
	forcedInclusionNS := []byte("forced-inclusion")

	mockDA.MockClient.On("GetHeaderNamespace").Return([]byte("header")).Maybe()
	mockDA.MockClient.On("GetDataNamespace").Return([]byte("data")).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return(forcedInclusionNS).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	// DA head is at 105 — multiple epochs ahead, triggers catch-up
	mockDA.MockClient.On("GetLatestDAHeight", mock.Anything).Return(uint64(105), nil).Once()

	// Epoch at height 100: timestamp 5 minutes ago
	epochTimestamp := time.Now().Add(-5 * time.Minute).UTC()
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(100), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: epochTimestamp},
		Data:       [][]byte{[]byte("forced-tx")},
	}).Once()

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq, err := NewSequencer(
		zerolog.Nop(),
		db,
		mockDA,
		config.DefaultConfig(),
		[]byte("test-chain"),
		1000,
		gen,
		createDefaultMockExecutor(t),
	)
	require.NoError(t, err)

	req := coresequencer.GetNextBatchRequest{
		Id:            []byte("test-chain"),
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	resp, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, seq.isCatchingUp(), "should be in catch-up mode")

	// During catch-up, the timestamp should be the DA epoch end time, not time.Now()
	assert.Equal(t, epochTimestamp, resp.Timestamp,
		"catch-up batch timestamp should match DA epoch timestamp")
}

func TestSequencer_CatchUp_ExitsCatchUpAtDAHead(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(zerolog.NewTestWriter(t))

	db := ds.NewMapDatastore()
	defer db.Close()

	mockDA := newMockFullDAClient(t)
	forcedInclusionNS := []byte("forced-inclusion")

	mockDA.MockClient.On("GetHeaderNamespace").Return([]byte("header")).Maybe()
	mockDA.MockClient.On("GetDataNamespace").Return([]byte("data")).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return(forcedInclusionNS).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	// DA head is at 105 — multiple epochs ahead, triggers catch-up
	mockDA.MockClient.On("GetLatestDAHeight", mock.Anything).Return(uint64(105), nil).Once()

	// Epoch 100: old (catch-up)
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(100), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: time.Now().Add(-5 * time.Minute)},
		Data:       [][]byte{[]byte("forced-old")},
	}).Once()

	// Epoch 101: fetched during catch-up, but returns HeightFromFuture to exit catch-up
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(101), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusHeightFromFuture},
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
		createDefaultMockExecutor(t),
	)
	require.NoError(t, err)

	// Submit mempool tx
	_, err = seq.SubmitBatchTxs(ctx, coresequencer.SubmitBatchTxsRequest{
		Id:    []byte("test-chain"),
		Batch: &coresequencer.Batch{Transactions: [][]byte{[]byte("mempool-tx")}},
	})
	require.NoError(t, err)

	req := coresequencer.GetNextBatchRequest{
		Id:            []byte("test-chain"),
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	// First batch: catch-up (old epoch 100)
	resp1, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	assert.True(t, seq.isCatchingUp(), "should be catching up during old epoch")
	assert.Equal(t, 1, len(resp1.Batch.Transactions), "catch-up: only forced tx")
	assert.Equal(t, []byte("forced-old"), resp1.Batch.Transactions[0])

	// Second batch: epoch 101 returns HeightFromFuture — should exit catch-up
	resp2, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	assert.False(t, seq.isCatchingUp(), "should have exited catch-up after reaching DA head")

	// Should include mempool tx now (no forced txs available)
	hasMempoolTx := false
	for _, tx := range resp2.Batch.Transactions {
		if bytes.Equal(tx, []byte("mempool-tx")) {
			hasMempoolTx = true
		}
	}
	assert.True(t, hasMempoolTx, "should contain mempool tx after exiting catch-up")
}

func TestSequencer_CatchUp_HeightFromFutureExitsCatchUp(t *testing.T) {
	ctx := context.Background()

	db := ds.NewMapDatastore()
	defer db.Close()

	mockDA := newMockFullDAClient(t)
	forcedInclusionNS := []byte("forced-inclusion")

	mockDA.MockClient.On("GetHeaderNamespace").Return([]byte("header")).Maybe()
	mockDA.MockClient.On("GetDataNamespace").Return([]byte("data")).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return(forcedInclusionNS).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	// DA head is at 105 — multiple epochs ahead, triggers catch-up
	mockDA.MockClient.On("GetLatestDAHeight", mock.Anything).Return(uint64(105), nil).Once()

	// Epoch 100: success, fetched during catch-up
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(100), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: time.Now().Add(-5 * time.Minute)},
		Data:       [][]byte{[]byte("forced-tx")},
	}).Once()

	// Epoch 101: from the future — DA head reached, exits catch-up
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(101), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusHeightFromFuture},
	}).Once()

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq, err := NewSequencer(
		zerolog.Nop(),
		db,
		mockDA,
		config.DefaultConfig(),
		[]byte("test-chain"),
		1000,
		gen,
		createDefaultMockExecutor(t),
	)
	require.NoError(t, err)

	req := coresequencer.GetNextBatchRequest{
		Id:            []byte("test-chain"),
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	// First call: fetches epoch 100, enters catch-up via epoch gap detection
	resp1, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	assert.True(t, seq.isCatchingUp())
	assert.Equal(t, 1, len(resp1.Batch.Transactions))

	// Second call: epoch 101 is from the future, should exit catch-up
	resp2, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	assert.False(t, seq.isCatchingUp(), "should exit catch-up when DA returns HeightFromFuture")
	// No forced txs available, batch is empty
	assert.Equal(t, 0, len(resp2.Batch.Transactions))
}

func TestSequencer_CatchUp_NoCatchUpWhenRecentEpoch(t *testing.T) {
	ctx := context.Background()

	db := ds.NewMapDatastore()
	defer db.Close()

	mockDA := newMockFullDAClient(t)
	forcedInclusionNS := []byte("forced-inclusion")

	mockDA.MockClient.On("GetHeaderNamespace").Return([]byte("header")).Maybe()
	mockDA.MockClient.On("GetDataNamespace").Return([]byte("data")).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return(forcedInclusionNS).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	// DA head is at 100 — sequencer starts at 100 with epoch size 1,
	// so it is within the same epoch (0 missed). No catch-up.
	mockDA.MockClient.On("GetLatestDAHeight", mock.Anything).Return(uint64(100), nil).Once()

	// Epoch at height 100: current epoch
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(100), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: time.Now()},
		Data:       [][]byte{[]byte("forced-tx")},
	}).Once()

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq, err := NewSequencer(
		zerolog.Nop(),
		db,
		mockDA,
		config.DefaultConfig(),
		[]byte("test-chain"),
		1000,
		gen,
		createDefaultMockExecutor(t),
	)
	require.NoError(t, err)

	// Submit a mempool tx
	_, err = seq.SubmitBatchTxs(ctx, coresequencer.SubmitBatchTxsRequest{
		Id:    []byte("test-chain"),
		Batch: &coresequencer.Batch{Transactions: [][]byte{[]byte("mempool-tx")}},
	})
	require.NoError(t, err)

	req := coresequencer.GetNextBatchRequest{
		Id:            []byte("test-chain"),
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	resp, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	assert.False(t, seq.isCatchingUp(), "should NOT be catching up when within one epoch of DA head")

	// Should have both forced and mempool txs (normal operation)
	assert.Equal(t, 2, len(resp.Batch.Transactions), "should have forced + mempool tx in normal mode")
}

func TestSequencer_CatchUp_MultiEpochReplay(t *testing.T) {
	// Simulates a sequencer that missed 3 DA epochs and must replay them all
	// before resuming normal operation.
	ctx := context.Background()
	logger := zerolog.New(zerolog.NewTestWriter(t))

	db := ds.NewMapDatastore()
	defer db.Close()

	mockDA := newMockFullDAClient(t)
	forcedInclusionNS := []byte("forced-inclusion")

	mockDA.MockClient.On("GetHeaderNamespace").Return([]byte("header")).Maybe()
	mockDA.MockClient.On("GetDataNamespace").Return([]byte("data")).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return(forcedInclusionNS).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	// DA head is at 106 — sequencer starts at 100 with epoch size 1,
	// so it has missed 6 epochs (>1), triggering catch-up.
	// Called once on first fetchNextDAEpoch.
	mockDA.MockClient.On("GetLatestDAHeight", mock.Anything).Return(uint64(106), nil).Once()

	// 3 old epochs (100, 101, 102) — all with timestamps far in the past
	for h := uint64(100); h <= 102; h++ {
		ts := time.Now().Add(-time.Duration(103-h) * time.Minute) // older epochs further in the past
		txData := []byte("forced-from-epoch-" + string(rune('0'+h-100)))
		mockDA.MockClient.On("Retrieve", mock.Anything, h, forcedInclusionNS).Return(datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: ts},
			Data:       [][]byte{txData},
		}).Once()
	}

	// Epoch 103: returns HeightFromFuture — DA head reached, exits catch-up
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(103), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusHeightFromFuture},
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
		createDefaultMockExecutor(t),
	)
	require.NoError(t, err)

	// Submit mempool txs
	_, err = seq.SubmitBatchTxs(ctx, coresequencer.SubmitBatchTxsRequest{
		Id:    []byte("test-chain"),
		Batch: &coresequencer.Batch{Transactions: [][]byte{[]byte("mempool-1"), []byte("mempool-2")}},
	})
	require.NoError(t, err)

	req := coresequencer.GetNextBatchRequest{
		Id:            []byte("test-chain"),
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	// Process the 3 old epochs — all should be catch-up (no mempool)
	for i := 0; i < 3; i++ {
		resp, err := seq.GetNextBatch(ctx, req)
		require.NoError(t, err)
		assert.True(t, seq.isCatchingUp(), "should be catching up during epoch %d", 100+i)
		assert.Equal(t, 1, len(resp.Batch.Transactions),
			"epoch %d: should have exactly 1 forced tx", 100+i)

		for _, tx := range resp.Batch.Transactions {
			assert.NotEqual(t, []byte("mempool-1"), tx, "no mempool during catch-up epoch %d", 100+i)
			assert.NotEqual(t, []byte("mempool-2"), tx, "no mempool during catch-up epoch %d", 100+i)
		}
	}

	// DA height should have advanced through the 3 old epochs
	assert.Equal(t, uint64(103), seq.GetDAHeight(), "DA height should be at 103 after replaying 3 epochs")

	// Next batch: epoch 103 returns HeightFromFuture — should exit catch-up and include mempool
	resp4, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	assert.False(t, seq.isCatchingUp(), "should have exited catch-up at DA head")

	hasMempoolTx := false
	for _, tx := range resp4.Batch.Transactions {
		if bytes.Equal(tx, []byte("mempool-1")) || bytes.Equal(tx, []byte("mempool-2")) {
			hasMempoolTx = true
		}
	}
	assert.True(t, hasMempoolTx, "should include mempool txs after exiting catch-up")
}

func TestSequencer_CatchUp_NoForcedInclusionConfigured(t *testing.T) {
	// When forced inclusion is not configured, catch-up should never activate.
	// GetLatestDAHeight should NOT be called because DAEpochForcedInclusion == 0
	// causes updateCatchUpState to bail out early.
	ctx := context.Background()

	db := ds.NewMapDatastore()
	defer db.Close()

	mockDA := newMockFullDAClient(t)
	// No forced inclusion namespace configured
	mockDA.MockClient.On("GetHeaderNamespace").Return([]byte("header")).Maybe()
	mockDA.MockClient.On("GetDataNamespace").Return([]byte("data")).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(false).Maybe()

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 0, // no epoch-based forced inclusion
	}

	seq, err := NewSequencer(
		zerolog.Nop(),
		db,
		mockDA,
		config.DefaultConfig(),
		[]byte("test-chain"),
		1000,
		gen,
		createDefaultMockExecutor(t),
	)
	require.NoError(t, err)

	// Submit mempool tx
	_, err = seq.SubmitBatchTxs(ctx, coresequencer.SubmitBatchTxsRequest{
		Id:    []byte("test-chain"),
		Batch: &coresequencer.Batch{Transactions: [][]byte{[]byte("mempool-tx")}},
	})
	require.NoError(t, err)

	req := coresequencer.GetNextBatchRequest{
		Id:            []byte("test-chain"),
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	resp, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	assert.False(t, seq.isCatchingUp(), "should never catch up when forced inclusion not configured")
	assert.Equal(t, 1, len(resp.Batch.Transactions))
	assert.Equal(t, []byte("mempool-tx"), resp.Batch.Transactions[0])
}

func TestSequencer_CatchUp_CheckpointAdvancesDuringCatchUp(t *testing.T) {
	// Verify that the checkpoint (DA epoch tracking) advances correctly during catch-up.
	ctx := context.Background()

	db := ds.NewMapDatastore()
	defer db.Close()

	mockDA := newMockFullDAClient(t)
	forcedInclusionNS := []byte("forced-inclusion")

	mockDA.MockClient.On("GetHeaderNamespace").Return([]byte("header")).Maybe()
	mockDA.MockClient.On("GetDataNamespace").Return([]byte("data")).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return(forcedInclusionNS).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	// DA head is at 105 — multiple epochs ahead, triggers catch-up
	mockDA.MockClient.On("GetLatestDAHeight", mock.Anything).Return(uint64(105), nil).Once()

	// Epoch 100: old
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(100), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: time.Now().Add(-5 * time.Minute)},
		Data:       [][]byte{[]byte("tx-a"), []byte("tx-b")},
	}).Once()

	// Epoch 101: old
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(101), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: time.Now().Add(-4 * time.Minute)},
		Data:       [][]byte{[]byte("tx-c")},
	}).Once()

	// Epoch 102: from the future
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(102), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusHeightFromFuture},
	}).Maybe()

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq, err := NewSequencer(
		zerolog.Nop(),
		db,
		mockDA,
		config.DefaultConfig(),
		[]byte("test-chain"),
		1000,
		gen,
		createDefaultMockExecutor(t),
	)
	require.NoError(t, err)

	// Initial checkpoint
	assert.Equal(t, uint64(100), seq.checkpoint.DAHeight)
	assert.Equal(t, uint64(0), seq.checkpoint.TxIndex)

	req := coresequencer.GetNextBatchRequest{
		Id:            []byte("test-chain"),
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	// Process epoch 100
	resp1, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 2, len(resp1.Batch.Transactions))

	// Checkpoint should advance to epoch 101
	assert.Equal(t, uint64(101), seq.checkpoint.DAHeight)
	assert.Equal(t, uint64(0), seq.checkpoint.TxIndex)
	assert.Equal(t, uint64(101), seq.GetDAHeight())

	// Process epoch 101
	resp2, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 1, len(resp2.Batch.Transactions))

	// Checkpoint should advance to epoch 102
	assert.Equal(t, uint64(102), seq.checkpoint.DAHeight)
	assert.Equal(t, uint64(0), seq.checkpoint.TxIndex)
	assert.Equal(t, uint64(102), seq.GetDAHeight())
}

func TestSequencer_CatchUp_MonotonicTimestamps(t *testing.T) {
	// When a single DA epoch has more forced txs than fit in one block,
	// catch-up must produce strictly monotonic timestamps across the
	// resulting blocks.  The jitter scheme is:
	//   epochStart     = daEndTime - totalEpochTxs * 1ms
	//   blockTimestamp = epochStart + txIndexForTimestamp * 1ms
	// where txIndexForTimestamp is the cumulative consumed-tx count
	// captured *before* the checkpoint resets at an epoch boundary.
	// The final block of an epoch therefore lands exactly on daEndTime.
	ctx := context.Background()
	logger := zerolog.New(zerolog.NewTestWriter(t))

	db := ds.NewMapDatastore()
	defer db.Close()

	mockDA := newMockFullDAClient(t)
	forcedInclusionNS := []byte("forced-inclusion")

	mockDA.MockClient.On("GetHeaderNamespace").Return([]byte("header")).Maybe()
	mockDA.MockClient.On("GetDataNamespace").Return([]byte("data")).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return(forcedInclusionNS).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	// DA head is far ahead — triggers catch-up
	mockDA.MockClient.On("GetLatestDAHeight", mock.Anything).Return(uint64(110), nil).Once()

	// Epoch at height 100: 3 forced txs, each 100 bytes
	epochTimestamp := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	tx1 := make([]byte, 100)
	tx2 := make([]byte, 100)
	tx3 := make([]byte, 100)
	copy(tx1, "forced-tx-1")
	copy(tx2, "forced-tx-2")
	copy(tx3, "forced-tx-3")
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(100), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: epochTimestamp},
		Data:       [][]byte{tx1, tx2, tx3},
	}).Once()

	// Epoch at height 101: single tx (to verify cross-epoch monotonicity)
	epoch2Timestamp := time.Date(2025, 1, 1, 12, 0, 10, 0, time.UTC) // 10 seconds later
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(101), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: epoch2Timestamp},
		Data:       [][]byte{[]byte("forced-tx-4")},
	}).Once()

	// Epoch 102: future — exits catch-up
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(102), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusHeightFromFuture},
	}).Maybe()

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	// Custom executor: only 1 tx fits per block (gas-limited)
	mockExec := mocks.NewMockExecutor(t)
	mockExec.On("GetExecutionInfo", mock.Anything).Return(execution.ExecutionInfo{MaxGas: 1000000}, nil).Maybe()
	mockExec.On("FilterTxs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, txs [][]byte, maxBytes, maxGas uint64, hasForceIncludedTransaction bool) []execution.FilterStatus {
			result := make([]execution.FilterStatus, len(txs))
			// Only first tx fits, rest are postponed
			for i := range result {
				if i == 0 {
					result[i] = execution.FilterOK
				} else {
					result[i] = execution.FilterPostpone
				}
			}
			return result
		},
		nil,
	).Maybe()

	seq, err := NewSequencer(
		logger,
		db,
		mockDA,
		config.DefaultConfig(),
		[]byte("test-chain"),
		1000,
		gen,
		mockExec,
	)
	require.NoError(t, err)

	req := coresequencer.GetNextBatchRequest{
		Id:            []byte("test-chain"),
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	// Produce 3 blocks from epoch 100 (1 tx each due to gas filter)
	var timestamps []time.Time
	for i := 0; i < 3; i++ {
		resp, err := seq.GetNextBatch(ctx, req)
		require.NoError(t, err)
		assert.True(t, seq.isCatchingUp(), "should be catching up during block %d", i)
		assert.Equal(t, 1, len(resp.Batch.Transactions), "block %d: exactly 1 forced tx", i)
		timestamps = append(timestamps, resp.Timestamp)
	}

	// All 3 timestamps must be strictly monotonically increasing
	for i := 1; i < len(timestamps); i++ {
		assert.True(t, timestamps[i].After(timestamps[i-1]),
			"timestamp[%d] (%v) must be strictly after timestamp[%d] (%v)",
			i, timestamps[i], i-1, timestamps[i-1])
	}

	// Verify exact jitter values using epochStart + txIndexForTimestamp formula:
	//   epochStart = T - 3ms  (3 total txs in epoch)
	//   Block 0: 1 consumed → txIndex=1 → epochStart + 1ms = T - 2ms
	//   Block 1: 1 consumed → txIndex=2 → epochStart + 2ms = T - 1ms
	//   Block 2: 1 consumed → txIndex=3 (pre-reset) → epochStart + 3ms = T
	assert.Equal(t, epochTimestamp.Add(-2*time.Millisecond), timestamps[0], "block 0: T - 2ms")
	assert.Equal(t, epochTimestamp.Add(-1*time.Millisecond), timestamps[1], "block 1: T - 1ms")
	assert.Equal(t, epochTimestamp, timestamps[2], "block 2: T (exact epoch end time)")

	// Block from epoch 101 should also be monotonically after epoch 100's last block
	resp4, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	assert.True(t, seq.isCatchingUp(), "should still be catching up")
	assert.Equal(t, 1, len(resp4.Batch.Transactions))
	assert.True(t, resp4.Timestamp.After(timestamps[2]),
		"epoch 101 timestamp (%v) must be after epoch 100 last timestamp (%v)",
		resp4.Timestamp, timestamps[2])
	// epoch 101 has 1 tx: epochStart = T2 - 1ms, txIndexForTimestamp=1 → T2 - 1ms + 1ms = T2
	assert.Equal(t, epoch2Timestamp, resp4.Timestamp, "single-tx epoch gets exact DA end time")
}

func TestSequencer_CatchUp_MonotonicTimestamps_EmptyEpoch(t *testing.T) {
	// Verify that an empty DA epoch (no forced txs) still advances the
	// checkpoint and updates currentDAEndTime so subsequent epochs get
	// correct timestamps.
	ctx := context.Background()

	db := ds.NewMapDatastore()
	defer db.Close()

	mockDA := newMockFullDAClient(t)
	forcedInclusionNS := []byte("forced-inclusion")

	mockDA.MockClient.On("GetHeaderNamespace").Return([]byte("header")).Maybe()
	mockDA.MockClient.On("GetDataNamespace").Return([]byte("data")).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return(forcedInclusionNS).Maybe()
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	mockDA.MockClient.On("GetLatestDAHeight", mock.Anything).Return(uint64(110), nil).Once()

	// Epoch 100: empty (no forced txs) but valid timestamp
	emptyEpochTimestamp := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(100), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: emptyEpochTimestamp},
		Data:       [][]byte{},
	}).Once()

	// Epoch 101: has a forced tx with a later timestamp
	epoch2Timestamp := time.Date(2025, 1, 1, 12, 0, 15, 0, time.UTC)
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(101), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: epoch2Timestamp},
		Data:       [][]byte{[]byte("forced-tx-after-empty")},
	}).Once()

	// Epoch 102: future
	mockDA.MockClient.On("Retrieve", mock.Anything, uint64(102), forcedInclusionNS).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusHeightFromFuture},
	}).Maybe()

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq, err := NewSequencer(
		zerolog.Nop(),
		db,
		mockDA,
		config.DefaultConfig(),
		[]byte("test-chain"),
		1000,
		gen,
		createDefaultMockExecutor(t),
	)
	require.NoError(t, err)

	req := coresequencer.GetNextBatchRequest{
		Id:            []byte("test-chain"),
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	// First call processes the empty epoch 100 — empty batch, but checkpoint advances
	resp1, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	assert.True(t, seq.isCatchingUp())
	assert.Equal(t, 0, len(resp1.Batch.Transactions), "empty epoch should produce empty batch")
	assert.Equal(t, emptyEpochTimestamp, resp1.Timestamp,
		"empty epoch batch should use epoch DA end time (0 remaining)")

	// Second call processes epoch 101 — should have later timestamp
	resp2, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	assert.True(t, seq.isCatchingUp())
	assert.Equal(t, 1, len(resp2.Batch.Transactions))
	assert.True(t, resp2.Timestamp.After(resp1.Timestamp),
		"epoch 101 timestamp (%v) must be after empty epoch 100 timestamp (%v)",
		resp2.Timestamp, resp1.Timestamp)
}

func TestSequencer_GetNextBatch_GasFilteringPreservesUnprocessedTxs(t *testing.T) {
	db := ds.NewMapDatastore()
	logger := zerolog.New(zerolog.NewTestWriter(t))

	gen := genesis.Genesis{
		ChainID:                "test-gas-preserve",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	mockDA := newMockFullDAClient(t)
	mockDA.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()
	mockDA.MockClient.On("GetForcedInclusionNamespace").Return([]byte("forced")).Maybe()
	mockDA.MockClient.On("MaxBlobSize", mock.Anything).Return(uint64(1000000), nil).Maybe()

	filterCallCount := 0
	mockExec := mocks.NewMockExecutor(t)
	mockExec.On("GetExecutionInfo", mock.Anything).Return(execution.ExecutionInfo{MaxGas: 1000000}, nil).Maybe()
	mockExec.On("FilterTxs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, txs [][]byte, maxBytes, maxGas uint64, hasForceIncludedTransaction bool) []execution.FilterStatus {
			filterCallCount++
			t.Logf("FilterTxs call #%d with %d txs", filterCallCount, len(txs))

			result := make([]execution.FilterStatus, len(txs))

			if filterCallCount == 1 && len(txs) >= 2 {
				// First call: simulate tx0 fits, tx1 is gas-limited (postponed)
				result[0] = execution.FilterOK
				result[1] = execution.FilterPostpone
				for i := 2; i < len(txs); i++ {
					result[i] = execution.FilterOK
				}
				return result
			}

			// Pass through all txs
			for i := range result {
				result[i] = execution.FilterOK
			}
			return result
		},
		nil,
	).Maybe()

	seq, err := NewSequencer(
		logger,
		db,
		mockDA,
		config.DefaultConfig(),
		[]byte("test-gas-preserve"),
		1000,
		gen,
		mockExec,
	)
	require.NoError(t, err)

	// Set up cached forced inclusion txs: 4 transactions of 50 bytes each
	tx0 := make([]byte, 50)
	tx1 := make([]byte, 50)
	tx2 := make([]byte, 50)
	tx3 := make([]byte, 50)
	copy(tx0, "tx0-valid-fits")
	copy(tx1, "tx1-valid-gas-limited")
	copy(tx2, "tx2-should-not-be-lost")
	copy(tx3, "tx3-should-not-be-lost")
	seq.cachedForcedInclusionTxs = [][]byte{tx0, tx1, tx2, tx3}
	seq.checkpoint.DAHeight = 100
	seq.checkpoint.TxIndex = 0
	seq.SetDAHeight(100)

	ctx := context.Background()
	allProcessedForcedTxs := make([][]byte, 0)

	// Process multiple batches to consume all forced txs
	// Use maxBytes=120 to fetch only 2 txs at a time (each is 50 bytes)
	for i := range 5 { // Max 5 iterations to prevent infinite loop
		req := coresequencer.GetNextBatchRequest{
			Id:       []byte("test-gas-preserve"),
			MaxBytes: 120, // Limits to ~2 txs per batch
		}

		resp, err := seq.GetNextBatch(ctx, req)
		require.NoError(t, err)

		// Extract forced txs from response (they come first)
		for _, tx := range resp.Batch.Transactions {
			// Check if this is one of our original forced txs
			if bytes.HasPrefix(tx, []byte("tx")) {
				allProcessedForcedTxs = append(allProcessedForcedTxs, tx)
			}
		}

		t.Logf("Batch %d: %d txs, checkpoint: height=%d, index=%d, cache size=%d",
			i+1, len(resp.Batch.Transactions), seq.checkpoint.DAHeight, seq.checkpoint.TxIndex, len(seq.cachedForcedInclusionTxs))

		if seq.checkpoint.DAHeight > 100 {
			break // Moved to next DA epoch
		}
		if len(seq.cachedForcedInclusionTxs) == 0 {
			break // Cache exhausted
		}
	}

	// Verify all 4 original forced transactions were processed
	assert.GreaterOrEqual(t, len(allProcessedForcedTxs), 4, "all 4 forced transactions should have been processed")

	// Check that each original tx appears
	txFound := make(map[string]bool)
	for _, tx := range allProcessedForcedTxs {
		txFound[string(tx)] = true
	}

	assert.True(t, txFound[string(tx0)], "tx0 should have been processed")
	assert.True(t, txFound[string(tx1)], "tx1 should have been processed (was gas-limited, retried later)")
	assert.True(t, txFound[string(tx2)], "tx2 should have been processed (must not be lost)")
	assert.True(t, txFound[string(tx3)], "tx3 should have been processed (must not be lost)")
}
