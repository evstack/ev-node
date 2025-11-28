package single

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/celestiaorg/go-square/v3/share"
	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/blob"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/namespace"
)

var (
	testNamespaceBytes = namespace.NamespaceFromString("test-namespace").Bytes()
	testNamespace      = func() share.Namespace {
		ns, err := share.NewNamespaceFromBytes(testNamespaceBytes)
		if err != nil {
			panic(err)
		}
		return ns
	}()
)

type noopBlobVerifier struct{}

func (noopBlobVerifier) GetProof(ctx context.Context, height uint64, ns share.Namespace, commitment blob.Commitment) (*blob.Proof, error) {
	return &blob.Proof{}, nil
}

func (noopBlobVerifier) Included(ctx context.Context, height uint64, ns share.Namespace, proof *blob.Proof, commitment blob.Commitment) (bool, error) {
	return true, nil
}

type mockBlobVerifier struct {
	mock.Mock
}

func (m *mockBlobVerifier) GetProof(ctx context.Context, height uint64, ns share.Namespace, commitment blob.Commitment) (*blob.Proof, error) {
	args := m.Called(ctx, height, ns, commitment)
	if proof, ok := args.Get(0).(*blob.Proof); ok {
		return proof, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockBlobVerifier) Included(ctx context.Context, height uint64, ns share.Namespace, proof *blob.Proof, commitment blob.Commitment) (bool, error) {
	args := m.Called(ctx, height, ns, proof, commitment)
	return args.Bool(0), args.Error(1)
}

type sequencerTestParams struct {
	ctx       context.Context
	logger    zerolog.Logger
	db        ds.Batching
	verifier  blobProofVerifier
	id        []byte
	namespace []byte
	batchTime time.Duration
	metrics   *Metrics
	proposer  bool
	queueSize int
}

func newSequencerForTest(t *testing.T, opts ...func(*sequencerTestParams)) *Sequencer {
	t.Helper()

	params := &sequencerTestParams{
		ctx:       context.Background(),
		logger:    zerolog.Nop(),
		db:        ds.NewMapDatastore(),
		verifier:  noopBlobVerifier{},
		id:        []byte("test"),
		namespace: testNamespaceBytes,
		batchTime: time.Second,
		metrics:   nil,
		proposer:  false,
		queueSize: 1000,
	}

	for _, opt := range opts {
		opt(params)
	}

	seq, err := NewSequencerWithQueueSize(
		params.ctx,
		params.logger,
		params.db,
		params.verifier,
		params.id,
		params.namespace,
		params.batchTime,
		params.metrics,
		params.proposer,
		params.queueSize,
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		if closer, ok := params.db.(interface{ Close() error }); ok {
			require.NoError(t, closer.Close())
		}
	})

	return seq
}

func makeBlobID(height uint64, commitment string) []byte {
	return blob.MakeID(height, blob.Commitment([]byte(commitment)))
}

func TestNewSequencer(t *testing.T) {
	metrics, err := NopMetrics()
	require.NoError(t, err)

	seq := newSequencerForTest(t, func(p *sequencerTestParams) {
		p.metrics = metrics
		p.id = []byte("test1")
		p.batchTime = 10 * time.Second
	})

	require.NotNil(t, seq)
	require.NotNil(t, seq.queue)
	require.NotNil(t, seq.verifier)
	assert.True(t, bytes.Equal(seq.Id, []byte("test1")))
}

func TestSequencer_SubmitBatchTxs(t *testing.T) {
	seq := newSequencerForTest(t, func(p *sequencerTestParams) {
		p.id = []byte("test1")
	})

	tx := []byte("transaction1")
	res, err := seq.SubmitBatchTxs(context.Background(), coresequencer.SubmitBatchTxsRequest{Id: seq.Id, Batch: &coresequencer.Batch{Transactions: [][]byte{tx}}})
	require.NoError(t, err)
	require.NotNil(t, res)

	nextResp, err := seq.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{Id: seq.Id})
	require.NoError(t, err)
	require.Len(t, nextResp.Batch.Transactions, 1)

	res, err = seq.SubmitBatchTxs(context.Background(), coresequencer.SubmitBatchTxsRequest{Id: []byte("other"), Batch: &coresequencer.Batch{Transactions: [][]byte{tx}}})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidId))
	assert.Nil(t, res)
}

func TestSequencer_SubmitBatchTxs_EmptyBatch(t *testing.T) {
	seq := newSequencerForTest(t)

	res, err := seq.SubmitBatchTxs(context.Background(), coresequencer.SubmitBatchTxsRequest{
		Id:    seq.Id,
		Batch: &coresequencer.Batch{Transactions: [][]byte{}},
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	res, err = seq.SubmitBatchTxs(context.Background(), coresequencer.SubmitBatchTxsRequest{
		Id:    seq.Id,
		Batch: nil,
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	nextResp, err := seq.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{Id: seq.Id})
	require.NoError(t, err)
	assert.Empty(t, nextResp.Batch.Transactions)
}

func TestSequencer_GetNextBatch_NoLastBatch(t *testing.T) {
	seq := newSequencerForTest(t)

	res, err := seq.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{Id: seq.Id})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Empty(t, res.Batch.Transactions)
}

func TestSequencer_GetNextBatch_Success(t *testing.T) {
	seq := newSequencerForTest(t)

	mockBatch := coresequencer.Batch{Transactions: [][]byte{[]byte("tx1"), []byte("tx2")}}
	err := seq.queue.AddBatch(context.Background(), mockBatch)
	require.NoError(t, err)

	res, err := seq.GetNextBatch(context.Background(), coresequencer.GetNextBatchRequest{Id: seq.Id})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Len(t, res.Batch.Transactions, 2)
}

func TestSequencer_VerifyBatch(t *testing.T) {
	ctx := context.Background()
	Id := []byte("test")
	batchData := [][]byte{
		makeBlobID(10, "commit1"),
		makeBlobID(11, "commit2"),
	}

	t.Run("Proposer Mode", func(t *testing.T) {
		seq := newSequencerForTest(t, func(p *sequencerTestParams) {
			p.id = Id
			p.proposer = true
		})

		res, err := seq.VerifyBatch(ctx, coresequencer.VerifyBatchRequest{Id: Id, BatchData: batchData})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.Status)
	})

	t.Run("Non-Proposer Mode", func(t *testing.T) {
		t.Run("Valid Proofs", func(t *testing.T) {
			verifier := new(mockBlobVerifier)
			seq := newSequencerForTest(t, func(p *sequencerTestParams) {
				p.id = Id
				p.verifier = verifier
			})

			proof := &blob.Proof{}
			verifier.On("GetProof", ctx, uint64(10), testNamespace, blob.Commitment([]byte("commit1"))).Return(proof, nil).Once()
			verifier.On("Included", ctx, uint64(10), testNamespace, proof, blob.Commitment([]byte("commit1"))).Return(true, nil).Once()
			verifier.On("GetProof", ctx, uint64(11), testNamespace, blob.Commitment([]byte("commit2"))).Return(proof, nil).Once()
			verifier.On("Included", ctx, uint64(11), testNamespace, proof, blob.Commitment([]byte("commit2"))).Return(true, nil).Once()

			res, err := seq.VerifyBatch(ctx, coresequencer.VerifyBatchRequest{Id: Id, BatchData: batchData})
			require.NoError(t, err)
			require.True(t, res.Status)
			verifier.AssertExpectations(t)
		})

		t.Run("Invalid Proof", func(t *testing.T) {
			verifier := new(mockBlobVerifier)
			seq := newSequencerForTest(t, func(p *sequencerTestParams) {
				p.id = Id
				p.verifier = verifier
			})

			proof := &blob.Proof{}
			verifier.On("GetProof", ctx, uint64(10), testNamespace, blob.Commitment([]byte("commit1"))).Return(proof, nil).Once()
			verifier.On("Included", ctx, uint64(10), testNamespace, proof, blob.Commitment([]byte("commit1"))).Return(true, nil).Once()
			verifier.On("GetProof", ctx, uint64(11), testNamespace, blob.Commitment([]byte("commit2"))).Return(proof, nil).Once()
			verifier.On("Included", ctx, uint64(11), testNamespace, proof, blob.Commitment([]byte("commit2"))).Return(false, nil).Once()

			res, err := seq.VerifyBatch(ctx, coresequencer.VerifyBatchRequest{Id: Id, BatchData: batchData})
			require.NoError(t, err)
			require.False(t, res.Status)
			verifier.AssertExpectations(t)
		})

		t.Run("GetProof Error", func(t *testing.T) {
			verifier := new(mockBlobVerifier)
			seq := newSequencerForTest(t, func(p *sequencerTestParams) {
				p.id = Id
				p.verifier = verifier
			})

			expectedErr := errors.New("get proof failed")
			verifier.On("GetProof", ctx, uint64(10), testNamespace, blob.Commitment([]byte("commit1"))).Return((*blob.Proof)(nil), expectedErr).Once()

			res, err := seq.VerifyBatch(ctx, coresequencer.VerifyBatchRequest{Id: Id, BatchData: batchData})
			require.Error(t, err)
			require.Nil(t, res)
			assert.ErrorContains(t, err, expectedErr.Error())
			verifier.AssertExpectations(t)
		})

		t.Run("Included Error", func(t *testing.T) {
			verifier := new(mockBlobVerifier)
			seq := newSequencerForTest(t, func(p *sequencerTestParams) {
				p.id = Id
				p.verifier = verifier
			})

			proof := &blob.Proof{}
			expectedErr := errors.New("included failed")
			verifier.On("GetProof", ctx, uint64(10), testNamespace, blob.Commitment([]byte("commit1"))).Return(proof, nil).Once()
			verifier.On("Included", ctx, uint64(10), testNamespace, proof, blob.Commitment([]byte("commit1"))).Return(false, expectedErr).Once()

			res, err := seq.VerifyBatch(ctx, coresequencer.VerifyBatchRequest{Id: Id, BatchData: batchData})
			require.Error(t, err)
			require.Nil(t, res)
			assert.ErrorContains(t, err, expectedErr.Error())
			verifier.AssertExpectations(t)
		})

		t.Run("Invalid Blob ID", func(t *testing.T) {
			verifier := new(mockBlobVerifier)
			seq := newSequencerForTest(t, func(p *sequencerTestParams) {
				p.id = Id
				p.verifier = verifier
			})

			res, err := seq.VerifyBatch(ctx, coresequencer.VerifyBatchRequest{Id: Id, BatchData: [][]byte{[]byte("short")}})
			require.Error(t, err)
			require.Nil(t, res)
		})

		t.Run("Invalid ID", func(t *testing.T) {
			verifier := new(mockBlobVerifier)
			seq := newSequencerForTest(t, func(p *sequencerTestParams) {
				p.id = Id
				p.verifier = verifier
			})

			res, err := seq.VerifyBatch(ctx, coresequencer.VerifyBatchRequest{Id: []byte("invalid"), BatchData: batchData})
			require.Error(t, err)
			require.Nil(t, res)
			assert.ErrorIs(t, err, ErrInvalidId)
			verifier.AssertNotCalled(t, "GetProof", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	})
}

func TestSequencer_RecordMetrics(t *testing.T) {
	t.Run("With Metrics", func(t *testing.T) {
		metrics, err := NopMetrics()
		require.NoError(t, err)

		seq := &Sequencer{
			logger:  zerolog.Nop(),
			metrics: metrics,
		}

		seq.RecordMetrics(1.5, 1024, datypes.StatusSuccess, 5, 100)
		assert.NotNil(t, seq.metrics)
	})

	t.Run("Without Metrics", func(t *testing.T) {
		seq := &Sequencer{logger: zerolog.Nop()}
		seq.RecordMetrics(2.0, 2048, datypes.StatusNotIncludedInBlock, 3, 200)
		assert.Nil(t, seq.metrics)
	})

	t.Run("With Different Status Codes", func(t *testing.T) {
		metrics, err := NopMetrics()
		require.NoError(t, err)

		seq := &Sequencer{
			logger:  zerolog.Nop(),
			metrics: metrics,
		}

		codes := []datypes.StatusCode{
			datypes.StatusSuccess,
			datypes.StatusNotIncludedInBlock,
			datypes.StatusAlreadyInMempool,
			datypes.StatusTooBig,
			datypes.StatusContextCanceled,
		}

		for _, code := range codes {
			seq.RecordMetrics(1.0, 512, code, 2, 50)
			assert.NotNil(t, seq.metrics)
		}
	})
}

func TestSequencer_QueueLimit_Integration(t *testing.T) {
	seq := newSequencerForTest(t, func(p *sequencerTestParams) {
		p.queueSize = 2
		p.proposer = true
	})

	ctx := context.Background()

	batch1 := createTestBatch(t, 3)
	resp, err := seq.SubmitBatchTxs(ctx, coresequencer.SubmitBatchTxsRequest{Id: seq.Id, Batch: &batch1})
	require.NoError(t, err)
	require.NotNil(t, resp)

	batch2 := createTestBatch(t, 4)
	resp, err = seq.SubmitBatchTxs(ctx, coresequencer.SubmitBatchTxsRequest{Id: seq.Id, Batch: &batch2})
	require.NoError(t, err)
	require.NotNil(t, resp)

	batch3 := createTestBatch(t, 5)
	resp, err = seq.SubmitBatchTxs(ctx, coresequencer.SubmitBatchTxsRequest{Id: seq.Id, Batch: &batch3})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrQueueFull))
	require.Nil(t, resp)

	_, err = seq.GetNextBatch(ctx, coresequencer.GetNextBatchRequest{Id: seq.Id})
	require.NoError(t, err)

	resp, err = seq.SubmitBatchTxs(ctx, coresequencer.SubmitBatchTxsRequest{Id: seq.Id, Batch: &batch3})
	require.NoError(t, err)
	require.NotNil(t, resp)

	emptyBatch := coresequencer.Batch{Transactions: nil}
	resp, err = seq.SubmitBatchTxs(ctx, coresequencer.SubmitBatchTxsRequest{Id: seq.Id, Batch: &emptyBatch})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestSequencer_QueueThrottling(t *testing.T) {
	queueSize := 3
	seq := newSequencerForTest(t, func(p *sequencerTestParams) {
		p.queueSize = queueSize
		p.proposer = true
	})

	ctx := context.Background()

	for i := 0; i < queueSize; i++ {
		batch := createTestBatch(t, i+1)
		resp, err := seq.SubmitBatchTxs(ctx, coresequencer.SubmitBatchTxsRequest{Id: seq.Id, Batch: &batch})
		require.NoError(t, err)
		require.NotNil(t, resp)
	}

	overflowBatch := createTestBatch(t, queueSize+1)
	resp, err := seq.SubmitBatchTxs(ctx, coresequencer.SubmitBatchTxsRequest{Id: seq.Id, Batch: &overflowBatch})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrQueueFull))
	require.Nil(t, resp)

	_, err = seq.GetNextBatch(ctx, coresequencer.GetNextBatchRequest{Id: seq.Id})
	require.NoError(t, err)

	resp, err = seq.SubmitBatchTxs(ctx, coresequencer.SubmitBatchTxsRequest{Id: seq.Id, Batch: &overflowBatch})
	require.NoError(t, err)
	require.NotNil(t, resp)

	for i := 0; i < 10; i++ {
		batch := createTestBatch(t, 100+i)
		resp, err := seq.SubmitBatchTxs(ctx, coresequencer.SubmitBatchTxsRequest{Id: seq.Id, Batch: &batch})
		if err != nil {
			require.True(t, errors.Is(err, ErrQueueFull))
			require.Nil(t, resp)
			break
		}
		require.NotNil(t, resp)
	}

	finalBatch := createTestBatch(t, 999)
	resp, err = seq.SubmitBatchTxs(ctx, coresequencer.SubmitBatchTxsRequest{Id: seq.Id, Batch: &finalBatch})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrQueueFull))
	require.Nil(t, resp)
}
