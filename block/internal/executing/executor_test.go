package executing

import (
	"context"
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
