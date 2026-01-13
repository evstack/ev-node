package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
	"github.com/evstack/ev-node/types/pb/evnode/v1/v1connect"
)

type mockExecutorServiceHandler struct {
	initChainFn  func(context.Context, *connect.Request[pb.InitChainRequest]) (*connect.Response[pb.InitChainResponse], error)
	getTxsFn     func(context.Context, *connect.Request[pb.GetTxsRequest]) (*connect.Response[pb.GetTxsResponse], error)
	executeTxsFn func(context.Context, *connect.Request[pb.ExecuteTxsRequest]) (*connect.Response[pb.ExecuteTxsResponse], error)
	setFinalFn   func(context.Context, *connect.Request[pb.SetFinalRequest]) (*connect.Response[pb.SetFinalResponse], error)
}

func (m *mockExecutorServiceHandler) InitChain(ctx context.Context, req *connect.Request[pb.InitChainRequest]) (*connect.Response[pb.InitChainResponse], error) {
	if m.initChainFn != nil {
		return m.initChainFn(ctx, req)
	}
	return connect.NewResponse(&pb.InitChainResponse{}), nil
}

func (m *mockExecutorServiceHandler) GetTxs(ctx context.Context, req *connect.Request[pb.GetTxsRequest]) (*connect.Response[pb.GetTxsResponse], error) {
	if m.getTxsFn != nil {
		return m.getTxsFn(ctx, req)
	}
	return connect.NewResponse(&pb.GetTxsResponse{}), nil
}

func (m *mockExecutorServiceHandler) ExecuteTxs(ctx context.Context, req *connect.Request[pb.ExecuteTxsRequest]) (*connect.Response[pb.ExecuteTxsResponse], error) {
	if m.executeTxsFn != nil {
		return m.executeTxsFn(ctx, req)
	}
	return connect.NewResponse(&pb.ExecuteTxsResponse{}), nil
}

func (m *mockExecutorServiceHandler) SetFinal(ctx context.Context, req *connect.Request[pb.SetFinalRequest]) (*connect.Response[pb.SetFinalResponse], error) {
	if m.setFinalFn != nil {
		return m.setFinalFn(ctx, req)
	}
	return connect.NewResponse(&pb.SetFinalResponse{}), nil
}

var _ v1connect.ExecutorServiceHandler = (*mockExecutorServiceHandler)(nil)

func setupExecutorTrace(t *testing.T, inner v1connect.ExecutorServiceHandler) (v1connect.ExecutorServiceHandler, *tracetest.SpanRecorder) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	otel.SetTracerProvider(tp)
	return WithTracingServer(inner), sr
}

func TestTracedExecutorService_InitChain_Success(t *testing.T) {
	mock := &mockExecutorServiceHandler{
		initChainFn: func(ctx context.Context, req *connect.Request[pb.InitChainRequest]) (*connect.Response[pb.InitChainResponse], error) {
			return connect.NewResponse(&pb.InitChainResponse{
				StateRoot: []byte{0x01, 0x02, 0x03},
				MaxBytes:  1000,
			}), nil
		},
	}
	handler, sr := setupExecutorTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&pb.InitChainRequest{
		GenesisTime:   timestamppb.New(time.Now()),
		InitialHeight: 1,
		ChainId:       "test-chain",
	})

	res, err := handler.InitChain(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "ExecutorService.InitChain", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	requireAttribute(t, attrs, "chain_id", "test-chain")
	requireAttribute(t, attrs, "initial_height", int64(1))
	requireAttribute(t, attrs, "state_root", "010203")
	requireAttribute(t, attrs, "max_bytes", int64(1000))
}

func TestTracedExecutorService_InitChain_Error(t *testing.T) {
	mock := &mockExecutorServiceHandler{
		initChainFn: func(ctx context.Context, req *connect.Request[pb.InitChainRequest]) (*connect.Response[pb.InitChainResponse], error) {
			return nil, connect.NewError(connect.CodeInternal, errors.New("init failed"))
		},
	}
	handler, sr := setupExecutorTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&pb.InitChainRequest{
		GenesisTime:   timestamppb.New(time.Now()),
		InitialHeight: 1,
		ChainId:       "test-chain",
	})

	_, err := handler.InitChain(ctx, req)
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
}

func TestTracedExecutorService_GetTxs_Success(t *testing.T) {
	mock := &mockExecutorServiceHandler{
		getTxsFn: func(ctx context.Context, req *connect.Request[pb.GetTxsRequest]) (*connect.Response[pb.GetTxsResponse], error) {
			return connect.NewResponse(&pb.GetTxsResponse{
				Txs: [][]byte{[]byte("tx1"), []byte("tx2")},
			}), nil
		},
	}
	handler, sr := setupExecutorTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&pb.GetTxsRequest{})
	res, err := handler.GetTxs(ctx, req)
	require.NoError(t, err)
	require.Len(t, res.Msg.Txs, 2)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "ExecutorService.GetTxs", span.Name())

	attrs := span.Attributes()
	requireAttribute(t, attrs, "tx_count", 2)
}

func TestTracedExecutorService_GetTxs_Error(t *testing.T) {
	mock := &mockExecutorServiceHandler{
		getTxsFn: func(ctx context.Context, req *connect.Request[pb.GetTxsRequest]) (*connect.Response[pb.GetTxsResponse], error) {
			return nil, connect.NewError(connect.CodeInternal, errors.New("get txs failed"))
		},
	}
	handler, sr := setupExecutorTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&pb.GetTxsRequest{})
	_, err := handler.GetTxs(ctx, req)
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
}

func TestTracedExecutorService_ExecuteTxs_Success(t *testing.T) {
	mock := &mockExecutorServiceHandler{
		executeTxsFn: func(ctx context.Context, req *connect.Request[pb.ExecuteTxsRequest]) (*connect.Response[pb.ExecuteTxsResponse], error) {
			return connect.NewResponse(&pb.ExecuteTxsResponse{
				UpdatedStateRoot: []byte{0xaa, 0xbb, 0xcc},
				MaxBytes:         2000,
			}), nil
		},
	}
	handler, sr := setupExecutorTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&pb.ExecuteTxsRequest{
		Txs:           [][]byte{[]byte("tx1"), []byte("tx2"), []byte("tx3")},
		BlockHeight:   10,
		Timestamp:     timestamppb.New(time.Now()),
		PrevStateRoot: []byte{0x01, 0x02},
	})

	res, err := handler.ExecuteTxs(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "ExecutorService.ExecuteTxs", span.Name())

	attrs := span.Attributes()
	requireAttribute(t, attrs, "tx_count", 3)
	requireAttribute(t, attrs, "block_height", int64(10))
	requireAttribute(t, attrs, "prev_state_root", "0102")
	requireAttribute(t, attrs, "updated_state_root", "aabbcc")
	requireAttribute(t, attrs, "max_bytes", int64(2000))
}

func TestTracedExecutorService_ExecuteTxs_Error(t *testing.T) {
	mock := &mockExecutorServiceHandler{
		executeTxsFn: func(ctx context.Context, req *connect.Request[pb.ExecuteTxsRequest]) (*connect.Response[pb.ExecuteTxsResponse], error) {
			return nil, connect.NewError(connect.CodeInternal, errors.New("execution failed"))
		},
	}
	handler, sr := setupExecutorTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&pb.ExecuteTxsRequest{
		Txs:           [][]byte{[]byte("tx1")},
		BlockHeight:   10,
		Timestamp:     timestamppb.New(time.Now()),
		PrevStateRoot: []byte{0x01},
	})

	_, err := handler.ExecuteTxs(ctx, req)
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
}

func TestTracedExecutorService_SetFinal_Success(t *testing.T) {
	mock := &mockExecutorServiceHandler{
		setFinalFn: func(ctx context.Context, req *connect.Request[pb.SetFinalRequest]) (*connect.Response[pb.SetFinalResponse], error) {
			return connect.NewResponse(&pb.SetFinalResponse{}), nil
		},
	}
	handler, sr := setupExecutorTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&pb.SetFinalRequest{
		BlockHeight: 100,
	})

	res, err := handler.SetFinal(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "ExecutorService.SetFinal", span.Name())

	attrs := span.Attributes()
	requireAttribute(t, attrs, "block_height", int64(100))
}

func TestTracedExecutorService_SetFinal_Error(t *testing.T) {
	mock := &mockExecutorServiceHandler{
		setFinalFn: func(ctx context.Context, req *connect.Request[pb.SetFinalRequest]) (*connect.Response[pb.SetFinalResponse], error) {
			return nil, connect.NewError(connect.CodeInternal, errors.New("set final failed"))
		},
	}
	handler, sr := setupExecutorTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&pb.SetFinalRequest{
		BlockHeight: 100,
	})

	_, err := handler.SetFinal(ctx, req)
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
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
