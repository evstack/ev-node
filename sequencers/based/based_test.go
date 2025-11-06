package based

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreda "github.com/evstack/ev-node/core/da"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
)

// MockDARetriever is a mock implementation of DARetriever for testing
type MockDARetriever struct {
	mock.Mock
}

func (m *MockDARetriever) RetrieveForcedIncludedTxsFromDA(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error) {
	args := m.Called(ctx, daHeight)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ForcedInclusionEvent), args.Error(1)
}

// MockDA is a mock implementation of DA for testing
type MockDA struct {
	mock.Mock
}

func (m *MockDA) Submit(ctx context.Context, blobs [][]byte, gasPrice float64, namespace []byte) ([][]byte, error) {
	args := m.Called(ctx, blobs, gasPrice, namespace)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([][]byte), args.Error(1)
}

func (m *MockDA) SubmitWithOptions(ctx context.Context, blobs [][]byte, gasPrice float64, namespace []byte, options []byte) ([][]byte, error) {
	args := m.Called(ctx, blobs, gasPrice, namespace, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([][]byte), args.Error(1)
}

func (m *MockDA) GetIDs(ctx context.Context, height uint64, namespace []byte) (*coreda.GetIDsResult, error) {
	args := m.Called(ctx, height, namespace)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*coreda.GetIDsResult), args.Error(1)
}

func (m *MockDA) Get(ctx context.Context, ids [][]byte, namespace []byte) ([][]byte, error) {
	args := m.Called(ctx, ids, namespace)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([][]byte), args.Error(1)
}

func (m *MockDA) GetProofs(ctx context.Context, ids [][]byte, namespace []byte) ([]coreda.Proof, error) {
	args := m.Called(ctx, ids, namespace)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]coreda.Proof), args.Error(1)
}

func (m *MockDA) Validate(ctx context.Context, ids [][]byte, proofs []coreda.Proof, namespace []byte) ([]bool, error) {
	args := m.Called(ctx, ids, proofs, namespace)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]bool), args.Error(1)
}

func (m *MockDA) Commit(ctx context.Context, blobs [][]byte, namespace []byte) ([][]byte, error) {
	args := m.Called(ctx, blobs, namespace)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([][]byte), args.Error(1)
}

func (m *MockDA) GasPrice(ctx context.Context) (float64, error) {
	args := m.Called(ctx)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockDA) GasMultiplier(ctx context.Context) (float64, error) {
	args := m.Called(ctx)
	return args.Get(0).(float64), args.Error(1)
}

func TestNewBasedSequencer(t *testing.T) {
	mockRetriever := new(MockDARetriever)
	mockDA := new(MockDA)
	cfg := config.DefaultConfig()
	gen := genesis.Genesis{
		ChainID:       "test-chain",
		DAStartHeight: 100,
	}

	seq := NewBasedSequencer(mockRetriever, mockDA, cfg, gen, zerolog.Nop())

	require.NotNil(t, seq)
	assert.Equal(t, uint64(100), seq.daHeight)
	assert.Equal(t, 0, len(seq.txQueue))
}

func TestBasedSequencer_SubmitBatchTxs(t *testing.T) {
	mockRetriever := new(MockDARetriever)
	mockDA := new(MockDA)
	cfg := config.DefaultConfig()
	gen := genesis.Genesis{ChainID: "test-chain"}

	seq := NewBasedSequencer(mockRetriever, mockDA, cfg, gen, zerolog.Nop())

	// Submit should succeed but be ignored
	req := coresequencer.SubmitBatchTxsRequest{
		Id: []byte("test-chain"),
		Batch: &coresequencer.Batch{
			Transactions: [][]byte{[]byte("tx1"), []byte("tx2")},
		},
	}

	resp, err := seq.SubmitBatchTxs(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Queue should still be empty
	assert.Equal(t, 0, len(seq.txQueue))
}

func TestBasedSequencer_GetNextBatch_WithForcedTxs(t *testing.T) {
	mockRetriever := new(MockDARetriever)
	mockDA := new(MockDA)
	cfg := config.DefaultConfig()
	gen := genesis.Genesis{
		ChainID:       "test-chain",
		DAStartHeight: 100,
	}

	seq := NewBasedSequencer(mockRetriever, mockDA, cfg, gen, zerolog.Nop())

	// Mock retriever to return forced inclusion transactions
	forcedTxs := &ForcedInclusionEvent{
		Txs:           [][]byte{[]byte("forced_tx1"), []byte("forced_tx2")},
		StartDaHeight: 101,
		EndDaHeight:   105,
	}
	mockRetriever.On("RetrieveForcedIncludedTxsFromDA", mock.Anything, uint64(100)).
		Return(forcedTxs, nil).Once()

	req := coresequencer.GetNextBatchRequest{
		Id:       []byte("test-chain"),
		MaxBytes: 10000,
	}

	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 2, len(resp.Batch.Transactions))
	assert.Equal(t, []byte("forced_tx1"), resp.Batch.Transactions[0])
	assert.Equal(t, []byte("forced_tx2"), resp.Batch.Transactions[1])

	// DA height should be updated
	assert.Equal(t, uint64(105), seq.GetDAHeight())

	mockRetriever.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_EmptyDA(t *testing.T) {
	mockRetriever := new(MockDARetriever)
	mockDA := new(MockDA)
	cfg := config.DefaultConfig()
	gen := genesis.Genesis{
		ChainID:       "test-chain",
		DAStartHeight: 100,
	}

	seq := NewBasedSequencer(mockRetriever, mockDA, cfg, gen, zerolog.Nop())

	// Mock retriever to return empty transactions
	emptyEvent := &ForcedInclusionEvent{
		Txs:           [][]byte{},
		StartDaHeight: 100,
		EndDaHeight:   100,
	}
	mockRetriever.On("RetrieveForcedIncludedTxsFromDA", mock.Anything, uint64(100)).
		Return(emptyEvent, nil).Once()

	req := coresequencer.GetNextBatchRequest{
		Id:       []byte("test-chain"),
		MaxBytes: 10000,
	}

	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 0, len(resp.Batch.Transactions))

	mockRetriever.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_NotConfigured(t *testing.T) {
	mockRetriever := new(MockDARetriever)
	mockDA := new(MockDA)
	cfg := config.DefaultConfig()
	gen := genesis.Genesis{
		ChainID:       "test-chain",
		DAStartHeight: 100,
	}

	seq := NewBasedSequencer(mockRetriever, mockDA, cfg, gen, zerolog.Nop())

	// Mock retriever to return not configured error
	mockRetriever.On("RetrieveForcedIncludedTxsFromDA", mock.Anything, uint64(100)).
		Return(nil, errors.New("forced inclusion namespace not configured")).Once()

	req := coresequencer.GetNextBatchRequest{
		Id:       []byte("test-chain"),
		MaxBytes: 10000,
	}

	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Nil(t, resp.Batch.Transactions)

	mockRetriever.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_HeightFromFuture(t *testing.T) {
	mockRetriever := new(MockDARetriever)
	mockDA := new(MockDA)
	cfg := config.DefaultConfig()
	gen := genesis.Genesis{
		ChainID:       "test-chain",
		DAStartHeight: 100,
	}

	seq := NewBasedSequencer(mockRetriever, mockDA, cfg, gen, zerolog.Nop())

	// Mock retriever to return height from future error
	mockRetriever.On("RetrieveForcedIncludedTxsFromDA", mock.Anything, uint64(100)).
		Return(nil, coreda.ErrHeightFromFuture).Once()

	req := coresequencer.GetNextBatchRequest{
		Id:       []byte("test-chain"),
		MaxBytes: 10000,
	}

	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Nil(t, resp.Batch.Transactions)

	// DA height should NOT increment on ErrHeightFromFuture - we wait for DA to catch up
	assert.Equal(t, uint64(100), seq.GetDAHeight())

	mockRetriever.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_WithMaxBytes(t *testing.T) {
	mockRetriever := new(MockDARetriever)
	mockDA := new(MockDA)
	cfg := config.DefaultConfig()
	gen := genesis.Genesis{
		ChainID:       "test-chain",
		DAStartHeight: 100,
	}

	seq := NewBasedSequencer(mockRetriever, mockDA, cfg, gen, zerolog.Nop())

	// Create transactions that will exceed max bytes
	tx1 := make([]byte, 50)
	tx2 := make([]byte, 50)
	tx3 := make([]byte, 50)

	forcedTxs := &ForcedInclusionEvent{
		Txs:           [][]byte{tx1, tx2, tx3},
		StartDaHeight: 101,
		EndDaHeight:   105,
	}
	mockRetriever.On("RetrieveForcedIncludedTxsFromDA", mock.Anything, uint64(100)).
		Return(forcedTxs, nil).Once()

	// Request with max bytes that only fits 2 transactions
	req := coresequencer.GetNextBatchRequest{
		Id:       []byte("test-chain"),
		MaxBytes: 100, // Only fits 2 transactions
	}

	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 2, len(resp.Batch.Transactions))

	// Third transaction should still be in queue
	assert.Equal(t, 1, len(seq.txQueue))

	// Next request should return the remaining transaction
	req2 := coresequencer.GetNextBatchRequest{
		Id:       []byte("test-chain"),
		MaxBytes: 100,
	}

	resp2, err := seq.GetNextBatch(context.Background(), req2)
	require.NoError(t, err)
	require.NotNil(t, resp2)
	require.NotNil(t, resp2.Batch)
	assert.Equal(t, 1, len(resp2.Batch.Transactions))
	assert.Equal(t, 0, len(seq.txQueue))

	mockRetriever.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_FromQueue(t *testing.T) {
	mockRetriever := new(MockDARetriever)
	mockDA := new(MockDA)
	cfg := config.DefaultConfig()
	gen := genesis.Genesis{
		ChainID:       "test-chain",
		DAStartHeight: 100,
	}

	seq := NewBasedSequencer(mockRetriever, mockDA, cfg, gen, zerolog.Nop())

	// Pre-populate the queue
	seq.txQueue = [][]byte{[]byte("queued_tx1"), []byte("queued_tx2")}

	req := coresequencer.GetNextBatchRequest{
		Id:       []byte("test-chain"),
		MaxBytes: 10000,
	}

	// Should return from queue without calling retriever
	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 2, len(resp.Batch.Transactions))
	assert.Equal(t, []byte("queued_tx1"), resp.Batch.Transactions[0])
	assert.Equal(t, []byte("queued_tx2"), resp.Batch.Transactions[1])
	assert.Equal(t, 0, len(seq.txQueue))

	// No expectations on retriever since it shouldn't be called
	mockRetriever.AssertExpectations(t)
}

func TestBasedSequencer_VerifyBatch(t *testing.T) {
	mockRetriever := new(MockDARetriever)
	mockDA := new(MockDA)
	cfg := config.DefaultConfig()
	gen := genesis.Genesis{ChainID: "test-chain"}

	seq := NewBasedSequencer(mockRetriever, mockDA, cfg, gen, zerolog.Nop())

	req := coresequencer.VerifyBatchRequest{
		Id:        []byte("test-chain"),
		BatchData: [][]byte{[]byte("tx1")},
	}

	resp, err := seq.VerifyBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Status)
}

func TestBasedSequencer_SetDAHeight(t *testing.T) {
	mockRetriever := new(MockDARetriever)
	mockDA := new(MockDA)
	cfg := config.DefaultConfig()
	gen := genesis.Genesis{
		ChainID:       "test-chain",
		DAStartHeight: 100,
	}

	seq := NewBasedSequencer(mockRetriever, mockDA, cfg, gen, zerolog.Nop())

	assert.Equal(t, uint64(100), seq.GetDAHeight())

	seq.SetDAHeight(200)
	assert.Equal(t, uint64(200), seq.GetDAHeight())
}

func TestBasedSequencer_ConcurrentAccess(t *testing.T) {
	mockRetriever := new(MockDARetriever)
	mockDA := new(MockDA)
	cfg := config.DefaultConfig()
	gen := genesis.Genesis{
		ChainID:       "test-chain",
		DAStartHeight: 100,
	}

	seq := NewBasedSequencer(mockRetriever, mockDA, cfg, gen, zerolog.Nop())

	// Mock retriever to return transactions
	forcedTxs := &ForcedInclusionEvent{
		Txs:           [][]byte{[]byte("tx1")},
		StartDaHeight: 101,
		EndDaHeight:   105,
	}
	mockRetriever.On("RetrieveForcedIncludedTxsFromDA", mock.Anything, mock.Anything).
		Return(forcedTxs, nil).Maybe()

	// Test concurrent access
	done := make(chan bool, 3)

	// Concurrent GetNextBatch calls
	go func() {
		req := coresequencer.GetNextBatchRequest{Id: []byte("test-chain"), MaxBytes: 1000}
		_, _ = seq.GetNextBatch(context.Background(), req)
		done <- true
	}()

	// Concurrent SetDAHeight calls
	go func() {
		seq.SetDAHeight(200)
		done <- true
	}()

	// Concurrent GetDAHeight calls
	go func() {
		_ = seq.GetDAHeight()
		done <- true
	}()

	// Wait for all goroutines
	timeout := time.After(5 * time.Second)
	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-timeout:
			t.Fatal("test timed out")
		}
	}
}

func TestBasedSequencer_GetNextBatch_ErrorHandling(t *testing.T) {
	mockRetriever := new(MockDARetriever)
	mockDA := new(MockDA)
	cfg := config.DefaultConfig()
	gen := genesis.Genesis{
		ChainID:       "test-chain",
		DAStartHeight: 100,
	}

	seq := NewBasedSequencer(mockRetriever, mockDA, cfg, gen, zerolog.Nop())

	// Mock retriever to return an unexpected error
	expectedErr := errors.New("unexpected DA error")
	mockRetriever.On("RetrieveForcedIncludedTxsFromDA", mock.Anything, uint64(100)).
		Return(nil, expectedErr).Once()

	req := coresequencer.GetNextBatchRequest{
		Id:       []byte("test-chain"),
		MaxBytes: 10000,
	}

	resp, err := seq.GetNextBatch(context.Background(), req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, expectedErr, err)

	mockRetriever.AssertExpectations(t)
}
