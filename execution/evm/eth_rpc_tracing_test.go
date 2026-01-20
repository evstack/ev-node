package evm

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/evstack/ev-node/pkg/telemetry/testutil"
)

// setupTestEthRPCTracing creates a traced eth RPC client with an in-memory span recorder
func setupTestEthRPCTracing(t *testing.T, mockClient EthRPCClient) (EthRPCClient, *tracetest.SpanRecorder) {
	t.Helper()

	// create in-memory span recorder
	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(
		trace.WithSpanProcessor(sr),
	)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})

	// set as global provider for the test
	otel.SetTracerProvider(tp)

	// create traced client
	traced := withTracingEthRPCClient(mockClient)

	return traced, sr
}

// mockEthRPCClient is a simple mock for testing
type mockEthRPCClient struct {
	headerByNumberFn func(ctx context.Context, number *big.Int) (*types.Header, error)
	getTxsFn         func(ctx context.Context) ([]string, error)
}

func (m *mockEthRPCClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	if m.headerByNumberFn != nil {
		return m.headerByNumberFn(ctx, number)
	}
	return nil, nil
}

func (m *mockEthRPCClient) GetTxs(ctx context.Context) ([]string, error) {
	if m.getTxsFn != nil {
		return m.getTxsFn(ctx)
	}
	return nil, nil
}

func TestTracedEthRPCClient_HeaderByNumber_Success(t *testing.T) {
	expectedHeader := &types.Header{
		GasLimit: 30000000,
		GasUsed:  15000000,
		Time:     1234567890,
	}

	mockClient := &mockEthRPCClient{
		headerByNumberFn: func(ctx context.Context, number *big.Int) (*types.Header, error) {
			return expectedHeader, nil
		},
	}

	traced, sr := setupTestEthRPCTracing(t, mockClient)

	ctx := context.Background()
	blockNumber := big.NewInt(100)

	header, err := traced.HeaderByNumber(ctx, blockNumber)

	require.NoError(t, err)
	require.Equal(t, expectedHeader, header)

	// verify span was created
	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	require.Equal(t, "Eth.GetBlockByNumber", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	// verify attributes
	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "method", "eth_getBlockByNumber")
	testutil.RequireAttribute(t, attrs, "block_number", "100")
	testutil.RequireAttribute(t, attrs, "block_hash", expectedHeader.Hash().Hex())
	testutil.RequireAttribute(t, attrs, "state_root", expectedHeader.Root.Hex())
	testutil.RequireAttribute(t, attrs, "gas_limit", int64(expectedHeader.GasLimit))
	testutil.RequireAttribute(t, attrs, "gas_used", int64(expectedHeader.GasUsed))
	testutil.RequireAttribute(t, attrs, "timestamp", int64(expectedHeader.Time))
}

func TestTracedEthRPCClient_HeaderByNumber_Latest(t *testing.T) {
	expectedHeader := &types.Header{
		GasLimit: 30000000,
		GasUsed:  15000000,
		Time:     1234567890,
	}

	mockClient := &mockEthRPCClient{
		headerByNumberFn: func(ctx context.Context, number *big.Int) (*types.Header, error) {
			require.Nil(t, number, "number should be nil for latest block")
			return expectedHeader, nil
		},
	}

	traced, sr := setupTestEthRPCTracing(t, mockClient)

	ctx := context.Background()

	header, err := traced.HeaderByNumber(ctx, nil)

	require.NoError(t, err)
	require.Equal(t, expectedHeader, header)

	// verify span
	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	require.Equal(t, "Eth.GetBlockByNumber", span.Name())

	// verify block_number is "latest" when nil
	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "block_number", "latest")
}

func TestTracedEthRPCClient_HeaderByNumber_Error(t *testing.T) {
	expectedErr := errors.New("failed to get block header")

	mockClient := &mockEthRPCClient{
		headerByNumberFn: func(ctx context.Context, number *big.Int) (*types.Header, error) {
			return nil, expectedErr
		},
	}

	traced, sr := setupTestEthRPCTracing(t, mockClient)

	ctx := context.Background()
	blockNumber := big.NewInt(100)

	_, err := traced.HeaderByNumber(ctx, blockNumber)

	require.Error(t, err)
	require.Equal(t, expectedErr, err)

	// verify span recorded the error
	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	require.Equal(t, "Eth.GetBlockByNumber", span.Name())
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, expectedErr.Error(), span.Status().Description)

	// verify error event was recorded
	events := span.Events()
	require.Len(t, events, 1)
	require.Equal(t, "exception", events[0].Name)

	// verify block header attributes NOT set on error
	attrs := span.Attributes()
	for _, attr := range attrs {
		key := string(attr.Key)
		require.NotEqual(t, "block_hash", key)
		require.NotEqual(t, "state_root", key)
		require.NotEqual(t, "gas_limit", key)
		require.NotEqual(t, "gas_used", key)
	}
}

func TestTracedEthRPCClient_GetTxs_Success(t *testing.T) {
	expectedTxs := []string{"0xabcd", "0xef01", "0x2345"}

	mockClient := &mockEthRPCClient{
		getTxsFn: func(ctx context.Context) ([]string, error) {
			return expectedTxs, nil
		},
	}

	traced, sr := setupTestEthRPCTracing(t, mockClient)

	ctx := context.Background()

	txs, err := traced.GetTxs(ctx)

	require.NoError(t, err)
	require.Equal(t, expectedTxs, txs)

	// verify span was created
	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	require.Equal(t, "TxPool.GetTxs", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	// verify attributes
	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "method", "txpoolExt_getTxs")
	testutil.RequireAttribute(t, attrs, "tx_count", len(expectedTxs))
}

func TestTracedEthRPCClient_GetTxs_EmptyPool(t *testing.T) {
	mockClient := &mockEthRPCClient{
		getTxsFn: func(ctx context.Context) ([]string, error) {
			return []string{}, nil
		},
	}

	traced, sr := setupTestEthRPCTracing(t, mockClient)

	ctx := context.Background()

	txs, err := traced.GetTxs(ctx)

	require.NoError(t, err)
	require.Empty(t, txs)

	// verify span
	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	require.Equal(t, "TxPool.GetTxs", span.Name())

	// verify tx_count is 0
	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "tx_count", 0)
}

func TestTracedEthRPCClient_GetTxs_Error(t *testing.T) {
	expectedErr := errors.New("failed to get transactions")

	mockClient := &mockEthRPCClient{
		getTxsFn: func(ctx context.Context) ([]string, error) {
			return nil, expectedErr
		},
	}

	traced, sr := setupTestEthRPCTracing(t, mockClient)

	ctx := context.Background()

	_, err := traced.GetTxs(ctx)

	require.Error(t, err)
	require.Equal(t, expectedErr, err)

	// verify span recorded the error
	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	require.Equal(t, "TxPool.GetTxs", span.Name())
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, expectedErr.Error(), span.Status().Description)

	// verify error event was recorded
	events := span.Events()
	require.Len(t, events, 1)
	require.Equal(t, "exception", events[0].Name)

	// verify tx_count NOT set on error
	attrs := span.Attributes()
	for _, attr := range attrs {
		require.NotEqual(t, "tx_count", string(attr.Key))
	}
}
