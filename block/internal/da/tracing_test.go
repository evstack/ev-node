package da

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/telemetry/testutil"
)

// mockFullClient provides function hooks for testing the tracing decorator.
type mockFullClient struct {
	submitFn    func(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) datypes.ResultSubmit
	retrieveFn  func(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve
	getFn       func(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error)
	getProofsFn func(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Proof, error)
	validateFn  func(ctx context.Context, ids []datypes.ID, proofs []datypes.Proof, namespace []byte) ([]bool, error)
}

func (m *mockFullClient) Submit(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) datypes.ResultSubmit {
	if m.submitFn != nil {
		return m.submitFn(ctx, data, gasPrice, namespace, options)
	}
	return datypes.ResultSubmit{}
}
func (m *mockFullClient) Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	if m.retrieveFn != nil {
		return m.retrieveFn(ctx, height, namespace)
	}
	return datypes.ResultRetrieve{}
}
func (m *mockFullClient) Get(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error) {
	if m.getFn != nil {
		return m.getFn(ctx, ids, namespace)
	}
	return nil, nil
}
func (m *mockFullClient) GetProofs(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Proof, error) {
	if m.getProofsFn != nil {
		return m.getProofsFn(ctx, ids, namespace)
	}
	return nil, nil
}
func (m *mockFullClient) Validate(ctx context.Context, ids []datypes.ID, proofs []datypes.Proof, namespace []byte) ([]bool, error) {
	if m.validateFn != nil {
		return m.validateFn(ctx, ids, proofs, namespace)
	}
	return nil, nil
}
func (m *mockFullClient) GetLatestDAHeight(_ context.Context) (uint64, error) { return 0, nil }
func (m *mockFullClient) GetHeaderNamespace() []byte                          { return []byte{0x01} }
func (m *mockFullClient) GetDataNamespace() []byte                            { return []byte{0x02} }
func (m *mockFullClient) GetForcedInclusionNamespace() []byte                 { return []byte{0x03} }
func (m *mockFullClient) HasForcedInclusionNamespace() bool                   { return true }

// setup a tracer provider + span recorder
func setupDATrace(t *testing.T, inner FullClient) (FullClient, *tracetest.SpanRecorder) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	otel.SetTracerProvider(tp)
	return WithTracingClient(inner), sr
}

func TestTracedDA_Submit_Success(t *testing.T) {
	mock := &mockFullClient{
		submitFn: func(ctx context.Context, data [][]byte, _ float64, _ []byte, _ []byte) datypes.ResultSubmit {
			return datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Height: 123}}
		},
	}
	client, sr := setupDATrace(t, mock)
	ctx := context.Background()

	_ = client.Submit(ctx, [][]byte{[]byte("a"), []byte("bc")}, -1.0, []byte{0xaa, 0xbb}, nil)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "DA.Submit", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "blob.count", 2)
	testutil.RequireAttribute(t, attrs, "blob.total_size_bytes", 3)
	// namespace hex string length assertion
	// 2 bytes = 4 hex characters
	foundNS := false
	for _, a := range attrs {
		if string(a.Key) == "da.namespace" {
			foundNS = true
			require.Equal(t, 4, len(a.Value.AsString()))
		}
	}
	require.True(t, foundNS, "attribute da.namespace not found")
}

func TestTracedDA_Submit_Error(t *testing.T) {
	mock := &mockFullClient{
		submitFn: func(ctx context.Context, data [][]byte, _ float64, _ []byte, _ []byte) datypes.ResultSubmit {
			return datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusError, Message: "boom"}}
		},
	}
	client, sr := setupDATrace(t, mock)
	ctx := context.Background()

	_ = client.Submit(ctx, [][]byte{[]byte("a")}, -1.0, []byte{0xaa}, nil)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, "boom", span.Status().Description)
}

func TestTracedDA_Retrieve_Success(t *testing.T) {
	mock := &mockFullClient{
		retrieveFn: func(ctx context.Context, height uint64, _ []byte) datypes.ResultRetrieve {
			return datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Height: height}, Data: []datypes.Blob{{}, {}}}
		},
	}
	client, sr := setupDATrace(t, mock)
	ctx := context.Background()

	_ = client.Retrieve(ctx, 42, []byte{0x01})
	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "DA.Retrieve", span.Name())
	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "ns.length", 1)
	testutil.RequireAttribute(t, attrs, "blob.count", 2)
}

func TestTracedDA_Retrieve_Error(t *testing.T) {
	mock := &mockFullClient{
		retrieveFn: func(ctx context.Context, height uint64, _ []byte) datypes.ResultRetrieve {
			return datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusError, Message: "oops"}}
		},
	}
	client, sr := setupDATrace(t, mock)
	ctx := context.Background()

	_ = client.Retrieve(ctx, 7, []byte{0x02})
	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, "oops", span.Status().Description)
}

func TestTracedDA_Get_Success(t *testing.T) {
	mock := &mockFullClient{
		getFn: func(ctx context.Context, ids []datypes.ID, _ []byte) ([]datypes.Blob, error) {
			return []datypes.Blob{{}, {}}, nil
		},
	}
	client, sr := setupDATrace(t, mock)
	ctx := context.Background()
	ids := []datypes.ID{[]byte{0x01}, []byte{0x02}}

	blobs, err := client.Get(ctx, ids, []byte{0x01})
	require.NoError(t, err)
	require.Len(t, blobs, 2)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "DA.Get", span.Name())
	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "id.count", 2)
	testutil.RequireAttribute(t, attrs, "blob.count", 2)
}

func TestTracedDA_Get_Error(t *testing.T) {
	mock := &mockFullClient{
		getFn: func(ctx context.Context, ids []datypes.ID, _ []byte) ([]datypes.Blob, error) {
			return nil, errors.New("get failed")
		},
	}
	client, sr := setupDATrace(t, mock)
	ctx := context.Background()
	ids := []datypes.ID{[]byte{0x01}}

	_, err := client.Get(ctx, ids, []byte{0x01})
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, "get failed", span.Status().Description)
}
