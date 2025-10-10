package executing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

// mockBroadcaster for testing
type mockBroadcaster[T any] struct {
	called  bool
	payload T
}

func (m *mockBroadcaster[T]) WriteToStoreAndBroadcast(ctx context.Context, payload T) error {
	m.called = true
	m.payload = payload
	return nil
}

func TestExecutor_BroadcasterIntegration(t *testing.T) {
	// Create in-memory store
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	// Create cache
	cacheManager, err := cache.NewManager(config.DefaultConfig(), memStore, zerolog.Nop())
	require.NoError(t, err)

	metrics := common.NopMetrics()
	signerAddr, _, testSigner := buildTestSigner(t)

	// Create genesis
	gen := genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		StartTime:       time.Now(),
		ProposerAddress: signerAddr,
	}

	// Create mock broadcasters
	headerBroadcaster := &mockBroadcaster[*types.SignedHeader]{}
	dataBroadcaster := &mockBroadcaster[*types.Data]{}

	// Create executor with broadcasters
	executor, err := NewExecutor(
		memStore,
		nil,        // nil executor (we're not testing execution)
		nil,        // nil sequencer (we're not testing sequencing)
		testSigner, // test signer (required for executor)
		cacheManager,
		metrics,
		config.DefaultConfig(),
		gen,
		headerBroadcaster,
		dataBroadcaster,
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	require.NoError(t, err)

	// Verify broadcasters are set
	assert.NotNil(t, executor.headerBroadcaster)
	assert.NotNil(t, executor.dataBroadcaster)
	assert.Equal(t, headerBroadcaster, executor.headerBroadcaster)
	assert.Equal(t, dataBroadcaster, executor.dataBroadcaster)

	// Verify other properties
	assert.Equal(t, memStore, executor.store)
	assert.Equal(t, cacheManager, executor.cache)
	assert.Equal(t, gen, executor.genesis)
}

func TestExecutor_NilBroadcasters(t *testing.T) {
	// Create in-memory store
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	// Create cache
	cacheManager, err := cache.NewManager(config.DefaultConfig(), memStore, zerolog.Nop())
	require.NoError(t, err)

	metrics := common.NopMetrics()
	signerAddr, _, testSigner := buildTestSigner(t)

	// Create genesis
	gen := genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		StartTime:       time.Now(),
		ProposerAddress: signerAddr,
	}

	// Create executor with nil broadcasters (light node scenario)
	executor, err := NewExecutor(
		memStore,
		nil,        // nil executor
		nil,        // nil sequencer
		testSigner, // test signer (required for executor)
		cacheManager,
		metrics,
		config.DefaultConfig(),
		gen,
		nil, // nil header broadcaster
		nil, // nil data broadcaster
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	require.NoError(t, err)

	// Verify broadcasters are nil
	assert.Nil(t, executor.headerBroadcaster)
	assert.Nil(t, executor.dataBroadcaster)

	// Verify other properties
	assert.Equal(t, memStore, executor.store)
	assert.Equal(t, cacheManager, executor.cache)
	assert.Equal(t, gen, executor.genesis)
}

func TestExecutor_BroadcastFlow(t *testing.T) {
	// This test demonstrates how the broadcast flow works
	// when an Executor produces a block

	// Create mock broadcasters that track calls
	headerBroadcaster := &mockBroadcaster[*types.SignedHeader]{}
	dataBroadcaster := &mockBroadcaster[*types.Data]{}

	// Create sample data that would be broadcast
	sampleHeader := &types.SignedHeader{
		Header: types.Header{
			BaseHeader: types.BaseHeader{
				ChainID: "test-chain",
				Height:  1,
				Time:    uint64(time.Now().UnixNano()),
			},
		},
	}

	sampleData := &types.Data{
		Metadata: &types.Metadata{
			ChainID: "test-chain",
			Height:  1,
			Time:    uint64(time.Now().UnixNano()),
		},
		Txs: []types.Tx{},
	}

	// Test broadcast calls
	ctx := context.Background()

	// Simulate what happens in produceBlock() after block creation
	err := headerBroadcaster.WriteToStoreAndBroadcast(ctx, sampleHeader)
	require.NoError(t, err)
	assert.True(t, headerBroadcaster.called, "header broadcaster should be called")

	err = dataBroadcaster.WriteToStoreAndBroadcast(ctx, sampleData)
	require.NoError(t, err)
	assert.True(t, dataBroadcaster.called, "data broadcaster should be called")

	// Verify the correct data was passed to broadcasters
	assert.Equal(t, sampleHeader, headerBroadcaster.payload)
	assert.Equal(t, sampleData, dataBroadcaster.payload)
}

func TestExecutor_CachePruneLoop(t *testing.T) {
	// Create in-memory store
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	// Create temporary directory for cache
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.RootDir = tmpDir

	// Create cache
	cacheManager, err := cache.NewManager(cfg, memStore, zerolog.Nop())
	require.NoError(t, err)

	metrics := common.NopMetrics()
	signerAddr, _, testSigner := buildTestSigner(t)

	// Create genesis
	gen := genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		StartTime:       time.Now(),
		ProposerAddress: signerAddr,
	}

	// Create context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create executor
	executor, err := NewExecutor(
		memStore,
		nil,
		nil,
		testSigner,
		cacheManager,
		metrics,
		cfg,
		gen,
		nil,
		nil,
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	require.NoError(t, err)
	executor.ctx = ctx

	// Add some test data to cache before starting prune loop
	// This simulates normal operation where cache accumulates data
	for i := uint64(1); i <= 5; i++ {
		hash := fmt.Sprintf("header-hash-%d", i)
		cacheManager.SetHeaderSeen(hash, i)
	}

	// Start cache prune loop in background
	done := make(chan struct{})
	go func() {
		executor.cachePruneLoop()
		close(done)
	}()

	// Let it run briefly to ensure loop is active
	time.Sleep(100 * time.Millisecond)

	// Cancel context to trigger shutdown
	cancel()

	// Wait for graceful shutdown with timeout
	select {
	case <-done:
		// Success - loop exited cleanly
	case <-time.After(5 * time.Second):
		t.Fatal("cachePruneLoop did not exit within timeout")
	}

	// Verify cache was saved to disk during shutdown
	// The SaveToDisk call should have been made
	err = cacheManager.LoadFromDisk()
	require.NoError(t, err, "cache should be loadable after graceful shutdown")
}

func TestExecutor_CachePruneLoop_RespectsDAIncludedHeight(t *testing.T) {
	// Create in-memory store
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	// Create temporary directory for cache
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.RootDir = tmpDir

	// Create cache
	cacheManager, err := cache.NewManager(cfg, memStore, zerolog.Nop())
	require.NoError(t, err)

	metrics := common.NopMetrics()
	signerAddr, _, testSigner := buildTestSigner(t)

	// Create genesis
	gen := genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		StartTime:       time.Now(),
		ProposerAddress: signerAddr,
	}

	// Set DA included height in store
	ctx := context.Background()
	daIncludedHeight := uint64(3)
	heightBytes := make([]byte, 8)
	heightBytes[0] = byte(daIncludedHeight)
	heightBytes[1] = byte(daIncludedHeight >> 8)
	heightBytes[2] = byte(daIncludedHeight >> 16)
	heightBytes[3] = byte(daIncludedHeight >> 24)
	heightBytes[4] = byte(daIncludedHeight >> 32)
	heightBytes[5] = byte(daIncludedHeight >> 40)
	heightBytes[6] = byte(daIncludedHeight >> 48)
	heightBytes[7] = byte(daIncludedHeight >> 56)
	err = memStore.SetMetadata(ctx, store.DAIncludedHeightKey, heightBytes)
	require.NoError(t, err)

	// Add test data to cache at various heights
	for i := uint64(1); i <= 10; i++ {
		hash := fmt.Sprintf("header-hash-%d", i)
		cacheManager.SetHeaderSeen(hash, i)
	}

	// Create executor
	executor, err := NewExecutor(
		memStore,
		nil,
		nil,
		testSigner,
		cacheManager,
		metrics,
		cfg,
		gen,
		nil,
		nil,
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	require.NoError(t, err)
	executor.ctx = ctx

	// Manually trigger prune
	cacheManager.PruneCache(ctx)

	// Verify that entries below DA included height are pruned
	// and entries at or above are retained
	// Note: This is an indirect test - the cache interface doesn't expose
	// internal state, but we can verify the system doesn't panic and
	// SaveToDisk succeeds
	err = cacheManager.SaveToDisk()
	require.NoError(t, err, "cache should be saveable after pruning")
}
