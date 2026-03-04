package executing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/telemetry/testutil"
	"github.com/evstack/ev-node/types"
)

// mockBlockProducer provides function hooks for testing the tracing decorator.
type mockBlockProducer struct {
	produceBlockFn  func(ctx context.Context) error
	retrieveBatchFn func(ctx context.Context) (*BatchData, error)
	createBlockFn   func(ctx context.Context, height uint64, batchData *BatchData) (*types.SignedHeader, *types.Data, error)
	applyBlockFn    func(ctx context.Context, header types.Header, data *types.Data) (types.State, error)
}

func (m *mockBlockProducer) ProduceBlock(ctx context.Context) error {
	if m.produceBlockFn != nil {
		return m.produceBlockFn(ctx)
	}
	return nil
}

func (m *mockBlockProducer) RetrieveBatch(ctx context.Context) (*BatchData, error) {
	if m.retrieveBatchFn != nil {
		return m.retrieveBatchFn(ctx)
	}
	return nil, nil
}

func (m *mockBlockProducer) CreateBlock(ctx context.Context, height uint64, batchData *BatchData) (*types.SignedHeader, *types.Data, error) {
	if m.createBlockFn != nil {
		return m.createBlockFn(ctx, height, batchData)
	}
	return nil, nil, nil
}

func (m *mockBlockProducer) ApplyBlock(ctx context.Context, header types.Header, data *types.Data) (types.State, error) {
	if m.applyBlockFn != nil {
		return m.applyBlockFn(ctx, header, data)
	}
	return types.State{}, nil
}

func setupBlockProducerTrace(t *testing.T, inner BlockProducer) (BlockProducer, *tracetest.SpanRecorder) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	otel.SetTracerProvider(tp)
	return WithTracingBlockProducer(inner), sr
}

func TestTracedBlockProducer_ProduceBlock_Success(t *testing.T) {
	mock := &mockBlockProducer{
		produceBlockFn: func(ctx context.Context) error {
			return nil
		},
	}
	producer, sr := setupBlockProducerTrace(t, mock)
	ctx := context.Background()

	err := producer.ProduceBlock(ctx)
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "BlockExecutor.ProduceBlock", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)
}

func TestTracedBlockProducer_ProduceBlock_Error(t *testing.T) {
	mock := &mockBlockProducer{
		produceBlockFn: func(ctx context.Context) error {
			return errors.New("production failed")
		},
	}
	producer, sr := setupBlockProducerTrace(t, mock)
	ctx := context.Background()

	err := producer.ProduceBlock(ctx)
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, "production failed", span.Status().Description)
}

func TestTracedBlockProducer_RetrieveBatch_Success(t *testing.T) {
	mock := &mockBlockProducer{
		retrieveBatchFn: func(ctx context.Context) (*BatchData, error) {
			return &BatchData{
				Batch: &coresequencer.Batch{
					Transactions: [][]byte{[]byte("tx1"), []byte("tx2")},
				},
			}, nil
		},
	}
	producer, sr := setupBlockProducerTrace(t, mock)
	ctx := context.Background()

	batch, err := producer.RetrieveBatch(ctx)
	require.NoError(t, err)
	require.NotNil(t, batch)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "BlockExecutor.RetrieveBatch", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "batch.tx_count", 2)
}

func TestTracedBlockProducer_RetrieveBatch_Error(t *testing.T) {
	mock := &mockBlockProducer{
		retrieveBatchFn: func(ctx context.Context) (*BatchData, error) {
			return nil, errors.New("no batch available")
		},
	}
	producer, sr := setupBlockProducerTrace(t, mock)
	ctx := context.Background()

	_, err := producer.RetrieveBatch(ctx)
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, "no batch available", span.Status().Description)
}

func TestTracedBlockProducer_CreateBlock_Success(t *testing.T) {
	mock := &mockBlockProducer{
		createBlockFn: func(ctx context.Context, height uint64, batchData *BatchData) (*types.SignedHeader, *types.Data, error) {
			return &types.SignedHeader{}, &types.Data{}, nil
		},
	}
	producer, sr := setupBlockProducerTrace(t, mock)
	ctx := context.Background()

	batchData := &BatchData{
		Batch: &coresequencer.Batch{
			Transactions: [][]byte{[]byte("tx1"), []byte("tx2"), []byte("tx3")},
		},
	}

	header, data, err := producer.CreateBlock(ctx, 100, batchData)
	require.NoError(t, err)
	require.NotNil(t, header)
	require.NotNil(t, data)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "BlockExecutor.CreateBlock", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "block.height", int64(100))
	testutil.RequireAttribute(t, attrs, "tx.count", 3)
}

func TestTracedBlockProducer_CreateBlock_Error(t *testing.T) {
	mock := &mockBlockProducer{
		createBlockFn: func(ctx context.Context, height uint64, batchData *BatchData) (*types.SignedHeader, *types.Data, error) {
			return nil, nil, errors.New("failed to create block")
		},
	}
	producer, sr := setupBlockProducerTrace(t, mock)
	ctx := context.Background()

	_, _, err := producer.CreateBlock(ctx, 100, nil)
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, "failed to create block", span.Status().Description)
}

func TestTracedBlockProducer_ApplyBlock_Success(t *testing.T) {
	mock := &mockBlockProducer{
		applyBlockFn: func(ctx context.Context, header types.Header, data *types.Data) (types.State, error) {
			return types.State{
				AppHash: []byte{0xde, 0xad, 0xbe, 0xef},
			}, nil
		},
	}
	producer, sr := setupBlockProducerTrace(t, mock)
	ctx := context.Background()

	header := types.Header{
		BaseHeader: types.BaseHeader{
			Height: 50,
		},
	}
	data := &types.Data{
		Txs: types.Txs{[]byte("tx1"), []byte("tx2")},
	}

	state, err := producer.ApplyBlock(ctx, header, data)
	require.NoError(t, err)
	require.NotEmpty(t, state.AppHash)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "BlockExecutor.ApplyBlock", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "block.height", int64(50))
	testutil.RequireAttribute(t, attrs, "tx.count", 2)
	testutil.RequireAttribute(t, attrs, "state_root", "deadbeef")
}

func TestTracedBlockProducer_ApplyBlock_Error(t *testing.T) {
	mock := &mockBlockProducer{
		applyBlockFn: func(ctx context.Context, header types.Header, data *types.Data) (types.State, error) {
			return types.State{}, errors.New("execution failed")
		},
	}
	producer, sr := setupBlockProducerTrace(t, mock)
	ctx := context.Background()

	header := types.Header{
		BaseHeader: types.BaseHeader{
			Height: 50,
		},
	}

	_, err := producer.ApplyBlock(ctx, header, &types.Data{})
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, "execution failed", span.Status().Description)
}

// TestTracedBlockProducer_RetrieveBatch_ErrorWithValue verifies that when the inner
// function returns both a value and an error, the value is passed through (not nil).
// this is important for cases like ErrNoTransactionsInBatch where valid data accompanies the error.
func TestTracedBlockProducer_RetrieveBatch_ErrorWithValue(t *testing.T) {
	expectedBatch := &BatchData{
		Batch: &coresequencer.Batch{
			Transactions: [][]byte{},
		},
	}
	mock := &mockBlockProducer{
		retrieveBatchFn: func(ctx context.Context) (*BatchData, error) {
			return expectedBatch, errors.New("no transactions in batch")
		},
	}
	producer, sr := setupBlockProducerTrace(t, mock)
	ctx := context.Background()

	batch, err := producer.RetrieveBatch(ctx)
	require.Error(t, err)
	require.Equal(t, "no transactions in batch", err.Error())
	require.NotNil(t, batch, "batch should not be nil when inner returns value with error")
	require.Same(t, expectedBatch, batch, "batch should be the same instance returned by inner")

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
}

// TestTracedBlockProducer_CreateBlock_ErrorWithValue verifies that when the inner
// function returns both values and an error, the values are passed through (not nil).
func TestTracedBlockProducer_CreateBlock_ErrorWithValue(t *testing.T) {
	expectedHeader := &types.SignedHeader{
		Header: types.Header{
			BaseHeader: types.BaseHeader{
				Height: 100,
			},
		},
	}
	expectedData := &types.Data{
		Txs: types.Txs{[]byte("tx1")},
	}
	mock := &mockBlockProducer{
		createBlockFn: func(ctx context.Context, height uint64, batchData *BatchData) (*types.SignedHeader, *types.Data, error) {
			return expectedHeader, expectedData, errors.New("partial failure")
		},
	}
	producer, sr := setupBlockProducerTrace(t, mock)
	ctx := context.Background()

	header, data, err := producer.CreateBlock(ctx, 100, nil)
	require.Error(t, err)
	require.Equal(t, "partial failure", err.Error())
	require.NotNil(t, header, "header should not be nil when inner returns value with error")
	require.NotNil(t, data, "data should not be nil when inner returns value with error")
	require.Same(t, expectedHeader, header, "header should be the same instance returned by inner")
	require.Same(t, expectedData, data, "data should be the same instance returned by inner")

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
}

// TestTracedBlockProducer_ApplyBlock_ErrorWithValue verifies that when the inner
// function returns both a state and an error, the state is passed through (not zero value).
func TestTracedBlockProducer_ApplyBlock_ErrorWithValue(t *testing.T) {
	expectedState := types.State{
		AppHash:         []byte{0xde, 0xad, 0xbe, 0xef},
		LastBlockHeight: 50,
	}
	mock := &mockBlockProducer{
		applyBlockFn: func(ctx context.Context, header types.Header, data *types.Data) (types.State, error) {
			return expectedState, errors.New("partial apply failure")
		},
	}
	producer, sr := setupBlockProducerTrace(t, mock)
	ctx := context.Background()

	header := types.Header{
		BaseHeader: types.BaseHeader{
			Height: 50,
		},
	}

	state, err := producer.ApplyBlock(ctx, header, &types.Data{})
	require.Error(t, err)
	require.Equal(t, "partial apply failure", err.Error())
	require.Equal(t, expectedState.AppHash, state.AppHash, "state should preserve AppHash when inner returns value with error")
	require.Equal(t, expectedState.LastBlockHeight, state.LastBlockHeight, "state should preserve LastBlockHeight when inner returns value with error")

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
}
