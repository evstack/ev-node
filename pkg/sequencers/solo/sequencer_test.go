package solo

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/test/mocks"
)

func createDefaultMockExecutor(t *testing.T) *mocks.MockExecutor {
	mockExec := mocks.NewMockExecutor(t)
	mockExec.On("GetExecutionInfo", mock.Anything).Return(execution.ExecutionInfo{MaxGas: 1000000}, nil).Maybe()
	mockExec.On("FilterTxs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, txs [][]byte, maxBytes, maxGas uint64, hasForceIncludedTransaction bool) []execution.FilterStatus {
			result := make([]execution.FilterStatus, len(txs))
			var cumulativeBytes uint64
			for i, tx := range txs {
				txBytes := uint64(len(tx))
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

func newTestSequencer(t *testing.T) *SoloSequencer {
	return NewSoloSequencer(
		zerolog.Nop(),
		[]byte("test"),
		createDefaultMockExecutor(t),
	)
}

func TestSoloSequencer_SubmitBatchTxs(t *testing.T) {
	seq := newTestSequencer(t)

	tx := []byte("transaction1")
	res, err := seq.SubmitBatchTxs(context.Background(), coresequencer.SubmitBatchTxsRequest{
		Id:    []byte("test"),
		Batch: &coresequencer.Batch{Transactions: [][]byte{tx}},
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	nextResp, err := seq.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{Id: []byte("test")})
	require.NoError(t, err)
	require.Len(t, nextResp.Batch.Transactions, 1)
	assert.Equal(t, tx, nextResp.Batch.Transactions[0])
}

func TestSoloSequencer_SubmitBatchTxs_InvalidID(t *testing.T) {
	seq := newTestSequencer(t)

	res, err := seq.SubmitBatchTxs(context.Background(), coresequencer.SubmitBatchTxsRequest{
		Id:    []byte("wrong"),
		Batch: &coresequencer.Batch{Transactions: [][]byte{{1}}},
	})
	assert.ErrorIs(t, err, ErrInvalidID)
	assert.Nil(t, res)
}

func TestSoloSequencer_SubmitBatchTxs_EmptyBatch(t *testing.T) {
	seq := newTestSequencer(t)

	res, err := seq.SubmitBatchTxs(context.Background(), coresequencer.SubmitBatchTxsRequest{
		Id:    []byte("test"),
		Batch: &coresequencer.Batch{},
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Empty(t, seq.queue)

	res, err = seq.SubmitBatchTxs(context.Background(), coresequencer.SubmitBatchTxsRequest{
		Id:    []byte("test"),
		Batch: nil,
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Empty(t, seq.queue)
}

func TestSoloSequencer_GetNextBatch_EmptyQueue(t *testing.T) {
	seq := newTestSequencer(t)

	resp, err := seq.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{Id: []byte("test")})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Batch.Transactions)
	assert.WithinDuration(t, time.Now(), resp.Timestamp, time.Second)
}

func TestSoloSequencer_GetNextBatch_InvalidID(t *testing.T) {
	seq := newTestSequencer(t)

	res, err := seq.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{Id: []byte("wrong")})
	assert.ErrorIs(t, err, ErrInvalidID)
	assert.Nil(t, res)
}

func TestSoloSequencer_GetNextBatch_DrainsAndFilters(t *testing.T) {
	seq := newTestSequencer(t)

	batch := coresequencer.Batch{Transactions: [][]byte{{1}, {2}, {3}}}
	_, err := seq.SubmitBatchTxs(context.Background(), coresequencer.SubmitBatchTxsRequest{
		Id:    []byte("test"),
		Batch: &batch,
	})
	require.NoError(t, err)

	resp, err := seq.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{Id: []byte("test")})
	require.NoError(t, err)
	assert.Len(t, resp.Batch.Transactions, 3)

	assert.Empty(t, seq.queue, "queue should be drained after GetNextBatch")
}

func TestSoloSequencer_GetNextBatch_PostponedTxsRequeued(t *testing.T) {
	mockExec := mocks.NewMockExecutor(t)
	mockExec.On("GetExecutionInfo", mock.Anything).Return(execution.ExecutionInfo{MaxGas: 1000000}, nil).Maybe()
	mockExec.On("FilterTxs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, txs [][]byte, maxBytes, maxGas uint64, hasForceIncludedTransaction bool) []execution.FilterStatus {
			result := make([]execution.FilterStatus, len(txs))
			for i := range txs {
				if i < 2 {
					result[i] = execution.FilterOK
				} else {
					result[i] = execution.FilterPostpone
				}
			}
			return result
		},
		nil,
	).Maybe()

	seq := NewSoloSequencer(
		zerolog.Nop(),
		[]byte("test"),
		mockExec,
	)

	batch := coresequencer.Batch{Transactions: [][]byte{{1}, {2}, {3}, {4}}}
	_, err := seq.SubmitBatchTxs(context.Background(), coresequencer.SubmitBatchTxsRequest{
		Id:    []byte("test"),
		Batch: &batch,
	})
	require.NoError(t, err)

	resp, err := seq.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{Id: []byte("test")})
	require.NoError(t, err)
	assert.Len(t, resp.Batch.Transactions, 2, "first 2 txs should pass filter")

	assert.Len(t, seq.queue, 2, "postponed txs should be re-queued")
	assert.Equal(t, []byte{3}, seq.queue[0])
	assert.Equal(t, []byte{4}, seq.queue[1])
}

func TestSoloSequencer_GetNextBatch_SubmitDuringProcessing(t *testing.T) {
	seq := newTestSequencer(t)

	batch := coresequencer.Batch{Transactions: [][]byte{{1}, {2}}}
	_, err := seq.SubmitBatchTxs(context.Background(), coresequencer.SubmitBatchTxsRequest{
		Id:    []byte("test"),
		Batch: &batch,
	})
	require.NoError(t, err)

	resp, err := seq.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{Id: []byte("test")})
	require.NoError(t, err)
	assert.Len(t, resp.Batch.Transactions, 2)

	_, err = seq.SubmitBatchTxs(context.Background(), coresequencer.SubmitBatchTxsRequest{
		Id:    []byte("test"),
		Batch: &coresequencer.Batch{Transactions: [][]byte{{3}}},
	})
	require.NoError(t, err)

	resp, err = seq.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{Id: []byte("test")})
	require.NoError(t, err)
	assert.Len(t, resp.Batch.Transactions, 1)
	assert.Equal(t, []byte{3}, resp.Batch.Transactions[0])
}

func TestSoloSequencer_VerifyBatch(t *testing.T) {
	seq := newTestSequencer(t)

	batchData := [][]byte{[]byte("batch1"), []byte("batch2")}

	res, err := seq.VerifyBatch(context.Background(), coresequencer.VerifyBatchRequest{
		Id:        []byte("test"),
		BatchData: batchData,
	})
	assert.NoError(t, err)
	assert.True(t, res.Status)
}

func TestSoloSequencer_DAHeight(t *testing.T) {
	seq := newTestSequencer(t)

	assert.Equal(t, uint64(0), seq.GetDAHeight())

	seq.SetDAHeight(42)
	assert.Equal(t, uint64(42), seq.GetDAHeight())
}
