package server

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
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
	"github.com/evstack/ev-node/types/pb/evnode/v1/v1connect"
)

// mock implementations for StoreService
type mockStoreServiceHandler struct {
	getBlockFn           func(context.Context, *connect.Request[pb.GetBlockRequest]) (*connect.Response[pb.GetBlockResponse], error)
	getStateFn           func(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetStateResponse], error)
	getMetadataFn        func(context.Context, *connect.Request[pb.GetMetadataRequest]) (*connect.Response[pb.GetMetadataResponse], error)
	getGenesisDaHeightFn func(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetGenesisDaHeightResponse], error)
	getP2PStoreInfoFn    func(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetP2PStoreInfoResponse], error)
}

func (m *mockStoreServiceHandler) GetBlock(ctx context.Context, req *connect.Request[pb.GetBlockRequest]) (*connect.Response[pb.GetBlockResponse], error) {
	if m.getBlockFn != nil {
		return m.getBlockFn(ctx, req)
	}
	return connect.NewResponse(&pb.GetBlockResponse{}), nil
}

func (m *mockStoreServiceHandler) GetState(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetStateResponse], error) {
	if m.getStateFn != nil {
		return m.getStateFn(ctx, req)
	}
	return connect.NewResponse(&pb.GetStateResponse{}), nil
}

func (m *mockStoreServiceHandler) GetMetadata(ctx context.Context, req *connect.Request[pb.GetMetadataRequest]) (*connect.Response[pb.GetMetadataResponse], error) {
	if m.getMetadataFn != nil {
		return m.getMetadataFn(ctx, req)
	}
	return connect.NewResponse(&pb.GetMetadataResponse{}), nil
}

func (m *mockStoreServiceHandler) GetGenesisDaHeight(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetGenesisDaHeightResponse], error) {
	if m.getGenesisDaHeightFn != nil {
		return m.getGenesisDaHeightFn(ctx, req)
	}
	return connect.NewResponse(&pb.GetGenesisDaHeightResponse{}), nil
}

func (m *mockStoreServiceHandler) GetP2PStoreInfo(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetP2PStoreInfoResponse], error) {
	if m.getP2PStoreInfoFn != nil {
		return m.getP2PStoreInfoFn(ctx, req)
	}
	return connect.NewResponse(&pb.GetP2PStoreInfoResponse{}), nil
}

var _ v1connect.StoreServiceHandler = (*mockStoreServiceHandler)(nil)

// mock implementations for P2PService
type mockP2PServiceHandler struct {
	getPeerInfoFn func(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetPeerInfoResponse], error)
	getNetInfoFn  func(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetNetInfoResponse], error)
}

func (m *mockP2PServiceHandler) GetPeerInfo(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetPeerInfoResponse], error) {
	if m.getPeerInfoFn != nil {
		return m.getPeerInfoFn(ctx, req)
	}
	return connect.NewResponse(&pb.GetPeerInfoResponse{}), nil
}

func (m *mockP2PServiceHandler) GetNetInfo(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetNetInfoResponse], error) {
	if m.getNetInfoFn != nil {
		return m.getNetInfoFn(ctx, req)
	}
	return connect.NewResponse(&pb.GetNetInfoResponse{}), nil
}

var _ v1connect.P2PServiceHandler = (*mockP2PServiceHandler)(nil)

// mock implementations for ConfigService
type mockConfigServiceHandler struct {
	getNamespaceFn  func(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetNamespaceResponse], error)
	getSignerInfoFn func(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetSignerInfoResponse], error)
}

func (m *mockConfigServiceHandler) GetNamespace(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetNamespaceResponse], error) {
	if m.getNamespaceFn != nil {
		return m.getNamespaceFn(ctx, req)
	}
	return connect.NewResponse(&pb.GetNamespaceResponse{}), nil
}

func (m *mockConfigServiceHandler) GetSignerInfo(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetSignerInfoResponse], error) {
	if m.getSignerInfoFn != nil {
		return m.getSignerInfoFn(ctx, req)
	}
	return connect.NewResponse(&pb.GetSignerInfoResponse{}), nil
}

var _ v1connect.ConfigServiceHandler = (*mockConfigServiceHandler)(nil)

func setupStoreTrace(t *testing.T, inner v1connect.StoreServiceHandler) (v1connect.StoreServiceHandler, *tracetest.SpanRecorder) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	otel.SetTracerProvider(tp)
	return WithTracingStoreServer(inner), sr
}

func setupP2PTrace(t *testing.T, inner v1connect.P2PServiceHandler) (v1connect.P2PServiceHandler, *tracetest.SpanRecorder) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	otel.SetTracerProvider(tp)
	return WithTracingP2PServer(inner), sr
}

func setupConfigTrace(t *testing.T, inner v1connect.ConfigServiceHandler) (v1connect.ConfigServiceHandler, *tracetest.SpanRecorder) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	otel.SetTracerProvider(tp)
	return WithTracingConfigServer(inner), sr
}

// StoreService tests

func TestTracedStoreService_GetBlock_Success(t *testing.T) {
	mock := &mockStoreServiceHandler{
		getBlockFn: func(ctx context.Context, req *connect.Request[pb.GetBlockRequest]) (*connect.Response[pb.GetBlockResponse], error) {
			return connect.NewResponse(&pb.GetBlockResponse{
				Block: &pb.Block{
					Header: &pb.SignedHeader{
						Header: &pb.Header{
							Height: 10,
						},
					},
					Data: &pb.Data{
						Txs: [][]byte{[]byte("tx1"), []byte("tx2")},
					},
				},
			}), nil
		},
	}
	handler, sr := setupStoreTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&pb.GetBlockRequest{
		Identifier: &pb.GetBlockRequest_Height{Height: 10},
	})

	res, err := handler.GetBlock(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "StoreService.GetBlock", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	requireAttribute(t, attrs, "height", int64(10))
	requireAttribute(t, attrs, "found", true)
	requireAttribute(t, attrs, "tx_count", 2)
}

func TestTracedStoreService_GetBlock_Error(t *testing.T) {
	mock := &mockStoreServiceHandler{
		getBlockFn: func(ctx context.Context, req *connect.Request[pb.GetBlockRequest]) (*connect.Response[pb.GetBlockResponse], error) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("block not found"))
		},
	}
	handler, sr := setupStoreTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&pb.GetBlockRequest{
		Identifier: &pb.GetBlockRequest_Height{Height: 999},
	})

	_, err := handler.GetBlock(ctx, req)
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
}

func TestTracedStoreService_GetState_Success(t *testing.T) {
	mock := &mockStoreServiceHandler{
		getStateFn: func(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetStateResponse], error) {
			return connect.NewResponse(&pb.GetStateResponse{
				State: &pb.State{
					LastBlockHeight: 100,
					AppHash:         []byte{0xaa, 0xbb},
					DaHeight:        50,
					LastBlockTime:   timestamppb.New(time.Now()),
				},
			}), nil
		},
	}
	handler, sr := setupStoreTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&emptypb.Empty{})
	res, err := handler.GetState(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "StoreService.GetState", span.Name())

	attrs := span.Attributes()
	requireAttribute(t, attrs, "height", int64(100))
	requireAttribute(t, attrs, "app_hash", "aabb")
	requireAttribute(t, attrs, "da_height", int64(50))
}

func TestTracedStoreService_GetMetadata_Success(t *testing.T) {
	mock := &mockStoreServiceHandler{
		getMetadataFn: func(ctx context.Context, req *connect.Request[pb.GetMetadataRequest]) (*connect.Response[pb.GetMetadataResponse], error) {
			return connect.NewResponse(&pb.GetMetadataResponse{
				Value: []byte("metadata_value"),
			}), nil
		},
	}
	handler, sr := setupStoreTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&pb.GetMetadataRequest{
		Key: "test_key",
	})

	res, err := handler.GetMetadata(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "StoreService.GetMetadata", span.Name())

	attrs := span.Attributes()
	requireAttribute(t, attrs, "key", "test_key")
	requireAttribute(t, attrs, "value_size_bytes", 14)
}

func TestTracedStoreService_GetGenesisDaHeight_Success(t *testing.T) {
	mock := &mockStoreServiceHandler{
		getGenesisDaHeightFn: func(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetGenesisDaHeightResponse], error) {
			return connect.NewResponse(&pb.GetGenesisDaHeightResponse{
				Height: 1000,
			}), nil
		},
	}
	handler, sr := setupStoreTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&emptypb.Empty{})
	res, err := handler.GetGenesisDaHeight(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "StoreService.GetGenesisDaHeight", span.Name())

	attrs := span.Attributes()
	requireAttribute(t, attrs, "genesis_da_height", int64(1000))
}

func TestTracedStoreService_GetP2PStoreInfo_Success(t *testing.T) {
	mock := &mockStoreServiceHandler{
		getP2PStoreInfoFn: func(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetP2PStoreInfoResponse], error) {
			return connect.NewResponse(&pb.GetP2PStoreInfoResponse{
				Stores: []*pb.P2PStoreSnapshot{
					{Label: "Header Store", Height: 100},
					{Label: "Data Store", Height: 99},
				},
			}), nil
		},
	}
	handler, sr := setupStoreTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&emptypb.Empty{})
	res, err := handler.GetP2PStoreInfo(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "StoreService.GetP2PStoreInfo", span.Name())

	attrs := span.Attributes()
	requireAttribute(t, attrs, "store_count", 2)
}

// P2PService tests

func TestTracedP2PService_GetPeerInfo_Success(t *testing.T) {
	mock := &mockP2PServiceHandler{
		getPeerInfoFn: func(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetPeerInfoResponse], error) {
			return connect.NewResponse(&pb.GetPeerInfoResponse{
				Peers: []*pb.PeerInfo{
					{Id: "peer1", Address: "addr1"},
					{Id: "peer2", Address: "addr2"},
				},
			}), nil
		},
	}
	handler, sr := setupP2PTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&emptypb.Empty{})
	res, err := handler.GetPeerInfo(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "P2PService.GetPeerInfo", span.Name())

	attrs := span.Attributes()
	requireAttribute(t, attrs, "peer_count", 2)
}

func TestTracedP2PService_GetPeerInfo_Error(t *testing.T) {
	mock := &mockP2PServiceHandler{
		getPeerInfoFn: func(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetPeerInfoResponse], error) {
			return nil, connect.NewError(connect.CodeInternal, errors.New("failed to get peers"))
		},
	}
	handler, sr := setupP2PTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&emptypb.Empty{})
	_, err := handler.GetPeerInfo(ctx, req)
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
}

func TestTracedP2PService_GetNetInfo_Success(t *testing.T) {
	mock := &mockP2PServiceHandler{
		getNetInfoFn: func(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetNetInfoResponse], error) {
			return connect.NewResponse(&pb.GetNetInfoResponse{
				NetInfo: &pb.NetInfo{
					Id:              "node123",
					ListenAddresses: []string{"/ip4/127.0.0.1/tcp/26656"},
				},
			}), nil
		},
	}
	handler, sr := setupP2PTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&emptypb.Empty{})
	res, err := handler.GetNetInfo(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "P2PService.GetNetInfo", span.Name())

	attrs := span.Attributes()
	requireAttribute(t, attrs, "node_id", "node123")
	requireAttribute(t, attrs, "listen_address_count", 1)
}

// ConfigService tests

func TestTracedConfigService_GetNamespace_Success(t *testing.T) {
	mock := &mockConfigServiceHandler{
		getNamespaceFn: func(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetNamespaceResponse], error) {
			return connect.NewResponse(&pb.GetNamespaceResponse{
				HeaderNamespace: "0x0001020304050607",
				DataNamespace:   "0x08090a0b0c0d0e0f",
			}), nil
		},
	}
	handler, sr := setupConfigTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&emptypb.Empty{})
	res, err := handler.GetNamespace(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "ConfigService.GetNamespace", span.Name())

	attrs := span.Attributes()
	requireAttribute(t, attrs, "header_namespace", "0x0001020304050607")
	requireAttribute(t, attrs, "data_namespace", "0x08090a0b0c0d0e0f")
}

func TestTracedConfigService_GetNamespace_Error(t *testing.T) {
	mock := &mockConfigServiceHandler{
		getNamespaceFn: func(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetNamespaceResponse], error) {
			return nil, connect.NewError(connect.CodeInternal, errors.New("failed to get namespace"))
		},
	}
	handler, sr := setupConfigTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&emptypb.Empty{})
	_, err := handler.GetNamespace(ctx, req)
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
}

func TestTracedConfigService_GetSignerInfo_Success(t *testing.T) {
	mock := &mockConfigServiceHandler{
		getSignerInfoFn: func(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetSignerInfoResponse], error) {
			return connect.NewResponse(&pb.GetSignerInfoResponse{
				Address: []byte{0x01, 0x02, 0x03, 0x04},
			}), nil
		},
	}
	handler, sr := setupConfigTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&emptypb.Empty{})
	res, err := handler.GetSignerInfo(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, "ConfigService.GetSignerInfo", span.Name())

	attrs := span.Attributes()
	requireAttribute(t, attrs, "signer_address", "01020304")
}

func TestTracedConfigService_GetSignerInfo_Error(t *testing.T) {
	mock := &mockConfigServiceHandler{
		getSignerInfoFn: func(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[pb.GetSignerInfoResponse], error) {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("signer not available"))
		},
	}
	handler, sr := setupConfigTrace(t, mock)
	ctx := context.Background()

	req := connect.NewRequest(&emptypb.Empty{})
	_, err := handler.GetSignerInfo(ctx, req)
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
			case bool:
				require.Equal(t, v, attr.Value.AsBool())
			default:
				t.Fatalf("unsupported attribute type: %T", expected)
			}
			break
		}
	}
	require.True(t, found, "attribute %s not found", key)
}
