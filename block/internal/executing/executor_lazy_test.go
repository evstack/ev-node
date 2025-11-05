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

func TestLazyMode_ProduceBlockLogic(t *testing.T) {
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	cacheManager, err := cache.NewManager(config.DefaultConfig(), memStore, zerolog.Nop())
	require.NoError(t, err)

	metrics := common.NopMetrics()

	addr, _, signerWrapper := buildTestSigner(t)

	cfg := config.DefaultConfig()
	cfg.Node.BlockTime = config.DurationWrapper{Duration: 10 * time.Millisecond}
	cfg.Node.LazyMode = true
	cfg.Node.MaxPendingHeadersAndData = 1000

	gen := genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		StartTime:       time.Now().Add(-time.Second),
		ProposerAddress: addr,
	}

	mockExec := testmocks.NewMockExecutor(t)
	mockSeq := testmocks.NewMockSequencer(t)
	hb := common.NewMockBroadcaster[*types.SignedHeader](t)
	hb.EXPECT().WriteToStoreAndBroadcast(mock.Anything, mock.Anything).Return(nil).Maybe()
	db := common.NewMockBroadcaster[*types.Data](t)
	db.EXPECT().WriteToStoreAndBroadcast(mock.Anything, mock.Anything).Return(nil).Maybe()

	exec, err := NewExecutor(
		memStore,
		mockExec,
		mockSeq,
		signerWrapper,
		cacheManager,
		metrics,
		cfg,
		gen,
		hb,
		db,
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	require.NoError(t, err)

	// Initialize state
	initStateRoot := []byte("init_root")
	mockExec.EXPECT().InitChain(mock.Anything, mock.AnythingOfType("time.Time"), gen.InitialHeight, gen.ChainID).
		Return(initStateRoot, uint64(1024), nil).Once()
	require.NoError(t, exec.initializeState())

	// Set up context for the executor (normally done in Start method)
	exec.ctx, exec.cancel = context.WithCancel(context.Background())
	defer exec.cancel()

	// Test 1: Lazy mode should produce blocks when called directly (simulating lazy timer)
	mockSeq.EXPECT().GetNextBatch(mock.Anything, mock.AnythingOfType("sequencer.GetNextBatchRequest")).
		RunAndReturn(func(ctx context.Context, req coreseq.GetNextBatchRequest) (*coreseq.GetNextBatchResponse, error) {
			return &coreseq.GetNextBatchResponse{
				Batch:     &coreseq.Batch{Transactions: nil}, // Empty batch
				Timestamp: time.Now(),
			}, nil
		}).Once()

	mockExec.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, uint64(1), mock.AnythingOfType("time.Time"), initStateRoot).
		Return([]byte("new_root_1"), uint64(1024), nil).Once()

	// Direct call to produceBlock should work (this is what lazy timer does)
	err = exec.produceBlock()
	require.NoError(t, err)

	h1, err := memStore.Height(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(1), h1, "lazy mode should produce block when called directly")

	// Test 2: Produce another block with transactions
	mockSeq.EXPECT().GetNextBatch(mock.Anything, mock.AnythingOfType("sequencer.GetNextBatchRequest")).
		RunAndReturn(func(ctx context.Context, req coreseq.GetNextBatchRequest) (*coreseq.GetNextBatchResponse, error) {
			return &coreseq.GetNextBatchResponse{
				Batch: &coreseq.Batch{
					Transactions: [][]byte{[]byte("tx1"), []byte("tx2")},
				},
				Timestamp: time.Now(),
			}, nil
		}).Once()

	mockExec.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, uint64(2), mock.AnythingOfType("time.Time"), []byte("new_root_1")).
		Return([]byte("new_root_2"), uint64(1024), nil).Once()

	err = exec.produceBlock()
	require.NoError(t, err)

	h2, err := memStore.Height(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(2), h2, "should produce block with transactions")

	// Verify blocks were stored correctly
	sh1, data1, err := memStore.GetBlockData(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 0, len(data1.Txs), "first block should be empty")
	assert.EqualValues(t, common.DataHashForEmptyTxs, sh1.DataHash)

	sh2, data2, err := memStore.GetBlockData(context.Background(), 2)
	require.NoError(t, err)
	assert.Equal(t, 2, len(data2.Txs), "second block should have 2 transactions")
	assert.NotEqual(t, common.DataHashForEmptyTxs, sh2.DataHash, "second block should not have empty data hash")
}

func TestRegularMode_ProduceBlockLogic(t *testing.T) {
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	cacheManager, err := cache.NewManager(config.DefaultConfig(), memStore, zerolog.Nop())
	require.NoError(t, err)

	metrics := common.NopMetrics()

	addr, _, signerWrapper := buildTestSigner(t)

	cfg := config.DefaultConfig()
	cfg.Node.BlockTime = config.DurationWrapper{Duration: 10 * time.Millisecond}
	cfg.Node.LazyMode = false // Regular mode
	cfg.Node.MaxPendingHeadersAndData = 1000

	gen := genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		StartTime:       time.Now().Add(-time.Second),
		ProposerAddress: addr,
	}

	mockExec := testmocks.NewMockExecutor(t)
	mockSeq := testmocks.NewMockSequencer(t)
	hb := common.NewMockBroadcaster[*types.SignedHeader](t)
	hb.EXPECT().WriteToStoreAndBroadcast(mock.Anything, mock.Anything).Return(nil).Maybe()
	db := common.NewMockBroadcaster[*types.Data](t)
	db.EXPECT().WriteToStoreAndBroadcast(mock.Anything, mock.Anything).Return(nil).Maybe()

	exec, err := NewExecutor(
		memStore,
		mockExec,
		mockSeq,
		signerWrapper,
		cacheManager,
		metrics,
		cfg,
		gen,
		hb,
		db,
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	require.NoError(t, err)

	// Initialize state
	initStateRoot := []byte("init_root")
	mockExec.EXPECT().InitChain(mock.Anything, mock.AnythingOfType("time.Time"), gen.InitialHeight, gen.ChainID).
		Return(initStateRoot, uint64(1024), nil).Once()
	require.NoError(t, exec.initializeState())

	// Set up context for the executor (normally done in Start method)
	exec.ctx, exec.cancel = context.WithCancel(context.Background())
	defer exec.cancel()

	// Test: Regular mode should produce blocks regardless of transaction availability
	mockSeq.EXPECT().GetNextBatch(mock.Anything, mock.AnythingOfType("sequencer.GetNextBatchRequest")).
		RunAndReturn(func(ctx context.Context, req coreseq.GetNextBatchRequest) (*coreseq.GetNextBatchResponse, error) {
			return &coreseq.GetNextBatchResponse{
				Batch:     &coreseq.Batch{Transactions: nil}, // Empty batch
				Timestamp: time.Now(),
			}, nil
		}).Once()

	mockExec.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, uint64(1), mock.AnythingOfType("time.Time"), initStateRoot).
		Return([]byte("new_root_1"), uint64(1024), nil).Once()

	err = exec.produceBlock()
	require.NoError(t, err)

	h1, err := memStore.Height(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(1), h1, "regular mode should produce block even without transactions")

	// Verify the block
	sh, data, err := memStore.GetBlockData(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 0, len(data.Txs), "block should be empty")
	assert.EqualValues(t, common.DataHashForEmptyTxs, sh.DataHash)
}
