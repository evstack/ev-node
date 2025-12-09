package syncing

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	da "github.com/evstack/ev-node/block/internal/da"
	datestclient "github.com/evstack/ev-node/block/internal/da/testclient"
	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/store"
	testmocks "github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
)

func TestVerifyForcedInclusionTxs_AllTransactionsIncluded(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)

	cm, err := cache.NewCacheManager(config.DefaultConfig(), zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{
		ChainID:                "tchain",
		InitialHeight:          1,
		StartTime:              time.Now().Add(-time.Second),
		ProposerAddress:        addr,
		DAStartHeight:          0,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.ForcedInclusionNamespace = "nsForcedInclusion"

	mockExec := testmocks.NewMockExecutor(t)
	mockExec.EXPECT().InitChain(mock.Anything, mock.Anything, uint64(1), "tchain").
		Return([]byte("app0"), uint64(1024), nil).Once()

	mockDA := testmocks.NewMockDA(t)

	daClient := datestclient.New(datestclient.Config{
		DA:                       mockDA,
		Namespace:                cfg.DA.Namespace,
		DataNamespace:            cfg.DA.DataNamespace,
		ForcedInclusionNamespace: cfg.DA.ForcedInclusionNamespace,
	})
	daRetriever := NewDARetriever(daClient, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	s := NewSyncer(
		st,
		mockExec,
		daClient,
		cm,
		common.NopMetrics(),
		cfg,
		gen,
		common.NewMockBroadcaster[*types.SignedHeader](t),
		common.NewMockBroadcaster[*types.Data](t),
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	s.daRetriever = daRetriever
	s.fiRetriever = fiRetriever

	require.NoError(t, s.initializeState())
	s.ctx = context.Background()

	// Mock DA to return forced inclusion transactions
	namespaceForcedInclusionBz := datypes.NamespaceFromString(cfg.DA.GetForcedInclusionNamespace()).Bytes()

	// Create forced inclusion transaction blob (SignedData) in DA
	dataBin, _ := makeSignedDataBytes(t, gen.ChainID, 10, addr, pub, signer, 2)

	// With DAStartHeight=0, epoch size=1, daHeight=0 -> epoch boundaries are [0, 0]
	// Check epoch start only (end check is skipped when same as start)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(0), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&datypes.GetIDsResult{IDs: [][]byte{[]byte("fi1")}, Timestamp: time.Now()}, nil).Once()

	// Fetch epoch start data
	mockDA.EXPECT().Get(mock.Anything, mock.Anything, mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return([][]byte{dataBin}, nil).Once()

	// Create block data that includes the forced transaction blob
	data := makeData(gen.ChainID, 1, 1)
	data.Txs[0] = types.Tx(dataBin)

	currentState := s.GetLastState()
	currentState.DAHeight = 0

	// Verify - should pass since all forced txs are included
	err = s.verifyForcedInclusionTxs(currentState, data)
	require.NoError(t, err)
}

func TestVerifyForcedInclusionTxs_MissingTransactions(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)

	cm, err := cache.NewCacheManager(config.DefaultConfig(), zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{
		ChainID:                "tchain",
		InitialHeight:          1,
		StartTime:              time.Now().Add(-time.Second),
		ProposerAddress:        addr,
		DAStartHeight:          0,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.ForcedInclusionNamespace = "nsForcedInclusion"

	mockExec := testmocks.NewMockExecutor(t)
	mockExec.EXPECT().InitChain(mock.Anything, mock.Anything, uint64(1), "tchain").
		Return([]byte("app0"), uint64(1024), nil).Once()

	mockDA := testmocks.NewMockDA(t)

	daClient := datestclient.New(datestclient.Config{
		DA:                       mockDA,
		Namespace:                cfg.DA.Namespace,
		DataNamespace:            cfg.DA.DataNamespace,
		ForcedInclusionNamespace: cfg.DA.ForcedInclusionNamespace,
	})
	daRetriever := NewDARetriever(daClient, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	s := NewSyncer(
		st,
		mockExec,
		daClient,
		cm,
		common.NopMetrics(),
		cfg,
		gen,
		common.NewMockBroadcaster[*types.SignedHeader](t),
		common.NewMockBroadcaster[*types.Data](t),
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	s.daRetriever = daRetriever
	s.fiRetriever = fiRetriever

	require.NoError(t, s.initializeState())
	s.ctx = context.Background()

	// Mock DA to return forced inclusion transactions
	namespaceForcedInclusionBz := datypes.NamespaceFromString(cfg.DA.GetForcedInclusionNamespace()).Bytes()

	// Create forced inclusion transaction blob (SignedData) in DA
	dataBin, _ := makeSignedDataBytes(t, gen.ChainID, 10, addr, pub, signer, 2)

	// With DAStartHeight=0, epoch size=1, daHeight=0 -> epoch boundaries are [0, 0]
	// Check epoch start only (end check is skipped when same as start)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(0), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&datypes.GetIDsResult{IDs: [][]byte{[]byte("fi1")}, Timestamp: time.Now()}, nil).Once()

	// Fetch epoch start data
	mockDA.EXPECT().Get(mock.Anything, mock.Anything, mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return([][]byte{dataBin}, nil).Once()

	// Create block data that does NOT include the forced transaction blob
	data := makeData(gen.ChainID, 1, 2)
	data.Txs[0] = types.Tx([]byte("regular_tx_1"))
	data.Txs[1] = types.Tx([]byte("regular_tx_2"))

	currentState := s.GetLastState()
	currentState.DAHeight = 0

	// Verify - should pass since forced tx blob may be legitimately deferred within the epoch
	err = s.verifyForcedInclusionTxs(currentState, data)
	require.NoError(t, err)

	// Mock DA for next epoch to return no forced inclusion transactions
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(1), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&datypes.GetIDsResult{IDs: [][]byte{}, Timestamp: time.Now()}, nil).Once()

	// Now simulate moving to next epoch - should fail if tx still not included
	currentState.DAHeight = 1 // Move past epoch end (epoch was [0, 0])
	data2 := makeData(gen.ChainID, 2, 1)
	data2.Txs[0] = types.Tx([]byte("regular_tx_3"))

	err = s.verifyForcedInclusionTxs(currentState, data2)
	require.Error(t, err)
	require.Contains(t, err.Error(), "sequencer is malicious")
	require.Contains(t, err.Error(), "forced inclusion transactions from past epoch(s) not included")
}

func TestVerifyForcedInclusionTxs_PartiallyIncluded(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)

	cm, err := cache.NewCacheManager(config.DefaultConfig(), zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{
		ChainID:                "tchain",
		InitialHeight:          1,
		StartTime:              time.Now().Add(-time.Second),
		ProposerAddress:        addr,
		DAStartHeight:          0,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.ForcedInclusionNamespace = "nsForcedInclusion"

	mockExec := testmocks.NewMockExecutor(t)
	mockExec.EXPECT().InitChain(mock.Anything, mock.Anything, uint64(1), "tchain").
		Return([]byte("app0"), uint64(1024), nil).Once()

	mockDA := testmocks.NewMockDA(t)

	daClient := datestclient.New(datestclient.Config{
		DA:                       mockDA,
		Namespace:                cfg.DA.Namespace,
		DataNamespace:            cfg.DA.DataNamespace,
		ForcedInclusionNamespace: cfg.DA.ForcedInclusionNamespace,
	})
	daRetriever := NewDARetriever(daClient, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	s := NewSyncer(
		st,
		mockExec,
		daClient,
		cm,
		common.NopMetrics(),
		cfg,
		gen,
		common.NewMockBroadcaster[*types.SignedHeader](t),
		common.NewMockBroadcaster[*types.Data](t),
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	s.daRetriever = daRetriever
	s.fiRetriever = fiRetriever

	require.NoError(t, s.initializeState())
	s.ctx = context.Background()

	// Mock DA to return two forced inclusion transaction blobs
	namespaceForcedInclusionBz := datypes.NamespaceFromString(cfg.DA.GetForcedInclusionNamespace()).Bytes()

	// Create two forced inclusion transaction blobs in DA
	dataBin1, _ := makeSignedDataBytes(t, gen.ChainID, 10, addr, pub, signer, 2)
	dataBin2, _ := makeSignedDataBytes(t, gen.ChainID, 11, addr, pub, signer, 1)

	// With DAStartHeight=0, epoch size=1, daHeight=0 -> epoch boundaries are [0, 0]
	// Check epoch start only (end check is skipped when same as start)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(0), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&datypes.GetIDsResult{IDs: [][]byte{[]byte("fi1"), []byte("fi2")}, Timestamp: time.Now()}, nil).Once()

	// Fetch epoch start data
	mockDA.EXPECT().Get(mock.Anything, mock.Anything, mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return([][]byte{dataBin1, dataBin2}, nil).Once()

	// Create block data that includes only one of the forced transaction blobs
	data := makeData(gen.ChainID, 1, 2)
	data.Txs[0] = types.Tx(dataBin1)
	data.Txs[1] = types.Tx([]byte("regular_tx"))
	// dataBin2 is missing

	currentState := s.GetLastState()
	currentState.DAHeight = 0

	// Verify - should pass since dataBin2 may be legitimately deferred within the epoch
	err = s.verifyForcedInclusionTxs(currentState, data)
	require.NoError(t, err)

	// Mock DA for next epoch to return no forced inclusion transactions
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(1), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&datypes.GetIDsResult{IDs: [][]byte{}, Timestamp: time.Now()}, nil).Once()

	// Now simulate moving to next epoch - should fail if dataBin2 still not included
	currentState.DAHeight = 1 // Move past epoch end (epoch was [0, 0])
	data2 := makeData(gen.ChainID, 2, 1)
	data2.Txs[0] = types.Tx([]byte("regular_tx_3"))

	err = s.verifyForcedInclusionTxs(currentState, data2)
	require.Error(t, err)
	require.Contains(t, err.Error(), "sequencer is malicious")
	require.Contains(t, err.Error(), "forced inclusion transactions from past epoch(s) not included")
}

func TestVerifyForcedInclusionTxs_NoForcedTransactions(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)

	cm, err := cache.NewCacheManager(config.DefaultConfig(), zerolog.Nop())
	require.NoError(t, err)

	addr, _, _ := buildSyncTestSigner(t)
	gen := genesis.Genesis{
		ChainID:                "tchain",
		InitialHeight:          1,
		StartTime:              time.Now().Add(-time.Second),
		ProposerAddress:        addr,
		DAStartHeight:          0,
		DAEpochForcedInclusion: 1,
	}

	cfg := config.DefaultConfig()
	cfg.DA.ForcedInclusionNamespace = "nsForcedInclusion"

	mockExec := testmocks.NewMockExecutor(t)
	mockExec.EXPECT().InitChain(mock.Anything, mock.Anything, uint64(1), "tchain").
		Return([]byte("app0"), uint64(1024), nil).Once()

	mockDA := testmocks.NewMockDA(t)

	daClient := datestclient.New(datestclient.Config{
		DA:                       mockDA,
		Namespace:                cfg.DA.Namespace,
		DataNamespace:            cfg.DA.DataNamespace,
		ForcedInclusionNamespace: cfg.DA.ForcedInclusionNamespace,
	})
	daRetriever := NewDARetriever(daClient, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	s := NewSyncer(
		st,
		mockExec,
		daClient,
		cm,
		common.NopMetrics(),
		cfg,
		gen,
		common.NewMockBroadcaster[*types.SignedHeader](t),
		common.NewMockBroadcaster[*types.Data](t),
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	s.daRetriever = daRetriever
	s.fiRetriever = fiRetriever

	require.NoError(t, s.initializeState())
	s.ctx = context.Background()

	// Mock DA to return no forced inclusion transactions
	namespaceForcedInclusionBz := datypes.NamespaceFromString(cfg.DA.GetForcedInclusionNamespace()).Bytes()

	// With DAStartHeight=0, epoch size=1, daHeight=0 -> epoch boundaries are [0, 0]
	// Check epoch start only (end check is skipped when same as start)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(0), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&datypes.GetIDsResult{IDs: [][]byte{}, Timestamp: time.Now()}, nil).Once()

	// Create block data
	data := makeData(gen.ChainID, 1, 2)

	currentState := s.GetLastState()
	currentState.DAHeight = 0

	// Verify - should pass since no forced txs to verify
	err = s.verifyForcedInclusionTxs(currentState, data)
	require.NoError(t, err)
}

func TestVerifyForcedInclusionTxs_NamespaceNotConfigured(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)

	cm, err := cache.NewCacheManager(config.DefaultConfig(), zerolog.Nop())
	require.NoError(t, err)

	addr, _, _ := buildSyncTestSigner(t)
	gen := genesis.Genesis{
		ChainID:         "tchain",
		InitialHeight:   1,
		StartTime:       time.Now().Add(-time.Second),
		ProposerAddress: addr,
	}

	cfg := config.DefaultConfig()
	// Leave ForcedInclusionNamespace empty

	mockExec := testmocks.NewMockExecutor(t)
	mockExec.EXPECT().InitChain(mock.Anything, mock.Anything, uint64(1), "tchain").
		Return([]byte("app0"), uint64(1024), nil).Once()

	mockDA := testmocks.NewMockDA(t)

	daClient := datestclient.New(datestclient.Config{
		DA:            mockDA,
		Namespace:     cfg.DA.Namespace,
		DataNamespace: cfg.DA.DataNamespace,
		// No ForcedInclusionNamespace - not configured
	})
	daRetriever := NewDARetriever(daClient, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	s := NewSyncer(
		st,
		mockExec,
		daClient,
		cm,
		common.NopMetrics(),
		cfg,
		gen,
		common.NewMockBroadcaster[*types.SignedHeader](t),
		common.NewMockBroadcaster[*types.Data](t),
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	s.daRetriever = daRetriever
	s.fiRetriever = fiRetriever

	require.NoError(t, s.initializeState())
	s.ctx = context.Background()

	// Create block data
	data := makeData(gen.ChainID, 1, 2)

	currentState := s.GetLastState()
	currentState.DAHeight = 0

	// Verify - should pass since namespace not configured
	err = s.verifyForcedInclusionTxs(currentState, data)
	require.NoError(t, err)
}

// TestVerifyForcedInclusionTxs_DeferralWithinEpoch tests that forced inclusion transactions
// can be legitimately deferred to a later block within the same epoch due to block size constraints
func TestVerifyForcedInclusionTxs_DeferralWithinEpoch(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)

	cm, err := cache.NewCacheManager(config.DefaultConfig(), zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{
		ChainID:                "tchain",
		InitialHeight:          1,
		StartTime:              time.Now().Add(-time.Second),
		ProposerAddress:        addr,
		DAStartHeight:          100,
		DAEpochForcedInclusion: 5, // Epoch spans 5 DA blocks
	}

	cfg := config.DefaultConfig()
	cfg.DA.ForcedInclusionNamespace = "nsForcedInclusion"

	mockExec := testmocks.NewMockExecutor(t)
	mockExec.EXPECT().InitChain(mock.Anything, mock.Anything, uint64(1), "tchain").
		Return([]byte("app0"), uint64(1024), nil).Once()

	mockDA := testmocks.NewMockDA(t)

	daClient := datestclient.New(datestclient.Config{
		DA:                       mockDA,
		Namespace:                cfg.DA.Namespace,
		DataNamespace:            cfg.DA.DataNamespace,
		ForcedInclusionNamespace: cfg.DA.ForcedInclusionNamespace,
	})
	daRetriever := NewDARetriever(daClient, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	s := NewSyncer(
		st,
		mockExec,
		daClient,
		cm,
		common.NopMetrics(),
		cfg,
		gen,
		common.NewMockBroadcaster[*types.SignedHeader](t),
		common.NewMockBroadcaster[*types.Data](t),
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	s.daRetriever = daRetriever
	s.fiRetriever = fiRetriever

	require.NoError(t, s.initializeState())
	s.ctx = context.Background()

	namespaceForcedInclusionBz := datypes.NamespaceFromString(cfg.DA.GetForcedInclusionNamespace()).Bytes()

	// Create forced inclusion transaction blobs
	dataBin1, _ := makeSignedDataBytes(t, gen.ChainID, 10, addr, pub, signer, 2)
	dataBin2, _ := makeSignedDataBytes(t, gen.ChainID, 11, addr, pub, signer, 1)

	// Mock DA retrieval for first block at DA height 104 (epoch end)
	// Epoch boundaries: [100, 104] (epoch size is 5)
	// The retriever will fetch all heights in the epoch: 100, 101, 102, 103, 104

	// Height 100 (epoch start)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(100), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&datypes.GetIDsResult{IDs: [][]byte{[]byte("fi1"), []byte("fi2")}, Timestamp: time.Now()}, nil).Once()

	mockDA.EXPECT().Get(mock.Anything, mock.Anything, mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return([][]byte{dataBin1, dataBin2}, nil).Once()

	// Heights 101, 102, 103 (intermediate heights in epoch)
	for height := uint64(101); height <= 103; height++ {
		mockDA.EXPECT().GetIDs(mock.Anything, height, mock.MatchedBy(func(ns []byte) bool {
			return bytes.Equal(ns, namespaceForcedInclusionBz)
		})).Return(&datypes.GetIDsResult{IDs: [][]byte{}, Timestamp: time.Now()}, nil).Once()
	}

	// Height 104 (epoch end)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(104), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&datypes.GetIDsResult{IDs: [][]byte{}, Timestamp: time.Now()}, nil).Once()

	// First block only includes dataBin1 (dataBin2 deferred due to size constraints)
	data1 := makeData(gen.ChainID, 1, 2)
	data1.Txs[0] = types.Tx(dataBin1)
	data1.Txs[1] = types.Tx([]byte("regular_tx_1"))

	currentState := s.GetLastState()
	currentState.DAHeight = 104

	// Verify - should pass since dataBin2 can be deferred within epoch
	err = s.verifyForcedInclusionTxs(currentState, data1)
	require.NoError(t, err)

	// Verify that dataBin2 is now tracked as pending
	pendingCount := 0
	s.pendingForcedInclusionTxs.Range(func(key, value any) bool {
		pendingCount++
		return true
	})
	require.Equal(t, 1, pendingCount, "should have 1 pending forced inclusion tx")

	// Mock DA for second verification at same epoch (height 104 - epoch end)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(100), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&datypes.GetIDsResult{IDs: [][]byte{[]byte("fi1"), []byte("fi2")}, Timestamp: time.Now()}, nil).Once()

	mockDA.EXPECT().Get(mock.Anything, mock.Anything, mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return([][]byte{dataBin1, dataBin2}, nil).Once()

	for height := uint64(101); height <= 103; height++ {
		mockDA.EXPECT().GetIDs(mock.Anything, height, mock.MatchedBy(func(ns []byte) bool {
			return bytes.Equal(ns, namespaceForcedInclusionBz)
		})).Return(&datypes.GetIDsResult{IDs: [][]byte{}, Timestamp: time.Now()}, nil).Once()
	}

	mockDA.EXPECT().GetIDs(mock.Anything, uint64(104), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&datypes.GetIDsResult{IDs: [][]byte{}, Timestamp: time.Now()}, nil).Once()

	// Second block includes BOTH the previously included dataBin1 AND the deferred dataBin2
	// This simulates the block containing both forced inclusion txs
	data2 := makeData(gen.ChainID, 2, 2)
	data2.Txs[0] = types.Tx(dataBin1) // Already included, but that's ok
	data2.Txs[1] = types.Tx(dataBin2) // The deferred one we're waiting for

	// Verify - should pass since dataBin2 is now included and clears pending
	err = s.verifyForcedInclusionTxs(currentState, data2)
	require.NoError(t, err)

	// Verify that pending queue is now empty (dataBin2 was included)
	pendingCount = 0
	s.pendingForcedInclusionTxs.Range(func(key, value any) bool {
		pendingCount++
		return true
	})
	require.Equal(t, 0, pendingCount, "should have no pending forced inclusion txs")
}

// TestVerifyForcedInclusionTxs_MaliciousAfterEpochEnd tests that missing forced inclusion
// transactions are detected as malicious when the epoch ends without them being included
func TestVerifyForcedInclusionTxs_MaliciousAfterEpochEnd(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)

	cm, err := cache.NewCacheManager(config.DefaultConfig(), zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{
		ChainID:                "tchain",
		InitialHeight:          1,
		StartTime:              time.Now().Add(-time.Second),
		ProposerAddress:        addr,
		DAStartHeight:          100,
		DAEpochForcedInclusion: 3, // Epoch spans 3 DA blocks
	}

	cfg := config.DefaultConfig()
	cfg.DA.ForcedInclusionNamespace = "nsForcedInclusion"

	mockExec := testmocks.NewMockExecutor(t)
	mockExec.EXPECT().InitChain(mock.Anything, mock.Anything, uint64(1), "tchain").
		Return([]byte("app0"), uint64(1024), nil).Once()

	mockDA := testmocks.NewMockDA(t)

	daClient := datestclient.New(datestclient.Config{
		DA:                       mockDA,
		Namespace:                cfg.DA.Namespace,
		DataNamespace:            cfg.DA.DataNamespace,
		ForcedInclusionNamespace: cfg.DA.ForcedInclusionNamespace,
	})
	daRetriever := NewDARetriever(daClient, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	s := NewSyncer(
		st,
		mockExec,
		daClient,
		cm,
		common.NopMetrics(),
		cfg,
		gen,
		common.NewMockBroadcaster[*types.SignedHeader](t),
		common.NewMockBroadcaster[*types.Data](t),
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	s.daRetriever = daRetriever
	s.fiRetriever = fiRetriever

	require.NoError(t, s.initializeState())
	s.ctx = context.Background()

	namespaceForcedInclusionBz := datypes.NamespaceFromString(cfg.DA.GetForcedInclusionNamespace()).Bytes()

	// Create forced inclusion transaction blob
	dataBin, _ := makeSignedDataBytes(t, gen.ChainID, 10, addr, pub, signer, 2)

	// Mock DA retrieval for DA height 100
	// Epoch boundaries: [100, 102] (epoch size is 3)
	// The retriever will fetch heights 100, 101, 102

	// Height 100 (epoch start)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(100), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&datypes.GetIDsResult{IDs: [][]byte{[]byte("fi1")}, Timestamp: time.Now()}, nil).Once()

	mockDA.EXPECT().Get(mock.Anything, mock.Anything, mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return([][]byte{dataBin}, nil).Once()

	// Height 101 (intermediate)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(101), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&datypes.GetIDsResult{IDs: [][]byte{}, Timestamp: time.Now()}, nil).Once()

	// Height 102 (epoch end)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(102), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&datypes.GetIDsResult{IDs: [][]byte{}, Timestamp: time.Now()}, nil).Once()

	// First block doesn't include the forced inclusion tx
	data1 := makeData(gen.ChainID, 1, 1)
	data1.Txs[0] = types.Tx([]byte("regular_tx_1"))

	currentState := s.GetLastState()
	currentState.DAHeight = 102

	// Verify - should pass, tx can be deferred within epoch
	err = s.verifyForcedInclusionTxs(currentState, data1)
	require.NoError(t, err)

	// Verify that the forced tx is tracked as pending
	pendingCount := 0
	s.pendingForcedInclusionTxs.Range(func(key, value any) bool {
		pendingCount++
		return true
	})
	require.Equal(t, 1, pendingCount, "should have 1 pending forced inclusion tx")

	// Process another block within same epoch - forced tx still not included
	// Mock DA for second verification at same epoch (height 102 - epoch end)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(100), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&datypes.GetIDsResult{IDs: [][]byte{[]byte("fi1")}, Timestamp: time.Now()}, nil).Once()

	mockDA.EXPECT().Get(mock.Anything, mock.Anything, mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return([][]byte{dataBin}, nil).Once()

	mockDA.EXPECT().GetIDs(mock.Anything, uint64(101), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&datypes.GetIDsResult{IDs: [][]byte{}, Timestamp: time.Now()}, nil).Once()

	mockDA.EXPECT().GetIDs(mock.Anything, uint64(102), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&datypes.GetIDsResult{IDs: [][]byte{}, Timestamp: time.Now()}, nil).Once()

	data2 := makeData(gen.ChainID, 2, 1)
	data2.Txs[0] = types.Tx([]byte("regular_tx_2"))

	// Still at epoch 100, should still pass
	err = s.verifyForcedInclusionTxs(currentState, data2)
	require.NoError(t, err)

	// Mock DA retrieval for next epoch (DA height 105 - epoch end)
	// Epoch boundaries: [103, 105]
	// The retriever will fetch heights 103, 104, 105

	// Height 103 (epoch start)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(103), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&datypes.GetIDsResult{IDs: [][]byte{}, Timestamp: time.Now()}, nil).Once()

	// Height 104 (intermediate)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(104), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&datypes.GetIDsResult{IDs: [][]byte{}, Timestamp: time.Now()}, nil).Once()

	// Height 105 (epoch end)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(105), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&datypes.GetIDsResult{IDs: [][]byte{}, Timestamp: time.Now()}, nil).Once()

	// Third block is in the next epoch (at epoch end 105) without including the forced tx
	data3 := makeData(gen.ChainID, 3, 1)
	data3.Txs[0] = types.Tx([]byte("regular_tx_3"))

	currentState.DAHeight = 105 // At epoch end [103, 105], past previous epoch [100, 102]

	// Verify - should FAIL since forced tx from previous epoch was never included
	err = s.verifyForcedInclusionTxs(currentState, data3)
	require.Error(t, err)
	require.Contains(t, err.Error(), "sequencer is malicious")
	require.Contains(t, err.Error(), "forced inclusion transactions from past epoch(s) not included")
}
