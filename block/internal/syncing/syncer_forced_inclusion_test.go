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
	"github.com/evstack/ev-node/block/internal/da"
	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/config"
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

	daClient := da.NewClient(da.Config{
		DA:                       mockDA,
		Logger:                   zerolog.Nop(),
		Namespace:                cfg.DA.Namespace,
		DataNamespace:            cfg.DA.DataNamespace,
		ForcedInclusionNamespace: cfg.DA.ForcedInclusionNamespace,
	})
	daRetriever := NewDARetriever(daClient, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	s := NewSyncer(
		st,
		mockExec,
		mockDA,
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
	namespaceForcedInclusionBz := coreda.NamespaceFromString(cfg.DA.GetForcedInclusionNamespace()).Bytes()

	// Create forced inclusion transaction blob (SignedData) in DA
	dataBin, _ := makeSignedDataBytes(t, gen.ChainID, 10, addr, pub, signer, 2)

	// With DAStartHeight=0, epoch size=1, daHeight=0 -> epoch boundaries are [0, 0]
	// Check epoch start only (end check is skipped when same as start)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(0), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&coreda.GetIDsResult{IDs: [][]byte{[]byte("fi1")}, Timestamp: time.Now()}, nil).Once()

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

	daClient := da.NewClient(da.Config{
		DA:                       mockDA,
		Logger:                   zerolog.Nop(),
		Namespace:                cfg.DA.Namespace,
		DataNamespace:            cfg.DA.DataNamespace,
		ForcedInclusionNamespace: cfg.DA.ForcedInclusionNamespace,
	})
	daRetriever := NewDARetriever(daClient, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	s := NewSyncer(
		st,
		mockExec,
		mockDA,
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
	namespaceForcedInclusionBz := coreda.NamespaceFromString(cfg.DA.GetForcedInclusionNamespace()).Bytes()

	// Create forced inclusion transaction blob (SignedData) in DA
	dataBin, _ := makeSignedDataBytes(t, gen.ChainID, 10, addr, pub, signer, 2)

	// With DAStartHeight=0, epoch size=1, daHeight=0 -> epoch boundaries are [0, 0]
	// Check epoch start only (end check is skipped when same as start)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(0), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&coreda.GetIDsResult{IDs: [][]byte{[]byte("fi1")}, Timestamp: time.Now()}, nil).Once()

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

	// Verify - should fail since forced tx blob is missing
	err = s.verifyForcedInclusionTxs(currentState, data)
	require.Error(t, err)
	require.Contains(t, err.Error(), "sequencer is malicious")
	require.Contains(t, err.Error(), "1 forced inclusion transactions not included")
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

	daClient := da.NewClient(da.Config{
		DA:                       mockDA,
		Logger:                   zerolog.Nop(),
		Namespace:                cfg.DA.Namespace,
		DataNamespace:            cfg.DA.DataNamespace,
		ForcedInclusionNamespace: cfg.DA.ForcedInclusionNamespace,
	})
	daRetriever := NewDARetriever(daClient, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	s := NewSyncer(
		st,
		mockExec,
		mockDA,
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
	namespaceForcedInclusionBz := coreda.NamespaceFromString(cfg.DA.GetForcedInclusionNamespace()).Bytes()

	// Create two forced inclusion transaction blobs in DA
	dataBin1, _ := makeSignedDataBytes(t, gen.ChainID, 10, addr, pub, signer, 2)
	dataBin2, _ := makeSignedDataBytes(t, gen.ChainID, 11, addr, pub, signer, 1)

	// With DAStartHeight=0, epoch size=1, daHeight=0 -> epoch boundaries are [0, 0]
	// Check epoch start only (end check is skipped when same as start)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(0), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&coreda.GetIDsResult{IDs: [][]byte{[]byte("fi1"), []byte("fi2")}, Timestamp: time.Now()}, nil).Once()

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

	// Verify - should fail since dataBin2 is missing
	err = s.verifyForcedInclusionTxs(currentState, data)
	require.Error(t, err)
	require.Contains(t, err.Error(), "sequencer is malicious")
	require.Contains(t, err.Error(), "1 forced inclusion transactions not included")
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

	daClient := da.NewClient(da.Config{
		DA:                       mockDA,
		Logger:                   zerolog.Nop(),
		Namespace:                cfg.DA.Namespace,
		DataNamespace:            cfg.DA.DataNamespace,
		ForcedInclusionNamespace: cfg.DA.ForcedInclusionNamespace,
	})
	daRetriever := NewDARetriever(daClient, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	s := NewSyncer(
		st,
		mockExec,
		mockDA,
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
	namespaceForcedInclusionBz := coreda.NamespaceFromString(cfg.DA.GetForcedInclusionNamespace()).Bytes()

	// With DAStartHeight=0, epoch size=1, daHeight=0 -> epoch boundaries are [0, 0]
	// Check epoch start only (end check is skipped when same as start)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(0), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&coreda.GetIDsResult{IDs: [][]byte{}, Timestamp: time.Now()}, nil).Once()

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

	daClient := da.NewClient(da.Config{
		DA:            mockDA,
		Logger:        zerolog.Nop(),
		Namespace:     cfg.DA.Namespace,
		DataNamespace: cfg.DA.DataNamespace,
		// No ForcedInclusionNamespace - not configured
	})
	daRetriever := NewDARetriever(daClient, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(daClient, gen, zerolog.Nop())

	s := NewSyncer(
		st,
		mockExec,
		mockDA,
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
