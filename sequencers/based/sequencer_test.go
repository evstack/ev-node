package based

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
)

type mockDAClient struct {
	forced map[uint64]datypes.ResultRetrieve
	hasFI  bool
}

func (m mockDAClient) Submit(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) datypes.ResultSubmit {
	return datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess}}
}

func (m mockDAClient) Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	return datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Height: height}}
}

func (m mockDAClient) RetrieveHeaders(ctx context.Context, height uint64) datypes.ResultRetrieve {
	return datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Height: height}}
}

func (m mockDAClient) RetrieveData(ctx context.Context, height uint64) datypes.ResultRetrieve {
	return datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Height: height}}
}

func (m mockDAClient) RetrieveForcedInclusion(ctx context.Context, height uint64) datypes.ResultRetrieve {
	if res, ok := m.forced[height]; ok {
		return res
	}
	return datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Height: height}}
}

func (m mockDAClient) Get(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error) {
	return nil, nil
}

func (m mockDAClient) GetHeaderNamespace() []byte { return []byte("hdr") }
func (m mockDAClient) GetDataNamespace() []byte   { return []byte("data") }
func (m mockDAClient) GetForcedInclusionNamespace() []byte {
	if m.hasFI {
		return []byte("fi")
	}
	return nil
}
func (m mockDAClient) HasForcedInclusionNamespace() bool { return m.hasFI }

func TestBasedSequencer_SubmitBatchTxs(t *testing.T) {
	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAEpochForcedInclusion: 10,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	daClient := mockDAClient{forced: map[uint64]datypes.ResultRetrieve{}, hasFI: true}
	fiRetriever := block.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	seq := NewBasedSequencer(fiRetriever, cfg, gen, zerolog.Nop())

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
	// Transactions should not be added to queue for based sequencer
	assert.Equal(t, 0, len(seq.txQueue))
}

func TestBasedSequencer_GetNextBatch_WithForcedTxs(t *testing.T) {
	testBlobs := [][]byte{[]byte("tx1"), []byte("tx2")}

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	daClient := mockDAClient{
		forced: map[uint64]datypes.ResultRetrieve{
			100: {
				BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Height: 100},
				Data:       testBlobs,
			},
		},
		hasFI: true,
	}
	fiRetriever := block.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	seq := NewBasedSequencer(fiRetriever, cfg, gen, zerolog.Nop())

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

	// DA height should be updated to epochEnd + 1
	assert.Equal(t, uint64(101), seq.GetDAHeight())
}

func TestBasedSequencer_GetNextBatch_EmptyDA(t *testing.T) {
	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	daClient := mockDAClient{forced: map[uint64]datypes.ResultRetrieve{}, hasFI: true}
	fiRetriever := block.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	seq := NewBasedSequencer(fiRetriever, cfg, gen, zerolog.Nop())

	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 0, len(resp.Batch.Transactions))
}

func TestBasedSequencer_GetNextBatch_NotConfigured(t *testing.T) {
	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	// Create config without forced inclusion namespace
	cfgNoFI := config.DefaultConfig()
	cfgNoFI.DA.ForcedInclusionNamespace = ""
	daClient := mockDAClient{forced: map[uint64]datypes.ResultRetrieve{}, hasFI: false}
	fiRetriever := block.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	seq := NewBasedSequencer(fiRetriever, cfgNoFI, gen, zerolog.Nop())

	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	resp, err := seq.GetNextBatch(context.Background(), req)
	require.Error(t, err)
	require.Nil(t, resp)
}

func TestBasedSequencer_GetNextBatch_WithMaxBytes(t *testing.T) {
	testBlobs := [][]byte{
		make([]byte, 50),  // 50 bytes
		make([]byte, 60),  // 60 bytes
		make([]byte, 100), // 100 bytes
	}

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	daClient := mockDAClient{
		forced: map[uint64]datypes.ResultRetrieve{
			100: {
				BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Height: 100},
				Data:       testBlobs,
			},
		},
		hasFI: true,
	}
	fiRetriever := block.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	seq := NewBasedSequencer(fiRetriever, cfg, gen, zerolog.Nop())

	// First call with max 100 bytes - should get first tx (50 bytes), second would exceed limit
	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      100,
		LastBatchData: nil,
	}

	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 1, len(resp.Batch.Transactions))
	assert.Equal(t, 2, len(seq.txQueue)) // 2 remaining in queue

	// Second call should get next tx from queue
	resp2, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp2)
	require.NotNil(t, resp2.Batch)
	assert.Equal(t, 1, len(resp2.Batch.Transactions))
	assert.Equal(t, 1, len(seq.txQueue)) // 1 remaining in queue

	// Third call with larger maxBytes to get the 100-byte tx
	req3 := coresequencer.GetNextBatchRequest{
		MaxBytes:      200,
		LastBatchData: nil,
	}
	resp3, err := seq.GetNextBatch(context.Background(), req3)
	require.NoError(t, err)
	require.NotNil(t, resp3)
	require.NotNil(t, resp3.Batch)
	assert.Equal(t, 1, len(resp3.Batch.Transactions))
	assert.Equal(t, 0, len(seq.txQueue)) // Queue should be empty

}

func TestBasedSequencer_GetNextBatch_FromQueue(t *testing.T) {
	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	daClient := mockDAClient{forced: map[uint64]datypes.ResultRetrieve{}, hasFI: true}
	fiRetriever := block.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	seq := NewBasedSequencer(fiRetriever, cfg, gen, zerolog.Nop())

	// Pre-populate the queue
	seq.txQueue = [][]byte{[]byte("queued_tx1"), []byte("queued_tx2")}

	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 2, len(resp.Batch.Transactions))
	assert.Equal(t, []byte("queued_tx1"), resp.Batch.Transactions[0])
	assert.Equal(t, []byte("queued_tx2"), resp.Batch.Transactions[1])

	// Queue should be empty now
	assert.Equal(t, 0, len(seq.txQueue))
}

func TestBasedSequencer_GetNextBatch_AlwaysCheckPendingForcedInclusion(t *testing.T) {
	forcedTx := make([]byte, 150)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	daClient := mockDAClient{
		forced: map[uint64]datypes.ResultRetrieve{
			100: {
				BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Height: 100},
				Data:       [][]byte{forcedTx},
			},
		},
		hasFI: true,
	}
	fiRetriever := block.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	seq := NewBasedSequencer(fiRetriever, cfg, gen, zerolog.Nop())

	// First call with maxBytes = 100
	// Forced tx (150 bytes) is added to queue, but batch will be empty since it exceeds maxBytes
	req1 := coresequencer.GetNextBatchRequest{
		MaxBytes:      100,
		LastBatchData: nil,
	}

	resp1, err := seq.GetNextBatch(context.Background(), req1)
	require.NoError(t, err)
	require.NotNil(t, resp1)
	require.NotNil(t, resp1.Batch)
	assert.Equal(t, 0, len(resp1.Batch.Transactions), "Should have no txs as forced tx exceeds maxBytes")

	// Verify forced tx is in queue
	assert.Equal(t, 1, len(seq.txQueue), "Forced tx should be in queue")

	// Second call with larger maxBytes = 200
	// Should process tx from queue
	req2 := coresequencer.GetNextBatchRequest{
		MaxBytes:      200,
		LastBatchData: nil,
	}

	resp2, err := seq.GetNextBatch(context.Background(), req2)
	require.NoError(t, err)
	require.NotNil(t, resp2)
	require.NotNil(t, resp2.Batch)
	assert.Equal(t, 1, len(resp2.Batch.Transactions), "Should include tx from queue")
	assert.Equal(t, 150, len(resp2.Batch.Transactions[0]))

	// Queue should now be empty
	assert.Equal(t, 0, len(seq.txQueue), "Queue should be empty")
}

func TestBasedSequencer_GetNextBatch_ForcedInclusionExceedsMaxBytes(t *testing.T) {
	// Return forced txs where combined they exceed maxBytes
	forcedTx1 := make([]byte, 100)
	forcedTx2 := make([]byte, 80)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	daClient := mockDAClient{
		forced: map[uint64]datypes.ResultRetrieve{
			100: {
				BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Height: 100},
				Data:       [][]byte{forcedTx1, forcedTx2},
			},
		},
		hasFI: true,
	}
	fiRetriever := block.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	seq := NewBasedSequencer(fiRetriever, cfg, gen, zerolog.Nop())

	// First call with maxBytes = 120
	// Should get only first forced tx (100 bytes), second stays in queue
	req1 := coresequencer.GetNextBatchRequest{
		MaxBytes:      120,
		LastBatchData: nil,
	}

	resp1, err := seq.GetNextBatch(context.Background(), req1)
	require.NoError(t, err)
	require.NotNil(t, resp1)
	require.NotNil(t, resp1.Batch)
	assert.Equal(t, 1, len(resp1.Batch.Transactions), "Should only include first forced tx")
	assert.Equal(t, 100, len(resp1.Batch.Transactions[0]))

	// Verify second tx is still in queue
	assert.Equal(t, 1, len(seq.txQueue), "Second tx should be in queue")

	// Second call - should get the second tx from queue
	req2 := coresequencer.GetNextBatchRequest{
		MaxBytes:      120,
		LastBatchData: nil,
	}

	resp2, err := seq.GetNextBatch(context.Background(), req2)
	require.NoError(t, err)
	require.NotNil(t, resp2)
	require.NotNil(t, resp2.Batch)
	assert.Equal(t, 1, len(resp2.Batch.Transactions), "Should include second tx from queue")
	assert.Equal(t, 80, len(resp2.Batch.Transactions[0]))

	// Queue should now be empty
	assert.Equal(t, 0, len(seq.txQueue), "Queue should be empty")
}

func TestBasedSequencer_VerifyBatch(t *testing.T) {
	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	daClient := mockDAClient{forced: map[uint64]datypes.ResultRetrieve{}, hasFI: true}
	fiRetriever := block.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	seq := NewBasedSequencer(fiRetriever, cfg, gen, zerolog.Nop())

	req := coresequencer.VerifyBatchRequest{
		Id:        []byte("test-chain"),
		BatchData: [][]byte{[]byte("tx1")},
	}

	resp, err := seq.VerifyBatch(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Status)
}

func TestBasedSequencer_SetDAHeight(t *testing.T) {
	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	daClient := mockDAClient{forced: map[uint64]datypes.ResultRetrieve{}, hasFI: true}
	fiRetriever := block.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	seq := NewBasedSequencer(fiRetriever, cfg, gen, zerolog.Nop())

	assert.Equal(t, uint64(100), seq.GetDAHeight())

	seq.SetDAHeight(200)
	assert.Equal(t, uint64(200), seq.GetDAHeight())
}

func TestBasedSequencer_GetNextBatch_ErrorHandling(t *testing.T) {
	gen := genesis.Genesis{
		ChainID:                "test-chain",
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	cfg.DA.ForcedInclusionNamespace = "test-fi-ns"

	daClient := mockDAClient{
		forced: map[uint64]datypes.ResultRetrieve{
			100: {BaseResult: datypes.BaseResult{Code: datypes.StatusError, Message: "DA connection error", Height: 100}},
		},
		hasFI: true,
	}
	fiRetriever := block.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	seq := NewBasedSequencer(fiRetriever, cfg, gen, zerolog.Nop())

	req := coresequencer.GetNextBatchRequest{
		MaxBytes:      1000000,
		LastBatchData: nil,
	}

	// With new error handling, errors during blob processing return empty batch instead of error
	resp, err := seq.GetNextBatch(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Batch)
	assert.Equal(t, 0, len(resp.Batch.Transactions), "Should return empty batch on DA error")

}
