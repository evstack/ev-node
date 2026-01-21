package telemetry

import (
	"context"
	"errors"
	"testing"
	"time"

	coreexec "github.com/evstack/ev-node/core/execution"
	"github.com/evstack/ev-node/pkg/telemetry/testutil"
	"github.com/evstack/ev-node/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// setupTestTracing creates a traced executor with an in-memory span recorder for testing
func setupTestTracing(t *testing.T, mockExec coreexec.Executor) (coreexec.Executor, *tracetest.SpanRecorder) {
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

	// create traced executor
	traced := WithTracingExecutor(mockExec)

	return traced, sr
}

func TestWithTracingExecutor_InitChain_Success(t *testing.T) {
	mockExec := mocks.NewMockExecutor(t)
	traced, sr := setupTestTracing(t, mockExec)

	ctx := context.Background()
	genesisTime := time.Now()
	initialHeight := uint64(1)
	chainID := "test-chain"
	expectedStateRoot := []byte("state-root")

	mockExec.EXPECT().
		InitChain(mock.Anything, genesisTime, initialHeight, chainID).
		Return(expectedStateRoot, nil)

	stateRoot, err := traced.InitChain(ctx, genesisTime, initialHeight, chainID)

	require.NoError(t, err)
	require.Equal(t, expectedStateRoot, stateRoot)

	// verify span was created
	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	require.Equal(t, "Executor.InitChain", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	// verify attributes
	attrs := span.Attributes()
	require.Len(t, attrs, 3)
	testutil.RequireAttribute(t, attrs, "chain.id", chainID)
	testutil.RequireAttribute(t, attrs, "initial.height", int64(initialHeight))
	testutil.RequireAttribute(t, attrs, "genesis.time_unix", genesisTime.Unix())
}

func TestWithTracingExecutor_InitChain_Error(t *testing.T) {
	mockExec := mocks.NewMockExecutor(t)
	traced, sr := setupTestTracing(t, mockExec)

	ctx := context.Background()
	genesisTime := time.Now()
	initialHeight := uint64(1)
	chainID := "test-chain"
	expectedErr := errors.New("init chain failed")

	mockExec.EXPECT().
		InitChain(mock.Anything, genesisTime, initialHeight, chainID).
		Return(nil, expectedErr)

	_, err := traced.InitChain(ctx, genesisTime, initialHeight, chainID)

	require.Error(t, err)
	require.Equal(t, expectedErr, err)

	// verify span recorded the error
	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	require.Equal(t, "Executor.InitChain", span.Name())
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, expectedErr.Error(), span.Status().Description)

	// verify error event was recorded
	events := span.Events()
	require.Len(t, events, 1)
	require.Equal(t, "exception", events[0].Name)
}

func TestWithTracingExecutor_GetTxs_Success(t *testing.T) {
	mockExec := mocks.NewMockExecutor(t)
	traced, sr := setupTestTracing(t, mockExec)

	ctx := context.Background()
	expectedTxs := [][]byte{[]byte("tx1"), []byte("tx2"), []byte("tx3")}

	mockExec.EXPECT().
		GetTxs(mock.Anything).
		Return(expectedTxs, nil)

	txs, err := traced.GetTxs(ctx)

	require.NoError(t, err)
	require.Equal(t, expectedTxs, txs)

	// verify span
	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	require.Equal(t, "Executor.GetTxs", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	// verify tx.count attribute
	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "tx.count", len(expectedTxs))
}

func TestWithTracingExecutor_GetTxs_Error(t *testing.T) {
	mockExec := mocks.NewMockExecutor(t)
	traced, sr := setupTestTracing(t, mockExec)

	ctx := context.Background()
	expectedErr := errors.New("get txs failed")

	mockExec.EXPECT().
		GetTxs(mock.Anything).
		Return(nil, expectedErr)

	_, err := traced.GetTxs(ctx)

	require.Error(t, err)
	require.Equal(t, expectedErr, err)

	// verify span recorded error and did NOT set tx.count
	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)

	// tx.count should not be present when error occurred
	attrs := span.Attributes()
	for _, attr := range attrs {
		require.NotEqual(t, "tx.count", string(attr.Key))
	}
}

func TestWithTracingExecutor_ExecuteTxs_Success(t *testing.T) {
	mockExec := mocks.NewMockExecutor(t)
	traced, sr := setupTestTracing(t, mockExec)

	ctx := context.Background()
	txs := [][]byte{[]byte("tx1"), []byte("tx2")}
	blockHeight := uint64(100)
	timestamp := time.Now()
	prevStateRoot := []byte("prev-state")
	expectedStateRoot := []byte("new-state-root")

	mockExec.EXPECT().
		ExecuteTxs(mock.Anything, txs, blockHeight, timestamp, prevStateRoot).
		Return(expectedStateRoot, nil)

	stateRoot, err := traced.ExecuteTxs(ctx, txs, blockHeight, timestamp, prevStateRoot)

	require.NoError(t, err)
	require.Equal(t, expectedStateRoot, stateRoot)

	// verify span
	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	require.Equal(t, "Executor.ExecuteTxs", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	// verify attributes
	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "tx.count", len(txs))
	testutil.RequireAttribute(t, attrs, "block.height", int64(blockHeight))
	testutil.RequireAttribute(t, attrs, "timestamp", timestamp.Unix())
}

func TestWithTracingExecutor_ExecuteTxs_Error(t *testing.T) {
	mockExec := mocks.NewMockExecutor(t)
	traced, sr := setupTestTracing(t, mockExec)

	ctx := context.Background()
	txs := [][]byte{[]byte("tx1")}
	blockHeight := uint64(100)
	timestamp := time.Now()
	prevStateRoot := []byte("prev-state")
	expectedErr := errors.New("execution failed")

	mockExec.EXPECT().
		ExecuteTxs(mock.Anything, txs, blockHeight, timestamp, prevStateRoot).
		Return(nil, expectedErr)

	_, err := traced.ExecuteTxs(ctx, txs, blockHeight, timestamp, prevStateRoot)

	require.Error(t, err)
	require.Equal(t, expectedErr, err)

	// verify error was recorded
	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, expectedErr.Error(), span.Status().Description)
}

func TestWithTracingExecutor_SetFinal_Success(t *testing.T) {
	mockExec := mocks.NewMockExecutor(t)
	traced, sr := setupTestTracing(t, mockExec)

	ctx := context.Background()
	blockHeight := uint64(50)

	mockExec.EXPECT().
		SetFinal(mock.Anything, blockHeight).
		Return(nil)

	err := traced.SetFinal(ctx, blockHeight)

	require.NoError(t, err)

	// verify span
	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	require.Equal(t, "Executor.SetFinal", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)

	attrs := span.Attributes()
	testutil.RequireAttribute(t, attrs, "block.height", int64(blockHeight))
}

func TestWithTracingExecutor_SetFinal_Error(t *testing.T) {
	mockExec := mocks.NewMockExecutor(t)
	traced, sr := setupTestTracing(t, mockExec)

	ctx := context.Background()
	blockHeight := uint64(50)
	expectedErr := errors.New("finalization failed")

	mockExec.EXPECT().
		SetFinal(mock.Anything, blockHeight).
		Return(expectedErr)

	err := traced.SetFinal(ctx, blockHeight)

	require.Error(t, err)
	require.Equal(t, expectedErr, err)

	// verify error was recorded
	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
}

func TestWithTracingExecutor_GetLatestHeight_NotImplemented(t *testing.T) {
	mockExec := mocks.NewMockExecutor(t)
	traced, sr := setupTestTracing(t, mockExec)

	ctx := context.Background()

	// executor does not implement HeightProvider, should return 0 without calling inner
	tracedWithHeight, ok := traced.(coreexec.HeightProvider)
	require.True(t, ok, "traced executor should implement HeightProvider")

	height, err := tracedWithHeight.GetLatestHeight(ctx)

	require.NoError(t, err)
	require.Equal(t, uint64(0), height)

	// no span should be created since HeightProvider is not implemented
	spans := sr.Ended()
	require.Len(t, spans, 0)
}

func TestWithTracingExecutor_GetLatestHeight_Success(t *testing.T) {
	// create a mock that implements HeightProvider
	mockExec := &mockExecutorWithHeight{
		MockExecutor: mocks.NewMockExecutor(t),
		height:       uint64(42),
	}
	traced, sr := setupTestTracing(t, mockExec)

	ctx := context.Background()

	tracedWithHeight, ok := traced.(coreexec.HeightProvider)
	require.True(t, ok, "traced executor should implement HeightProvider")

	height, err := tracedWithHeight.GetLatestHeight(ctx)

	require.NoError(t, err)
	require.Equal(t, uint64(42), height)

	// verify span was created
	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	require.Equal(t, "Executor.GetLatestHeight", span.Name())
	require.Equal(t, codes.Unset, span.Status().Code)
}

func TestWithTracingExecutor_GetLatestHeight_Error(t *testing.T) {
	expectedErr := errors.New("height retrieval failed")
	mockExec := &mockExecutorWithHeight{
		MockExecutor: mocks.NewMockExecutor(t),
		err:          expectedErr,
	}
	traced, sr := setupTestTracing(t, mockExec)

	ctx := context.Background()

	tracedWithHeight, ok := traced.(coreexec.HeightProvider)
	require.True(t, ok, "traced executor should implement HeightProvider")

	_, err := tracedWithHeight.GetLatestHeight(ctx)

	require.Error(t, err)
	require.Equal(t, expectedErr, err)

	// verify error was recorded
	spans := sr.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	require.Equal(t, codes.Error, span.Status().Code)
}

// mockExecutorWithHeight implements both Executor and HeightProvider for testing
type mockExecutorWithHeight struct {
	*mocks.MockExecutor
	height uint64
	err    error
}

func (m *mockExecutorWithHeight) GetLatestHeight(ctx context.Context) (uint64, error) {
	return m.height, m.err
}
