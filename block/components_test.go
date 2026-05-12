package block

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/crypto"
	crand "crypto/rand"
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
)

func TestBlockComponents_ExecutionClientFailure_StopsNode(t *testing.T) {
	errorCh := make(chan error, 1)
	criticalError := errors.New("execution client connection lost")

	bc := &Components{
		errorCh: errorCh,
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		errorCh <- criticalError
	}()

	ctx := context.Background()
	err := bc.Start(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "critical execution client failure")
	assert.Contains(t, err.Error(), "execution client connection lost")
}

func TestBlockComponents_StartStop_Lifecycle(t *testing.T) {
	bc := &Components{
		errorCh: make(chan error, 1),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

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

	components, err := NewSyncComponents(
		cfg,
		gen,
		memStore,
		mockExec,
		daClient,
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
	assert.Nil(t, components.Executor)
}

func TestNewAggregatorComponents_Creation(t *testing.T) {
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	cfg := config.DefaultConfig()

	priv, _, err := crypto.GenerateEd25519Key(crand.Reader)
	require.NoError(t, err)
	mockSigner, err := noop.NewNoopSigner(priv)
	require.NoError(t, err)

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
		zerolog.Nop(),
		NopMetrics(),
		DefaultBlockOptions(),
		nil,
	)

	require.NoError(t, err)
	assert.NotNil(t, components)
	assert.NotNil(t, components.Executor)
	assert.NotNil(t, components.Submitter)
	assert.NotNil(t, components.Cache)
	assert.NotNil(t, components.Pruner)
	assert.NotNil(t, components.errorCh)
	assert.Nil(t, components.Syncer)
}

func TestExecutor_RealExecutionClientFailure_StopsNode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ds := sync.MutexWrap(datastore.NewMapDatastore())
		memStore := store.New(ds)

		cfg := config.DefaultConfig()
		cfg.Node.BlockTime.Duration = 50 * time.Millisecond

		priv, _, err := crypto.GenerateEd25519Key(crand.Reader)
		require.NoError(t, err)
		testSigner, err := noop.NewNoopSigner(priv)
		require.NoError(t, err)
		addr, err := testSigner.GetAddress()
		require.NoError(t, err)

		gen := genesis.Genesis{
			ChainID:         "test-chain",
			InitialHeight:   1,
			StartTime:       time.Now().Add(-time.Second),
			ProposerAddress: addr,
		}

		mockExec := testmocks.NewMockExecutor(t)
		mockSeq := testmocks.NewMockSequencer(t)
		daClient := testmocks.NewMockClient(t)
		daClient.On("GetHeaderNamespace").Return(datypes.NamespaceFromString("ns").Bytes()).Maybe()
		daClient.On("GetDataNamespace").Return(datypes.NamespaceFromString("data-ns").Bytes()).Maybe()
		daClient.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()
		daClient.On("HasForcedInclusionNamespace").Return(false).Maybe()

		mockExec.On("InitChain", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]byte("state-root"), nil).Once()

		mockSeq.On("SetDAHeight", uint64(0)).Return().Once()

		mockSeq.On("GetNextBatch", mock.Anything, mock.Anything).
			Return(&coresequencer.GetNextBatchResponse{
				Batch:     &coresequencer.Batch{Transactions: nil},
				Timestamp: time.Now(),
			}, nil).Maybe()

		mockExec.On("GetTxs", mock.Anything).
			Return([][]byte{}, nil).Maybe()

		criticalError := errors.New("execution client RPC connection failed")
		mockExec.On("ExecuteTxs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, criticalError).Maybe()

		components, err := newAggregatorComponents(
			cfg,
			gen,
			memStore,
			mockExec,
			mockSeq,
			daClient,
			testSigner,
			zerolog.Nop(),
			NopMetrics(),
			DefaultBlockOptions(),
			nil,
		)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(t.Context(), 35*time.Second)
		defer cancel()

		startErrCh := make(chan error, 1)
		go func() {
			startErrCh <- components.Start(ctx)
		}()

		synctest.Wait()
		select {
		case err = <-startErrCh:
			require.Error(t, err)
			assert.Contains(t, err.Error(), "critical execution client failure")
			assert.Contains(t, err.Error(), "execution client RPC connection failed")
		case <-ctx.Done():
			t.Fatal("timeout waiting for critical error to propagate")
		}

		stopErr := components.Stop()
		assert.NoError(t, stopErr)
	})
}
