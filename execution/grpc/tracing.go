package grpc

import (
	"context"
	"encoding/hex"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
	"github.com/evstack/ev-node/types/pb/evnode/v1/v1connect"
)

// tracedServer decorates an ExecutorServiceHandler with OpenTelemetry spans.
type tracedServer struct {
	inner  v1connect.ExecutorServiceHandler
	tracer trace.Tracer
}

// WithTracingServer decorates the provided executor service handler with tracing spans.
func WithTracingServer(inner v1connect.ExecutorServiceHandler) v1connect.ExecutorServiceHandler {
	return &tracedServer{
		inner:  inner,
		tracer: otel.Tracer("ev-node/executor-service"),
	}
}

func (t *tracedServer) InitChain(
	ctx context.Context,
	req *connect.Request[pb.InitChainRequest],
) (*connect.Response[pb.InitChainResponse], error) {
	ctx, span := t.tracer.Start(ctx, "ExecutorService.InitChain",
		trace.WithAttributes(
			attribute.String("chain_id", req.Msg.ChainId),
			attribute.Int64("initial_height", int64(req.Msg.InitialHeight)),
		),
	)
	defer span.End()

	res, err := t.inner.InitChain(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.String("state_root", hex.EncodeToString(res.Msg.StateRoot)),
		attribute.Int64("max_bytes", int64(res.Msg.MaxBytes)),
	)
	return res, nil
}

func (t *tracedServer) GetTxs(
	ctx context.Context,
	req *connect.Request[pb.GetTxsRequest],
) (*connect.Response[pb.GetTxsResponse], error) {
	ctx, span := t.tracer.Start(ctx, "ExecutorService.GetTxs")
	defer span.End()

	res, err := t.inner.GetTxs(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("tx_count", len(res.Msg.Txs)),
	)
	return res, nil
}

func (t *tracedServer) ExecuteTxs(
	ctx context.Context,
	req *connect.Request[pb.ExecuteTxsRequest],
) (*connect.Response[pb.ExecuteTxsResponse], error) {
	ctx, span := t.tracer.Start(ctx, "ExecutorService.ExecuteTxs",
		trace.WithAttributes(
			attribute.Int("tx_count", len(req.Msg.Txs)),
			attribute.Int64("block_height", int64(req.Msg.BlockHeight)),
			attribute.String("prev_state_root", hex.EncodeToString(req.Msg.PrevStateRoot)),
		),
	)
	defer span.End()

	res, err := t.inner.ExecuteTxs(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.String("updated_state_root", hex.EncodeToString(res.Msg.UpdatedStateRoot)),
		attribute.Int64("max_bytes", int64(res.Msg.MaxBytes)),
	)
	return res, nil
}

func (t *tracedServer) SetFinal(
	ctx context.Context,
	req *connect.Request[pb.SetFinalRequest],
) (*connect.Response[pb.SetFinalResponse], error) {
	ctx, span := t.tracer.Start(ctx, "ExecutorService.SetFinal",
		trace.WithAttributes(
			attribute.Int64("block_height", int64(req.Msg.BlockHeight)),
		),
	)
	defer span.End()

	res, err := t.inner.SetFinal(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return res, nil
}
