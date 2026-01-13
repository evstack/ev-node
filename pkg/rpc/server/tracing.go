package server

import (
	"context"
	"encoding/hex"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
	"github.com/evstack/ev-node/types/pb/evnode/v1/v1connect"
)

// tracedStoreServer decorates a StoreServiceHandler with OpenTelemetry spans.
type tracedStoreServer struct {
	inner  v1connect.StoreServiceHandler
	tracer trace.Tracer
}

// WithTracingStoreServer decorates the provided store service handler with tracing spans.
func WithTracingStoreServer(inner v1connect.StoreServiceHandler) v1connect.StoreServiceHandler {
	return &tracedStoreServer{
		inner:  inner,
		tracer: otel.Tracer("ev-node/store-service"),
	}
}

func (t *tracedStoreServer) GetBlock(
	ctx context.Context,
	req *connect.Request[pb.GetBlockRequest],
) (*connect.Response[pb.GetBlockResponse], error) {
	var height uint64
	switch identifier := req.Msg.Identifier.(type) {
	case *pb.GetBlockRequest_Height:
		height = identifier.Height
	case *pb.GetBlockRequest_Hash:
		// for hash-based queries, we'll add the hash as an attribute
	}

	ctx, span := t.tracer.Start(ctx, "StoreService.GetBlock",
		trace.WithAttributes(
			attribute.Int64("height", int64(height)),
		),
	)
	defer span.End()

	// add hash attribute if hash-based query
	if hashIdentifier, ok := req.Msg.Identifier.(*pb.GetBlockRequest_Hash); ok {
		span.SetAttributes(attribute.String("hash", hex.EncodeToString(hashIdentifier.Hash)))
	}

	res, err := t.inner.GetBlock(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.Bool("found", res.Msg.Block != nil),
	)
	if res.Msg.Block != nil && res.Msg.Block.Data != nil {
		totalSize := 0
		for _, tx := range res.Msg.Block.Data.Txs {
			totalSize += len(tx)
		}
		span.SetAttributes(
			attribute.Int("block_size_bytes", totalSize),
			attribute.Int("tx_count", len(res.Msg.Block.Data.Txs)),
		)
	}
	return res, nil
}

func (t *tracedStoreServer) GetState(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[pb.GetStateResponse], error) {
	ctx, span := t.tracer.Start(ctx, "StoreService.GetState")
	defer span.End()

	res, err := t.inner.GetState(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if res.Msg.State != nil {
		span.SetAttributes(
			attribute.Int64("height", int64(res.Msg.State.LastBlockHeight)),
			attribute.String("app_hash", hex.EncodeToString(res.Msg.State.AppHash)),
			attribute.Int64("da_height", int64(res.Msg.State.DaHeight)),
		)
	}
	return res, nil
}

func (t *tracedStoreServer) GetMetadata(
	ctx context.Context,
	req *connect.Request[pb.GetMetadataRequest],
) (*connect.Response[pb.GetMetadataResponse], error) {
	ctx, span := t.tracer.Start(ctx, "StoreService.GetMetadata",
		trace.WithAttributes(
			attribute.String("key", req.Msg.Key),
		),
	)
	defer span.End()

	res, err := t.inner.GetMetadata(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("value_size_bytes", len(res.Msg.Value)),
	)
	return res, nil
}

func (t *tracedStoreServer) GetGenesisDaHeight(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[pb.GetGenesisDaHeightResponse], error) {
	ctx, span := t.tracer.Start(ctx, "StoreService.GetGenesisDaHeight")
	defer span.End()

	res, err := t.inner.GetGenesisDaHeight(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.Int64("genesis_da_height", int64(res.Msg.Height)),
	)
	return res, nil
}

func (t *tracedStoreServer) GetP2PStoreInfo(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[pb.GetP2PStoreInfoResponse], error) {
	ctx, span := t.tracer.Start(ctx, "StoreService.GetP2PStoreInfo")
	defer span.End()

	res, err := t.inner.GetP2PStoreInfo(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("store_count", len(res.Msg.Stores)),
	)
	for i, store := range res.Msg.Stores {
		span.SetAttributes(
			attribute.Int64("store_"+string(rune(i))+"_height", int64(store.Height)),
		)
	}
	return res, nil
}

// tracedP2PServer decorates a P2PServiceHandler with OpenTelemetry spans.
type tracedP2PServer struct {
	inner  v1connect.P2PServiceHandler
	tracer trace.Tracer
}

// WithTracingP2PServer decorates the provided P2P service handler with tracing spans.
func WithTracingP2PServer(inner v1connect.P2PServiceHandler) v1connect.P2PServiceHandler {
	return &tracedP2PServer{
		inner:  inner,
		tracer: otel.Tracer("ev-node/p2p-service"),
	}
}

func (t *tracedP2PServer) GetPeerInfo(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[pb.GetPeerInfoResponse], error) {
	ctx, span := t.tracer.Start(ctx, "P2PService.GetPeerInfo")
	defer span.End()

	res, err := t.inner.GetPeerInfo(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("peer_count", len(res.Msg.Peers)),
	)
	return res, nil
}

func (t *tracedP2PServer) GetNetInfo(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[pb.GetNetInfoResponse], error) {
	ctx, span := t.tracer.Start(ctx, "P2PService.GetNetInfo")
	defer span.End()

	res, err := t.inner.GetNetInfo(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if res.Msg.NetInfo != nil {
		span.SetAttributes(
			attribute.String("node_id", res.Msg.NetInfo.Id),
			attribute.Int("listen_address_count", len(res.Msg.NetInfo.ListenAddresses)),
		)
	}
	return res, nil
}

// tracedConfigServer decorates a ConfigServiceHandler with OpenTelemetry spans.
type tracedConfigServer struct {
	inner  v1connect.ConfigServiceHandler
	tracer trace.Tracer
}

// WithTracingConfigServer decorates the provided config service handler with tracing spans.
func WithTracingConfigServer(inner v1connect.ConfigServiceHandler) v1connect.ConfigServiceHandler {
	return &tracedConfigServer{
		inner:  inner,
		tracer: otel.Tracer("ev-node/config-service"),
	}
}

func (t *tracedConfigServer) GetNamespace(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[pb.GetNamespaceResponse], error) {
	ctx, span := t.tracer.Start(ctx, "ConfigService.GetNamespace")
	defer span.End()

	res, err := t.inner.GetNamespace(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.String("header_namespace", res.Msg.HeaderNamespace),
		attribute.String("data_namespace", res.Msg.DataNamespace),
	)
	return res, nil
}

func (t *tracedConfigServer) GetSignerInfo(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[pb.GetSignerInfoResponse], error) {
	ctx, span := t.tracer.Start(ctx, "ConfigService.GetSignerInfo")
	defer span.End()

	res, err := t.inner.GetSignerInfo(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.String("signer_address", hex.EncodeToString(res.Msg.Address)),
	)
	return res, nil
}
