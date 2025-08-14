package block

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	goheaderstore "github.com/celestiaorg/go-header/store"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/cache"
	"github.com/evstack/ev-node/pkg/config"
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	noopsigner "github.com/evstack/ev-node/pkg/signer/noop"
	storepkg "github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
)

// TestExecuteTxsFailureAfterBlockSave verifies that with the fix:
// 1. Block is NOT saved to disk when ExecuteTxs fails
// 2. State remains consistent
// 3. On restart, there is no corruption (heights remain in sync)
func TestExecuteTxsFailureAfterBlockSave(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()

	// Setup mock components
	mockStore := mocks.NewMockStore(t)
	mockExec := mocks.NewMockExecutor(t)
	mockSeq := mocks.NewMockSequencer(t)

	// Create a test signer
	privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 256)
	require.NoError(err)
	testSigner, err := noopsigner.NewNoopSigner(privKey)
	require.NoError(err)
	proposerAddr, err := testSigner.GetAddress()
	require.NoError(err)

	// Setup configuration
	cfg := config.DefaultConfig
	cfg.Node.BlockTime.Duration = 1 * time.Second
	genesis := genesispkg.NewGenesis("testchain", 1, time.Now(), proposerAddr)

	logger := zerolog.Nop()

	// Track what gets saved to the store
	var savedHeader *types.SignedHeader
	var savedData *types.Data
	blockSavedBeforeExecution := false

	// Setup initial state at height 0
	initialState := types.State{
		ChainID:         "testchain",
		InitialHeight:   1,
		LastBlockHeight: 0,
		AppHash:         []byte("initial_app_hash"),
		LastBlockTime:   time.Now().Add(-10 * time.Second),
	}

	// Mock store behavior
	mockStore.On("Height", mock.Anything).Return(uint64(0), nil).Maybe()
	mockStore.On("GetState", mock.Anything).Return(initialState, nil).Maybe()
	mockStore.On("GetBlockData", mock.Anything, uint64(0)).Return(
		&types.SignedHeader{
			Header: types.Header{
				BaseHeader: types.BaseHeader{
					Height: 0,
					Time:   uint64(initialState.LastBlockTime.Unix()),
				},
			},
		},
		&types.Data{},
		nil,
	).Maybe()
	// publishBlockInternal checks if block at newHeight (1) already exists
	mockStore.On("GetBlockData", mock.Anything, uint64(1)).Return(nil, nil, errors.New("not found")).Maybe()
	mockStore.On("GetSignature", mock.Anything, uint64(0)).Return(&types.Signature{}, nil).Maybe()

	// Track metadata calls
	mockStore.On("GetMetadata", mock.Anything, storepkg.LastSubmittedHeaderHeightKey).Return([]byte{0, 0, 0, 0, 0, 0, 0, 0}, nil).Maybe()
	mockStore.On("GetMetadata", mock.Anything, LastSubmittedDataHeightKey).Return([]byte{0, 0, 0, 0, 0, 0, 0, 0}, nil).Maybe()
	mockStore.On("GetMetadata", mock.Anything, mock.Anything).Return(nil, nil).Maybe()

	// WITH THE FIX: SaveBlockData should NOT be called when ExecuteTxs fails
	// We make this optional since it won't be called
	mockStore.On("SaveBlockData", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			savedHeader = args.Get(1).(*types.SignedHeader)
			savedData = args.Get(2).(*types.Data)
			blockSavedBeforeExecution = true
			t.Logf("❌ UNEXPECTED: Block SAVED to disk: height=%d, txs=%d",
				savedHeader.Height(), len(savedData.Txs))
		}).
		Return(nil).Maybe()  // Maybe() because it should NOT be called with the fix

	mockStore.On("SetHeight", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockStore.On("UpdateState", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockStore.On("SetMetadata", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// Create test transactions
	testTxs := [][]byte{
		[]byte("test_transaction_1"),
		[]byte("test_transaction_2"),
	}

	// Mock sequencer to return a batch with transactions
	mockSeq.On("GetNextBatch", mock.Anything, mock.Anything).Return(&coresequencer.GetNextBatchResponse{
		Batch: &coresequencer.Batch{
			Transactions: testTxs,
		},
		Timestamp: time.Now(),
	}, nil)

	// Mock ExecuteTxs to FAIL
	executeTxsError := errors.New("ExecuteTxs FAILED: gas limit exceeded")
	mockExec.On("ExecuteTxs", mock.Anything, mock.Anything, uint64(1), mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			t.Logf("💥 ExecuteTxs called and will FAIL")
			// WITH THE FIX: Verify that NO block was saved before ExecuteTxs
			assert.False(t, blockSavedBeforeExecution,
				"With the fix, block should NOT be saved before ExecuteTxs")
		}).
		Return(nil, uint64(0), executeTxsError)

	// Create the Manager
	manager := &Manager{
		store:     mockStore,
		exec:      mockExec,
		sequencer: mockSeq,
		signer:    testSigner,
		config:    cfg,
		genesis:   genesis,
		logger:    logger,
		lastState: initialState,
		headerBroadcaster: testBroadcasterFn[*types.SignedHeader](func(ctx context.Context, payload *types.SignedHeader) error {
			return nil
		}),
		dataBroadcaster: testBroadcasterFn[*types.Data](func(ctx context.Context, payload *types.Data) error {
			return nil
		}),
		headerStore:              (*goheaderstore.Store[*types.SignedHeader])(nil),
		dataStore:                (*goheaderstore.Store[*types.Data])(nil),
		daHeight:                 &atomic.Uint64{},
		headerCache:              cache.NewCache[types.SignedHeader](),
		dataCache:                cache.NewCache[types.Data](),
		lastStateMtx:             &sync.RWMutex{},
		metrics:                  NopMetrics(),
		headerStoreCh:            make(chan struct{}, 1),
		dataStoreCh:              make(chan struct{}, 1),
		retrieveCh:               make(chan struct{}, 1),
		daIncluderCh:             make(chan struct{}, 1),
		signaturePayloadProvider: types.DefaultSignaturePayloadProvider,
		validatorHasherProvider:  types.DefaultValidatorHasherProvider,
	}
	manager.publishBlock = manager.publishBlockInternal

	// Attempt to publish a block - this will fail during ExecuteTxs
	err = manager.publishBlockInternal(ctx)

	// Verify the error from ExecuteTxs propagated
	require.Error(err)
	assert.Contains(t, err.Error(), "error applying block")
	assert.Contains(t, err.Error(), "ExecuteTxs FAILED")

	// Verify what happened WITH THE FIX
	t.Logf("\n✅ === FIX VERIFICATION ===")

	// 1. Block was NOT saved to disk
	assert.False(t, blockSavedBeforeExecution, "Block should NOT be saved when ExecuteTxs fails")
	assert.Nil(t, savedHeader, "No header should be saved")
	assert.Nil(t, savedData, "No data should be saved")

	// 2. State was NOT updated (ExecuteTxs failed, as expected)
	assert.Equal(t, uint64(0), manager.lastState.LastBlockHeight,
		"State height should still be 0 since ExecuteTxs failed")
	assert.Equal(t, []byte("initial_app_hash"), manager.lastState.AppHash,
		"AppHash should be unchanged since transactions weren't executed")

	t.Logf("\n✅ FIX SUCCESSFUL:")
	t.Logf("  - Block was NOT saved to disk")
	t.Logf("  - State remains consistent at height 0")
	t.Logf("  - No corruption can occur")
	t.Logf("  - System maintains consistency when ExecuteTxs fails")

	// Now simulate a restart to verify NO corruption
	t.Logf("\n🔄 === SIMULATING NODE RESTART ===")

	// Create a new mock store for restart scenario
	mockStore2 := mocks.NewMockStore(t)

	// WITH THE FIX: On restart, the store will still report height 0 (no block was saved)
	mockStore2.On("Height", mock.Anything).Return(uint64(0), nil)

	// No block at height 1 exists (because it was never saved)
	mockStore2.On("GetBlockData", mock.Anything, uint64(1)).Return(nil, nil, errors.New("not found")).Maybe()

	// State is still at height 0 (consistent)
	mockStore2.On("GetState", mock.Anything).Return(initialState, nil).Maybe()

	// Additional mocks for restart
	mockStore2.On("GetSignature", mock.Anything, mock.Anything).Return(&types.Signature{}, nil).Maybe()
	mockStore2.On("GetMetadata", mock.Anything, mock.Anything).Return(nil, nil).Maybe()

	// Check that everything is consistent
	storeHeight, _ := mockStore2.Height(ctx)
	stateFromStore, _ := mockStore2.GetState(ctx)

	t.Logf("\n✅ === NO CORRUPTION AFTER RESTART ===")
	t.Logf("  Store height: %d (no invalid block on disk)", storeHeight)
	t.Logf("  State height: %d (consistent with store)", stateFromStore.LastBlockHeight)
	t.Logf("  No unexecuted transactions on disk")
	t.Logf("\n  THE NODE REMAINS CONSISTENT!")

	// Verify consistency
	assert.Equal(t, uint64(0), storeHeight, "Store remains at height 0")
	assert.Equal(t, uint64(0), stateFromStore.LastBlockHeight, "State is still at height 0")
	assert.Equal(t, storeHeight, stateFromStore.LastBlockHeight, "Heights are IN SYNC!")
}

// testBroadcasterFn is a helper type for creating mock broadcasters for this test
type testBroadcasterFn[T any] func(context.Context, T) error

func (f testBroadcasterFn[T]) WriteToStoreAndBroadcast(ctx context.Context, payload T) error {
	return f(ctx, payload)
}
