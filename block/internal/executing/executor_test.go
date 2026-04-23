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
	coreseq "github.com/evstack/ev-node/core/sequencer"
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

func TestExecutor_CreateBlock_UsesScheduledProposerForHeight(t *testing.T) {
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	cacheManager, err := cache.NewManager(config.DefaultConfig(), memStore, zerolog.Nop())
	require.NoError(t, err)

	metrics := common.NopMetrics()
	oldAddr, oldSignerInfo, _ := buildTestSigner(t)
	newAddr, newSignerInfo, newSigner := buildTestSigner(t)

	entry1, err := genesis.NewProposerScheduleEntry(1, oldSignerInfo.PubKey)
	require.NoError(t, err)
	entry2, err := genesis.NewProposerScheduleEntry(2, newSignerInfo.PubKey)
	require.NoError(t, err)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		InitialHeight:          1,
		StartTime:              time.Now().Add(-time.Second),
		ProposerAddress:        entry1.Address,
		ProposerSchedule:       []genesis.ProposerScheduleEntry{entry1, entry2},
		DAEpochForcedInclusion: 1,
	}

	executor, err := NewExecutor(
		memStore,
		nil,
		nil,
		newSigner,
		cacheManager,
		metrics,
		config.DefaultConfig(),
		gen,
		nil,
		nil,
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
		nil,
	)
	require.NoError(t, err)

	prevHeader := &types.SignedHeader{
		Header: types.Header{
			Version: types.InitStateVersion,
			BaseHeader: types.BaseHeader{
				ChainID: gen.ChainID,
				Height:  1,
				Time:    uint64(gen.StartTime.UnixNano()),
			},
			AppHash:         []byte("state-root-0"),
			ProposerAddress: oldAddr,
			DataHash:        common.DataHashForEmptyTxs,
		},
		Signature: types.Signature([]byte("sig-1")),
		Signer:    oldSignerInfo,
	}
	prevData := &types.Data{
		Metadata: &types.Metadata{
			ChainID: gen.ChainID,
			Height:  1,
			Time:    prevHeader.BaseHeader.Time,
		},
		Txs: nil,
	}

	batch, err := memStore.NewBatch(context.Background())
	require.NoError(t, err)
	require.NoError(t, batch.SaveBlockData(prevHeader, prevData, &prevHeader.Signature))
	require.NoError(t, batch.SetHeight(1))
	require.NoError(t, batch.Commit())

	executor.setLastState(types.State{
		Version:         types.InitStateVersion,
		ChainID:         gen.ChainID,
		InitialHeight:   gen.InitialHeight,
		LastBlockHeight: 1,
		LastBlockTime:   prevHeader.Time(),
		LastHeaderHash:  prevHeader.Hash(),
		AppHash:         []byte("state-root-1"),
	})

	header, data, err := executor.CreateBlock(context.Background(), 2, &BatchData{
		Batch: &coreseq.Batch{},
		Time:  time.Now(),
	})
	require.NoError(t, err)
	require.Equal(t, newAddr, header.ProposerAddress)
	require.Equal(t, newAddr, header.Signer.Address)
	require.Equal(t, uint64(2), data.Height())
}
