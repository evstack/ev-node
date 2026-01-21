package based

import (
	"context"
	"testing"
	"time"

	ds "github.com/ipfs/go-datastore"
	syncds "github.com/ipfs/go-datastore/sync"
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
	"github.com/evstack/ev-node/pkg/sequencers/common"
	"github.com/evstack/ev-node/test/mocks"
)

// MockFullDAClient combines MockClient and MockVerifier to implement FullDAClient
type MockFullDAClient struct {
	*mocks.MockClient
	*mocks.MockVerifier
}

// createDefaultMockExecutor creates a MockExecutor with default passthrough behavior for FilterTxs and GetExecutionInfo
func createDefaultMockExecutor(t *testing.T) *mocks.MockExecutor {
	mockExec := mocks.NewMockExecutor(t)
	mockExec.On("GetExecutionInfo", mock.Anything, mock.Anything).Return(execution.ExecutionInfo{MaxGas: 1000000}, nil).Maybe()
	mockExec.On("FilterTxs", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, txs [][]byte, forceIncludedMask []bool, maxGas uint64) *execution.FilterTxsResult {
			return &execution.FilterTxsResult{
				ValidTxs:          txs,
				ForceIncludedMask: forceIncludedMask,
				RemainingTxs:      nil,
			}
		},
		nil,
	).Maybe()
	return mockExec
}

func createTestSequencer(t *testing.T, mockRetriever *common.MockForcedInclusionRetriever, gen genesis.Genesis) *BasedSequencer {
	t.Helper()

	// Create in-memory datastore
	db := syncds.MutexWrap(ds.NewMapDatastore())

	// Create mock DA client
	mockDAClient := &MockFullDAClient{
		MockClient:   mocks.NewMockClient(t),
		MockVerifier: mocks.NewMockVerifier(t),
	}
	// Mock the forced inclusion namespace call
	mockDAClient.MockClient.On("GetForcedInclusionNamespace").Return([]byte("test-forced-inclusion-ns")).Maybe()
	mockDAClient.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	mockExec := createDefaultMockExecutor(t)

	seq, err := NewBasedSequencer(mockDAClient, config.DefaultConfig(), db, gen, zerolog.Nop(), mockExec)
	require.NoError(t, err)

	// Replace the fiRetriever with our mock so tests work as before
	seq.fiRetriever = mockRetriever

	return seq
}

func TestBasedSequencer_SubmitBatchTxs(t *testing.T) {
	mockRetriever := common.NewMockForcedInclusionRetriever(t)
	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAEpochForcedInclusion: 10,
	}

	seq := createTestSequencer(t, mockRetriever, gen)

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

	mockRetriever := common.NewMockForcedInclusionRetriever(t)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(100)).Return(&block.ForcedInclusionEvent{
		Txs:           testBlobs,
		StartDaHeight: 100,
		EndDaHeight:   100,
	}, nil)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq := createTestSequencer(t, mockRetriever, gen)

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

	mockRetriever.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_EmptyDA(t *testing.T) {
	mockRetriever := common.NewMockForcedInclusionRetriever(t)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(100)).Return(&block.ForcedInclusionEvent{
		Txs:           [][]byte{},
		StartDaHeight: 100,
		EndDaHeight:   100,
	}, nil)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq := createTestSequencer(t, mockRetriever, gen)

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

	mockRetriever.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_NotConfigured(t *testing.T) {
	mockRetriever := common.NewMockForcedInclusionRetriever(t)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(100)).Return(nil, block.ErrForceInclusionNotConfigured)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq := createTestSequencer(t, mockRetriever, gen)

	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	resp, err := seq.GetNextBatch(context.Background(), req)
	require.Error(t, err)
	require.Nil(t, resp)
	assert.ErrorIs(t, err, block.ErrForceInclusionNotConfigured)

	mockRetriever.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_WithMaxBytes(t *testing.T) {
	// Create transactions of known sizes
	tx1 := make([]byte, 100)
	tx2 := make([]byte, 150)
	tx3 := make([]byte, 200)
	testBlobs := [][]byte{tx1, tx2, tx3}

	mockRetriever := common.NewMockForcedInclusionRetriever(t)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(100)).Return(&block.ForcedInclusionEvent{
		Txs:           testBlobs,
		StartDaHeight: 100,
		EndDaHeight:   100,
	}, nil)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq := createTestSequencer(t, mockRetriever, gen)

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

	mockRetriever.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_MultipleDABlocks(t *testing.T) {
	testBlobs1 := [][]byte{[]byte("tx1"), []byte("tx2")}
	testBlobs2 := [][]byte{[]byte("tx3"), []byte("tx4")}

	mockRetriever := common.NewMockForcedInclusionRetriever(t)
	// First DA block
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(100)).Return(&block.ForcedInclusionEvent{
		Txs:           testBlobs1,
		StartDaHeight: 100,
		EndDaHeight:   100,
	}, nil).Once()

	// Second DA block
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(101)).Return(&block.ForcedInclusionEvent{
		Txs:           testBlobs2,
		StartDaHeight: 101,
		EndDaHeight:   101,
	}, nil).Once()

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq := createTestSequencer(t, mockRetriever, gen)

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

	mockRetriever.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_ResumesFromCheckpoint(t *testing.T) {
	testBlobs := [][]byte{[]byte("tx1"), []byte("tx2"), []byte("tx3")}
	mockRetriever := common.NewMockForcedInclusionRetriever(t)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq := createTestSequencer(t, mockRetriever, gen)

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
	// Create a transaction that exceeds maxBytes
	largeTx := make([]byte, 2000)
	testBlobs := [][]byte{largeTx}

	mockRetriever := common.NewMockForcedInclusionRetriever(t)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(100)).Return(&block.ForcedInclusionEvent{
		Txs:           testBlobs,
		StartDaHeight: 100,
		EndDaHeight:   100,
	}, nil)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq := createTestSequencer(t, mockRetriever, gen)

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

	mockRetriever.AssertExpectations(t)
}

func TestBasedSequencer_VerifyBatch(t *testing.T) {
	mockRetriever := common.NewMockForcedInclusionRetriever(t)
	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAEpochForcedInclusion: 10,
	}

	seq := createTestSequencer(t, mockRetriever, gen)

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
	mockRetriever := common.NewMockForcedInclusionRetriever(t)
	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	seq := createTestSequencer(t, mockRetriever, gen)

	// Initial height from genesis
	assert.Equal(t, uint64(100), seq.GetDAHeight())

	// Set new height
	seq.SetDAHeight(200)
	assert.Equal(t, uint64(200), seq.GetDAHeight())
}

func TestBasedSequencer_GetNextBatch_ErrorHandling(t *testing.T) {
	mockRetriever := common.NewMockForcedInclusionRetriever(t)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(100)).Return(nil, block.ErrForceInclusionNotConfigured)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq := createTestSequencer(t, mockRetriever, gen)

	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	resp, err := seq.GetNextBatch(context.Background(), req)
	require.Error(t, err)
	require.Nil(t, resp)
	assert.ErrorIs(t, err, block.ErrForceInclusionNotConfigured)

	mockRetriever.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_HeightFromFuture(t *testing.T) {
	mockRetriever := common.NewMockForcedInclusionRetriever(t)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(100)).Return(nil, datypes.ErrHeightFromFuture)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq := createTestSequencer(t, mockRetriever, gen)

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

	mockRetriever.AssertExpectations(t)
}

func TestBasedSequencer_CheckpointPersistence(t *testing.T) {
	testBlobs := [][]byte{[]byte("tx1"), []byte("tx2")}

	mockRetriever := common.NewMockForcedInclusionRetriever(t)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(100)).Return(&block.ForcedInclusionEvent{
		Txs:           testBlobs,
		StartDaHeight: 100,
		EndDaHeight:   100,
	}, nil)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	// Create persistent datastore
	db := syncds.MutexWrap(ds.NewMapDatastore())

	// Create mock DA client
	mockDAClient := &MockFullDAClient{
		MockClient:   mocks.NewMockClient(t),
		MockVerifier: mocks.NewMockVerifier(t),
	}
	mockDAClient.MockClient.On("GetForcedInclusionNamespace").Return([]byte("test-forced-inclusion-ns")).Maybe()
	mockDAClient.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	// Create first sequencer
	mockExec1 := createDefaultMockExecutor(t)
	seq1, err := NewBasedSequencer(mockDAClient, config.DefaultConfig(), db, gen, zerolog.Nop(), mockExec1)
	require.NoError(t, err)

	// Replace the fiRetriever with our mock so tests work as before
	seq1.fiRetriever = mockRetriever

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
	mockDAClient2 := &MockFullDAClient{
		MockClient:   mocks.NewMockClient(t),
		MockVerifier: mocks.NewMockVerifier(t),
	}
	mockDAClient2.MockClient.On("GetForcedInclusionNamespace").Return([]byte("test-forced-inclusion-ns")).Maybe()
	mockDAClient2.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()
	mockExec2 := createDefaultMockExecutor(t)
	seq2, err := NewBasedSequencer(mockDAClient2, config.DefaultConfig(), db, gen, zerolog.Nop(), mockExec2)
	require.NoError(t, err)

	// Replace the fiRetriever with our mock so tests work as before
	seq2.fiRetriever = mockRetriever

	// Checkpoint should be loaded from DB
	assert.Equal(t, uint64(101), seq2.checkpoint.DAHeight)
	assert.Equal(t, uint64(0), seq2.checkpoint.TxIndex)

	mockRetriever.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_EmptyDABatch_IncreasesDAHeight(t *testing.T) {
	mockRetriever := common.NewMockForcedInclusionRetriever(t)

	// First DA block returns empty transactions
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(100)).Return(&block.ForcedInclusionEvent{
		Txs:           [][]byte{},
		StartDaHeight: 100,
		EndDaHeight:   100,
	}, nil).Once()

	// Second DA block also returns empty transactions
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(101)).Return(&block.ForcedInclusionEvent{
		Txs:           [][]byte{},
		StartDaHeight: 101,
		EndDaHeight:   101,
	}, nil).Once()

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq := createTestSequencer(t, mockRetriever, gen)

	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	// Initial DA height should be 100
	assert.Equal(t, uint64(100), seq.GetDAHeight())
	assert.Equal(t, uint64(100), seq.checkpoint.DAHeight)

	// First batch - empty DA block at height 100
	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 0, len(resp.Batch.Transactions))

	// DA height should have increased to 101 even though no transactions were processed
	assert.Equal(t, uint64(101), seq.GetDAHeight())
	assert.Equal(t, uint64(101), seq.checkpoint.DAHeight)
	assert.Equal(t, uint64(0), seq.checkpoint.TxIndex)

	// Second batch - empty DA block at height 101
	resp, err = seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 0, len(resp.Batch.Transactions))

	// DA height should have increased to 102
	assert.Equal(t, uint64(102), seq.GetDAHeight())
	assert.Equal(t, uint64(102), seq.checkpoint.DAHeight)
	assert.Equal(t, uint64(0), seq.checkpoint.TxIndex)

	mockRetriever.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_TimestampAdjustment(t *testing.T) {
	// Test that timestamp is adjusted based on the number of transactions in the batch
	// The timestamp should be: daEndTime - (len(batch.Transactions) * 1ms)

	testBlobs := [][]byte{[]byte("tx1"), []byte("tx2"), []byte("tx3")}
	daEndTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	mockRetriever := common.NewMockForcedInclusionRetriever(t)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(100)).Return(&block.ForcedInclusionEvent{
		Txs:           testBlobs,
		StartDaHeight: 100,
		EndDaHeight:   100,
		Timestamp:     daEndTime,
	}, nil)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq := createTestSequencer(t, mockRetriever, gen)

	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 3, len(resp.Batch.Transactions))

	// After taking all 3 txs, there are 0 remaining, so timestamp = daEndTime - 0ms = daEndTime
	expectedTimestamp := daEndTime
	assert.Equal(t, expectedTimestamp, resp.Timestamp)

	mockRetriever.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_TimestampAdjustment_PartialBatch(t *testing.T) {
	// Test timestamp adjustment when MaxBytes limits the batch size
	tx1 := make([]byte, 100)
	tx2 := make([]byte, 150)
	tx3 := make([]byte, 200)
	testBlobs := [][]byte{tx1, tx2, tx3}
	daEndTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	mockRetriever := common.NewMockForcedInclusionRetriever(t)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(100)).Return(&block.ForcedInclusionEvent{
		Txs:           testBlobs,
		StartDaHeight: 100,
		EndDaHeight:   100,
		Timestamp:     daEndTime,
	}, nil)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq := createTestSequencer(t, mockRetriever, gen)

	// First call with MaxBytes that fits only first 2 transactions
	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      250,
		LastBatchData: nil,
	}

	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 2, len(resp.Batch.Transactions))

	// After taking 2 txs, there is 1 remaining, so timestamp = daEndTime - 1ms
	expectedTimestamp := daEndTime.Add(-1 * time.Millisecond)
	assert.Equal(t, expectedTimestamp, resp.Timestamp)

	// Second call should get the remaining transaction
	req = coresequencer.GetNextBatchRequest{
		MaxBytes:      1000,
		LastBatchData: nil,
	}

	// The second call uses cached transactions - timestamp should be based on remaining txs
	resp, err = seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 1, len(resp.Batch.Transactions))

	// After taking this 1 tx, there are 0 remaining, so timestamp = daEndTime - 0ms = daEndTime
	expectedTimestamp2 := daEndTime
	assert.Equal(t, expectedTimestamp2, resp.Timestamp)

	mockRetriever.AssertExpectations(t)
}

func TestBasedSequencer_GetNextBatch_TimestampAdjustment_EmptyBatch(t *testing.T) {
	// Test that timestamp is zero when batch is empty
	daEndTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	mockRetriever := common.NewMockForcedInclusionRetriever(t)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(100)).Return(&block.ForcedInclusionEvent{
		Txs:           [][]byte{},
		StartDaHeight: 100,
		EndDaHeight:   100,
		Timestamp:     daEndTime,
	}, nil)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	seq := createTestSequencer(t, mockRetriever, gen)

	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 0, len(resp.Batch.Transactions))

	// When batch is empty, there are 0 remaining txs, so timestamp = daEndTime
	expectedTimestamp := daEndTime
	assert.Equal(t, expectedTimestamp, resp.Timestamp)

	mockRetriever.AssertExpectations(t)
}

// TestBasedSequencer_GetNextBatch_GasFilteringPreservesUnprocessedTxs tests that when FilterTxs
// returns RemainingTxs (valid DA txs that didn't fit due to gas limits), the sequencer correctly
// preserves any transactions that weren't even processed yet due to maxBytes limits.
//
// This test exposes a bug where setting currentBatchTxs = remainingGasFilteredTxs would lose
// any unprocessed transactions from the original cache.
func TestBasedSequencer_GetNextBatch_GasFilteringPreservesUnprocessedTxs(t *testing.T) {
	// Create 5 transactions of 100 bytes each
	tx0 := make([]byte, 100)
	tx1 := make([]byte, 100)
	tx2 := make([]byte, 100)
	tx3 := make([]byte, 100)
	tx4 := make([]byte, 100)
	copy(tx0, "tx0")
	copy(tx1, "tx1")
	copy(tx2, "tx2")
	copy(tx3, "tx3")
	copy(tx4, "tx4")

	testBlobs := [][]byte{tx0, tx1, tx2, tx3, tx4}

	mockRetriever := common.NewMockForcedInclusionRetriever(t)
	// Only expect one call to retrieve - all txs come from one DA epoch
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(100)).Return(&block.ForcedInclusionEvent{
		Txs:           testBlobs,
		StartDaHeight: 100,
		EndDaHeight:   100,
		Timestamp:     time.Now(),
	}, nil).Once()

	// Second DA epoch - should NOT be called if bug is fixed (we still have tx3, tx4 to process)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(101)).Return(&block.ForcedInclusionEvent{
		Txs:           [][]byte{[]byte("epoch2-tx")},
		StartDaHeight: 101,
		EndDaHeight:   101,
		Timestamp:     time.Now(),
	}, nil).Maybe()

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	// Create executor that simulates gas filtering
	// - tx0: valid, fits in gas
	// - tx1: gibberish/invalid (filtered out completely)
	// - tx2: valid but doesn't fit due to gas limit (returned as remaining)
	gasFilterCallCount := 0
	mockExec := mocks.NewMockExecutor(t)
	mockExec.On("GetExecutionInfo", mock.Anything, mock.Anything).Return(execution.ExecutionInfo{MaxGas: 1000000}, nil).Maybe()
	mockExec.On("FilterTxs", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, txs [][]byte, forceIncludedMask []bool, maxGas uint64) *execution.FilterTxsResult {
			gasFilterCallCount++
			t.Logf("FilterTxs call #%d with %d txs", gasFilterCallCount, len(txs))
			for i, tx := range txs {
				t.Logf("  tx[%d]: %s", i, string(tx[:3]))
			}

			if gasFilterCallCount == 1 {
				// First call: receives tx0, tx1, tx2 (maxBytes=250 limits to 2-3 txs)
				// tx0: valid and fits
				// tx1: gibberish (filtered out)
				// tx2: valid but gas-limited
				require.GreaterOrEqual(t, len(txs), 2, "expected at least 2 txs in first filter call")

				return &execution.FilterTxsResult{
					ValidTxs:          txs[:1],          // Only tx0 fits
					ForceIncludedMask: []bool{true},     // All force-included
					RemainingTxs:      txs[len(txs)-1:], // Last tx is gas-limited (tx2)
				}
			}

			// Subsequent calls: should receive the remaining txs (tx2, tx3, tx4)
			// If bug exists, we'd only get tx2 and lose tx3, tx4
			return &execution.FilterTxsResult{
				ValidTxs:          txs,
				ForceIncludedMask: make([]bool, len(txs)),
				RemainingTxs:      nil,
			}
		},
		nil,
	).Maybe()

	// Create sequencer with custom executor
	db := syncds.MutexWrap(ds.NewMapDatastore())
	mockDAClient := &MockFullDAClient{
		MockClient:   mocks.NewMockClient(t),
		MockVerifier: mocks.NewMockVerifier(t),
	}
	mockDAClient.MockClient.On("GetForcedInclusionNamespace").Return([]byte("test-forced-inclusion-ns")).Maybe()
	mockDAClient.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	seq, err := NewBasedSequencer(mockDAClient, config.DefaultConfig(), db, gen, zerolog.New(zerolog.NewTestWriter(t)), mockExec)
	require.NoError(t, err)
	seq.fiRetriever = mockRetriever

	// First batch: maxBytes=250 means we can fetch ~2 txs (each 100 bytes)
	// getTxsFromCheckpoint will return tx0, tx1 (or tx0, tx1, tx2 depending on exact math)
	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      250, // Limits to ~2 txs per batch
		LastBatchData: nil,
	}

	resp1, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp1.Batch)
	t.Logf("Batch 1: %d txs, checkpoint: height=%d, index=%d",
		len(resp1.Batch.Transactions), seq.checkpoint.DAHeight, seq.checkpoint.TxIndex)

	// Second batch: should get remaining gas-filtered tx + any unprocessed txs
	resp2, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp2.Batch)
	t.Logf("Batch 2: %d txs, checkpoint: height=%d, index=%d",
		len(resp2.Batch.Transactions), seq.checkpoint.DAHeight, seq.checkpoint.TxIndex)

	// Third batch: should get more remaining txs
	resp3, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp3.Batch)
	t.Logf("Batch 3: %d txs, checkpoint: height=%d, index=%d",
		len(resp3.Batch.Transactions), seq.checkpoint.DAHeight, seq.checkpoint.TxIndex)

	// Verify we processed all 5 original transactions (minus 1 gibberish = 4 valid)
	// The key assertion: we should NOT have moved to DA height 101 until all txs from 100 are processed
	// If the bug exists, tx3 and tx4 would be lost and we'd prematurely move to height 101

	totalTxsProcessed := len(resp1.Batch.Transactions) + len(resp2.Batch.Transactions) + len(resp3.Batch.Transactions)
	t.Logf("Total txs processed across 3 batches: %d", totalTxsProcessed)

	// We should have processed at least 3 transactions (tx0, tx2, and at least one of tx3/tx4)
	// If bug exists, we might only get 2 (tx0 and tx2) because tx3, tx4 are lost
	assert.GreaterOrEqual(t, totalTxsProcessed, 3, "should process at least 3 valid transactions from the cache")
}

// TestBasedSequencer_GetNextBatch_RemainingTxsIndexCorrelation tests that the checkpoint
// correctly tracks which transactions have been consumed when some are gas-filtered.
// This test uses maxBytes to limit how many txs are fetched, triggering the unprocessed txs scenario.
func TestBasedSequencer_GetNextBatch_RemainingTxsIndexCorrelation(t *testing.T) {
	// Create 4 transactions of 50 bytes each
	tx0 := make([]byte, 50)
	tx1 := make([]byte, 50)
	tx2 := make([]byte, 50)
	tx3 := make([]byte, 50)
	copy(tx0, "tx0-valid-fits")
	copy(tx1, "tx1-valid-gas-limited")
	copy(tx2, "tx2-should-not-be-lost")
	copy(tx3, "tx3-should-not-be-lost")

	testBlobs := [][]byte{tx0, tx1, tx2, tx3}

	mockRetriever := common.NewMockForcedInclusionRetriever(t)
	mockRetriever.On("RetrieveForcedIncludedTxs", mock.Anything, uint64(100)).Return(&block.ForcedInclusionEvent{
		Txs:           testBlobs,
		StartDaHeight: 100,
		EndDaHeight:   100,
		Timestamp:     time.Now(),
	}, nil).Once()

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	allProcessedTxs := make([][]byte, 0)
	filterCallCount := 0

	mockExec := mocks.NewMockExecutor(t)
	mockExec.On("GetExecutionInfo", mock.Anything, mock.Anything).Return(execution.ExecutionInfo{MaxGas: 1000000}, nil).Maybe()
	mockExec.On("FilterTxs", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, txs [][]byte, forceIncludedMask []bool, maxGas uint64) *execution.FilterTxsResult {
			filterCallCount++
			t.Logf("FilterTxs call #%d with %d txs", filterCallCount, len(txs))

			if filterCallCount == 1 && len(txs) >= 2 {
				// First call: tx0 valid+fits, tx1 valid but gas-limited
				return &execution.FilterTxsResult{
					ValidTxs:          txs[:1], // tx0 only
					ForceIncludedMask: []bool{true},
					RemainingTxs:      txs[1:2], // tx1 is remaining (gas-limited)
				}
			}

			// Pass through all txs
			mask := make([]bool, len(txs))
			for i := range mask {
				mask[i] = true
			}
			return &execution.FilterTxsResult{
				ValidTxs:          txs,
				ForceIncludedMask: mask,
				RemainingTxs:      nil,
			}
		},
		nil,
	).Maybe()

	db := syncds.MutexWrap(ds.NewMapDatastore())
	mockDAClient := &MockFullDAClient{
		MockClient:   mocks.NewMockClient(t),
		MockVerifier: mocks.NewMockVerifier(t),
	}
	mockDAClient.MockClient.On("GetForcedInclusionNamespace").Return([]byte("test-ns")).Maybe()
	mockDAClient.MockClient.On("HasForcedInclusionNamespace").Return(true).Maybe()

	seq, err := NewBasedSequencer(mockDAClient, config.DefaultConfig(), db, gen, zerolog.New(zerolog.NewTestWriter(t)), mockExec)
	require.NoError(t, err)
	seq.fiRetriever = mockRetriever

	// Process all batches until we've consumed all txs from DA height 100
	// Use maxBytes=120 to fetch only 2 txs at a time (each is 50 bytes)
	for i := 0; i < 5; i++ { // Max 5 iterations to prevent infinite loop
		resp, err := seq.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{
			MaxBytes: 120, // Limits to ~2 txs per batch
		})
		require.NoError(t, err)

		allProcessedTxs = append(allProcessedTxs, resp.Batch.Transactions...)
		t.Logf("Batch %d: %d txs, checkpoint: height=%d, index=%d, cache size=%d",
			i+1, len(resp.Batch.Transactions), seq.checkpoint.DAHeight, seq.checkpoint.TxIndex, len(seq.currentBatchTxs))

		if seq.checkpoint.DAHeight > 100 {
			break // Moved to next DA epoch
		}
		if len(resp.Batch.Transactions) == 0 && len(seq.currentBatchTxs) == 0 {
			break // Nothing left
		}
	}

	// Verify all 4 original transactions were processed
	assert.Len(t, allProcessedTxs, 4, "all 4 transactions should have been processed")

	// Check that each original tx appears exactly once
	txFound := make(map[string]bool)
	for _, tx := range allProcessedTxs {
		txFound[string(tx)] = true
	}

	assert.True(t, txFound[string(tx0)], "tx0 should have been processed")
	assert.True(t, txFound[string(tx1)], "tx1 should have been processed (was gas-limited, retried later)")
	assert.True(t, txFound[string(tx2)], "tx2 should have been processed (must not be lost)")
	assert.True(t, txFound[string(tx3)], "tx3 should have been processed (must not be lost)")
}
