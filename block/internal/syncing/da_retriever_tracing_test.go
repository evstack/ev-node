package syncing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/telemetry/testutil"
	"github.com/evstack/ev-node/types"
)

type mockDARetriever struct {
	retrieveFromDAFn func(ctx context.Context, daHeight uint64) ([]common.DAHeightEvent, error)
}

func (m *mockDARetriever) RetrieveFromDA(ctx context.Context, daHeight uint64) ([]common.DAHeightEvent, error) {
	if m.retrieveFromDAFn != nil {
		return m.retrieveFromDAFn(ctx, daHeight)
	}
	return nil, nil
}

func (m *mockDARetriever) QueuePriorityHeight(daHeight uint64) {}

func (m *mockDARetriever) PopPriorityHeight() uint64 { return 0 }

func setupDARetrieverTrace(t *testing.T, inner DARetriever) (DARetriever, *tracetest.SpanRecorder) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	otel.SetTracerProvider(tp)
	return WithTracingDARetriever(inner), sr
}

func TestTracedDARetriever_RetrieveFromDA_Success(t *testing.T) {
	mock := &mockDARetriever{
		retrieveFromDAFn: func(ctx context.Context, daHeight uint64) ([]common.DAHeightEvent, error) {
			return []common.DAHeightEvent{
				{
					Header: &types.SignedHeader{
						Header: types.Header{
							BaseHeader: types.BaseHeader{Height: 100},
						},
					},
					DaHeight: daHeight,
					Source:   common.SourceDA,
				},
				{
					Header: &types.SignedHeader{
						Header: types.Header{
							BaseHeader: types.BaseHeader{Height: 101},
						},
					},
					DaHeight: daHeight,
					Source:   common.SourceDA,
				},
			}, nil
		},
	}
	retriever, sr := setupDARetrieverTrace(t, mock)
	ctx := context.Background()

	events, err := retriever.RetrieveFromDA(ctx, 50)
	require.NoError(t, err)
	require.Len(t, events, 2)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "DARetriever.RetrieveFromDA", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "da.height", int64(50))
	testutil.RequireAttribute(t, attrs, "event.count", 2)
}

func TestTracedDARetriever_RetrieveFromDA_NoEvents(t *testing.T) {
	mock := &mockDARetriever{
		retrieveFromDAFn: func(ctx context.Context, daHeight uint64) ([]common.DAHeightEvent, error) {
			return []common.DAHeightEvent{}, nil
		},
	}
	retriever, sr := setupDARetrieverTrace(t, mock)
	ctx := context.Background()

	events, err := retriever.RetrieveFromDA(ctx, 50)
	require.NoError(t, err)
	require.Empty(t, events)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "event.count", 0)
}

func TestTracedDARetriever_RetrieveFromDA_Error(t *testing.T) {
	expectedErr := errors.New("DA retrieval failed")
	mock := &mockDARetriever{
		retrieveFromDAFn: func(ctx context.Context, daHeight uint64) ([]common.DAHeightEvent, error) {
			return nil, expectedErr
		},
	}
	retriever, sr := setupDARetrieverTrace(t, mock)
	ctx := context.Background()

	_, err := retriever.RetrieveFromDA(ctx, 50)
	require.Error(t, err)
	require.Equal(t, expectedErr, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, expectedErr.Error(), span.Status().Description)
}
