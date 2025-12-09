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
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	damocks "github.com/evstack/ev-node/test/mocks"
)

// MockForcedInclusionRetriever is a mock implementation of DARetriever for testing
type MockForcedInclusionRetriever struct {
	mock.Mock
}

func (m *MockForcedInclusionRetriever) RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*block.ForcedInclusionEvent, error) {
	args := m.Called(ctx, daHeight)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*block.ForcedInclusionEvent), args.Error(1)
}

// testDAClient is a lightweight adapter implementing block.DAClient for tests.
type testDAClient struct {
	da       datypes.DA
	headerNS []byte
	dataNS   []byte
	forcedNS []byte
}

func newTestDAClient(da datypes.DA) *testDAClient {
	return &testDAClient{
		da:       da,
		headerNS: []byte("h"),
		dataNS:   []byte("d"),
		forcedNS: []byte("f"),
	}
}

func (t *testDAClient) Submit(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) datypes.ResultSubmit {
	return datypes.ResultSubmit{}
}

func (t *testDAClient) Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	return datypes.ResultRetrieve{}
}

func (t *testDAClient) RetrieveHeaders(ctx context.Context, height uint64) datypes.ResultRetrieve {
	return t.Retrieve(ctx, height, t.headerNS)
}

func (t *testDAClient) RetrieveData(ctx context.Context, height uint64) datypes.ResultRetrieve {
	return t.Retrieve(ctx, height, t.dataNS)
}

func (t *testDAClient) RetrieveForcedInclusion(ctx context.Context, height uint64) datypes.ResultRetrieve {
	return t.Retrieve(ctx, height, t.forcedNS)
}

func (t *testDAClient) Get(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error) {
	if t.da == nil {
		return nil, nil
	}
	return t.da.Get(ctx, ids, namespace)
}

func (t *testDAClient) GetProofs(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Proof, error) {
	if t.da == nil {
		return []datypes.Proof{}, nil
	}
	return t.da.GetProofs(ctx, ids, namespace)
}

func (t *testDAClient) Validate(ctx context.Context, ids []datypes.ID, proofs []datypes.Proof, namespace []byte) ([]bool, error) {
	if t.da == nil {
		return []bool{}, nil
	}
	return t.da.Validate(ctx, ids, proofs, namespace)
}

func (t *testDAClient) GetHeaderNamespace() []byte {
	return t.headerNS
}

func (t *testDAClient) GetDataNamespace() []byte {
	return t.dataNS
}

func (t *testDAClient) GetForcedInclusionNamespace() []byte {
	return t.forcedNS
}

func (t *testDAClient) HasForcedInclusionNamespace() bool {
	return len(t.forcedNS) > 0
}

// mockDAAdapter wraps the generated DA mock to satisfy block.DAClient.
type mockDAAdapter struct {
	*damocks.MockDA
}

func (m *mockDAAdapter) Submit(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) datypes.ResultSubmit {
	return datypes.ResultSubmit{}
}

func (m *mockDAAdapter) Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	return datypes.ResultRetrieve{}
}

func (m *mockDAAdapter) RetrieveHeaders(ctx context.Context, height uint64) datypes.ResultRetrieve {
	return datypes.ResultRetrieve{}
}

func (m *mockDAAdapter) RetrieveData(ctx context.Context, height uint64) datypes.ResultRetrieve {
	return datypes.ResultRetrieve{}
}

func (m *mockDAAdapter) RetrieveForcedInclusion(ctx context.Context, height uint64) datypes.ResultRetrieve {
	return datypes.ResultRetrieve{}
}

func (m *mockDAAdapter) Get(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error) {
	args := m.Called(ctx, ids, namespace)
	if blobs, ok := args.Get(0).([]datypes.Blob); ok {
		return blobs, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockDAAdapter) GetProofs(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Proof, error) {
	args := m.Called(ctx, ids, namespace)
	if proofs, ok := args.Get(0).([]datypes.Proof); ok {
		return proofs, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockDAAdapter) Validate(ctx context.Context, ids []datypes.ID, proofs []datypes.Proof, namespace []byte) ([]bool, error) {
	args := m.Called(ctx, ids, proofs, namespace)
	if results, ok := args.Get(0).([]bool); ok {
		return results, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockDAAdapter) GetHeaderNamespace() []byte { return []byte("h") }
func (m *mockDAAdapter) GetDataNamespace() []byte   { return []byte("d") }
func (m *mockDAAdapter) GetForcedInclusionNamespace() []byte {
	return []byte("f")
}
func (m *mockDAAdapter) HasForcedInclusionNamespace() bool { return true }

func TestSequencer_SubmitBatchTxs(t *testing.T) {
	dummyDA := datypes.NewDummyDA(100_000_000, 10*time.Second)
	daClient := newTestDAClient(dummyDA)
	db := ds.NewMapDatastore()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	Id := []byte("test1")
	logger := zerolog.Nop()
	mockRetriever := new(MockForcedInclusionRetriever)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, mock.Anything).
		Return(nil, block.ErrForceInclusionNotConfigured).Maybe()
	seq, err := NewSequencer(ctx, logger, db, daClient, Id, 10*time.Second, false, 1000, mockRetriever, genesis.Genesis{})
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
	dummyDA := datypes.NewDummyDA(100_000_000, 10*time.Second)
	daClient := newTestDAClient(dummyDA)
	db := ds.NewMapDatastore()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	Id := []byte("test1")
	logger := zerolog.Nop()
	mockRetriever := new(MockForcedInclusionRetriever)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, mock.Anything).
		Return(nil, block.ErrForceInclusionNotConfigured).Maybe()
	seq, err := NewSequencer(ctx, logger, db, daClient, Id, 10*time.Second, false, 1000, mockRetriever, genesis.Genesis{})
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
	logger := zerolog.Nop()

	mockRetriever := new(MockForcedInclusionRetriever)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, mock.Anything).
		Return(nil, block.ErrForceInclusionNotConfigured).Maybe()
	seq := &Sequencer{
		logger:      logger,
		queue:       NewBatchQueue(db, "batches", 0), // 0 = unlimited for test
		Id:          []byte("test"),
		fiRetriever: mockRetriever,
	}
	defer func() {
		err := db.Close()
		if err != nil {
			t.Fatalf("Failed to close sequencer: %v", err)
		}
	}()

	// Test case where lastBatchHash and seq.lastBatchHash are both nil
	res, err := seq.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{Id: seq.Id})
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
	logger := zerolog.Nop()

	mockRetriever := new(MockForcedInclusionRetriever)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, mock.Anything).
		Return(nil, block.ErrForceInclusionNotConfigured).Maybe()
	seq := &Sequencer{
		logger:      logger,
		queue:       NewBatchQueue(db, "batches", 0), // 0 = unlimited for test
		Id:          []byte("test"),
		fiRetriever: mockRetriever,
	}
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

	Id := []byte("test")
	batchData := [][]byte{[]byte("batch1"), []byte("batch2")}
	proofs := [][]byte{[]byte("proof1"), []byte("proof2")}

	t.Run("Proposer Mode", func(t *testing.T) {
		mockDA := damocks.NewMockDA(t)
		logger := zerolog.Nop()
		mockRetriever := new(MockForcedInclusionRetriever)
		mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, mock.Anything).
			Return(nil, block.ErrForceInclusionNotConfigured).Maybe()

		seq := &Sequencer{
			logger:      logger,
			Id:          Id,
			proposer:    true,
			da:          &mockDAAdapter{MockDA: mockDA},
			queue:       NewBatchQueue(db, "proposer_queue", 0), // 0 = unlimited for test
			fiRetriever: mockRetriever,
		}

		res, err := seq.VerifyBatch(context.Background(), coresequencer.VerifyBatchRequest{Id: seq.Id, BatchData: batchData})
		assert.NoError(err)
		assert.NotNil(res)
		assert.True(res.Status, "Expected status to be true in proposer mode")

		mockDA.AssertNotCalled(t, "GetProofs", context.Background(), mock.Anything)
		mockDA.AssertNotCalled(t, "Validate", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("Non-Proposer Mode", func(t *testing.T) {
		t.Run("Valid Proofs", func(t *testing.T) {
			mockDA := damocks.NewMockDA(t)
			logger := zerolog.Nop()
			mockRetriever := new(MockForcedInclusionRetriever)
			mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, mock.Anything).
				Return(nil, block.ErrForceInclusionNotConfigured).Maybe()
			seq := &Sequencer{
				logger:      logger,
				Id:          Id,
				proposer:    false,
				da:          &mockDAAdapter{MockDA: mockDA},
				queue:       NewBatchQueue(db, "valid_proofs_queue", 0),
				fiRetriever: mockRetriever,
			}

			mockDA.On("GetProofs", context.Background(), batchData, Id).Return(proofs, nil).Once()
			mockDA.On("Validate", mock.Anything, batchData, proofs, Id).Return([]bool{true, true}, nil).Once()

			res, err := seq.VerifyBatch(context.Background(), coresequencer.VerifyBatchRequest{Id: seq.Id, BatchData: batchData})
			assert.NoError(err)
			assert.NotNil(res)
			assert.True(res.Status, "Expected status to be true for valid proofs")
			mockDA.AssertExpectations(t)
		})

		t.Run("Invalid Proof", func(t *testing.T) {
			mockDA := damocks.NewMockDA(t)
			logger := zerolog.Nop()
			mockRetriever := new(MockForcedInclusionRetriever)
			mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, mock.Anything).
				Return(nil, block.ErrForceInclusionNotConfigured).Maybe()
			seq := &Sequencer{
				logger:      logger,
				Id:          Id,
				proposer:    false,
				da:          &mockDAAdapter{MockDA: mockDA},
				queue:       NewBatchQueue(db, "invalid_proof_queue", 0),
				fiRetriever: mockRetriever,
			}

			mockDA.On("GetProofs", context.Background(), batchData, Id).Return(proofs, nil).Once()
			mockDA.On("Validate", mock.Anything, batchData, proofs, Id).Return([]bool{true, false}, nil).Once()

			res, err := seq.VerifyBatch(context.Background(), coresequencer.VerifyBatchRequest{Id: seq.Id, BatchData: batchData})
			assert.NoError(err)
			assert.NotNil(res)
			assert.False(res.Status, "Expected status to be false for invalid proof")
			mockDA.AssertExpectations(t)
		})

		t.Run("GetProofs Error", func(t *testing.T) {
			mockDA := damocks.NewMockDA(t)
			logger := zerolog.Nop()
			mockRetriever := new(MockForcedInclusionRetriever)
			mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, mock.Anything).
				Return(nil, block.ErrForceInclusionNotConfigured).Maybe()
			seq := &Sequencer{
				logger:      logger,
				Id:          Id,
				proposer:    false,
				da:          &mockDAAdapter{MockDA: mockDA},
				queue:       NewBatchQueue(db, "getproofs_err_queue", 0),
				fiRetriever: mockRetriever,
			}
			expectedErr := errors.New("get proofs failed")

			mockDA.On("GetProofs", context.Background(), batchData, Id).Return(nil, expectedErr).Once()

			res, err := seq.VerifyBatch(context.Background(), coresequencer.VerifyBatchRequest{Id: seq.Id, BatchData: batchData})
			assert.Error(err)
			assert.Nil(res)
			assert.Contains(err.Error(), expectedErr.Error())
			mockDA.AssertExpectations(t)
			mockDA.AssertNotCalled(t, "Validate", mock.Anything, mock.Anything, mock.Anything)
		})

		t.Run("Validate Error", func(t *testing.T) {
			mockDA := damocks.NewMockDA(t)
			logger := zerolog.Nop()
			mockRetriever := new(MockForcedInclusionRetriever)
			mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, mock.Anything).
				Return(nil, block.ErrForceInclusionNotConfigured).Maybe()
			seq := &Sequencer{
				logger:      logger,
				Id:          Id,
				proposer:    false,
				da:          &mockDAAdapter{MockDA: mockDA},
				queue:       NewBatchQueue(db, "validate_err_queue", 0),
				fiRetriever: mockRetriever,
			}
			expectedErr := errors.New("validate failed")

			mockDA.On("GetProofs", context.Background(), batchData, Id).Return(proofs, nil).Once()
			mockDA.On("Validate", mock.Anything, batchData, proofs, Id).Return(nil, expectedErr).Once()

			res, err := seq.VerifyBatch(context.Background(), coresequencer.VerifyBatchRequest{Id: seq.Id, BatchData: batchData})
			assert.Error(err)
			assert.Nil(res)
			assert.Contains(err.Error(), expectedErr.Error())
			mockDA.AssertExpectations(t)
		})

		t.Run("Invalid ID", func(t *testing.T) {
			mockDA := damocks.NewMockDA(t)
			logger := zerolog.Nop()
			mockRetriever := new(MockForcedInclusionRetriever)
			mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, mock.Anything).
				Return(nil, block.ErrForceInclusionNotConfigured).Maybe()

			seq := &Sequencer{
				logger:      logger,
				Id:          Id,
				proposer:    false,
				da:          &mockDAAdapter{MockDA: mockDA},
				queue:       NewBatchQueue(db, "invalid_queue", 0),
				fiRetriever: mockRetriever,
			}

			invalidId := []byte("invalid")
			res, err := seq.VerifyBatch(context.Background(), coresequencer.VerifyBatchRequest{Id: invalidId, BatchData: batchData})
			assert.Error(err)
			assert.Nil(res)
			assert.ErrorIs(err, ErrInvalidId)

			mockDA.AssertNotCalled(t, "GetProofs", context.Background(), mock.Anything)
			mockDA.AssertNotCalled(t, "Validate", mock.Anything, mock.Anything, mock.Anything)
		})
	})
}

func TestSequencer_GetNextBatch_BeforeDASubmission(t *testing.T) {
	t.Skip()
	// Initialize a new sequencer with mock DA
	mockDA := &damocks.MockDA{}
	db := ds.NewMapDatastore()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logger := zerolog.Nop()
	mockRetriever := new(MockForcedInclusionRetriever)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, mock.Anything).
		Return(nil, block.ErrForceInclusionNotConfigured).Maybe()
	seq, err := NewSequencer(ctx, logger, db, &mockDAAdapter{MockDA: mockDA}, []byte("test1"), 1*time.Second, false, 1000, mockRetriever, genesis.Genesis{})
	if err != nil {
		t.Fatalf("Failed to create sequencer: %v", err)
	}
	defer func() {
		err := db.Close()
		if err != nil {
			t.Fatalf("Failed to close sequencer: %v", err)
		}
	}()

	// Set up mock expectations
	mockDA.On("Submit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("mock DA always rejects submissions"))

	// Submit a batch
	Id := []byte("test1")
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

	// Verify all mock expectations were met
	mockDA.AssertExpectations(t)
}

func TestSequencer_GetNextBatch_ForcedInclusionAndBatch_MaxBytes(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(zerolog.NewConsoleWriter())

	// Create in-memory datastore
	db := ds.NewMapDatastore()

	// Create mock forced inclusion retriever with txs that are 50 bytes each
	mockFI := &MockForcedInclusionRetriever{}
	forcedTx1 := make([]byte, 50)
	forcedTx2 := make([]byte, 60)
	mockFI.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(100)).Return(&block.ForcedInclusionEvent{
		Txs:           [][]byte{forcedTx1, forcedTx2}, // Total 110 bytes
		StartDaHeight: 100,
		EndDaHeight:   100,
	}, nil)

	gen := genesis.Genesis{
		ChainID:       "test-chain",
		DAStartHeight: 100,
	}

	seq, err := NewSequencer(
		ctx,
		logger,
		db,
		nil,
		[]byte("test-chain"),
		1*time.Second,
		true,
		100,
		mockFI,
		gen,
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

	mockFI.AssertExpectations(t)
}

func TestSequencer_GetNextBatch_ForcedInclusion_ExceedsMaxBytes(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(zerolog.NewConsoleWriter())

	db := ds.NewMapDatastore()

	// Create forced inclusion txs where combined they exceed maxBytes
	mockFI := &MockForcedInclusionRetriever{}
	forcedTx1 := make([]byte, 100)
	forcedTx2 := make([]byte, 80) // This would be deferred
	mockFI.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(100)).Return(&block.ForcedInclusionEvent{
		Txs:           [][]byte{forcedTx1, forcedTx2},
		StartDaHeight: 100,
		EndDaHeight:   100,
	}, nil).Once()

	// Second call should process pending tx at DA height 101 (after first call bumped it to epochEnd + 1)
	mockFI.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(101)).Return(&block.ForcedInclusionEvent{
		Txs:           [][]byte{},
		StartDaHeight: 101,
		EndDaHeight:   101,
	}, nil).Once()

	gen := genesis.Genesis{
		ChainID:       "test-chain",
		DAStartHeight: 100,
	}

	seq, err := NewSequencer(
		ctx,
		logger,
		db,
		nil,
		[]byte("test-chain"),
		1*time.Second,
		true,
		100,
		mockFI,
		gen,
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

	// Verify pending tx is stored
	assert.Equal(t, 1, len(seq.pendingForcedInclusionTxs), "Second tx should be pending")

	// Second call - should get the pending forced tx
	resp2, err := seq.GetNextBatch(ctx, getReq)
	require.NoError(t, err)
	require.NotNil(t, resp2.Batch)
	assert.Equal(t, 1, len(resp2.Batch.Transactions), "Should include pending forced tx")
	assert.Equal(t, 80, len(resp2.Batch.Transactions[0]))

	// Pending queue should now be empty
	assert.Equal(t, 0, len(seq.pendingForcedInclusionTxs), "Pending queue should be empty")

	mockFI.AssertExpectations(t)
}

func TestSequencer_GetNextBatch_AlwaysCheckPendingForcedInclusion(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(zerolog.NewConsoleWriter())

	db := ds.NewMapDatastore()

	mockFI := &MockForcedInclusionRetriever{}

	// First call returns a large forced tx that gets deferred
	largeForcedTx := make([]byte, 150)
	mockFI.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(100)).Return(&block.ForcedInclusionEvent{
		Txs:           [][]byte{largeForcedTx},
		StartDaHeight: 100,
		EndDaHeight:   100,
	}, nil).Once()

	// Second call returns no new forced txs at height 101 (after first call bumped DA height to epochEnd + 1), but pending should still be processed
	mockFI.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(101)).Return(&block.ForcedInclusionEvent{
		Txs:           [][]byte{},
		StartDaHeight: 101,
		EndDaHeight:   101,
	}, nil).Once()

	gen := genesis.Genesis{
		ChainID:       "test-chain",
		DAStartHeight: 100,
	}

	seq, err := NewSequencer(
		ctx,
		logger,
		db,
		nil,
		[]byte("test-chain"),
		1*time.Second,
		true,
		100,
		mockFI,
		gen,
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

	// First call with maxBytes = 100
	// Large forced tx (150 bytes) won't fit, gets deferred
	// Batch tx (50 bytes) should be returned
	getReq := coresequencer.GetNextBatchRequest{
		Id:            []byte("test-chain"),
		MaxBytes:      100,
		LastBatchData: nil,
	}

	resp, err := seq.GetNextBatch(ctx, getReq)
	require.NoError(t, err)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 1, len(resp.Batch.Transactions), "Should have batch tx only")
	assert.Equal(t, 50, len(resp.Batch.Transactions[0]))

	// Verify pending forced tx is stored
	assert.Equal(t, 1, len(seq.pendingForcedInclusionTxs), "Large forced tx should be pending")

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
	assert.Equal(t, 150, len(resp2.Batch.Transactions[0]))

	// Pending queue should now be empty
	assert.Equal(t, 0, len(seq.pendingForcedInclusionTxs), "Pending queue should be empty")

	mockFI.AssertExpectations(t)
}

func TestSequencer_QueueLimit_Integration(t *testing.T) {
	// Test integration between sequencer and queue limits to demonstrate backpressure
	db := ds.NewMapDatastore()
	defer db.Close()

	mockDA := &damocks.MockDA{}
	mockRetriever := new(MockForcedInclusionRetriever)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, mock.Anything).
		Return(nil, block.ErrForceInclusionNotConfigured).Maybe()

	// Create a sequencer with a small queue limit for testing
	logger := zerolog.Nop()
	seq := &Sequencer{
		logger:      logger,
		da:          &mockDAAdapter{MockDA: mockDA},
		batchTime:   time.Second,
		Id:          []byte("test"),
		queue:       NewBatchQueue(db, "test_queue", 2), // Very small limit for testing
		proposer:    true,
		fiRetriever: mockRetriever,
	}

	ctx := context.Background()

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
	dummyDA := datypes.NewDummyDA(100_000, 100*time.Millisecond)
	dummyDA.StartHeightTicker()
	defer dummyDA.StopHeightTicker()
	daClient := newTestDAClient(dummyDA)

	// Create sequencer with small queue size to trigger throttling quickly
	queueSize := 3 // Small for testing
	logger := zerolog.Nop()
	mockRetriever := new(MockForcedInclusionRetriever)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, mock.Anything).
		Return(nil, block.ErrForceInclusionNotConfigured).Maybe()
	seq, err := NewSequencer(
		context.Background(),
		logger,
		db,
		daClient,
		[]byte("test-chain"),
		100*time.Millisecond,
		true, // proposer
		queueSize,
		mockRetriever,     // fiRetriever
		genesis.Genesis{}, // genesis
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
