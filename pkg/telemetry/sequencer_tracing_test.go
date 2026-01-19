package telemetry

import (
	"context"
	"errors"
	"testing"
	"time"

	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/telemetry/testutil"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type mockSequencer struct {
	submitBatchTxsFn func(context.Context, coresequencer.SubmitBatchTxsRequest) (*coresequencer.SubmitBatchTxsResponse, error)
	getNextBatchFn   func(context.Context, coresequencer.GetNextBatchRequest) (*coresequencer.GetNextBatchResponse, error)
	verifyBatchFn    func(context.Context, coresequencer.VerifyBatchRequest) (*coresequencer.VerifyBatchResponse, error)
	daHeight         uint64
}

func (m *mockSequencer) SubmitBatchTxs(ctx context.Context, req coresequencer.SubmitBatchTxsRequest) (*coresequencer.SubmitBatchTxsResponse, error) {
	if m.submitBatchTxsFn != nil {
		return m.submitBatchTxsFn(ctx, req)
	}
	return &coresequencer.SubmitBatchTxsResponse{}, nil
}

func (m *mockSequencer) GetNextBatch(ctx context.Context, req coresequencer.GetNextBatchRequest) (*coresequencer.GetNextBatchResponse, error) {
	if m.getNextBatchFn != nil {
		return m.getNextBatchFn(ctx, req)
	}
	return &coresequencer.GetNextBatchResponse{
		Batch:     &coresequencer.Batch{},
		Timestamp: time.Now(),
	}, nil
}

func (m *mockSequencer) VerifyBatch(ctx context.Context, req coresequencer.VerifyBatchRequest) (*coresequencer.VerifyBatchResponse, error) {
	if m.verifyBatchFn != nil {
		return m.verifyBatchFn(ctx, req)
	}
	return &coresequencer.VerifyBatchResponse{Status: true}, nil
}

func (m *mockSequencer) SetDAHeight(height uint64) {
	m.daHeight = height
}

func (m *mockSequencer) GetDAHeight() uint64 {
	return m.daHeight
}

var _ coresequencer.Sequencer = (*mockSequencer)(nil)

func setupSequencerTrace(t *testing.T, inner coresequencer.Sequencer) (coresequencer.Sequencer, *tracetest.SpanRecorder) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	otel.SetTracerProvider(tp)
	return WithTracingSequencer(inner), sr
}

func TestTracedSequencer_SubmitBatchTxs_Success(t *testing.T) {
	mock := &mockSequencer{
		submitBatchTxsFn: func(ctx context.Context, req coresequencer.SubmitBatchTxsRequest) (*coresequencer.SubmitBatchTxsResponse, error) {
			return &coresequencer.SubmitBatchTxsResponse{}, nil
		},
	}
	seq, sr := setupSequencerTrace(t, mock)
	ctx := context.Background()

	req := coresequencer.SubmitBatchTxsRequest{
		Id: []byte("test-chain"),
		Batch: &coresequencer.Batch{
			Transactions: [][]byte{[]byte("tx1"), []byte("tx2"), []byte("tx3")},
		},
	}

	res, err := seq.SubmitBatchTxs(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "Sequencer.SubmitBatchTxs", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "tx.count", 3)
	testutil.RequireAttribute(t, attrs, "batch.size_bytes", 9) // "tx1" + "tx2" + "tx3" = 9 bytes
}

func TestTracedSequencer_SubmitBatchTxs_Error(t *testing.T) {
	mock := &mockSequencer{
		submitBatchTxsFn: func(ctx context.Context, req coresequencer.SubmitBatchTxsRequest) (*coresequencer.SubmitBatchTxsResponse, error) {
			return nil, errors.New("queue full")
		},
	}
	seq, sr := setupSequencerTrace(t, mock)
	ctx := context.Background()

	req := coresequencer.SubmitBatchTxsRequest{
		Id:    []byte("test-chain"),
		Batch: &coresequencer.Batch{Transactions: [][]byte{[]byte("tx1")}},
	}

	_, err := seq.SubmitBatchTxs(ctx, req)
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
}

func TestTracedSequencer_GetNextBatch_Success(t *testing.T) {
	mock := &mockSequencer{
		getNextBatchFn: func(ctx context.Context, req coresequencer.GetNextBatchRequest) (*coresequencer.GetNextBatchResponse, error) {
			return &coresequencer.GetNextBatchResponse{
				Batch: &coresequencer.Batch{
					Transactions:      [][]byte{[]byte("tx1"), []byte("forced-tx")},
					ForceIncludedMask: []bool{false, true},
				},
				Timestamp: time.Unix(1700000000, 0),
			}, nil
		},
	}
	seq, sr := setupSequencerTrace(t, mock)
	ctx := context.Background()

	req := coresequencer.GetNextBatchRequest{
		Id:       []byte("test-chain"),
		MaxBytes: 1000,
	}

	res, err := seq.GetNextBatch(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "Sequencer.GetNextBatch", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "tx.count", 2)
	testutil.RequireAttribute(t, attrs, "forced_inclusion.count", 1)
	testutil.RequireAttribute(t, attrs, "max_bytes", int64(1000))
}

func TestTracedSequencer_GetNextBatch_Error(t *testing.T) {
	mock := &mockSequencer{
		getNextBatchFn: func(ctx context.Context, req coresequencer.GetNextBatchRequest) (*coresequencer.GetNextBatchResponse, error) {
			return nil, errors.New("failed to fetch from DA")
		},
	}
	seq, sr := setupSequencerTrace(t, mock)
	ctx := context.Background()

	req := coresequencer.GetNextBatchRequest{
		Id:       []byte("test-chain"),
		MaxBytes: 1000,
	}

	_, err := seq.GetNextBatch(ctx, req)
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
}

func TestTracedSequencer_VerifyBatch_Success(t *testing.T) {
	mock := &mockSequencer{
		verifyBatchFn: func(ctx context.Context, req coresequencer.VerifyBatchRequest) (*coresequencer.VerifyBatchResponse, error) {
			return &coresequencer.VerifyBatchResponse{Status: true}, nil
		},
	}
	seq, sr := setupSequencerTrace(t, mock)
	ctx := context.Background()

	req := coresequencer.VerifyBatchRequest{
		Id:        []byte("test-chain"),
		BatchData: [][]byte{[]byte("proof1"), []byte("proof2")},
	}

	res, err := seq.VerifyBatch(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.True(t, res.Status)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "Sequencer.VerifyBatch", span.Name())

	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "batch_data.count", 2)
	testutil.RequireAttribute(t, attrs, "verified", true)
}

func TestTracedSequencer_VerifyBatch_Failure(t *testing.T) {
	mock := &mockSequencer{
		verifyBatchFn: func(ctx context.Context, req coresequencer.VerifyBatchRequest) (*coresequencer.VerifyBatchResponse, error) {
			return &coresequencer.VerifyBatchResponse{Status: false}, nil
		},
	}
	seq, sr := setupSequencerTrace(t, mock)
	ctx := context.Background()

	req := coresequencer.VerifyBatchRequest{
		Id:        []byte("test-chain"),
		BatchData: [][]byte{[]byte("invalid-proof")},
	}

	res, err := seq.VerifyBatch(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.False(t, res.Status)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]

	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "verified", false)
}

func TestTracedSequencer_VerifyBatch_Error(t *testing.T) {
	mock := &mockSequencer{
		verifyBatchFn: func(ctx context.Context, req coresequencer.VerifyBatchRequest) (*coresequencer.VerifyBatchResponse, error) {
			return nil, errors.New("failed to get proofs")
		},
	}
	seq, sr := setupSequencerTrace(t, mock)
	ctx := context.Background()

	req := coresequencer.VerifyBatchRequest{
		Id:        []byte("test-chain"),
		BatchData: [][]byte{[]byte("proof")},
	}

	_, err := seq.VerifyBatch(ctx, req)
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
}

func TestTracedSequencer_DAHeightPassthrough(t *testing.T) {
	mock := &mockSequencer{}
	seq, _ := setupSequencerTrace(t, mock)

	seq.SetDAHeight(100)
	require.Equal(t, uint64(100), seq.GetDAHeight())

	seq.SetDAHeight(200)
	require.Equal(t, uint64(200), seq.GetDAHeight())
}
