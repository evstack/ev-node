//go:build !integration

package node

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreexecutor "github.com/evstack/ev-node/core/execution"
	testmocks "github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
)

func TestBasicExecutionFlow(t *testing.T) {
	require := require.New(t)

	node, cleanup := createNodeWithCleanup(t, getTestConfig(t, 1))
	defer cleanup()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		_ = node.Run(ctx)
	}()

	// Wait for node initialization
	err := waitForNodeInitialization(node)
	require.NoError(err)

	// Get the original executor to retrieve transactions
	executor := getExecutorFromNode(t, node)
	require.NotNil(executor, "Executor should not be nil")
	txs := getTransactions(t, executor, t.Context())

	// Use the generated mock executor for testing execution steps
	mockExec := testmocks.NewMockExecutor(t)

	// Define expected state and parameters
	expectedInitialStateRoot := []byte("initial state root")
	expectedNewStateRoot := []byte("new state root")
	blockHeight := uint64(1)
	chainID := "test-chain"

	// Set expectations on the mock executor
	mockExec.On("InitChain", mock.Anything, mock.AnythingOfType("time.Time"), blockHeight, chainID).
		Return(expectedInitialStateRoot, nil).Once()
	mockExec.On("ExecuteTxs", mock.Anything, txs, blockHeight, mock.AnythingOfType("time.Time"), expectedInitialStateRoot).
		Return(expectedNewStateRoot, nil).Once()
	mockExec.On("SetFinal", mock.Anything, blockHeight).
		Return(nil).Once()

	// Call helper functions with the mock executor
	stateRoot := initializeChain(t, mockExec, t.Context())
	require.Equal(expectedInitialStateRoot, stateRoot)

	newStateRoot := executeTransactions(t, mockExec, t.Context(), txs, stateRoot)
	require.Equal(expectedNewStateRoot, newStateRoot)

	finalizeExecution(t, mockExec, t.Context())

	require.NotEmpty(newStateRoot)
	cancel()
	time.Sleep(100 * time.Millisecond) // grace period for node shutdown and cleanup
}

func waitForNodeInitialization(node *FullNode) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if node.IsRunning() {
				return nil
			}
		case <-ctx.Done():
			return errors.New("timeout waiting for node initialization")
		}
	}
}

func getExecutorFromNode(t *testing.T, node *FullNode) coreexecutor.Executor {
	le := node.leaderElection
	sle, ok := le.(*singleRoleElector)
	require.True(t, ok, "Leader election is not singleRoleElector")
	state := sle.state()
	require.NotNil(t, state)
	require.NotNil(t, state.bc)
	require.NotNil(t, state.bc.Executor)
	return nil
}

func getTransactions(t *testing.T, executor coreexecutor.Executor, ctx context.Context) [][]byte {
	txs, err := executor.GetTxs(ctx)
	require.NoError(t, err)
	return txs
}

func initializeChain(t *testing.T, executor coreexecutor.Executor, ctx context.Context) []byte {
	genesisTime := time.Now()
	initialHeight := uint64(1)
	chainID := "test-chain"
	stateRoot, err := executor.InitChain(ctx, genesisTime, initialHeight, chainID)
	require.NoError(t, err)
	return stateRoot
}

func executeTransactions(t *testing.T, executor coreexecutor.Executor, ctx context.Context, txs [][]byte, stateRoot types.Hash) []byte {
	blockHeight := uint64(1)
	timestamp := time.Now()
	newStateRoot, err := executor.ExecuteTxs(ctx, txs, blockHeight, timestamp, stateRoot)
	require.NoError(t, err)
	return newStateRoot
}

func finalizeExecution(t *testing.T, executor coreexecutor.Executor, ctx context.Context) {
	err := executor.SetFinal(ctx, 1)
	require.NoError(t, err)
}
