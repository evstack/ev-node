package submitting

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/telemetry/testutil"
	"github.com/evstack/ev-node/types"
)

type mockDASubmitterAPI struct {
	submitBlocksFn func(ctx context.Context, headers []*types.SignedHeader, data []*types.Data, cacheMgr cache.Manager, signer signer.Signer, onSubmitError func(error)) error
}

func (m *mockDASubmitterAPI) SubmitBlocks(ctx context.Context, headers []*types.SignedHeader, data []*types.Data, cacheMgr cache.Manager, signer signer.Signer, onSubmitError func(error)) error {
	if m.submitBlocksFn != nil {
		return m.submitBlocksFn(ctx, headers, data, cacheMgr, signer, onSubmitError)
	}
	return nil
}

func (m *mockDASubmitterAPI) Close() {}

func setupDASubmitterTrace(t *testing.T, inner DASubmitterAPI) (DASubmitterAPI, *tracetest.SpanRecorder) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	otel.SetTracerProvider(tp)
	return WithTracingDASubmitter(inner), sr
}

func TestTracedDASubmitter_SubmitBlocks_Success(t *testing.T) {
	mock := &mockDASubmitterAPI{
		submitBlocksFn: func(ctx context.Context, headers []*types.SignedHeader, data []*types.Data, cacheMgr cache.Manager, signer signer.Signer, onSubmitError func(error)) error {
			return nil
		},
	}
	submitter, sr := setupDASubmitterTrace(t, mock)
	ctx := context.Background()

	headers := []*types.SignedHeader{
		{Header: types.Header{BaseHeader: types.BaseHeader{Height: 100}}},
		{Header: types.Header{BaseHeader: types.BaseHeader{Height: 101}}},
		{Header: types.Header{BaseHeader: types.BaseHeader{Height: 102}}},
	}
	data := []*types.Data{
		{Metadata: &types.Metadata{Height: 100}},
		{Metadata: &types.Metadata{Height: 101}},
		{Metadata: &types.Metadata{Height: 102}},
	}

	err := submitter.SubmitBlocks(ctx, headers, data, nil, nil, nil)
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "DASubmitter.SubmitBlocks", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "block.count", 3)
	testutil.RequireAttribute(t, attrs, "block.start_height", int64(100))
	testutil.RequireAttribute(t, attrs, "block.end_height", int64(102))
}

func TestTracedDASubmitter_SubmitBlocks_Error(t *testing.T) {
	expectedErr := errors.New("DA submission failed")
	mock := &mockDASubmitterAPI{
		submitBlocksFn: func(ctx context.Context, headers []*types.SignedHeader, data []*types.Data, cacheMgr cache.Manager, signer signer.Signer, onSubmitError func(error)) error {
			return expectedErr
		},
	}
	submitter, sr := setupDASubmitterTrace(t, mock)
	ctx := context.Background()

	headers := []*types.SignedHeader{
		{Header: types.Header{BaseHeader: types.BaseHeader{Height: 100}}},
	}
	data := []*types.Data{
		{Metadata: &types.Metadata{Height: 100}},
	}

	err := submitter.SubmitBlocks(ctx, headers, data, nil, nil, nil)
	require.Error(t, err)
	require.Equal(t, expectedErr, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, expectedErr.Error(), span.Status().Description)
}

func TestTracedDASubmitter_SubmitBlocks_Empty(t *testing.T) {
	mock := &mockDASubmitterAPI{
		submitBlocksFn: func(ctx context.Context, headers []*types.SignedHeader, data []*types.Data, cacheMgr cache.Manager, signer signer.Signer, onSubmitError func(error)) error {
			return nil
		},
	}
	submitter, sr := setupDASubmitterTrace(t, mock)
	ctx := context.Background()

	err := submitter.SubmitBlocks(ctx, []*types.SignedHeader{}, []*types.Data{}, nil, nil, nil)
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]

	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "block.count", 0)
}
