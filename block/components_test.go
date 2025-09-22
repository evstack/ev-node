package block

import (
	"context"
	crand "crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreda "github.com/evstack/ev-node/core/da"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/signer/noop"
	"github.com/evstack/ev-node/pkg/store"
	testmocks "github.com/evstack/ev-node/test/mocks"
)

func TestBlockComponents_ExecutionClientFailure_StopsNode(t *testing.T) {
	// Test the error channel mechanism works as intended

	// Create a mock component that simulates execution client failure
	errorCh := make(chan error, 1)
	criticalError := errors.New("execution client connection lost")

	// Create BlockComponents with error channel
	bc := &Components{
		errorCh: errorCh,
	}

	// Simulate an execution client failure by sending error to channel
	go func() {
		time.Sleep(50 * time.Millisecond) // Small delay to ensure Start() is running
		errorCh <- criticalError
	}()

	// Start should block until error is received, then return the error
	ctx := context.Background()
	err := bc.Start(ctx)

	// Verify the error is properly wrapped and returned
	require.Error(t, err)
	assert.Contains(t, err.Error(), "critical execution client failure")
	assert.Contains(t, err.Error(), "execution client connection lost")
}

func TestBlockComponents_GetLastState(t *testing.T) {
	// Test that GetLastState works correctly for different component types

	t.Run("Empty state", func(t *testing.T) {
		// When neither is present, return empty state
		bc := &Components{}

		result := bc.GetLastState()
		assert.Equal(t, uint64(0), result.LastBlockHeight)
	})
}

func TestBlockComponents_StartStop_Lifecycle(t *testing.T) {
	// Simple lifecycle test without creating full components
	bc := &Components{
		errorCh: make(chan error, 1),
	}

	// Test that Start and Stop work without hanging
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start should complete when context is cancelled
	err := bc.Start(ctx)
	assert.Contains(t, err.Error(), "context")
}

func TestNewSyncComponents_Creation(t *testing.T) {
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	cfg := config.DefaultConfig()
	gen := genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		StartTime:       time.Now(),
		ProposerAddress: []byte("test-proposer"),
	}

	mockExec := testmocks.NewMockExecutor(t)
	dummyDA := coreda.NewDummyDA(10_000_000, 0, 0, 10*time.Millisecond)

	// Just test that the constructor doesn't panic - don't start the components
	// to avoid P2P store dependencies
	components, err := NewSyncComponents(
		cfg,
		gen,
		memStore,
		mockExec,
		dummyDA,
		nil,
		nil,
		zerolog.Nop(),
		NopMetrics(),
		DefaultBlockOptions(),
	)

	require.NoError(t, err)
	assert.NotNil(t, components)
	assert.NotNil(t, components.Syncer)
	assert.NotNil(t, components.Submitter)
	assert.NotNil(t, components.Cache)
	assert.NotNil(t, components.errorCh)
	assert.Nil(t, components.Executor) // Sync nodes don't have executors
}

func TestNewAggregatorComponents_Creation(t *testing.T) {
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	cfg := config.DefaultConfig()

	// Create a test signer first
	priv, _, err := crypto.GenerateEd25519Key(crand.Reader)
	require.NoError(t, err)
	mockSigner, err := noop.NewNoopSigner(priv)
	require.NoError(t, err)

	// Get the signer's address to use as proposer
	signerAddr, err := mockSigner.GetAddress()
	require.NoError(t, err)

	gen := genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		StartTime:       time.Now(),
		ProposerAddress: signerAddr,
	}

	mockExec := testmocks.NewMockExecutor(t)
	mockSeq := testmocks.NewMockSequencer(t)
	dummyDA := coreda.NewDummyDA(10_000_000, 0, 0, 10*time.Millisecond)

	components, err := NewAggregatorComponents(
		cfg,
		gen,
		memStore,
		mockExec,
		mockSeq,
		dummyDA,
		mockSigner,
		nil, // header broadcaster
		nil, // data broadcaster
		zerolog.Nop(),
		NopMetrics(),
		DefaultBlockOptions(),
	)

	require.NoError(t, err)
	assert.NotNil(t, components)
	assert.NotNil(t, components.Executor)
	assert.NotNil(t, components.Submitter)
	assert.NotNil(t, components.Cache)
	assert.NotNil(t, components.errorCh)
	assert.Nil(t, components.Syncer) // Aggregator nodes currently don't create syncers in this constructor
}

func TestExecutor_RealExecutionClientFailure_StopsNode(t *testing.T) {
	// This test verifies that when the executor's execution client calls fail,
	// the error is properly propagated through the error channel and stops the node

	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	cfg := config.DefaultConfig()
	cfg.Node.BlockTime.Duration = 50 * time.Millisecond // Fast for testing

	// Create test signer
	priv, _, err := crypto.GenerateEd25519Key(crand.Reader)
	require.NoError(t, err)
	testSigner, err := noop.NewNoopSigner(priv)
	require.NoError(t, err)
	addr, err := testSigner.GetAddress()
	require.NoError(t, err)

	gen := genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		StartTime:       time.Now().Add(-time.Second), // Start in past to trigger immediate execution
		ProposerAddress: addr,
	}

	// Create mock executor that will fail on ExecuteTxs
	mockExec := testmocks.NewMockExecutor(t)
	mockSeq := testmocks.NewMockSequencer(t)
	dummyDA := coreda.NewDummyDA(10_000_000, 0, 0, 10*time.Millisecond)

	// Mock InitChain to succeed initially
	mockExec.On("InitChain", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("state-root"), uint64(1024), nil).Once()

	// Mock GetNextBatch to return empty batch
	mockSeq.On("GetNextBatch", mock.Anything, mock.Anything).
		Return(&coresequencer.GetNextBatchResponse{
			Batch:     &coresequencer.Batch{Transactions: nil},
			Timestamp: time.Now(),
		}, nil).Maybe()

	// Mock ExecuteTxs to fail with a critical error
	criticalError := errors.New("execution client RPC connection failed")
	mockExec.On("ExecuteTxs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, uint64(0), criticalError).Maybe()

	// Create aggregator node
	components, err := NewAggregatorComponents(
		cfg,
		gen,
		memStore,
		mockExec,
		mockSeq,
		dummyDA,
		testSigner,
		nil, // header broadcaster
		nil, // data broadcaster
		zerolog.Nop(),
		NopMetrics(),
		DefaultBlockOptions(),
	)
	require.NoError(t, err)

	// Start should return with error when execution client fails
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = components.Start(ctx)

	// We expect an error containing the critical execution client failure
	require.Error(t, err)
	assert.Contains(t, err.Error(), "critical execution client failure")
	assert.Contains(t, err.Error(), "execution client RPC connection failed")

	// Clean up
	stopErr := components.Stop()
	assert.NoError(t, stopErr)
}
