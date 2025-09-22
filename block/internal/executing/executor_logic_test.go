package executing

import (
	"context"
	crand "crypto/rand"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	coreseq "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	pkgsigner "github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/signer/noop"
	"github.com/evstack/ev-node/pkg/store"
	testmocks "github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
	"github.com/stretchr/testify/mock"
)

// buildTestSigner returns a signer and its address for use in tests
func buildTestSigner(t *testing.T) (signerAddr []byte, tSigner types.Signer, s pkgsigner.Signer) {
	t.Helper()
	priv, _, err := crypto.GenerateEd25519Key(crand.Reader)
	require.NoError(t, err)
	n, err := noop.NewNoopSigner(priv)
	require.NoError(t, err)
	addr, err := n.GetAddress()
	require.NoError(t, err)
	pub, err := n.GetPublic()
	require.NoError(t, err)
	return addr, types.Signer{PubKey: pub, Address: addr}, n
}

func TestProduceBlock_EmptyBatch_SetsEmptyDataHash(t *testing.T) {
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	cacheManager, err := cache.NewManager(config.DefaultConfig(), memStore, zerolog.Nop())
	require.NoError(t, err)

	metrics := common.NopMetrics()

	// signer and genesis with correct proposer
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

	// Use mocks for executor and sequencer
	mockExec := testmocks.NewMockExecutor(t)
	mockSeq := testmocks.NewMockSequencer(t)

	// Broadcasters are required by produceBlock; use simple mocks
	hb := &mockBroadcaster[*types.SignedHeader]{}
	db := &mockBroadcaster[*types.Data]{}

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

	// Expect InitChain to be called
	initStateRoot := []byte("init_root")
	mockExec.EXPECT().InitChain(mock.Anything, mock.AnythingOfType("time.Time"), gen.InitialHeight, gen.ChainID).
		Return(initStateRoot, uint64(1024), nil).Once()

	// initialize state (creates genesis block in store and sets state)
	require.NoError(t, exec.initializeState())

	// Set up context for the executor (normally done in Start method)
	exec.ctx, exec.cancel = context.WithCancel(context.Background())
	defer exec.cancel()

	// sequencer returns empty batch
	mockSeq.EXPECT().GetNextBatch(mock.Anything, mock.AnythingOfType("sequencer.GetNextBatchRequest")).
		RunAndReturn(func(ctx context.Context, req coreseq.GetNextBatchRequest) (*coreseq.GetNextBatchResponse, error) {
			return &coreseq.GetNextBatchResponse{Batch: &coreseq.Batch{Transactions: nil}, Timestamp: time.Now()}, nil
		}).Once()

	// executor ExecuteTxs called with empty txs and previous state root
	mockExec.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, uint64(1), mock.AnythingOfType("time.Time"), initStateRoot).
		Return([]byte("new_root"), uint64(1024), nil).Once()

	// produce one block
	err = exec.produceBlock()
	require.NoError(t, err)

	// Verify height and stored block
	h, err := memStore.Height(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(1), h)

	sh, data, err := memStore.GetBlockData(context.Background(), 1)
	require.NoError(t, err)
	// Expect empty txs and special empty data hash marker
	assert.Equal(t, 0, len(data.Txs))
	assert.EqualValues(t, common.DataHashForEmptyTxs, sh.DataHash)

	// Broadcasters should have been called with the produced header and data
	assert.True(t, hb.called)
	assert.True(t, db.called)
}

func TestPendingLimit_SkipsProduction(t *testing.T) {
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	cacheManager, err := cache.NewManager(config.DefaultConfig(), memStore, zerolog.Nop())
	require.NoError(t, err)

	metrics := common.NopMetrics()

	addr, _, signerWrapper := buildTestSigner(t)

	cfg := config.DefaultConfig()
	cfg.Node.BlockTime = config.DurationWrapper{Duration: 10 * time.Millisecond}
	cfg.Node.MaxPendingHeadersAndData = 1 // low limit to trigger skip quickly

	gen := genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		StartTime:       time.Now().Add(-time.Second),
		ProposerAddress: addr,
	}

	mockExec := testmocks.NewMockExecutor(t)
	mockSeq := testmocks.NewMockSequencer(t)
	hb := &mockBroadcaster[*types.SignedHeader]{}
	db := &mockBroadcaster[*types.Data]{}

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

	mockExec.EXPECT().InitChain(mock.Anything, mock.AnythingOfType("time.Time"), gen.InitialHeight, gen.ChainID).
		Return([]byte("i0"), uint64(1024), nil).Once()
	require.NoError(t, exec.initializeState())

	// Set up context for the executor (normally done in Start method)
	exec.ctx, exec.cancel = context.WithCancel(context.Background())
	defer exec.cancel()

	// First production should succeed
	// Return empty batch again
	mockSeq.EXPECT().GetNextBatch(mock.Anything, mock.AnythingOfType("sequencer.GetNextBatchRequest")).
		RunAndReturn(func(ctx context.Context, req coreseq.GetNextBatchRequest) (*coreseq.GetNextBatchResponse, error) {
			return &coreseq.GetNextBatchResponse{Batch: &coreseq.Batch{Transactions: nil}, Timestamp: time.Now()}, nil
		}).Once()
	// ExecuteTxs with empty
	mockExec.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, uint64(1), mock.AnythingOfType("time.Time"), []byte("i0")).
		Return([]byte("i1"), uint64(1024), nil).Once()

	require.NoError(t, exec.produceBlock())
	h1, err := memStore.Height(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(1), h1)

	// With limit=1 and lastSubmitted default 0, pending >= 1 so next production should be skipped
	// No new expectations; produceBlock should return early before hitting sequencer
	require.NoError(t, exec.produceBlock())
	h2, err := memStore.Height(context.Background())
	require.NoError(t, err)
	assert.Equal(t, h1, h2, "height should not change when production is skipped")
}
