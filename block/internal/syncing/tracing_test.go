package syncing

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

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/types"
)

// mockBlockSyncer provides function hooks for testing the tracing decorator.
type mockBlockSyncer struct {
	trySyncNextBlockFn      func(ctx context.Context, event *common.DAHeightEvent) error
	applyBlockFn            func(ctx context.Context, header types.Header, data *types.Data, currentState types.State) (types.State, error)
	validateBlockFn         func(ctx context.Context, currState types.State, data *types.Data, header *types.SignedHeader) error
	verifyForcedInclusionFn func(ctx context.Context, currentState types.State, data *types.Data) error
}

func (m *mockBlockSyncer) TrySyncNextBlock(ctx context.Context, event *common.DAHeightEvent) error {
	if m.trySyncNextBlockFn != nil {
		return m.trySyncNextBlockFn(ctx, event)
	}
	return nil
}

func (m *mockBlockSyncer) ApplyBlock(ctx context.Context, header types.Header, data *types.Data, currentState types.State) (types.State, error) {
	if m.applyBlockFn != nil {
		return m.applyBlockFn(ctx, header, data, currentState)
	}
	return types.State{}, nil
}

func (m *mockBlockSyncer) ValidateBlock(ctx context.Context, currState types.State, data *types.Data, header *types.SignedHeader) error {
	if m.validateBlockFn != nil {
		return m.validateBlockFn(ctx, currState, data, header)
	}
	return nil
}

func (m *mockBlockSyncer) VerifyForcedInclusionTxs(ctx context.Context, currentState types.State, data *types.Data) error {
	if m.verifyForcedInclusionFn != nil {
		return m.verifyForcedInclusionFn(ctx, currentState, data)
	}
	return nil
}

func setupBlockSyncerTrace(t *testing.T, inner BlockSyncer) (BlockSyncer, *tracetest.SpanRecorder) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	otel.SetTracerProvider(tp)
	return WithTracingBlockSyncer(inner), sr
}

func TestTracedBlockSyncer_TrySyncNextBlock_Success(t *testing.T) {
	mock := &mockBlockSyncer{
		trySyncNextBlockFn: func(ctx context.Context, event *common.DAHeightEvent) error {
			return nil
		},
	}
	syncer, sr := setupBlockSyncerTrace(t, mock)
	ctx := context.Background()

	event := &common.DAHeightEvent{
		Header: &types.SignedHeader{
			Header: types.Header{
				BaseHeader: types.BaseHeader{
					Height: 100,
				},
			},
		},
		DaHeight: 50,
		Source:   common.SourceDA,
	}

	err := syncer.TrySyncNextBlock(ctx, event)
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "BlockSyncer.SyncBlock", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	requireAttribute(t, attrs, "block.height", int64(100))
	requireAttribute(t, attrs, "da.height", int64(50))
	requireAttribute(t, attrs, "source", string(common.SourceDA))
}

func TestTracedBlockSyncer_TrySyncNextBlock_Error(t *testing.T) {
	mock := &mockBlockSyncer{
		trySyncNextBlockFn: func(ctx context.Context, event *common.DAHeightEvent) error {
			return errors.New("sync failed")
		},
	}
	syncer, sr := setupBlockSyncerTrace(t, mock)
	ctx := context.Background()

	event := &common.DAHeightEvent{
		Header: &types.SignedHeader{
			Header: types.Header{
				BaseHeader: types.BaseHeader{
					Height: 100,
				},
			},
		},
		DaHeight: 50,
		Source:   common.SourceDA,
	}

	err := syncer.TrySyncNextBlock(ctx, event)
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, "sync failed", span.Status().Description)
}

func TestTracedBlockSyncer_ApplyBlock_Success(t *testing.T) {
	mock := &mockBlockSyncer{
		applyBlockFn: func(ctx context.Context, header types.Header, data *types.Data, currentState types.State) (types.State, error) {
			return types.State{
				AppHash: []byte{0xde, 0xad, 0xbe, 0xef},
			}, nil
		},
	}
	syncer, sr := setupBlockSyncerTrace(t, mock)
	ctx := context.Background()

	header := types.Header{
		BaseHeader: types.BaseHeader{
			Height: 50,
		},
	}
	data := &types.Data{
		Txs: types.Txs{[]byte("tx1"), []byte("tx2")},
	}

	state, err := syncer.ApplyBlock(ctx, header, data, types.State{})
	require.NoError(t, err)
	require.NotEmpty(t, state.AppHash)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "BlockSyncer.ApplyBlock", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	requireAttribute(t, attrs, "block.height", int64(50))
	requireAttribute(t, attrs, "tx.count", 2)
	requireAttribute(t, attrs, "state_root", "deadbeef")
}

func TestTracedBlockSyncer_ApplyBlock_Error(t *testing.T) {
	mock := &mockBlockSyncer{
		applyBlockFn: func(ctx context.Context, header types.Header, data *types.Data, currentState types.State) (types.State, error) {
			return types.State{}, errors.New("execution failed")
		},
	}
	syncer, sr := setupBlockSyncerTrace(t, mock)
	ctx := context.Background()

	header := types.Header{
		BaseHeader: types.BaseHeader{
			Height: 50,
		},
	}

	_, err := syncer.ApplyBlock(ctx, header, &types.Data{}, types.State{})
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, "execution failed", span.Status().Description)
}

func TestTracedBlockSyncer_ValidateBlock_Success(t *testing.T) {
	mock := &mockBlockSyncer{
		validateBlockFn: func(ctx context.Context, currState types.State, data *types.Data, header *types.SignedHeader) error {
			return nil
		},
	}
	syncer, sr := setupBlockSyncerTrace(t, mock)
	ctx := context.Background()

	header := &types.SignedHeader{
		Header: types.Header{
			BaseHeader: types.BaseHeader{
				Height: 75,
			},
		},
	}

	err := syncer.ValidateBlock(ctx, types.State{}, &types.Data{}, header)
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "BlockSyncer.ValidateBlock", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	requireAttribute(t, attrs, "block.height", int64(75))
}

func TestTracedBlockSyncer_ValidateBlock_Error(t *testing.T) {
	mock := &mockBlockSyncer{
		validateBlockFn: func(ctx context.Context, currState types.State, data *types.Data, header *types.SignedHeader) error {
			return errors.New("validation failed")
		},
	}
	syncer, sr := setupBlockSyncerTrace(t, mock)
	ctx := context.Background()

	header := &types.SignedHeader{
		Header: types.Header{
			BaseHeader: types.BaseHeader{
				Height: 75,
			},
		},
	}

	err := syncer.ValidateBlock(ctx, types.State{}, &types.Data{}, header)
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, "validation failed", span.Status().Description)
}

func TestTracedBlockSyncer_VerifyForcedInclusionTxs_Success(t *testing.T) {
	mock := &mockBlockSyncer{
		verifyForcedInclusionFn: func(ctx context.Context, currentState types.State, data *types.Data) error {
			return nil
		},
	}
	syncer, sr := setupBlockSyncerTrace(t, mock)
	ctx := context.Background()

	data := &types.Data{
		Metadata: &types.Metadata{
			Height: 100,
		},
	}
	state := types.State{
		DAHeight: 50,
	}

	err := syncer.VerifyForcedInclusionTxs(ctx, state, data)
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "BlockSyncer.VerifyForcedInclusion", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	requireAttribute(t, attrs, "block.height", int64(100))
	requireAttribute(t, attrs, "da.height", int64(50))
}

func TestTracedBlockSyncer_VerifyForcedInclusionTxs_Error(t *testing.T) {
	mock := &mockBlockSyncer{
		verifyForcedInclusionFn: func(ctx context.Context, currentState types.State, data *types.Data) error {
			return errors.New("forced inclusion verification failed")
		},
	}
	syncer, sr := setupBlockSyncerTrace(t, mock)
	ctx := context.Background()

	data := &types.Data{
		Metadata: &types.Metadata{
			Height: 100,
		},
	}
	state := types.State{
		DAHeight: 50,
	}

	err := syncer.VerifyForcedInclusionTxs(ctx, state, data)
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, "forced inclusion verification failed", span.Status().Description)
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
