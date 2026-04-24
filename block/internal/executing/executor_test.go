package executing

import (
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
	headerBroadcaster := common.NewMockBroadcaster[*types.P2PSignedHeader](t)
	dataBroadcaster := common.NewMockBroadcaster[*types.P2PData](t)

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
		nil,
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
		nil,
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
