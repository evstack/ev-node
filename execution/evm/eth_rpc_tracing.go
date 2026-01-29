package evm

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// tracedEthRPCClient wraps an EthRPCClient and records spans for observability.
type tracedEthRPCClient struct {
	inner  EthRPCClient
	tracer trace.Tracer
}

// withTracingEthRPCClient decorates an EthRPCClient with OpenTelemetry tracing.
func withTracingEthRPCClient(inner EthRPCClient) EthRPCClient {
	return &tracedEthRPCClient{
		inner:  inner,
		tracer: otel.Tracer("ev-node/execution/eth-rpc"),
	}
}

func (t *tracedEthRPCClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	var blockNumber string
	if number == nil {
		blockNumber = "latest"
	} else {
		blockNumber = number.String()
	}

	ctx, span := t.tracer.Start(ctx, "Eth.GetBlockByNumber",
		trace.WithAttributes(
			attribute.String("method", "eth_getBlockByNumber"),
			attribute.String("block_number", blockNumber),
		),
	)
	defer span.End()

	result, err := t.inner.HeaderByNumber(ctx, number)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.String("block_hash", result.Hash().Hex()),
		attribute.String("state_root", result.Root.Hex()),
		attribute.Int64("gas_limit", int64(result.GasLimit)),
		attribute.Int64("gas_used", int64(result.GasUsed)),
		attribute.Int64("timestamp", int64(result.Time)),
	)

	return result, nil
}

// GetTxs works only on custom execution clients exposing txpoolExt_getTxs.
// Standard Ethereum nodes do not support this RPC method.
func (t *tracedEthRPCClient) GetTxs(ctx context.Context) ([]string, error) {
	ctx, span := t.tracer.Start(ctx, "TxPool.GetTxs",
		trace.WithAttributes(
			attribute.String("method", "GetTxs"),
		),
	)
	defer span.End()

	result, err := t.inner.GetTxs(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("tx_count", len(result)),
	)

	return result, nil
}
