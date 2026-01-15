package executing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/types"
)

// mockBlockProducer provides function hooks for testing the tracing decorator.
type mockBlockProducer struct {
	produceBlockFn  func(ctx context.Context) error
	retrieveBatchFn func(ctx context.Context) (*BatchData, error)
	createBlockFn   func(ctx context.Context, height uint64, batchData *BatchData) (*types.SignedHeader, *types.Data, error)
	applyBlockFn    func(ctx context.Context, header types.Header, data *types.Data) (types.State, error)
	validateBlockFn func(ctx context.Context, lastState types.State, header *types.SignedHeader, data *types.Data) error
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

func (m *mockBlockProducer) ValidateBlock(ctx context.Context, lastState types.State, header *types.SignedHeader, data *types.Data) error {
	if m.validateBlockFn != nil {
		return m.validateBlockFn(ctx, lastState, header, data)
	}
	return nil
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
	requireAttribute(t, attrs, "batch.tx_count", 2)
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
	requireAttribute(t, attrs, "block.height", int64(100))
	requireAttribute(t, attrs, "tx.count", 3)
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
	requireAttribute(t, attrs, "block.height", int64(50))
	requireAttribute(t, attrs, "tx.count", 2)
	requireAttribute(t, attrs, "state_root", "deadbeef")
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

func TestTracedBlockProducer_ValidateBlock_Success(t *testing.T) {
	mock := &mockBlockProducer{
		validateBlockFn: func(ctx context.Context, lastState types.State, header *types.SignedHeader, data *types.Data) error {
			return nil
		},
	}
	producer, sr := setupBlockProducerTrace(t, mock)
	ctx := context.Background()

	header := &types.SignedHeader{
		Header: types.Header{
			BaseHeader: types.BaseHeader{
				Height: 75,
			},
		},
	}

	err := producer.ValidateBlock(ctx, types.State{}, header, &types.Data{})
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "BlockExecutor.ValidateBlock", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	requireAttribute(t, attrs, "block.height", int64(75))
}

func TestTracedBlockProducer_ValidateBlock_Error(t *testing.T) {
	mock := &mockBlockProducer{
		validateBlockFn: func(ctx context.Context, lastState types.State, header *types.SignedHeader, data *types.Data) error {
			return errors.New("validation failed")
		},
	}
	producer, sr := setupBlockProducerTrace(t, mock)
	ctx := context.Background()

	header := &types.SignedHeader{
		Header: types.Header{
			BaseHeader: types.BaseHeader{
				Height: 75,
			},
		},
	}

	err := producer.ValidateBlock(ctx, types.State{}, header, &types.Data{})
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, "validation failed", span.Status().Description)
}

func requireAttribute(t *testing.T, attrs []attribute.KeyValue, key string, expected interface{}) {
	t.Helper()
	found := false
	for _, attr := range attrs {
		if string(attr.Key) == key {
			found = true
			switch v := expected.(type) {
			case string:
				require.Equal(t, v, attr.Value.AsString())
			case int64:
				require.Equal(t, v, attr.Value.AsInt64())
			case int:
				require.Equal(t, int64(v), attr.Value.AsInt64())
			default:
				t.Fatalf("unsupported attribute type: %T", expected)
			}
			break
		}
	}
	require.True(t, found, "attribute %s not found", key)
}
