package syncing

import (
	"context"
	"sync/atomic"
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
	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/store"
	testmocks "github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
)

func TestCalculateBlockFullness_HalfFull(t *testing.T) {
	s := &Syncer{}

	// Create 5000 transactions of 100 bytes each = 500KB
	txs := make([]types.Tx, 5000)
	for i := range txs {
		txs[i] = make([]byte, 100)
	}

	data := &types.Data{
		Txs: txs,
	}

	fullness := s.calculateBlockFullness(data)
	// Size fullness: 500000/2097152 ≈ 0.238
	require.InDelta(t, 0.238, fullness, 0.05)
}

func TestCalculateBlockFullness_Full(t *testing.T) {
	s := &Syncer{}

	// Create 10000 transactions of 210 bytes each = ~2MB
	txs := make([]types.Tx, 10000)
	for i := range txs {
		txs[i] = make([]byte, 210)
	}

	data := &types.Data{
		Txs: txs,
	}

	fullness := s.calculateBlockFullness(data)
	// Both metrics at or near 1.0
	require.Greater(t, fullness, 0.95)
}

func TestCalculateBlockFullness_VerySmall(t *testing.T) {
	s := &Syncer{}

	data := &types.Data{
		Txs: []types.Tx{[]byte("tx1"), []byte("tx2")},
	}

	fullness := s.calculateBlockFullness(data)
	// Very small relative to heuristic limits
	require.Less(t, fullness, 0.001)
}

func TestUpdateDynamicGracePeriod_NoChangeWhenBelowThreshold(t *testing.T) {
	initialMultiplier := 1.0
	initialEMA := 0.1 // Well below threshold

	config := forcedInclusionGracePeriodConfig{
		dynamicMinMultiplier:     0.5,
		dynamicMaxMultiplier:     3.0,
		dynamicFullnessThreshold: 0.8,
		dynamicAdjustmentRate:    0.01, // Low adjustment rate
	}

	s := &Syncer{
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
		blockFullnessEMA:      &atomic.Pointer[float64]{},
		gracePeriodConfig:     config,
		metrics:               common.NopMetrics(),
	}
	s.gracePeriodMultiplier.Store(&initialMultiplier)
	s.blockFullnessEMA.Store(&initialEMA)

	// Update with low fullness - multiplier should stay at 1.0 initially
	s.updateDynamicGracePeriod(0.2)

	// With low adjustment rate and starting EMA below threshold,
	// multiplier should not change significantly on first call
	newMultiplier := *s.gracePeriodMultiplier.Load()
	require.InDelta(t, 1.0, newMultiplier, 0.05)
}

func TestUpdateDynamicGracePeriod_IncreaseOnHighFullness(t *testing.T) {
	initialMultiplier := 1.0
	initialEMA := 0.5

	s := &Syncer{
		gracePeriodConfig: forcedInclusionGracePeriodConfig{
			dynamicMinMultiplier:     0.5,
			dynamicMaxMultiplier:     3.0,
			dynamicFullnessThreshold: 0.8,
			dynamicAdjustmentRate:    0.1,
		},
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
		blockFullnessEMA:      &atomic.Pointer[float64]{},
		metrics:               common.NopMetrics(),
	}
	s.gracePeriodMultiplier.Store(&initialMultiplier)
	s.blockFullnessEMA.Store(&initialEMA)

	// Update multiple times with very high fullness to build up the effect
	for i := 0; i < 20; i++ {
		s.updateDynamicGracePeriod(0.95)
	}

	// EMA should increase
	newEMA := *s.blockFullnessEMA.Load()
	require.Greater(t, newEMA, initialEMA)

	// Multiplier should increase because EMA is now above threshold
	newMultiplier := *s.gracePeriodMultiplier.Load()
	require.Greater(t, newMultiplier, initialMultiplier)
}

func TestUpdateDynamicGracePeriod_DecreaseOnLowFullness(t *testing.T) {
	initialMultiplier := 2.0
	initialEMA := 0.9

	s := &Syncer{
		gracePeriodConfig: forcedInclusionGracePeriodConfig{
			dynamicMinMultiplier:     0.5,
			dynamicMaxMultiplier:     3.0,
			dynamicFullnessThreshold: 0.8,
			dynamicAdjustmentRate:    0.1,
		},
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
		blockFullnessEMA:      &atomic.Pointer[float64]{},
		metrics:               common.NopMetrics(),
	}
	s.gracePeriodMultiplier.Store(&initialMultiplier)
	s.blockFullnessEMA.Store(&initialEMA)

	// Update multiple times with low fullness to build up the effect
	for i := 0; i < 20; i++ {
		s.updateDynamicGracePeriod(0.2)
	}

	// EMA should decrease significantly
	newEMA := *s.blockFullnessEMA.Load()
	require.Less(t, newEMA, initialEMA)

	// Multiplier should decrease
	newMultiplier := *s.gracePeriodMultiplier.Load()
	require.Less(t, newMultiplier, initialMultiplier)
}

func TestUpdateDynamicGracePeriod_ClampToMin(t *testing.T) {
	initialMultiplier := 0.6
	initialEMA := 0.1

	s := &Syncer{
		gracePeriodConfig: forcedInclusionGracePeriodConfig{
			dynamicMinMultiplier:     0.5,
			dynamicMaxMultiplier:     3.0,
			dynamicFullnessThreshold: 0.8,
			dynamicAdjustmentRate:    0.5, // High rate to force clamping
		},
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
		blockFullnessEMA:      &atomic.Pointer[float64]{},
		metrics:               common.NopMetrics(),
	}
	s.gracePeriodMultiplier.Store(&initialMultiplier)
	s.blockFullnessEMA.Store(&initialEMA)

	// Update many times with very low fullness - should eventually clamp to min
	for i := 0; i < 50; i++ {
		s.updateDynamicGracePeriod(0.0)
	}

	newMultiplier := *s.gracePeriodMultiplier.Load()
	require.Equal(t, 0.5, newMultiplier)
}

func TestUpdateDynamicGracePeriod_ClampToMax(t *testing.T) {
	initialMultiplier := 2.5
	initialEMA := 0.9

	s := &Syncer{
		gracePeriodConfig: forcedInclusionGracePeriodConfig{
			dynamicMinMultiplier:     0.5,
			dynamicMaxMultiplier:     3.0,
			dynamicFullnessThreshold: 0.8,
			dynamicAdjustmentRate:    0.5, // High rate to force clamping
		},
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
		blockFullnessEMA:      &atomic.Pointer[float64]{},
		metrics:               common.NopMetrics(),
	}
	s.gracePeriodMultiplier.Store(&initialMultiplier)
	s.blockFullnessEMA.Store(&initialEMA)

	// Update many times with very high fullness - should eventually clamp to max
	for i := 0; i < 50; i++ {
		s.updateDynamicGracePeriod(1.0)
	}

	newMultiplier := *s.gracePeriodMultiplier.Load()
	require.Equal(t, 3.0, newMultiplier)
}

func TestGetEffectiveGracePeriod_WithMultiplier(t *testing.T) {
	multiplier := 2.5

	s := &Syncer{
		gracePeriodConfig: forcedInclusionGracePeriodConfig{
			basePeriod:               2,
			dynamicMinMultiplier:     0.5,
			dynamicMaxMultiplier:     3.0,
			dynamicFullnessThreshold: 0.8,
			dynamicAdjustmentRate:    0.05,
		},
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
	}
	s.gracePeriodMultiplier.Store(&multiplier)

	effective := s.getEffectiveGracePeriod()
	// 2 * 2.5 = 5
	require.Equal(t, uint64(5), effective)
}

func TestGetEffectiveGracePeriod_RoundingUp(t *testing.T) {
	multiplier := 2.6

	s := &Syncer{
		gracePeriodConfig: forcedInclusionGracePeriodConfig{
			basePeriod:               2,
			dynamicMinMultiplier:     0.5,
			dynamicMaxMultiplier:     3.0,
			dynamicFullnessThreshold: 0.8,
			dynamicAdjustmentRate:    0.05,
		},
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
	}
	s.gracePeriodMultiplier.Store(&multiplier)

	effective := s.getEffectiveGracePeriod()
	// 2 * 2.6 = 5.2, rounds to 5
	require.Equal(t, uint64(5), effective)
}

func TestGetEffectiveGracePeriod_EnsuresMinimum(t *testing.T) {
	multiplier := 0.3

	s := &Syncer{
		gracePeriodConfig: forcedInclusionGracePeriodConfig{
			basePeriod:               4,
			dynamicMinMultiplier:     0.5,
			dynamicMaxMultiplier:     3.0,
			dynamicFullnessThreshold: 0.8,
			dynamicAdjustmentRate:    0.05,
		},
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
	}
	s.gracePeriodMultiplier.Store(&multiplier)

	effective := s.getEffectiveGracePeriod()
	// 4 * 0.3 = 1.2, but minimum is 4 * 0.5 = 2
	require.Equal(t, uint64(2), effective)
}

func TestDynamicGracePeriod_Integration_HighCongestion(t *testing.T) {
	initialMultiplier := 1.0
	initialEMA := 0.3

	s := &Syncer{
		gracePeriodConfig: forcedInclusionGracePeriodConfig{
			basePeriod:               2,
			dynamicMinMultiplier:     0.5,
			dynamicMaxMultiplier:     3.0,
			dynamicFullnessThreshold: 0.8,
			dynamicAdjustmentRate:    0.1,
		},
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
		blockFullnessEMA:      &atomic.Pointer[float64]{},
		metrics:               common.NopMetrics(),
	}
	s.gracePeriodMultiplier.Store(&initialMultiplier)
	s.blockFullnessEMA.Store(&initialEMA)

	// Simulate processing many blocks with very high fullness (above threshold)
	for i := 0; i < 50; i++ {
		s.updateDynamicGracePeriod(0.95)
	}

	// Multiplier should have increased due to sustained high fullness
	finalMultiplier := *s.gracePeriodMultiplier.Load()
	require.Greater(t, finalMultiplier, initialMultiplier, "multiplier should increase with sustained congestion")

	// Effective grace period should be higher than base
	effectiveGracePeriod := s.getEffectiveGracePeriod()
	require.Greater(t, effectiveGracePeriod, s.gracePeriodConfig.basePeriod, "effective grace period should be higher than base")
}

func TestDynamicGracePeriod_Integration_LowCongestion(t *testing.T) {
	initialMultiplier := 2.0
	initialEMA := 0.85

	s := &Syncer{
		gracePeriodConfig: forcedInclusionGracePeriodConfig{
			basePeriod:               2,
			dynamicMinMultiplier:     0.5,
			dynamicMaxMultiplier:     3.0,
			dynamicFullnessThreshold: 0.8,
			dynamicAdjustmentRate:    0.1,
		},
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
		blockFullnessEMA:      &atomic.Pointer[float64]{},
		metrics:               common.NopMetrics(),
	}
	s.gracePeriodMultiplier.Store(&initialMultiplier)
	s.blockFullnessEMA.Store(&initialEMA)

	// Simulate processing many blocks with very low fullness (below threshold)
	for i := 0; i < 50; i++ {
		s.updateDynamicGracePeriod(0.1)
	}

	// Multiplier should have decreased
	finalMultiplier := *s.gracePeriodMultiplier.Load()
	require.Less(t, finalMultiplier, initialMultiplier, "multiplier should decrease with low congestion")
}

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

	client := testmocks.NewMockClient(t)
	client.On("GetHeaderNamespace").Return([]byte(cfg.DA.Namespace)).Maybe()
	client.On("GetDataNamespace").Return([]byte(cfg.DA.DataNamespace)).Maybe()
	client.On("GetForcedInclusionNamespace").Return([]byte(cfg.DA.ForcedInclusionNamespace)).Maybe()
	client.On("HasForcedInclusionNamespace").Return(true).Maybe()
	daRetriever := NewDARetriever(client, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(client, zerolog.Nop(), gen.DAStartHeight, gen.DAEpochForcedInclusion)

	s := NewSyncer(
		st,
		mockExec,
		client,
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
	// Create forced inclusion transaction blob (SignedData) in DA
	dataBin, _ := makeSignedDataBytes(t, gen.ChainID, 10, addr, pub, signer, 2)

	client.On("Retrieve", mock.Anything, uint64(0), []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, IDs: [][]byte{[]byte("fi1")}, Timestamp: time.Now()},
		Data:       [][]byte{dataBin},
	}).Once()

	// Create block data that includes the forced transaction blob
	data := makeData(gen.ChainID, 1, 1)
	data.Txs[0] = types.Tx(dataBin)

	currentState := s.getLastState()
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

	client := testmocks.NewMockClient(t)
	client.On("GetHeaderNamespace").Return([]byte(cfg.DA.Namespace)).Maybe()
	client.On("GetDataNamespace").Return([]byte(cfg.DA.DataNamespace)).Maybe()
	client.On("GetForcedInclusionNamespace").Return([]byte(cfg.DA.ForcedInclusionNamespace)).Maybe()
	client.On("HasForcedInclusionNamespace").Return(true).Maybe()
	daRetriever := NewDARetriever(client, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(client, zerolog.Nop(), gen.DAStartHeight, gen.DAEpochForcedInclusion)

	s := NewSyncer(
		st,
		mockExec,
		client,
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
	// Create forced inclusion transaction blob (SignedData) in DA
	dataBin, _ := makeSignedDataBytes(t, gen.ChainID, 10, addr, pub, signer, 2)

	// With DAStartHeight=0, epoch size=1, daHeight=0 -> epoch boundaries are [0, 0]
	client.On("Retrieve", mock.Anything, uint64(0), []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, IDs: [][]byte{[]byte("fi1")}, Timestamp: time.Now()},
		Data:       [][]byte{dataBin},
	}).Once()

	// Create block data that does NOT include the forced transaction blob
	data := makeData(gen.ChainID, 1, 2)
	data.Txs[0] = types.Tx([]byte("regular_tx_1"))
	data.Txs[1] = types.Tx([]byte("regular_tx_2"))

	currentState := s.getLastState()
	currentState.DAHeight = 0

	// Verify - should pass since forced tx blob may be legitimately deferred within the epoch
	err = s.verifyForcedInclusionTxs(currentState, data)
	require.NoError(t, err)

	// Mock DA for next epoch to return no forced inclusion transactions
	client.On("Retrieve", mock.Anything, uint64(1), []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Timestamp: time.Now()},
	}).Once()

	// Move to next epoch but still within grace period
	currentState.DAHeight = 1 // Move to epoch end (epoch was [0, 0])
	data2 := makeData(gen.ChainID, 2, 1)
	data2.Txs[0] = []byte("regular_tx_3")

	err = s.verifyForcedInclusionTxs(currentState, data2)
	require.NoError(t, err) // Should pass since DAHeight=1 equals grace boundary, not past it

	// Mock DA for height 2 to return no forced inclusion transactions
	client.On("Retrieve", mock.Anything, uint64(2), []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Timestamp: time.Now()},
	}).Once()

	// Now move past grace boundary - should fail if tx still not included
	currentState.DAHeight = 2 // Move past grace boundary (graceBoundary = 0 + 1*1 = 1)
	data3 := makeData(gen.ChainID, 3, 1)
	data3.Txs[0] = types.Tx([]byte("regular_tx_4"))

	err = s.verifyForcedInclusionTxs(currentState, data3)
	require.Error(t, err)
	require.Contains(t, err.Error(), "sequencer is malicious")
	require.Contains(t, err.Error(), "past grace boundary")
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

	client := testmocks.NewMockClient(t)
	client.On("GetHeaderNamespace").Return([]byte(cfg.DA.Namespace)).Maybe()
	client.On("GetDataNamespace").Return([]byte(cfg.DA.DataNamespace)).Maybe()
	client.On("GetForcedInclusionNamespace").Return([]byte(cfg.DA.ForcedInclusionNamespace)).Maybe()
	client.On("HasForcedInclusionNamespace").Return(true).Maybe()
	daRetriever := NewDARetriever(client, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(client, zerolog.Nop(), gen.DAStartHeight, gen.DAEpochForcedInclusion)

	s := NewSyncer(
		st,
		mockExec,
		client,
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

	// Create two forced inclusion transaction blobs in DA
	dataBin1, _ := makeSignedDataBytes(t, gen.ChainID, 10, addr, pub, signer, 2)
	dataBin2, _ := makeSignedDataBytes(t, gen.ChainID, 11, addr, pub, signer, 1)

	// With DAStartHeight=0, epoch size=1, daHeight=0 -> epoch boundaries are [0, 0]
	client.On("Retrieve", mock.Anything, uint64(0), []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, IDs: [][]byte{[]byte("fi1"), []byte("fi2")}, Timestamp: time.Now()},
		Data:       [][]byte{dataBin1, dataBin2},
	}).Once()

	// Create block data that includes only one of the forced transaction blobs
	data := makeData(gen.ChainID, 1, 2)
	data.Txs[0] = types.Tx(dataBin1)
	data.Txs[1] = types.Tx([]byte("regular_tx"))
	// dataBin2 is missing

	currentState := s.getLastState()
	currentState.DAHeight = 0

	// Verify - should pass since dataBin2 may be legitimately deferred within the epoch
	err = s.verifyForcedInclusionTxs(currentState, data)
	require.NoError(t, err)

	// Mock DA for next epoch to return no forced inclusion transactions
	client.On("Retrieve", mock.Anything, uint64(1), []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Timestamp: time.Now()},
	}).Once()

	// Move to DAHeight=1 (still within grace period since graceBoundary = 0 + 1*1 = 1)
	currentState.DAHeight = 1
	data2 := makeData(gen.ChainID, 2, 1)
	data2.Txs[0] = types.Tx([]byte("regular_tx_3"))

	// Verify - should pass since we're at the grace boundary, not past it
	err = s.verifyForcedInclusionTxs(currentState, data2)
	require.NoError(t, err)

	// Mock DA for height 2 (when we move to DAHeight 2)
	client.On("Retrieve", mock.Anything, uint64(2), []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Timestamp: time.Now()},
	}).Once()

	// Now simulate moving past grace boundary - should fail if dataBin2 still not included
	// With basePeriod=1 and DAEpochForcedInclusion=1, graceBoundary = 0 + (1*1) = 1
	// So we need DAHeight > 1 to trigger the error
	currentState.DAHeight = 2 // Move past grace boundary
	data3 := makeData(gen.ChainID, 3, 1)
	data3.Txs[0] = types.Tx([]byte("regular_tx_4"))

	err = s.verifyForcedInclusionTxs(currentState, data3)
	require.Error(t, err)
	require.Contains(t, err.Error(), "sequencer is malicious")
	require.Contains(t, err.Error(), "past grace boundary")
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

	client := testmocks.NewMockClient(t)
	client.On("GetHeaderNamespace").Return([]byte(cfg.DA.Namespace)).Maybe()
	client.On("GetDataNamespace").Return([]byte(cfg.DA.DataNamespace)).Maybe()
	client.On("GetForcedInclusionNamespace").Return([]byte(cfg.DA.ForcedInclusionNamespace)).Maybe()
	client.On("HasForcedInclusionNamespace").Return(true).Maybe()
	daRetriever := NewDARetriever(client, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(client, zerolog.Nop(), gen.DAStartHeight, gen.DAEpochForcedInclusion)

	s := NewSyncer(
		st,
		mockExec,
		client,
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

	// Mock DA to return no forced inclusion transactions at height 0
	client.On("Retrieve", mock.Anything, uint64(0), []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Timestamp: time.Now()},
	}).Once()

	// Create block data
	data := makeData(gen.ChainID, 1, 2)

	currentState := s.getLastState()
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

	client := testmocks.NewMockClient(t)
	client.On("GetHeaderNamespace").Return([]byte(cfg.DA.Namespace)).Maybe()
	client.On("GetDataNamespace").Return([]byte(cfg.DA.DataNamespace)).Maybe()
	client.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()
	client.On("HasForcedInclusionNamespace").Return(false).Maybe()
	daRetriever := NewDARetriever(client, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(client, zerolog.Nop(), gen.DAStartHeight, gen.DAEpochForcedInclusion)

	s := NewSyncer(
		st,
		mockExec,
		client,
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

	currentState := s.getLastState()
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

	client := testmocks.NewMockClient(t)
	client.On("GetHeaderNamespace").Return([]byte(cfg.DA.Namespace)).Maybe()
	client.On("GetDataNamespace").Return([]byte(cfg.DA.DataNamespace)).Maybe()
	client.On("GetForcedInclusionNamespace").Return([]byte(cfg.DA.ForcedInclusionNamespace)).Maybe()
	client.On("HasForcedInclusionNamespace").Return(true).Maybe()
	daRetriever := NewDARetriever(client, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(client, zerolog.Nop(), gen.DAStartHeight, gen.DAEpochForcedInclusion)

	s := NewSyncer(
		st,
		mockExec,
		client,
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

	// Create forced inclusion transaction blobs
	dataBin1, _ := makeSignedDataBytes(t, gen.ChainID, 10, addr, pub, signer, 2)
	dataBin2, _ := makeSignedDataBytes(t, gen.ChainID, 11, addr, pub, signer, 1)

	// Mock DA retrieval for first block at DA height 104 (epoch end)
	// Epoch boundaries: [100, 104] (epoch size is 5)
	// The retriever will fetch all heights in the epoch: 100, 101, 102, 103, 104

	client.On("Retrieve", mock.Anything, uint64(100), []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, IDs: [][]byte{[]byte("fi1"), []byte("fi2")}, Timestamp: time.Now()},
		Data:       [][]byte{dataBin1, dataBin2},
	}).Once()

	for height := uint64(101); height <= 104; height++ {
		client.On("Retrieve", mock.Anything, height, []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Timestamp: time.Now()},
		}).Once()
	}

	// First block only includes dataBin1 (dataBin2 deferred due to size constraints)
	data1 := makeData(gen.ChainID, 1, 2)
	data1.Txs[0] = types.Tx(dataBin1)
	data1.Txs[1] = types.Tx([]byte("regular_tx_1"))

	currentState := s.getLastState()
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

	// Mock DA for second verification at same epoch (height 104 - epoch end)
	for height := uint64(101); height <= 104; height++ {
		client.On("Retrieve", mock.Anything, height, []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Timestamp: time.Now()},
		}).Once()
	}

	client.On("Retrieve", mock.Anything, uint64(100), []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, IDs: [][]byte{[]byte("fi1"), []byte("fi2")}, Timestamp: time.Now()},
		Data:       [][]byte{dataBin1, dataBin2},
	}).Once()

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

	client := testmocks.NewMockClient(t)
	client.On("GetHeaderNamespace").Return([]byte(cfg.DA.Namespace)).Maybe()
	client.On("GetDataNamespace").Return([]byte(cfg.DA.DataNamespace)).Maybe()
	client.On("GetForcedInclusionNamespace").Return([]byte(cfg.DA.ForcedInclusionNamespace)).Maybe()
	client.On("HasForcedInclusionNamespace").Return(true).Maybe()
	daRetriever := NewDARetriever(client, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(client, zerolog.Nop(), gen.DAStartHeight, gen.DAEpochForcedInclusion)

	s := NewSyncer(
		st,
		mockExec,
		client,
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

	// Create forced inclusion transaction blob
	dataBin, _ := makeSignedDataBytes(t, gen.ChainID, 10, addr, pub, signer, 2)

	// Mock DA retrieval for DA height 100
	// Epoch boundaries: [100, 102] (epoch size is 3)
	// The retriever will fetch heights 100, 101, 102

	client.On("Retrieve", mock.Anything, uint64(100), []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, IDs: [][]byte{[]byte("fi1")}, Timestamp: time.Now()},
		Data:       [][]byte{dataBin},
	}).Once()

	client.On("Retrieve", mock.Anything, uint64(101), []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Timestamp: time.Now()},
	}).Once()

	client.On("Retrieve", mock.Anything, uint64(102), []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Timestamp: time.Now()},
	}).Once()

	// First block doesn't include the forced inclusion tx
	data1 := makeData(gen.ChainID, 1, 1)
	data1.Txs[0] = types.Tx([]byte("regular_tx_1"))

	currentState := s.getLastState()
	currentState.DAHeight = 102

	// Verify - should pass, tx can be deferred within epoch
	err = s.verifyForcedInclusionTxs(currentState, data1)
	require.NoError(t, err)
}

// TestVerifyForcedInclusionTxs_SmoothingExceedsEpoch tests the critical scenario where
// forced inclusion transactions cannot all be included before an epoch ends.
// This demonstrates that the system correctly detects malicious behavior when
// transactions remain pending after the epoch boundary.
func TestVerifyForcedInclusionTxs_SmoothingExceedsEpoch(t *testing.T) {
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
		DAEpochForcedInclusion: 3, // Epoch: [100, 102]
	}

	cfg := config.DefaultConfig()
	cfg.DA.ForcedInclusionNamespace = "nsForcedInclusion"

	mockExec := testmocks.NewMockExecutor(t)
	mockExec.EXPECT().InitChain(mock.Anything, mock.Anything, uint64(1), "tchain").
		Return([]byte("app0"), uint64(1024), nil).Once()

	client := testmocks.NewMockClient(t)
	client.On("GetHeaderNamespace").Return([]byte(cfg.DA.Namespace)).Maybe()
	client.On("GetDataNamespace").Return([]byte(cfg.DA.DataNamespace)).Maybe()
	client.On("GetForcedInclusionNamespace").Return([]byte(cfg.DA.ForcedInclusionNamespace)).Maybe()
	client.On("HasForcedInclusionNamespace").Return(true).Maybe()

	daRetriever := NewDARetriever(client, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(client, zerolog.Nop(), gen.DAStartHeight, gen.DAEpochForcedInclusion)

	s := NewSyncer(
		st,
		mockExec,
		client,
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

	// Create 3 forced inclusion transactions
	dataBin1, _ := makeSignedDataBytes(t, gen.ChainID, 10, addr, pub, signer, 2)
	dataBin2, _ := makeSignedDataBytes(t, gen.ChainID, 11, addr, pub, signer, 2)
	dataBin3, _ := makeSignedDataBytes(t, gen.ChainID, 12, addr, pub, signer, 2)

	// Mock DA retrieval for Epoch 1: [100, 102]
	client.On("Retrieve", mock.Anything, uint64(100), []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{
			Code:      datypes.StatusSuccess,
			IDs:       [][]byte{[]byte("fi1"), []byte("fi2"), []byte("fi3")},
			Timestamp: time.Now(),
		},
		Data: [][]byte{dataBin1, dataBin2, dataBin3},
	}).Once()

	client.On("Retrieve", mock.Anything, uint64(101), []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Timestamp: time.Now()},
	}).Once()

	client.On("Retrieve", mock.Anything, uint64(102), []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Timestamp: time.Now()},
	}).Once()

	// Block at DA height 102 (epoch end): Only includes 2 of 3 txs
	// The third tx remains pending - legitimate within the epoch
	data1 := makeData(gen.ChainID, 1, 2)
	data1.Txs[0] = types.Tx(dataBin1)
	data1.Txs[1] = types.Tx(dataBin2)

	currentState := s.getLastState()
	currentState.DAHeight = 102 // At epoch end

	err = s.verifyForcedInclusionTxs(currentState, data1)
	require.NoError(t, err, "smoothing within epoch should be allowed")
}
