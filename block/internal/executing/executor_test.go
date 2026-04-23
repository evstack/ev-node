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

// TestNewExecutor_RejectsSignerOutsideSchedule verifies that a signer whose
// address does not appear anywhere in the proposer schedule cannot start the
// executor. This prevents a misconfigured replacement key from coming up as
// an aggregator on a chain it was never scheduled on.
func TestNewExecutor_RejectsSignerOutsideSchedule(t *testing.T) {
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	cacheManager, err := cache.NewManager(config.DefaultConfig(), memStore, zerolog.Nop())
	require.NoError(t, err)

	_, scheduledSigner, _ := buildTestSigner(t)
	_, _, strayerSigner := buildTestSigner(t)

	entry, err := genesis.NewProposerScheduleEntry(1, scheduledSigner.PubKey)
	require.NoError(t, err)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		InitialHeight:          1,
		StartTime:              time.Now(),
		ProposerAddress:        entry.Address,
		ProposerSchedule:       []genesis.ProposerScheduleEntry{entry},
		DAEpochForcedInclusion: 1,
	}

	_, err = NewExecutor(
		memStore, nil, nil, strayerSigner, cacheManager,
		common.NopMetrics(), config.DefaultConfig(), gen,
		nil, nil, zerolog.Nop(), common.DefaultBlockOptions(),
		make(chan error, 1), nil,
	)
	require.ErrorIs(t, err, common.ErrNotProposer)
}

// TestExecutor_CreateBlock_RejectsSignerAtWrongHeight verifies that a signer
// which is scheduled (so startup succeeds) but not active at the current
// height cannot produce a block. This guards the per-height proposer check
// inside CreateBlock — without it, a rotation could be jumped ahead or
// rolled back by whichever signer the operator happens to start.
func TestExecutor_CreateBlock_RejectsSignerAtWrongHeight(t *testing.T) {
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	memStore := store.New(ds)

	cacheManager, err := cache.NewManager(config.DefaultConfig(), memStore, zerolog.Nop())
	require.NoError(t, err)

	oldAddr, oldSignerInfo, oldSigner := buildTestSigner(t)
	_, newSignerInfo, _ := buildTestSigner(t)

	entry1, err := genesis.NewProposerScheduleEntry(1, oldSignerInfo.PubKey)
	require.NoError(t, err)
	// Second entry activates at height 5. The old signer is scheduled at
	// height 1 and is NOT the proposer for height 5+.
	entry2, err := genesis.NewProposerScheduleEntry(5, newSignerInfo.PubKey)
	require.NoError(t, err)

	gen := genesis.Genesis{
		ChainID:                "test-chain",
		InitialHeight:          1,
		StartTime:              time.Now().Add(-time.Second),
		ProposerAddress:        entry1.Address,
		ProposerSchedule:       []genesis.ProposerScheduleEntry{entry1, entry2},
		DAEpochForcedInclusion: 1,
	}

	// Start the executor as the old signer — it IS in the schedule at
	// height 1, so NewExecutor must accept it.
	executor, err := NewExecutor(
		memStore, nil, nil, oldSigner, cacheManager,
		common.NopMetrics(), config.DefaultConfig(), gen,
		nil, nil, zerolog.Nop(), common.DefaultBlockOptions(),
		make(chan error, 1), nil,
	)
	require.NoError(t, err)

	// Seed a height-4 block so CreateBlock(5) has a parent to reference.
	prevHeader := &types.SignedHeader{
		Header: types.Header{
			Version: types.InitStateVersion,
			BaseHeader: types.BaseHeader{
				ChainID: gen.ChainID,
				Height:  4,
				Time:    uint64(gen.StartTime.UnixNano()),
			},
			AppHash:         []byte("state-root-4"),
			ProposerAddress: oldAddr,
			DataHash:        common.DataHashForEmptyTxs,
		},
		Signature: types.Signature([]byte("sig-4")),
		Signer:    oldSignerInfo,
	}
	prevData := &types.Data{
		Metadata: &types.Metadata{
			ChainID: gen.ChainID,
			Height:  4,
			Time:    prevHeader.BaseHeader.Time,
		},
	}

	batch, err := memStore.NewBatch(context.Background())
	require.NoError(t, err)
	require.NoError(t, batch.SaveBlockData(prevHeader, prevData, &prevHeader.Signature))
	require.NoError(t, batch.SetHeight(4))
	require.NoError(t, batch.Commit())

	executor.setLastState(types.State{
		Version:         types.InitStateVersion,
		ChainID:         gen.ChainID,
		InitialHeight:   gen.InitialHeight,
		LastBlockHeight: 4,
		LastBlockTime:   prevHeader.Time(),
		LastHeaderHash:  prevHeader.Hash(),
		AppHash:         []byte("state-root-4"),
	})

	// Height 5 belongs to the NEW signer per the schedule — the old
	// signer must be rejected even though it's a known schedule member.
	_, _, err = executor.CreateBlock(context.Background(), 5, &BatchData{
		Batch: &coreseq.Batch{},
		Time:  time.Now(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "proposer")
}
