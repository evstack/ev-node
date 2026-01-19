package block

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/evstack/ev-node/pkg/telemetry/testutil"
)

type mockForcedInclusionRetriever struct {
	retrieveFn func(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error)
	stopCalled bool
}

func (m *mockForcedInclusionRetriever) RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error) {
	if m.retrieveFn != nil {
		return m.retrieveFn(ctx, daHeight)
	}
	return nil, nil
}

func (m *mockForcedInclusionRetriever) Stop() {
	m.stopCalled = true
}

func setupForcedInclusionTrace(t *testing.T, inner ForcedInclusionRetriever) (ForcedInclusionRetriever, *tracetest.SpanRecorder) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	otel.SetTracerProvider(tp)
	return WithTracingForcedInclusionRetriever(inner), sr
}

func TestTracedForcedInclusionRetriever_RetrieveForcedIncludedTxs_Success(t *testing.T) {
	expectedEvent := &ForcedInclusionEvent{
		Timestamp:     time.Now(),
		StartDaHeight: 100,
		EndDaHeight:   109,
		Txs:           [][]byte{[]byte("tx1"), []byte("tx2"), []byte("tx3")},
	}

	mock := &mockForcedInclusionRetriever{
		retrieveFn: func(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error) {
			return expectedEvent, nil
		},
	}
	retriever, sr := setupForcedInclusionTrace(t, mock)
	ctx := context.Background()

	event, err := retriever.RetrieveForcedIncludedTxs(ctx, 109)
	require.NoError(t, err)
	require.Equal(t, expectedEvent, event)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "ForcedInclusionRetriever.RetrieveForcedIncludedTxs", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "da.height", int64(109))
	testutil.RequireAttribute(t, attrs, "event.start_da_height", int64(100))
	testutil.RequireAttribute(t, attrs, "event.end_da_height", int64(109))
	testutil.RequireAttribute(t, attrs, "event.tx_count", 3)
}

func TestTracedForcedInclusionRetriever_RetrieveForcedIncludedTxs_Error(t *testing.T) {
	expectedErr := errors.New("retrieval failed")
	mock := &mockForcedInclusionRetriever{
		retrieveFn: func(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error) {
			return nil, expectedErr
		},
	}
	retriever, sr := setupForcedInclusionTrace(t, mock)
	ctx := context.Background()

	event, err := retriever.RetrieveForcedIncludedTxs(ctx, 100)
	require.Error(t, err)
	require.Equal(t, expectedErr, err)
	require.Nil(t, event)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, expectedErr.Error(), span.Status().Description)
}

func TestTracedForcedInclusionRetriever_RetrieveForcedIncludedTxs_NilEvent(t *testing.T) {
	mock := &mockForcedInclusionRetriever{
		retrieveFn: func(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error) {
			return nil, nil
		},
	}
	retriever, sr := setupForcedInclusionTrace(t, mock)
	ctx := context.Background()

	event, err := retriever.RetrieveForcedIncludedTxs(ctx, 100)
	require.NoError(t, err)
	require.Nil(t, event)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Unset, span.Status().Code)

	// only da.height should be present since event is nil
	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "da.height", int64(100))
}

func TestTracedForcedInclusionRetriever_RetrieveForcedIncludedTxs_EmptyTxs(t *testing.T) {
	expectedEvent := &ForcedInclusionEvent{
		Timestamp:     time.Now(),
		StartDaHeight: 100,
		EndDaHeight:   100,
		Txs:           [][]byte{},
	}

	mock := &mockForcedInclusionRetriever{
		retrieveFn: func(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error) {
			return expectedEvent, nil
		},
	}
	retriever, sr := setupForcedInclusionTrace(t, mock)
	ctx := context.Background()

	event, err := retriever.RetrieveForcedIncludedTxs(ctx, 100)
	require.NoError(t, err)
	require.Equal(t, expectedEvent, event)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]

	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "event.tx_count", 0)
}

func TestTracedForcedInclusionRetriever_Stop(t *testing.T) {
	mock := &mockForcedInclusionRetriever{}
	retriever, _ := setupForcedInclusionTrace(t, mock)

	retriever.Stop()
	require.True(t, mock.stopCalled)
}
