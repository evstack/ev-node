package based

import (
	"context"
	"errors"
	"testing"
	"time"

	ds "github.com/ipfs/go-datastore"
	syncds "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block"
	coreda "github.com/evstack/ev-node/core/da"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
)

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

// createTestSequencer is a helper function to create a sequencer for testing
func createTestSequencer(t *testing.T, mockDA *MockDA, cfg config.Config, gen genesis.Genesis) *BasedSequencer {
	t.Helper()
	daClient := block.NewDAClient(mockDA, cfg, zerolog.Nop())
	fiRetriever := block.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	// Create in-memory datastore
	db := syncds.MutexWrap(ds.NewMapDatastore())

	seq, err := NewBasedSequencer(context.Background(), fiRetriever, db, gen, zerolog.Nop())
	require.NoError(t, err)
	return seq
}

func TestBasedSequencer_SubmitBatchTxs(t *testing.T) {
	mockDA := new(MockDA)
	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAEpochForcedInclusion: 10,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	seq := createTestSequencer(t, mockDA, cfg, gen)

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
	// Transactions should not be processed for based sequencer
	assert.Equal(t, uint64(0), seq.checkpoint.TxIndex)
}

func TestBasedSequencer_GetNextBatch_WithForcedTxs(t *testing.T) {
	testBlobs := [][]byte{[]byte("tx1"), []byte("tx2")}

	mockDA := new(MockDA)
	mockDA.On("GetIDs", mock.Anything, uint64(100), mock.Anything).Return(&coreda.GetIDsResult{
		IDs:       []coreda.ID{[]byte("id1"), []byte("id2")},
		Timestamp: time.Now(),
	}, nil)
	mockDA.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(testBlobs, nil)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	seq := createTestSequencer(t, mockDA, cfg, gen)

	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 2, len(resp.Batch.Transactions))
	assert.Equal(t, []byte("tx1"), resp.Batch.Transactions[0])
	assert.Equal(t, []byte("tx2"), resp.Batch.Transactions[1])

	// Checkpoint should have moved to next DA height
	assert.Equal(t, uint64(101), seq.checkpoint.DAHeight)
	assert.Equal(t, uint64(0), seq.checkpoint.TxIndex)

	mockDA.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_EmptyDA(t *testing.T) {
	mockDA := new(MockDA)
	mockDA.On("GetIDs", mock.Anything, uint64(100), mock.Anything).Return(&coreda.GetIDsResult{
		IDs:       []coreda.ID{},
		Timestamp: time.Now(),
	}, nil)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	seq := createTestSequencer(t, mockDA, cfg, gen)

	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	// Should return empty batch when DA has no transactions
	assert.Equal(t, 0, len(resp.Batch.Transactions))

	mockDA.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_NotConfigured(t *testing.T) {
	mockDA := new(MockDA)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 0, // Not configured
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "" // Empty to trigger not configured

	daClient := block.NewDAClient(mockDA, cfg, zerolog.Nop())
	fiRetriever := block.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	// Create in-memory datastore
	db := syncds.MutexWrap(ds.NewMapDatastore())

	seq, err := NewBasedSequencer(context.Background(), fiRetriever, db, gen, zerolog.Nop())
	require.NoError(t, err)

	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	_, err = seq.GetNextBatch(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forced inclusion not configured")
}

func TestBasedSequencer_GetNextBatch_WithMaxBytes(t *testing.T) {
	// Create transactions of known sizes
	tx1 := make([]byte, 100)
	tx2 := make([]byte, 150)
	tx3 := make([]byte, 200)
	testBlobs := [][]byte{tx1, tx2, tx3}

	mockDA := new(MockDA)
	mockDA.On("GetIDs", mock.Anything, uint64(100), mock.Anything).Return(&coreda.GetIDsResult{
		IDs:       []coreda.ID{[]byte("id1"), []byte("id2"), []byte("id3")},
		Timestamp: time.Now(),
	}, nil)
	mockDA.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(testBlobs, nil)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	seq := createTestSequencer(t, mockDA, cfg, gen)

	// First call with MaxBytes that fits only first 2 transactions
	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      250,
		LastBatchData: nil,
	}

	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	// Should only get first 2 transactions (100 + 150 = 250 bytes)
	assert.Equal(t, 2, len(resp.Batch.Transactions))
	assert.Equal(t, uint64(100), seq.checkpoint.DAHeight)
	assert.Equal(t, uint64(2), seq.checkpoint.TxIndex)

	// Second call should get the remaining transaction
	req = coresequencer.GetNextBatchRequest{
		MaxBytes:      1000,
		LastBatchData: nil,
	}

	resp, err = seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 1, len(resp.Batch.Transactions))
	assert.Equal(t, 200, len(resp.Batch.Transactions[0]))

	// After consuming all transactions, checkpoint should move to next DA height
	assert.Equal(t, uint64(101), seq.checkpoint.DAHeight)
	assert.Equal(t, uint64(0), seq.checkpoint.TxIndex)

	mockDA.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_MultipleDABlocks(t *testing.T) {
	testBlobs1 := [][]byte{[]byte("tx1"), []byte("tx2")}
	testBlobs2 := [][]byte{[]byte("tx3"), []byte("tx4")}

	mockDA := new(MockDA)
	// First DA block
	mockDA.On("GetIDs", mock.Anything, uint64(100), mock.Anything).Return(&coreda.GetIDsResult{
		IDs:       []coreda.ID{[]byte("id1"), []byte("id2")},
		Timestamp: time.Now(),
	}, nil).Once()
	mockDA.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(testBlobs1, nil).Once()

	// Second DA block
	mockDA.On("GetIDs", mock.Anything, uint64(101), mock.Anything).Return(&coreda.GetIDsResult{
		IDs:       []coreda.ID{[]byte("id3"), []byte("id4")},
		Timestamp: time.Now(),
	}, nil).Once()
	mockDA.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(testBlobs2, nil).Once()

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	seq := createTestSequencer(t, mockDA, cfg, gen)

	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	// First batch from first DA block
	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 2, len(resp.Batch.Transactions))
	assert.Equal(t, []byte("tx1"), resp.Batch.Transactions[0])
	assert.Equal(t, []byte("tx2"), resp.Batch.Transactions[1])
	assert.Equal(t, uint64(101), seq.checkpoint.DAHeight)

	// Second batch from second DA block
	resp, err = seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 2, len(resp.Batch.Transactions))
	assert.Equal(t, []byte("tx3"), resp.Batch.Transactions[0])
	assert.Equal(t, []byte("tx4"), resp.Batch.Transactions[1])
	assert.Equal(t, uint64(102), seq.checkpoint.DAHeight)

	mockDA.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_ResumesFromCheckpoint(t *testing.T) {
	testBlobs := [][]byte{[]byte("tx1"), []byte("tx2"), []byte("tx3")}

	mockDA := new(MockDA)
	// No DA calls expected since we manually set the state

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	seq := createTestSequencer(t, mockDA, cfg, gen)

	// Simulate processing first transaction (resuming from checkpoint after restart)
	seq.checkpoint.DAHeight = 100
	seq.checkpoint.TxIndex = 1
	seq.currentBatchTxs = testBlobs

	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	// Should resume from index 1, getting tx2 and tx3
	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 2, len(resp.Batch.Transactions))
	assert.Equal(t, []byte("tx2"), resp.Batch.Transactions[0])
	assert.Equal(t, []byte("tx3"), resp.Batch.Transactions[1])

	// Should have moved to next DA height
	assert.Equal(t, uint64(101), seq.checkpoint.DAHeight)
	assert.Equal(t, uint64(0), seq.checkpoint.TxIndex)
}

func TestBasedSequencer_GetNextBatch_ForcedInclusionExceedsMaxBytes(t *testing.T) {
	// Create a transaction larger than max bytes
	largeTx := make([]byte, 2000000) // 2MB
	testBlobs := [][]byte{largeTx}

	mockDA := new(MockDA)
	mockDA.On("GetIDs", mock.Anything, uint64(100), mock.Anything).Return(&coreda.GetIDsResult{
		IDs:       []coreda.ID{[]byte("id1")},
		Timestamp: time.Now(),
	}, nil)
	mockDA.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(testBlobs, nil)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	seq := createTestSequencer(t, mockDA, cfg, gen)

	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      1000, // Much smaller than the transaction
		LastBatchData: nil,
	}

	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	// Should return empty batch since transaction exceeds max bytes
	assert.Equal(t, 0, len(resp.Batch.Transactions))

	mockDA.AssertExpectations(t)
}

func TestBasedSequencer_VerifyBatch(t *testing.T) {
	mockDA := new(MockDA)
	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAEpochForcedInclusion: 10,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	seq := createTestSequencer(t, mockDA, cfg, gen)

	req := coresequencer.VerifyBatchRequest{
		Id:        []byte("test-chain"),
		BatchData: [][]byte{[]byte("tx1"), []byte("tx2")},
	}

	resp, err := seq.VerifyBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	// Based sequencer always verifies as true since all txs come from DA
	assert.True(t, resp.Status)
}

func TestBasedSequencer_SetDAHeight(t *testing.T) {
	mockDA := new(MockDA)
	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	seq := createTestSequencer(t, mockDA, cfg, gen)

	// Initial height from genesis
	assert.Equal(t, uint64(100), seq.GetDAHeight())

	// Set new height
	seq.SetDAHeight(200)
	assert.Equal(t, uint64(200), seq.GetDAHeight())
}

func TestBasedSequencer_GetNextBatch_ErrorHandling(t *testing.T) {
	mockDA := new(MockDA)
	mockDA.On("GetIDs", mock.Anything, uint64(100), mock.Anything).Return(nil, errors.New("DA connection error"))

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	seq := createTestSequencer(t, mockDA, cfg, gen)

	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	// DA errors are handled gracefully by returning empty batch and retrying
	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 0, len(resp.Batch.Transactions), "Should return empty batch on DA error")

	mockDA.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_HeightFromFuture(t *testing.T) {
	mockDA := new(MockDA)
	mockDA.On("GetIDs", mock.Anything, uint64(100), mock.Anything).Return(nil, coreda.ErrHeightFromFuture)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	seq := createTestSequencer(t, mockDA, cfg, gen)

	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	// Should not error, but return empty batch
	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 0, len(resp.Batch.Transactions))

	// DA height should stay the same
	assert.Equal(t, uint64(100), seq.checkpoint.DAHeight)

	mockDA.AssertExpectations(t)
}

func TestBasedSequencer_CheckpointPersistence(t *testing.T) {
	testBlobs := [][]byte{[]byte("tx1"), []byte("tx2")}

	mockDA := new(MockDA)
	mockDA.On("GetIDs", mock.Anything, uint64(100), mock.Anything).Return(&coreda.GetIDsResult{
		IDs:       []coreda.ID{[]byte("id1"), []byte("id2")},
		Timestamp: time.Now(),
	}, nil)
	mockDA.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(testBlobs, nil)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	// Create persistent datastore
	db := syncds.MutexWrap(ds.NewMapDatastore())

	// Create first sequencer
	daClient := block.NewDAClient(mockDA, cfg, zerolog.Nop())
	fiRetriever := block.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	seq1, err := NewBasedSequencer(context.Background(), fiRetriever, db, gen, zerolog.Nop())
	require.NoError(t, err)

	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	// Process a batch
	resp, err := seq1.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 2, len(resp.Batch.Transactions))

	// Create a new sequencer with the same datastore (simulating restart)
	seq2, err := NewBasedSequencer(context.Background(), fiRetriever, db, gen, zerolog.Nop())
	require.NoError(t, err)

	// Checkpoint should be loaded from DB
	assert.Equal(t, uint64(101), seq2.checkpoint.DAHeight)
	assert.Equal(t, uint64(0), seq2.checkpoint.TxIndex)

	mockDA.AssertExpectations(t)
}
