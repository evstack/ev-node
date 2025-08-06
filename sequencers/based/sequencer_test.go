package based_test

import (
	"context"
	"testing"
	"time"

	coreda "github.com/evstack/ev-node/core/da"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/sequencers/based"
	"github.com/stretchr/testify/assert"
)

func getTestDABlockTime() time.Duration {
	return 100 * time.Millisecond
}

func newTestSequencer(t *testing.T) *based.Sequencer {
	dummyDA := coreda.NewDummyDA(100_000_000, 1.0, 1.5, getTestDABlockTime())
	dummyDA.StartHeightTicker()
	store := ds.NewMapDatastore()
	logger := zerolog.Nop()
	seq, err := based.NewSequencer(logger, dummyDA, []byte("test1"), 0, 2, store, []byte("test-namespace"))
	assert.NoError(t, err)
	return seq
}

func TestSequencer_SubmitBatchTxs_Valid(t *testing.T) {
	sequencer := newTestSequencer(t)

	batch := &coresequencer.Batch{Transactions: [][]byte{[]byte("tx1"), []byte("tx2")}}
	resp, err := sequencer.SubmitBatchTxs(context.Background(), coresequencer.SubmitBatchTxsRequest{
		Id:    []byte("test1"),
		Batch: batch,
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestSequencer_SubmitBatchTxs_Invalid(t *testing.T) {
	sequencer := newTestSequencer(t)

	batch := &coresequencer.Batch{Transactions: [][]byte{[]byte("tx1")}}
	resp, err := sequencer.SubmitBatchTxs(context.Background(), coresequencer.SubmitBatchTxsRequest{
		Id:    []byte("invalid"),
		Batch: batch,
	})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestSequencer_GetNextBatch_OnlyPendingQueue(t *testing.T) {
	sequencer := newTestSequencer(t)

	timestamp := time.Now()
	sequencer.AddToPendingTxs([][]byte{[]byte("tx1")}, [][]byte{[]byte("id1")}, timestamp)

	resp, err := sequencer.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{Id: []byte("test1")})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resp.Batch.Transactions))
	assert.Equal(t, timestamp.Unix(), resp.Timestamp.Unix())
}

func TestSequencer_GetNextBatch_FromDALayer(t *testing.T) {
	sequencer := newTestSequencer(t)
	ctx := context.Background()

	blobs := []coreda.Blob{[]byte("tx2"), []byte("tx3")}
	_, err := sequencer.DA.Submit(ctx, blobs, 1.0, []byte("ns"))
	assert.NoError(t, err)
	time.Sleep(getTestDABlockTime())

	resp, err := sequencer.GetNextBatch(ctx, coresequencer.GetNextBatchRequest{
		Id: []byte("test1"),
	})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(resp.Batch.Transactions), 1)
	assert.GreaterOrEqual(t, len(resp.BatchData), 1)
}

func TestSequencer_GetNextBatch_Invalid(t *testing.T) {
	sequencer := newTestSequencer(t)

	resp, err := sequencer.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{
		Id: []byte("invalid"),
	})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestSequencer_GetNextBatch_ExceedsMaxDrift(t *testing.T) {
	dummyDA := coreda.NewDummyDA(100_000_000, 1.0, 1.5, 10*time.Second)
	store := ds.NewMapDatastore()
	logger := zerolog.Nop()
	sequencer, err := based.NewSequencer(logger, dummyDA, []byte("test1"), 0, 0, store, []byte("test-namespace"))
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = dummyDA.Submit(ctx, []coreda.Blob{[]byte("tx4")}, 1.0, []byte("ns"))
	assert.NoError(t, err)
	time.Sleep(getTestDABlockTime())

	resp, err := sequencer.GetNextBatch(ctx, coresequencer.GetNextBatchRequest{
		Id: []byte("test1"),
	})
	assert.NoError(t, err)
	if resp != nil {
		assert.LessOrEqual(t, len(resp.Batch.Transactions), 1)
	}
}

func TestSequencer_VerifyBatch_Success(t *testing.T) {
	sequencer := newTestSequencer(t)

	ctx := context.Background()
	ids, err := sequencer.DA.Submit(ctx, []coreda.Blob{[]byte("tx1")}, 1.0, []byte("ns"))
	assert.NoError(t, err)
	time.Sleep(getTestDABlockTime())

	resp, err := sequencer.VerifyBatch(ctx, coresequencer.VerifyBatchRequest{
		Id:        []byte("test1"),
		BatchData: ids,
	})
	assert.NoError(t, err)
	assert.True(t, resp.Status)
}

func TestSequencer_VerifyBatch_Invalid(t *testing.T) {
	sequencer := newTestSequencer(t)

	ctx := context.Background()
	resp, err := sequencer.VerifyBatch(ctx, coresequencer.VerifyBatchRequest{
		Id:        []byte("invalid"),
		BatchData: [][]byte{[]byte("someID")},
	})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestSequencer_VerifyBatch_InvalidProof(t *testing.T) {
	sequencer := newTestSequencer(t)

	ctx := context.Background()
	resp, err := sequencer.VerifyBatch(ctx, coresequencer.VerifyBatchRequest{
		Id:        []byte("test1"),
		BatchData: [][]byte{[]byte("invalid")},
	})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestSequencer_NamespaceUsage(t *testing.T) {
	// Create sequencer with custom namespace
	customNamespace := []byte("custom-data-namespace")
	dummyDA := coreda.NewDummyDA(100_000_000, 1.0, 1.5, getTestDABlockTime())
	dummyDA.StartHeightTicker()
	store := ds.NewMapDatastore()
	logger := zerolog.Nop()
	sequencer, err := based.NewSequencer(logger, dummyDA, []byte("test1"), 0, 2, store, customNamespace)
	assert.NoError(t, err)

	ctx := context.Background()
	
	// Test that submit uses the correct namespace
	batch := &coresequencer.Batch{Transactions: [][]byte{[]byte("tx1")}}
	_, err = sequencer.SubmitBatchTxs(ctx, coresequencer.SubmitBatchTxsRequest{
		Id:    []byte("test1"),
		Batch: batch,
	})
	assert.NoError(t, err)

	// Submit some data using the custom namespace to verify retrieval works with that namespace
	_, err = dummyDA.Submit(ctx, []coreda.Blob{[]byte("namespace-tx")}, 1.0, customNamespace)
	assert.NoError(t, err)
	time.Sleep(getTestDABlockTime())

	// Test that get retrieves from the correct namespace  
	resp, err := sequencer.GetNextBatch(ctx, coresequencer.GetNextBatchRequest{
		Id: []byte("test1"),
	})
	assert.NoError(t, err)
	
	// Should retrieve the data submitted with the custom namespace
	if resp != nil {
		assert.GreaterOrEqual(t, len(resp.Batch.Transactions), 1)
		assert.GreaterOrEqual(t, len(resp.BatchData), 1)
	}
}

func TestSequencer_NamespaceIsolation(t *testing.T) {
	// Create sequencer with one namespace
	namespace1 := []byte("namespace-1")
	dummyDA := coreda.NewDummyDA(100_000_000, 1.0, 1.5, getTestDABlockTime())
	dummyDA.StartHeightTicker()
	store := ds.NewMapDatastore()
	logger := zerolog.Nop()
	sequencer, err := based.NewSequencer(logger, dummyDA, []byte("test1"), 0, 2, store, namespace1)
	assert.NoError(t, err)

	ctx := context.Background()
	
	// Submit data to a different namespace
	namespace2 := []byte("namespace-2")
	_, err = dummyDA.Submit(ctx, []coreda.Blob{[]byte("wrong-namespace-tx")}, 1.0, namespace2)
	assert.NoError(t, err)
	time.Sleep(getTestDABlockTime())

	// The sequencer should not retrieve data from the wrong namespace
	resp, err := sequencer.GetNextBatch(ctx, coresequencer.GetNextBatchRequest{
		Id: []byte("test1"),
	})
	assert.NoError(t, err)
	
	// Should not retrieve any transactions since they were in a different namespace
	if resp != nil {
		assert.Equal(t, 0, len(resp.Batch.Transactions), "Should not retrieve transactions from different namespace")
	}
}
