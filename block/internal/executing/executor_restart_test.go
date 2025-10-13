package executing

import (
	"context"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	coreseq "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/store"
	testmocks "github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
)

func TestExecutor_RestartUsesPendingHeader(t *testing.T) {
	// Create persistent datastore
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	cacheManager, err := cache.NewManager(config.DefaultConfig(), memStore, zerolog.Nop())
	require.NoError(t, err)

	metrics := common.NopMetrics()
	addr, _, signerWrapper := buildTestSigner(t)

	cfg := config.DefaultConfig()
	cfg.Node.BlockTime = config.DurationWrapper{Duration: 10 * time.Millisecond}
	cfg.Node.MaxPendingHeadersAndData = 1000

	gen := genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		StartTime:       time.Now().Add(-time.Second),
		ProposerAddress: addr,
	}

	// Create first executor instance
	mockExec1 := testmocks.NewMockExecutor(t)
	mockSeq1 := testmocks.NewMockSequencer(t)
	hb1 := common.NewMockBroadcaster[*types.SignedHeader](t)
	hb1.EXPECT().WriteToStoreAndBroadcast(mock.Anything, mock.Anything).Return(nil).Maybe()
	db1 := common.NewMockBroadcaster[*types.Data](t)
	db1.EXPECT().WriteToStoreAndBroadcast(mock.Anything, mock.Anything).Return(nil).Maybe()

	exec1, err := NewExecutor(
		memStore,
		mockExec1,
		mockSeq1,
		signerWrapper,
		cacheManager,
		metrics,
		cfg,
		gen,
		hb1,
		db1,
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	require.NoError(t, err)

	// Initialize state for first executor
	initStateRoot := []byte("init_root")
	mockExec1.EXPECT().InitChain(mock.Anything, mock.AnythingOfType("time.Time"), gen.InitialHeight, gen.ChainID).
		Return(initStateRoot, uint64(1024), nil).Once()
	require.NoError(t, exec1.initializeState())

	// Set up context for first executor
	exec1.ctx, exec1.cancel = context.WithCancel(context.Background())

	// First executor produces a block normally
	mockSeq1.EXPECT().GetNextBatch(mock.Anything, mock.AnythingOfType("sequencer.GetNextBatchRequest")).
		RunAndReturn(func(ctx context.Context, req coreseq.GetNextBatchRequest) (*coreseq.GetNextBatchResponse, error) {
			return &coreseq.GetNextBatchResponse{
				Batch: &coreseq.Batch{
					Transactions: [][]byte{[]byte("tx1"), []byte("tx2")},
				},
				Timestamp: time.Now(),
			}, nil
		}).Once()

	mockExec1.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, uint64(1), mock.AnythingOfType("time.Time"), initStateRoot).
		Return([]byte("new_root_1"), uint64(1024), nil).Once()

	err = exec1.produceBlock()
	require.NoError(t, err)

	// Verify first block was produced
	h1, err := memStore.Height(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(1), h1)

	// Store the produced block data for later verification
	originalHeader, originalData, err := memStore.GetBlockData(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 2, len(originalData.Txs), "first block should have 2 transactions")

	// Now simulate creating a pending block at height 2 but not fully completing it
	// This simulates a crash scenario where block data is saved but state isn't updated
	currentState := exec1.getLastState()
	newHeight := currentState.LastBlockHeight + 1 // height 2

	// Get validator hash properly
	pubKey, err := signerWrapper.GetPublic()
	require.NoError(t, err)
	validatorHash, err := common.DefaultBlockOptions().ValidatorHasherProvider(gen.ProposerAddress, pubKey)
	require.NoError(t, err)

	// Create pending block data manually (simulating partial block creation)
	pendingHeader := &types.SignedHeader{
		Header: types.Header{
			Version: types.Version{
				Block: currentState.Version.Block,
				App:   currentState.Version.App,
			},
			BaseHeader: types.BaseHeader{
				ChainID: gen.ChainID,
				Height:  newHeight,
				Time:    uint64(time.Now().UnixNano()),
			},
			LastHeaderHash:  originalHeader.Hash(),
			ConsensusHash:   make(types.Hash, 32),
			AppHash:         currentState.AppHash,
			ProposerAddress: gen.ProposerAddress,
			ValidatorHash:   validatorHash,
		},
		Signer: types.Signer{
			PubKey:  pubKey,
			Address: gen.ProposerAddress,
		},
	}

	pendingData := &types.Data{
		Txs: []types.Tx{[]byte("pending_tx1"), []byte("pending_tx2"), []byte("pending_tx3")},
		Metadata: &types.Metadata{
			ChainID:      pendingHeader.ChainID(),
			Height:       pendingHeader.Height(),
			Time:         pendingHeader.BaseHeader.Time,
			LastDataHash: originalData.Hash(),
		},
	}

	// Set data hash
	pendingHeader.DataHash = pendingData.DACommitment()

	// Save pending block data (this is what would happen during a crash)
	batch, err := memStore.NewBatch(context.Background())
	require.NoError(t, err)
	err = batch.SaveBlockData(pendingHeader, pendingData, &types.Signature{})
	require.NoError(t, err)
	err = batch.Commit()
	require.NoError(t, err)

	// Stop first executor (simulating crash/restart)
	exec1.cancel()

	// Create second executor instance (restart scenario)
	mockExec2 := testmocks.NewMockExecutor(t)
	mockSeq2 := testmocks.NewMockSequencer(t)
	hb2 := common.NewMockBroadcaster[*types.SignedHeader](t)
	hb2.EXPECT().WriteToStoreAndBroadcast(mock.Anything, mock.Anything).Return(nil).Maybe()
	db2 := common.NewMockBroadcaster[*types.Data](t)
	db2.EXPECT().WriteToStoreAndBroadcast(mock.Anything, mock.Anything).Return(nil).Maybe()

	exec2, err := NewExecutor(
		memStore, // same store
		mockExec2,
		mockSeq2,
		signerWrapper,
		cacheManager,
		metrics,
		cfg,
		gen,
		hb2,
		db2,
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	require.NoError(t, err)

	// Initialize state for second executor (should load existing state)
	require.NoError(t, exec2.initializeState())

	// Set up context for second executor
	exec2.ctx, exec2.cancel = context.WithCancel(context.Background())
	defer exec2.cancel()

	// Verify that the state is at height 1 (pending block at height 2 wasn't committed)
	currentState2 := exec2.getLastState()
	assert.Equal(t, uint64(1), currentState2.LastBlockHeight)

	// When second executor tries to produce block at height 2, it should use pending data
	// The sequencer should NOT be called because pending block exists
	// The executor should be called to apply the pending block

	mockExec2.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, uint64(2), mock.AnythingOfType("time.Time"), currentState2.AppHash).
		Return([]byte("new_root_2"), uint64(1024), nil).Once()

	// Note: mockSeq2 should NOT receive any calls because pending block should be used

	err = exec2.produceBlock()
	require.NoError(t, err)

	// Verify height advanced to 2
	h2, err := memStore.Height(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(2), h2, "height should advance to 2 using pending block")

	// Verify the block at height 2 matches the pending block data
	finalHeader, finalData, err := memStore.GetBlockData(context.Background(), 2)
	require.NoError(t, err)
	assert.Equal(t, 3, len(finalData.Txs), "should use pending block with 3 transactions")
	assert.Equal(t, []byte("pending_tx1"), []byte(finalData.Txs[0]))
	assert.Equal(t, []byte("pending_tx2"), []byte(finalData.Txs[1]))
	assert.Equal(t, []byte("pending_tx3"), []byte(finalData.Txs[2]))

	// Verify that the header data hash matches the pending block
	assert.Equal(t, pendingData.DACommitment(), finalHeader.DataHash)

	// Verify broadcasters were called with the pending block data
	// The testify mock framework tracks calls automatically

	// Verify the executor state was updated correctly
	finalState := exec2.getLastState()
	assert.Equal(t, uint64(2), finalState.LastBlockHeight)
	assert.Equal(t, []byte("new_root_2"), finalState.AppHash)

	// Verify that no calls were made to the sequencer (confirming pending block was used)
	mockSeq2.AssertNotCalled(t, "GetNextBatch")
}

func TestExecutor_RestartNoPendingHeader(t *testing.T) {
	// Test that restart works normally when there's no pending header
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	cacheManager, err := cache.NewManager(config.DefaultConfig(), memStore, zerolog.Nop())
	require.NoError(t, err)

	metrics := common.NopMetrics()
	addr, _, signerWrapper := buildTestSigner(t)

	cfg := config.DefaultConfig()
	cfg.Node.BlockTime = config.DurationWrapper{Duration: 10 * time.Millisecond}
	cfg.Node.MaxPendingHeadersAndData = 1000

	gen := genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		StartTime:       time.Now().Add(-time.Second),
		ProposerAddress: addr,
	}

	// Create first executor and produce one block
	mockExec1 := testmocks.NewMockExecutor(t)
	mockSeq1 := testmocks.NewMockSequencer(t)
	hb1 := common.NewMockBroadcaster[*types.SignedHeader](t)
	hb1.EXPECT().WriteToStoreAndBroadcast(mock.Anything, mock.Anything).Return(nil).Maybe()
	db1 := common.NewMockBroadcaster[*types.Data](t)
	db1.EXPECT().WriteToStoreAndBroadcast(mock.Anything, mock.Anything).Return(nil).Maybe()

	exec1, err := NewExecutor(
		memStore,
		mockExec1,
		mockSeq1,
		signerWrapper,
		cacheManager,
		metrics,
		cfg,
		gen,
		hb1,
		db1,
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	require.NoError(t, err)

	initStateRoot := []byte("init_root")
	mockExec1.EXPECT().InitChain(mock.Anything, mock.AnythingOfType("time.Time"), gen.InitialHeight, gen.ChainID).
		Return(initStateRoot, uint64(1024), nil).Once()
	require.NoError(t, exec1.initializeState())

	exec1.ctx, exec1.cancel = context.WithCancel(context.Background())

	// Produce first block
	mockSeq1.EXPECT().GetNextBatch(mock.Anything, mock.AnythingOfType("sequencer.GetNextBatchRequest")).
		RunAndReturn(func(ctx context.Context, req coreseq.GetNextBatchRequest) (*coreseq.GetNextBatchResponse, error) {
			return &coreseq.GetNextBatchResponse{
				Batch: &coreseq.Batch{
					Transactions: [][]byte{[]byte("tx1")},
				},
				Timestamp: time.Now(),
			}, nil
		}).Once()

	mockExec1.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, uint64(1), mock.AnythingOfType("time.Time"), initStateRoot).
		Return([]byte("new_root_1"), uint64(1024), nil).Once()

	err = exec1.produceBlock()
	require.NoError(t, err)

	// Stop first executor
	exec1.cancel()

	// Create second executor (restart)
	mockExec2 := testmocks.NewMockExecutor(t)
	mockSeq2 := testmocks.NewMockSequencer(t)
	hb2 := common.NewMockBroadcaster[*types.SignedHeader](t)
	hb2.EXPECT().WriteToStoreAndBroadcast(mock.Anything, mock.Anything).Return(nil).Maybe()
	db2 := common.NewMockBroadcaster[*types.Data](t)
	db2.EXPECT().WriteToStoreAndBroadcast(mock.Anything, mock.Anything).Return(nil).Maybe()

	exec2, err := NewExecutor(
		memStore,
		mockExec2,
		mockSeq2,
		signerWrapper,
		cacheManager,
		metrics,
		cfg,
		gen,
		hb2,
		db2,
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	require.NoError(t, err)

	require.NoError(t, exec2.initializeState())
	exec2.ctx, exec2.cancel = context.WithCancel(context.Background())
	defer exec2.cancel()

	// Verify state loaded correctly
	state := exec2.getLastState()
	assert.Equal(t, uint64(1), state.LastBlockHeight)

	// Now produce next block - should go through normal sequencer flow since no pending block
	mockSeq2.EXPECT().GetNextBatch(mock.Anything, mock.AnythingOfType("sequencer.GetNextBatchRequest")).
		RunAndReturn(func(ctx context.Context, req coreseq.GetNextBatchRequest) (*coreseq.GetNextBatchResponse, error) {
			return &coreseq.GetNextBatchResponse{
				Batch: &coreseq.Batch{
					Transactions: [][]byte{[]byte("restart_tx")},
				},
				Timestamp: time.Now(),
			}, nil
		}).Once()

	mockExec2.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, uint64(2), mock.AnythingOfType("time.Time"), []byte("new_root_1")).
		Return([]byte("new_root_2"), uint64(1024), nil).Once()

	err = exec2.produceBlock()
	require.NoError(t, err)

	// Verify normal operation
	h, err := memStore.Height(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(2), h)

	// Verify sequencer was called (normal flow)
	mockSeq2.AssertCalled(t, "GetNextBatch", mock.Anything, mock.AnythingOfType("sequencer.GetNextBatchRequest"))
}
