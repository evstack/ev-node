package block

import (
	"context"
	crand "crypto/rand"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/signer/noop"
	"github.com/evstack/ev-node/pkg/store"
	testmocks "github.com/evstack/ev-node/test/mocks"
	extmocks "github.com/evstack/ev-node/test/mocks/external"
	"github.com/evstack/ev-node/types"
)

// noopDAHintAppender is a no-op implementation of DAHintAppender for testing
type noopDAHintAppender struct{}

func (n noopDAHintAppender) AppendDAHint(ctx context.Context, daHeight uint64, heights ...uint64) error {
	return nil
}

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
	daClient := testmocks.NewMockClient(t)
	daClient.On("GetHeaderNamespace").Return(datypes.NamespaceFromString("ns").Bytes()).Maybe()
	daClient.On("GetDataNamespace").Return(datypes.NamespaceFromString("data-ns").Bytes()).Maybe()
	daClient.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()
	daClient.On("HasForcedInclusionNamespace").Return(false).Maybe()

	// Create mock P2P stores
	mockHeaderStore := extmocks.NewMockStore[*types.P2PSignedHeader](t)
	mockDataStore := extmocks.NewMockStore[*types.P2PData](t)

	// Create noop DAHintAppenders for testing
	headerHintAppender := noopDAHintAppender{}
	dataHintAppender := noopDAHintAppender{}

	// Just test that the constructor doesn't panic - don't start the components
	// to avoid P2P store dependencies
	components, err := NewSyncComponents(
		cfg,
		gen,
		memStore,
		mockExec,
		daClient,
		mockHeaderStore,
		mockDataStore,
		headerHintAppender,
		dataHintAppender,
		zerolog.Nop(),
		NopMetrics(),
		DefaultBlockOptions(),
		nil,
	)

	require.NoError(t, err)
	assert.NotNil(t, components)
	assert.NotNil(t, components.Syncer)
	assert.NotNil(t, components.Submitter)
	assert.NotNil(t, components.Cache)
	assert.NotNil(t, components.Pruner)
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
	daClient := testmocks.NewMockClient(t)
	daClient.On("GetHeaderNamespace").Return(datypes.NamespaceFromString("ns").Bytes()).Maybe()
	daClient.On("GetDataNamespace").Return(datypes.NamespaceFromString("data-ns").Bytes()).Maybe()
	daClient.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()
	daClient.On("HasForcedInclusionNamespace").Return(false).Maybe()

	components, err := newAggregatorComponents(
		cfg,
		gen,
		memStore,
		mockExec,
		mockSeq,
		daClient,
		mockSigner,
		nil, // header broadcaster
		nil, // data broadcaster
		zerolog.Nop(),
		NopMetrics(),
		DefaultBlockOptions(),
		nil, // raftNode
	)

	require.NoError(t, err)
	assert.NotNil(t, components)
	assert.NotNil(t, components.Executor)
	assert.NotNil(t, components.Submitter)
	assert.NotNil(t, components.Cache)
	assert.NotNil(t, components.Pruner)
	assert.NotNil(t, components.errorCh)
	assert.Nil(t, components.Syncer) // Aggregator nodes currently don't create syncers in this constructor
}

func TestExecutor_RealExecutionClientFailure_StopsNode(t *testing.T) {
	// This test verifies that when the executor's execution client calls fail,
	// the error is properly propagated through the error channel and stops the node
	synctest.Test(t, func(t *testing.T) {
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
		daClient := testmocks.NewMockClient(t)
		daClient.On("GetHeaderNamespace").Return(datypes.NamespaceFromString("ns").Bytes()).Maybe()
		daClient.On("GetDataNamespace").Return(datypes.NamespaceFromString("data-ns").Bytes()).Maybe()
		daClient.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()
		daClient.On("HasForcedInclusionNamespace").Return(false).Maybe()

		// Mock InitChain to succeed initially
		mockExec.On("InitChain", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]byte("state-root"), nil).Once()

		// Mock SetDAHeight to be called during initialization
		mockSeq.On("SetDAHeight", uint64(0)).Return().Once()

		// Mock GetNextBatch to return empty batch
		mockSeq.On("GetNextBatch", mock.Anything, mock.Anything).
			Return(&coresequencer.GetNextBatchResponse{
				Batch:     &coresequencer.Batch{Transactions: nil},
				Timestamp: time.Now(),
			}, nil).Maybe()

		// Mock GetTxs for reaper (return empty to avoid interfering with test)
		mockExec.On("GetTxs", mock.Anything).
			Return([][]byte{}, nil).Maybe()

		// Mock ExecuteTxs to fail with a critical error
		criticalError := errors.New("execution client RPC connection failed")
		mockExec.On("ExecuteTxs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, criticalError).Maybe()

		// Create aggregator node
		components, err := newAggregatorComponents(
			cfg,
			gen,
			memStore,
			mockExec,
			mockSeq,
			daClient,
			testSigner,
			nil, // header broadcaster
			nil, // data broadcaster
			zerolog.Nop(),
			NopMetrics(),
			DefaultBlockOptions(),
			nil, // raftNode
		)
		require.NoError(t, err)

		// Start should return with error when execution client fails.
		// With synctest the fake clock advances the retry delays instantly.
		ctx, cancel := context.WithTimeout(t.Context(), 35*time.Second)
		defer cancel()

		// Run Start in a goroutine to handle the blocking call
		startErrCh := make(chan error, 1)
		go func() {
			startErrCh <- components.Start(ctx)
		}()

		// Wait for either the error or timeout
		synctest.Wait()
		select {
		case err = <-startErrCh:
			// We expect an error containing the critical execution client failure
			require.Error(t, err)
			assert.Contains(t, err.Error(), "critical execution client failure")
			assert.Contains(t, err.Error(), "execution client RPC connection failed")
		case <-ctx.Done():
			t.Fatal("timeout waiting for critical error to propagate")
		}

		// Clean up
		stopErr := components.Stop()
		assert.NoError(t, stopErr)
	})
}
