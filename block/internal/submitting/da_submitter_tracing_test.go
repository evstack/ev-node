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
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/telemetry/testutil"
	"github.com/evstack/ev-node/types"
)

type mockDASubmitterAPI struct {
	submitHeadersFn func(ctx context.Context, headers []*types.SignedHeader, marshalledHeaders [][]byte, cache cache.Manager, signer signer.Signer) error
	submitDataFn    func(ctx context.Context, signedDataList []*types.SignedData, marshalledData [][]byte, cache cache.Manager, signer signer.Signer, genesis genesis.Genesis) error
}

func (m *mockDASubmitterAPI) SubmitHeaders(ctx context.Context, headers []*types.SignedHeader, marshalledHeaders [][]byte, cache cache.Manager, signer signer.Signer) error {
	if m.submitHeadersFn != nil {
		return m.submitHeadersFn(ctx, headers, marshalledHeaders, cache, signer)
	}
	return nil
}

func (m *mockDASubmitterAPI) SubmitData(ctx context.Context, signedDataList []*types.SignedData, marshalledData [][]byte, cache cache.Manager, signer signer.Signer, genesis genesis.Genesis) error {
	if m.submitDataFn != nil {
		return m.submitDataFn(ctx, signedDataList, marshalledData, cache, signer, genesis)
	}
	return nil
}

func setupDASubmitterTrace(t *testing.T, inner DASubmitterAPI) (DASubmitterAPI, *tracetest.SpanRecorder) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	otel.SetTracerProvider(tp)
	return WithTracingDASubmitter(inner), sr
}

func TestTracedDASubmitter_SubmitHeaders_Success(t *testing.T) {
	mock := &mockDASubmitterAPI{
		submitHeadersFn: func(ctx context.Context, headers []*types.SignedHeader, marshalledHeaders [][]byte, cache cache.Manager, signer signer.Signer) error {
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
	marshalledHeaders := [][]byte{
		[]byte("header1"),
		[]byte("header2"),
		[]byte("header3"),
	}

	err := submitter.SubmitHeaders(ctx, headers, marshalledHeaders, nil, nil)
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "DASubmitter.SubmitHeaders", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "header.count", 3)
	testutil.RequireAttribute(t, attrs, "header.total_bytes", 21) // 7+7+7
	testutil.RequireAttribute(t, attrs, "header.start_height", int64(100))
	testutil.RequireAttribute(t, attrs, "header.end_height", int64(102))
}

func TestTracedDASubmitter_SubmitHeaders_Error(t *testing.T) {
	expectedErr := errors.New("DA submission failed")
	mock := &mockDASubmitterAPI{
		submitHeadersFn: func(ctx context.Context, headers []*types.SignedHeader, marshalledHeaders [][]byte, cache cache.Manager, signer signer.Signer) error {
			return expectedErr
		},
	}
	submitter, sr := setupDASubmitterTrace(t, mock)
	ctx := context.Background()

	headers := []*types.SignedHeader{
		{Header: types.Header{BaseHeader: types.BaseHeader{Height: 100}}},
	}
	marshalledHeaders := [][]byte{[]byte("header1")}

	err := submitter.SubmitHeaders(ctx, headers, marshalledHeaders, nil, nil)
	require.Error(t, err)
	require.Equal(t, expectedErr, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, expectedErr.Error(), span.Status().Description)
}

func TestTracedDASubmitter_SubmitHeaders_Empty(t *testing.T) {
	mock := &mockDASubmitterAPI{
		submitHeadersFn: func(ctx context.Context, headers []*types.SignedHeader, marshalledHeaders [][]byte, cache cache.Manager, signer signer.Signer) error {
			return nil
		},
	}
	submitter, sr := setupDASubmitterTrace(t, mock)
	ctx := context.Background()

	err := submitter.SubmitHeaders(ctx, []*types.SignedHeader{}, [][]byte{}, nil, nil)
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]

	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "header.count", 0)
	testutil.RequireAttribute(t, attrs, "header.total_bytes", 0)
}

func TestTracedDASubmitter_SubmitData_Success(t *testing.T) {
	mock := &mockDASubmitterAPI{
		submitDataFn: func(ctx context.Context, signedDataList []*types.SignedData, marshalledData [][]byte, cache cache.Manager, signer signer.Signer, genesis genesis.Genesis) error {
			return nil
		},
	}
	submitter, sr := setupDASubmitterTrace(t, mock)
	ctx := context.Background()

	signedDataList := []*types.SignedData{
		{Data: types.Data{Metadata: &types.Metadata{Height: 100}}},
		{Data: types.Data{Metadata: &types.Metadata{Height: 101}}},
	}
	marshalledData := [][]byte{
		[]byte("data1data1"),
		[]byte("data2data2"),
	}

	err := submitter.SubmitData(ctx, signedDataList, marshalledData, nil, nil, genesis.Genesis{})
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "DASubmitter.SubmitData", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "data.count", 2)
	testutil.RequireAttribute(t, attrs, "data.total_bytes", 20) // 10+10
	testutil.RequireAttribute(t, attrs, "data.start_height", int64(100))
	testutil.RequireAttribute(t, attrs, "data.end_height", int64(101))
}

func TestTracedDASubmitter_SubmitData_Error(t *testing.T) {
	expectedErr := errors.New("data submission failed")
	mock := &mockDASubmitterAPI{
		submitDataFn: func(ctx context.Context, signedDataList []*types.SignedData, marshalledData [][]byte, cache cache.Manager, signer signer.Signer, genesis genesis.Genesis) error {
			return expectedErr
		},
	}
	submitter, sr := setupDASubmitterTrace(t, mock)
	ctx := context.Background()

	signedDataList := []*types.SignedData{
		{Data: types.Data{Metadata: &types.Metadata{Height: 100}}},
	}
	marshalledData := [][]byte{[]byte("data1")}

	err := submitter.SubmitData(ctx, signedDataList, marshalledData, nil, nil, genesis.Genesis{})
	require.Error(t, err)
	require.Equal(t, expectedErr, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, expectedErr.Error(), span.Status().Description)
}
