package syncing

import (
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
	"github.com/evstack/ev-node/core/execution"
	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/store"
	testmocks "github.com/evstack/ev-node/test/mocks"
	extmocks "github.com/evstack/ev-node/test/mocks/external"
	"github.com/evstack/ev-node/types"
)

// setupExecMocks stubs GetExecutionInfo (always succeeds) and FilterTxs (accepts all txs).
func setupExecMocks(mockExec *testmocks.MockExecutor) {
	mockExec.On("GetExecutionInfo", mock.Anything).Return(execution.ExecutionInfo{MaxGas: 1000000}, nil).Maybe()
	mockExec.On("FilterTxs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		func(_ context.Context, txs [][]byte, _, _ uint64, _ bool) []execution.FilterStatus {
			result := make([]execution.FilterStatus, len(txs))
			for i := range result {
				result[i] = execution.FilterOK
			}
			return result
		},
		nil,
	).Maybe()
}

// newForcedInclusionSyncer builds a minimal Syncer for forced-inclusion tests.
// FilterTxs is intentionally not pre-registered; call setupExecMocks(mockExec)
// or register a custom handler before exercising the epoch-check path.
func newForcedInclusionSyncer(t *testing.T, daStart, epochSize uint64) (*Syncer, *testmocks.MockClient, *testmocks.MockExecutor) {
	t.Helper()

	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)

	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, _, _ := buildSyncTestSigner(t)
	gen := genesis.Genesis{
		ChainID:                "tchain",
		InitialHeight:          1,
		StartTime:              time.Now().Add(-time.Second),
		ProposerAddress:        addr,
		DAStartHeight:          daStart,
		DAEpochForcedInclusion: epochSize,
	}

	cfg := config.DefaultConfig()
	cfg.DA.ForcedInclusionNamespace = "nsForcedInclusion"

	mockExec := testmocks.NewMockExecutor(t)
	mockExec.EXPECT().InitChain(mock.Anything, mock.Anything, uint64(1), "tchain").
		Return([]byte("app0"), nil).Once()
	mockExec.On("GetExecutionInfo", mock.Anything).Return(execution.ExecutionInfo{MaxGas: 1000000}, nil).Maybe()

	client := testmocks.NewMockClient(t)
	client.On("GetHeaderNamespace").Return([]byte(cfg.DA.Namespace)).Maybe()
	client.On("GetDataNamespace").Return([]byte(cfg.DA.DataNamespace)).Maybe()
	client.On("GetForcedInclusionNamespace").Return([]byte(cfg.DA.ForcedInclusionNamespace)).Maybe()
	client.On("HasForcedInclusionNamespace").Return(true).Maybe()

	daRetriever := NewDARetriever(client, cm, gen, zerolog.Nop())
	fiRetriever := da.NewForcedInclusionRetriever(client, zerolog.Nop(), cfg, gen.DAStartHeight, gen.DAEpochForcedInclusion)
	t.Cleanup(fiRetriever.Stop)

	s := NewSyncer(
		st, mockExec, client, cm, common.NopMetrics(), cfg, gen,
		extmocks.NewMockStore[*types.P2PSignedHeader](t),
		extmocks.NewMockStore[*types.P2PData](t),
		zerolog.Nop(), common.DefaultBlockOptions(), make(chan error, 1), nil,
	)
	s.daRetriever = daRetriever
	s.fiRetriever = fiRetriever

	require.NoError(t, s.initializeState())
	s.ctx = t.Context()

	return s, client, mockExec
}

// mockFIEmpty stubs the FI namespace at every height in [start, end] as not found.
// Maybe() covers calls from both the synchronous path and the async prefetcher.
func mockFIEmpty(client *testmocks.MockClient, start, end uint64) {
	for h := start; h <= end; h++ {
		client.On("Retrieve", mock.Anything, h, []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Timestamp: time.Now()},
		}).Maybe()
	}
}

// mockFIWithTxs stubs the FI namespace: epochStart returns blobs, remaining heights not found.
func mockFIWithTxs(client *testmocks.MockClient, epochStart, epochEnd uint64, blobs [][]byte) {
	client.On("Retrieve", mock.Anything, epochStart, []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: time.Now()},
		Data:       blobs,
	}).Maybe()
	for h := epochStart + 1; h <= epochEnd; h++ {
		client.On("Retrieve", mock.Anything, h, []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Timestamp: time.Now()},
		}).Maybe()
	}
}

// TestVerifyForcedInclusionTxs_NamespaceNotConfigured verifies that without a
// configured FI namespace the function returns nil immediately.
func TestVerifyForcedInclusionTxs_NamespaceNotConfigured(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, _, _ := buildSyncTestSigner(t)
	gen := genesis.Genesis{
		ChainID:         "tchain",
		InitialHeight:   1,
		StartTime:       time.Now().Add(-time.Second),
		ProposerAddress: addr,
	}

	cfg := config.DefaultConfig() // intentionally no ForcedInclusionNamespace

	mockExec := testmocks.NewMockExecutor(t)
	mockExec.EXPECT().InitChain(mock.Anything, mock.Anything, uint64(1), "tchain").
		Return([]byte("app0"), nil).Once()
	setupExecMocks(mockExec)

	client := testmocks.NewMockClient(t)
	client.On("GetHeaderNamespace").Return([]byte(cfg.DA.Namespace)).Maybe()
	client.On("GetDataNamespace").Return([]byte(cfg.DA.DataNamespace)).Maybe()
	client.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()
	client.On("HasForcedInclusionNamespace").Return(false).Maybe()

	fiRetriever := da.NewForcedInclusionRetriever(client, zerolog.Nop(), cfg, gen.DAStartHeight, gen.DAEpochForcedInclusion)
	t.Cleanup(fiRetriever.Stop)

	s := NewSyncer(
		st, mockExec, client, cm, common.NopMetrics(), cfg, gen,
		extmocks.NewMockStore[*types.P2PSignedHeader](t),
		extmocks.NewMockStore[*types.P2PData](t),
		zerolog.Nop(), common.DefaultBlockOptions(), make(chan error, 1), nil,
	)
	s.fiRetriever = fiRetriever
	require.NoError(t, s.initializeState())
	s.ctx = t.Context()

	data := makeData(gen.ChainID, 1, 1)
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 5, data))
}

// TestVerifyForcedInclusionTxs_NoForcedTxs verifies that epochs with no forced
// txs in DA produce no error even when their grace boundaries are crossed.
//
// daStart=10, epochSize=2, daHeight=16: epochs [10,11] and [12,13] are checked,
// [14,15] is not (graceBoundary 17 >= 16).
func TestVerifyForcedInclusionTxs_NoForcedTxs(t *testing.T) {
	s, client, mockExec := newForcedInclusionSyncer(t, 10, 2)
	setupExecMocks(mockExec)

	mockFIEmpty(client, 10, 11) // epoch 1
	mockFIEmpty(client, 12, 13) // epoch 2

	data := makeData("tchain", 1, 1)
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 16, data))
}

// TestVerifyForcedInclusionTxs_AllIncludedBeforeGraceBoundary verifies the happy
// path: F1 is included before epoch1's grace boundary (13) is crossed.
//
// daStart=10, epochSize=2: epoch1=[10,11], graceBoundary=13.
// F1 is recorded at daHeight=12; epoch1 is checked at daHeight=14.
func TestVerifyForcedInclusionTxs_AllIncludedBeforeGraceBoundary(t *testing.T) {
	s, client, mockExec := newForcedInclusionSyncer(t, 10, 2)
	setupExecMocks(mockExec)

	addr, pub, signer := buildSyncTestSigner(t)
	forcedTx, _ := makeSignedDataBytes(t, "tchain", 10, addr, pub, signer, 1)

	mockFIWithTxs(client, 10, 11, [][]byte{forcedTx})

	// daHeight=12: graceBoundary(13) >= 12 → epoch1 not yet checked; F1 recorded.
	data1 := makeData("tchain", 1, 1)
	data1.Txs[0] = types.Tx(forcedTx)
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 12, data1))

	// daHeight=14: graceBoundary(13) < 14 → epoch1 checked; F1 in seenBlockTxs → OK.
	data2 := makeData("tchain", 2, 1)
	data2.Txs[0] = types.Tx([]byte("regular"))
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 14, data2))
}

// TestVerifyForcedInclusionTxs_SmoothingWithinGracePeriod verifies that a forced
// tx included after its epoch ends but before the grace boundary is not flagged.
//
// daStart=0, epochSize=3: epoch1=[0,2], graceBoundary=5.
// F1 is included at daHeight=4 (inside grace window); epoch1 checked at daHeight=6.
func TestVerifyForcedInclusionTxs_SmoothingWithinGracePeriod(t *testing.T) {
	s, client, mockExec := newForcedInclusionSyncer(t, 0, 3)
	setupExecMocks(mockExec)

	addr, pub, signer := buildSyncTestSigner(t)
	forcedTx, _ := makeSignedDataBytes(t, "tchain", 10, addr, pub, signer, 1)

	// daHeight=4: graceBoundary(5) >= 4 → not yet checked; F1 recorded.
	data1 := makeData("tchain", 1, 1)
	data1.Txs[0] = types.Tx(forcedTx)
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 4, data1))

	// daHeight=6: graceBoundary(5) < 6 → epoch1 checked; F1 in seenBlockTxs → OK.
	mockFIWithTxs(client, 0, 2, [][]byte{forcedTx})
	data2 := makeData("tchain", 2, 1)
	data2.Txs[0] = types.Tx([]byte("regular"))
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 6, data2))
}

// TestVerifyForcedInclusionTxs_MaliciousNeverIncluded verifies that a forced tx
// never included before its grace boundary triggers errMaliciousProposer.
//
// daStart=0, epochSize=1: epoch1=[0,0], graceBoundary=1. Checked at daHeight=2.
func TestVerifyForcedInclusionTxs_MaliciousNeverIncluded(t *testing.T) {
	s, client, mockExec := newForcedInclusionSyncer(t, 0, 1)
	setupExecMocks(mockExec)

	addr, pub, signer := buildSyncTestSigner(t)
	forcedTx, _ := makeSignedDataBytes(t, "tchain", 10, addr, pub, signer, 1)

	mockFIWithTxs(client, 0, 0, [][]byte{forcedTx})

	// daHeight=2: past graceBoundary(1); F1 not included → malicious.
	data := makeData("tchain", 1, 1)
	data.Txs[0] = types.Tx([]byte("regular_only"))

	err := s.VerifyForcedInclusionTxs(t.Context(), 2, data)
	require.Error(t, err)
	require.ErrorIs(t, err, errMaliciousProposer)
	require.Contains(t, err.Error(), "sequencer is malicious")
}

// TestVerifyForcedInclusionTxs_MaliciousPartialInclusion verifies that when only
// some forced txs are included before the grace boundary, the missing ones are flagged.
//
// daStart=0, epochSize=1: epoch1=[0,0], graceBoundary=1. F1 included, F2 not.
func TestVerifyForcedInclusionTxs_MaliciousPartialInclusion(t *testing.T) {
	s, client, mockExec := newForcedInclusionSyncer(t, 0, 1)
	setupExecMocks(mockExec)

	addr, pub, signer := buildSyncTestSigner(t)
	forcedTx1, _ := makeSignedDataBytes(t, "tchain", 10, addr, pub, signer, 1)
	forcedTx2, _ := makeSignedDataBytes(t, "tchain", 11, addr, pub, signer, 1)

	mockFIWithTxs(client, 0, 0, [][]byte{forcedTx1, forcedTx2})

	data1 := makeData("tchain", 1, 1)
	data1.Txs[0] = types.Tx(forcedTx1)
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 1, data1))

	// daHeight=2: past graceBoundary(1); F2 missing → malicious.
	data2 := makeData("tchain", 2, 1)
	data2.Txs[0] = types.Tx([]byte("regular"))
	err := s.VerifyForcedInclusionTxs(t.Context(), 2, data2)
	require.Error(t, err)
	require.ErrorIs(t, err, errMaliciousProposer)
}

// TestVerifyForcedInclusionTxs_SmoothingAcrossMultipleBlocks verifies that forced
// txs spread across several blocks within the grace window are all credited.
//
// daStart=0, epochSize=5: epoch1=[0,4], graceBoundary=9.
// F1/F2/F3 each land in separate blocks at daHeights 5, 6, 7; checked at daHeight=10.
func TestVerifyForcedInclusionTxs_SmoothingAcrossMultipleBlocks(t *testing.T) {
	s, client, mockExec := newForcedInclusionSyncer(t, 0, 5)
	setupExecMocks(mockExec)

	addr, pub, signer := buildSyncTestSigner(t)
	f1, _ := makeSignedDataBytes(t, "tchain", 10, addr, pub, signer, 1)
	f2, _ := makeSignedDataBytes(t, "tchain", 11, addr, pub, signer, 1)
	f3, _ := makeSignedDataBytes(t, "tchain", 12, addr, pub, signer, 1)

	// daHeights 5, 6, 7: graceBoundary(9) not yet crossed; one forced tx each.
	for i, forcedTx := range [][]byte{f1, f2, f3} {
		d := makeData("tchain", uint64(i+1), 1)
		d.Txs[0] = types.Tx(forcedTx)
		require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), uint64(5+i), d))
	}

	// daHeight=10: graceBoundary(9) < 10 → epoch1 checked; all three in seenBlockTxs → OK.
	mockFIWithTxs(client, 0, 4, [][]byte{f1, f2, f3})
	d := makeData("tchain", 4, 1)
	d.Txs[0] = types.Tx([]byte("regular"))
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 10, d))
}

// TestVerifyForcedInclusionTxs_WithinGracePeriodNoError verifies that a missing
// forced tx is not flagged while the grace window is still open.
//
// daStart=0, epochSize=2: epoch1=[0,1], graceBoundary=3.
// daHeight=3 → boundary not crossed (>=), no error. daHeight=4 → crossed, malicious.
func TestVerifyForcedInclusionTxs_WithinGracePeriodNoError(t *testing.T) {
	s, client, mockExec := newForcedInclusionSyncer(t, 0, 2)
	setupExecMocks(mockExec)

	addr, pub, signer := buildSyncTestSigner(t)
	forcedTx, _ := makeSignedDataBytes(t, "tchain", 10, addr, pub, signer, 1)

	// daHeight=3: graceBoundary(3) >= 3 → not checked → OK.
	data := makeData("tchain", 1, 1)
	data.Txs[0] = types.Tx([]byte("regular"))
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 3, data))

	// daHeight=4: graceBoundary(3) < 4 → epoch1 checked; F1 missing → malicious.
	mockFIWithTxs(client, 0, 1, [][]byte{forcedTx})
	data2 := makeData("tchain", 2, 1)
	data2.Txs[0] = types.Tx([]byte("regular"))
	err := s.VerifyForcedInclusionTxs(t.Context(), 4, data2)
	require.Error(t, err)
	require.ErrorIs(t, err, errMaliciousProposer)
}

// TestVerifyForcedInclusionTxs_MultipleEpochsFirstOKSecondMalicious verifies
// per-epoch independence: F1 (epoch1) is included, F2 (epoch2) is not.
//
// daStart=0, epochSize=2: epoch1 graceBoundary=3, epoch2 graceBoundary=5.
// At daHeight=6 both are past their boundaries; F1 seen → OK, F2 missing → malicious.
func TestVerifyForcedInclusionTxs_MultipleEpochsFirstOKSecondMalicious(t *testing.T) {
	s, client, mockExec := newForcedInclusionSyncer(t, 0, 2)
	setupExecMocks(mockExec)

	addr, pub, signer := buildSyncTestSigner(t)
	f1, _ := makeSignedDataBytes(t, "tchain", 10, addr, pub, signer, 1)
	f2, _ := makeSignedDataBytes(t, "tchain", 11, addr, pub, signer, 1)

	mockFIWithTxs(client, 0, 1, [][]byte{f1})
	mockFIWithTxs(client, 2, 3, [][]byte{f2})

	// daHeight=2: F1 recorded; epoch1 graceBoundary(3) not yet crossed.
	data1 := makeData("tchain", 1, 1)
	data1.Txs[0] = types.Tx(f1)
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 2, data1))

	// daHeight=6: epoch1 (boundary 3) and epoch2 (boundary 5) both checked.
	// F1 seen → OK; F2 missing → malicious.
	data2 := makeData("tchain", 2, 1)
	data2.Txs[0] = types.Tx([]byte("regular"))
	err := s.VerifyForcedInclusionTxs(t.Context(), 6, data2)
	require.Error(t, err)
	require.ErrorIs(t, err, errMaliciousProposer)
}

// TestVerifyForcedInclusionTxs_BeforeDaStart verifies that daHeight < daStartHeight is a no-op.
func TestVerifyForcedInclusionTxs_BeforeDaStart(t *testing.T) {
	s, _, _ := newForcedInclusionSyncer(t, 100, 5)

	data := makeData("tchain", 1, 1)
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 50, data))
}

// TestVerifyForcedInclusionTxs_InvalidForcedTxsFiltered verifies that txs rejected
// by FilterTxs are not counted against the sequencer.
//
// daStart=0, epochSize=1: epoch1=[0,0], graceBoundary=1. Checked at daHeight=2.
func TestVerifyForcedInclusionTxs_InvalidForcedTxsFiltered(t *testing.T) {
	s, client, mockExec := newForcedInclusionSyncer(t, 0, 1)

	addr, pub, signer := buildSyncTestSigner(t)
	validTx, _ := makeSignedDataBytes(t, "tchain", 10, addr, pub, signer, 1)
	invalidTx := []byte("this-is-garbage")

	// Custom FilterTxs registered first so it takes priority over any later pass-through.
	mockExec.On("FilterTxs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		func(_ context.Context, txs [][]byte, _, _ uint64, _ bool) []execution.FilterStatus {
			out := make([]execution.FilterStatus, len(txs))
			for i, tx := range txs {
				if string(tx) == string(invalidTx) {
					out[i] = execution.FilterRemove
				} else {
					out[i] = execution.FilterOK
				}
			}
			return out
		}, nil,
	).Maybe()
	mockExec.On("GetExecutionInfo", mock.Anything).Return(execution.ExecutionInfo{MaxGas: 1000000}, nil).Maybe()

	// epoch1 [0,0]: validTx + invalidTx in DA; only validTx is included in the block.
	client.On("Retrieve", mock.Anything, uint64(0), []byte("nsForcedInclusion")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: time.Now()},
		Data:       [][]byte{validTx, invalidTx},
	}).Maybe()

	// daHeight=2: past graceBoundary(1); invalidTx absence must not be flagged.
	data := makeData("tchain", 1, 1)
	data.Txs[0] = types.Tx(validTx)
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 2, data))
}

// TestVerifyForcedInclusionTxs_EpochSizeZeroDisabled verifies that epochSize=0
// (forced inclusion disabled) causes an immediate nil return.
func TestVerifyForcedInclusionTxs_EpochSizeZeroDisabled(t *testing.T) {
	s, _, _ := newForcedInclusionSyncer(t, 0, 0)

	data := makeData("tchain", 1, 1)
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 100, data))
}

// TestVerifyForcedInclusionTxs_SeenTxsAccumulateAcrossCalls verifies that a tx
// recorded in an earlier call is credited when its epoch is checked in a later call.
//
// daStart=0, epochSize=2: F1 included at daHeight=1; epoch1 checked at daHeight=4.
func TestVerifyForcedInclusionTxs_SeenTxsAccumulateAcrossCalls(t *testing.T) {
	s, client, mockExec := newForcedInclusionSyncer(t, 0, 2)
	setupExecMocks(mockExec)

	addr, pub, signer := buildSyncTestSigner(t)
	forcedTx, _ := makeSignedDataBytes(t, "tchain", 10, addr, pub, signer, 1)

	mockFIWithTxs(client, 0, 1, [][]byte{forcedTx})

	// daHeight=1: graceBoundary(3) >= 1 → not checked; F1 recorded.
	d1 := makeData("tchain", 1, 1)
	d1.Txs[0] = types.Tx(forcedTx)
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 1, d1))

	// daHeight=4: graceBoundary(3) < 4 → epoch1 checked; F1 in seenBlockTxs → OK.
	d2 := makeData("tchain", 2, 1)
	d2.Txs[0] = types.Tx([]byte("regular"))
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 4, d2))
}

// TestGracePeriodForEpoch_NoBlocksSeen verifies the base grace period when no blocks are recorded.
func TestGracePeriodForEpoch_NoBlocksSeen(t *testing.T) {
	s := &Syncer{daBlockBytes: make(map[uint64]uint64)}
	grace := s.gracePeriodForEpoch(0, 4)
	require.Equal(t, baseGracePeriodEpochs, grace)
}

// TestGracePeriodForEpoch_LightBlocks verifies the base grace period for near-empty blocks.
func TestGracePeriodForEpoch_LightBlocks(t *testing.T) {
	s := &Syncer{daBlockBytes: make(map[uint64]uint64)}
	// 1 KB << 0.8·DefaultMaxBlobSize → extra=0.
	for h := uint64(0); h <= 4; h++ {
		s.daBlockBytes[h] = 1024
	}
	grace := s.gracePeriodForEpoch(0, 4)
	require.Equal(t, baseGracePeriodEpochs, grace)
}

// TestGracePeriodForEpoch_FullBlocks verifies that 100%-full blocks return at least the base grace.
// avgBytes=M, threshold=0.8·M → extra=(M-0.8·M)/0.8·M=0 (integer) → grace=base.
func TestGracePeriodForEpoch_FullBlocks(t *testing.T) {
	s := &Syncer{daBlockBytes: make(map[uint64]uint64)}
	for h := uint64(0); h <= 4; h++ {
		s.daBlockBytes[h] = uint64(common.DefaultMaxBlobSize)
	}
	grace := s.gracePeriodForEpoch(0, 4)
	require.GreaterOrEqual(t, grace, baseGracePeriodEpochs)
}

// TestGracePeriodForEpoch_ExtendedUnderHighCongestion verifies extra grace for congested epochs.
// avgBytes=1.6·M, threshold=0.8·M → extra=1 → grace=base+1.
func TestGracePeriodForEpoch_ExtendedUnderHighCongestion(t *testing.T) {
	s := &Syncer{daBlockBytes: make(map[uint64]uint64)}
	congested := uint64(float64(common.DefaultMaxBlobSize) * 1.6)
	for h := uint64(0); h <= 2; h++ {
		s.daBlockBytes[h] = congested
	}
	grace := s.gracePeriodForEpoch(0, 2)
	require.Equal(t, baseGracePeriodEpochs+1, grace)
}

// TestGracePeriodForEpoch_CappedAtMax verifies the grace period never exceeds maxGracePeriodEpochs.
func TestGracePeriodForEpoch_CappedAtMax(t *testing.T) {
	s := &Syncer{daBlockBytes: make(map[uint64]uint64)}
	huge := uint64(common.DefaultMaxBlobSize) * 100
	for h := uint64(0); h <= 4; h++ {
		s.daBlockBytes[h] = huge
	}
	grace := s.gracePeriodForEpoch(0, 4)
	require.Equal(t, maxGracePeriodEpochs, grace)
}

// TestVerifyForcedInclusionTxs_DynamicGrace_CongestedEpochGetsExtraTime verifies
// that a congested epoch extends the grace window so F1 is not flagged prematurely.
//
// daStart=0, epochSize=2: blocks at avgBytes=1.6·M → extra=1 → graceBoundary=5.
// F1 not flagged at daHeight=5 (boundary not crossed); included at daHeight=6 → OK.
func TestVerifyForcedInclusionTxs_DynamicGrace_CongestedEpochGetsExtraTime(t *testing.T) {
	// asyncFetcher lookahead = epochSize*2 = 4; mock heights 0–9 to cover it.
	s, client, mockExec := newForcedInclusionSyncer(t, 0, 2)
	setupExecMocks(mockExec)

	addr, pub, signer := buildSyncTestSigner(t)
	forcedTx, _ := makeSignedDataBytes(t, "tchain", 10, addr, pub, signer, 1)

	mockFIWithTxs(client, 0, 1, [][]byte{forcedTx})
	mockFIEmpty(client, 2, 9)

	// avgBytes = 1.6·M → extra=1 → gracePeriodForEpoch(0,1)=2 → graceBoundary=5.
	blockBytes := uint64(float64(common.DefaultMaxBlobSize) * 1.6)

	d0 := makeData("tchain", 1, 1)
	d0.Txs[0] = types.Tx(make([]byte, blockBytes))
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 0, d0))

	d1 := makeData("tchain", 2, 1)
	d1.Txs[0] = types.Tx(make([]byte, blockBytes))
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 1, d1))

	require.Equal(t, baseGracePeriodEpochs+uint64(1), s.gracePeriodForEpoch(0, 1))

	// daHeight=5: graceBoundary(5) >= 5 → not yet checked → OK.
	d5 := makeData("tchain", 3, 1)
	d5.Txs[0] = types.Tx([]byte("regular"))
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 5, d5))

	// daHeight=6: graceBoundary(5) < 6 → epoch1 checked; F1 included here → OK.
	d6 := makeData("tchain", 4, 1)
	d6.Txs[0] = types.Tx(forcedTx)
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 6, d6))
}

// TestVerifyForcedInclusionTxs_DynamicGrace_LightEpochBaseOnly verifies that
// near-empty blocks use the base grace period and a missing tx is flagged on time.
//
// daStart=0, epochSize=2: light blocks → grace=base → graceBoundary=3. F1 missing at daHeight=4.
func TestVerifyForcedInclusionTxs_DynamicGrace_LightEpochBaseOnly(t *testing.T) {
	s, client, mockExec := newForcedInclusionSyncer(t, 0, 2)
	setupExecMocks(mockExec)

	addr, pub, signer := buildSyncTestSigner(t)
	forcedTx, _ := makeSignedDataBytes(t, "tchain", 10, addr, pub, signer, 1)

	mockFIWithTxs(client, 0, 1, [][]byte{forcedTx})

	d0 := makeData("tchain", 1, 1)
	d0.Txs[0] = types.Tx([]byte("tiny"))
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 0, d0))

	// daHeight=4: graceBoundary(3) < 4 → F1 missing → malicious.
	d4 := makeData("tchain", 2, 1)
	d4.Txs[0] = types.Tx([]byte("regular"))
	err := s.VerifyForcedInclusionTxs(t.Context(), 4, d4)
	require.Error(t, err)
	require.ErrorIs(t, err, errMaliciousProposer)
}

// TestPruning_daBlockBytesRemovedAfterEpochCheck verifies that daBlockBytes entries
// are deleted for heights in a checked epoch.
//
// daStart=0, epochSize=2: epoch1=[0,1], graceBoundary=3. Heights 0,1 pruned at daHeight=4.
func TestPruning_daBlockBytesRemovedAfterEpochCheck(t *testing.T) {
	s, client, mockExec := newForcedInclusionSyncer(t, 0, 2)
	setupExecMocks(mockExec)
	mockFIEmpty(client, 0, 1) // epoch1 has no forced txs

	d0 := makeData("tchain", 1, 1)
	d0.Txs[0] = types.Tx([]byte("tx-at-0"))
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 0, d0))

	d1 := makeData("tchain", 2, 1)
	d1.Txs[0] = types.Tx([]byte("tx-at-1"))
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 1, d1))

	s.forcedInclusionMu.RLock()
	require.Contains(t, s.daBlockBytes, uint64(0), "height 0 should be present before pruning")
	require.Contains(t, s.daBlockBytes, uint64(1), "height 1 should be present before pruning")
	s.forcedInclusionMu.RUnlock()

	// daHeight=4: epoch1 checked and pruned.
	d4 := makeData("tchain", 3, 1)
	d4.Txs[0] = types.Tx([]byte("regular"))
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 4, d4))

	s.forcedInclusionMu.RLock()
	require.NotContains(t, s.daBlockBytes, uint64(0), "height 0 should be pruned after epoch1 check")
	require.NotContains(t, s.daBlockBytes, uint64(1), "height 1 should be pruned after epoch1 check")
	s.forcedInclusionMu.RUnlock()
}

// TestPruning_seenBlockTxsRemovedAfterEpochCheck verifies that seenBlockTxs entries
// are deleted for txs first seen within a checked epoch.
//
// daStart=0, epochSize=2: F1 included at height 0; hash pruned after daHeight=4.
func TestPruning_seenBlockTxsRemovedAfterEpochCheck(t *testing.T) {
	s, client, mockExec := newForcedInclusionSyncer(t, 0, 2)
	setupExecMocks(mockExec)

	addr, pub, signer := buildSyncTestSigner(t)
	forcedTx, _ := makeSignedDataBytes(t, "tchain", 10, addr, pub, signer, 1)

	mockFIWithTxs(client, 0, 1, [][]byte{forcedTx})

	d0 := makeData("tchain", 1, 1)
	d0.Txs[0] = types.Tx(forcedTx)
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 0, d0))

	txHash := hashTx(forcedTx)
	s.forcedInclusionMu.RLock()
	_, presentBefore := s.seenBlockTxs[txHash]
	s.forcedInclusionMu.RUnlock()
	require.True(t, presentBefore, "F1 hash should be in seenBlockTxs after inclusion")

	// daHeight=4: epoch1 checked and pruned.
	d4 := makeData("tchain", 2, 1)
	d4.Txs[0] = types.Tx([]byte("regular"))
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 4, d4))

	s.forcedInclusionMu.RLock()
	_, presentAfter := s.seenBlockTxs[txHash]
	s.forcedInclusionMu.RUnlock()
	require.False(t, presentAfter, "F1 hash should be pruned from seenBlockTxs after epoch1 check")
}

// TestPruning_futureHeightsUntouched verifies that heights in unchecked epochs are not pruned.
//
// daStart=0, epochSize=2: epoch1 pruned at daHeight=4; epoch2 (graceBoundary=5) untouched.
func TestPruning_futureHeightsUntouched(t *testing.T) {
	s, client, mockExec := newForcedInclusionSyncer(t, 0, 2)
	setupExecMocks(mockExec)
	mockFIEmpty(client, 0, 1) // epoch1: no forced txs

	for h := uint64(0); h <= 3; h++ {
		d := makeData("tchain", h+1, 1)
		d.Txs[0] = types.Tx([]byte("tx"))
		require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), h, d))
	}

	// daHeight=4: epoch1 pruned; epoch2 not yet due.
	d4 := makeData("tchain", 5, 1)
	d4.Txs[0] = types.Tx([]byte("regular"))
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 4, d4))

	s.forcedInclusionMu.RLock()
	require.NotContains(t, s.daBlockBytes, uint64(0), "height 0 should be pruned")
	require.NotContains(t, s.daBlockBytes, uint64(1), "height 1 should be pruned")
	require.Contains(t, s.daBlockBytes, uint64(2), "height 2 should NOT be pruned yet")
	require.Contains(t, s.daBlockBytes, uint64(3), "height 3 should NOT be pruned yet")
	s.forcedInclusionMu.RUnlock()
}

// TestPruning_multipleEpochsPrunedTogether verifies that skipping ahead prunes all
// epochs whose grace boundaries were crossed.
//
// daStart=0, epochSize=2: epoch1 graceBoundary=3, epoch2 graceBoundary=5. Both pruned at daHeight=6.
func TestPruning_multipleEpochsPrunedTogether(t *testing.T) {
	s, client, mockExec := newForcedInclusionSyncer(t, 0, 2)
	setupExecMocks(mockExec)
	mockFIEmpty(client, 0, 1) // epoch1: no forced txs
	mockFIEmpty(client, 2, 3) // epoch2: no forced txs

	for h := uint64(0); h <= 3; h++ {
		d := makeData("tchain", h+1, 1)
		d.Txs[0] = types.Tx([]byte("tx"))
		require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), h, d))
	}

	// daHeight=6: both epoch grace boundaries crossed at once.
	d6 := makeData("tchain", 5, 1)
	d6.Txs[0] = types.Tx([]byte("regular"))
	require.NoError(t, s.VerifyForcedInclusionTxs(t.Context(), 6, d6))

	s.forcedInclusionMu.RLock()
	require.NotContains(t, s.daBlockBytes, uint64(0), "height 0 should be pruned")
	require.NotContains(t, s.daBlockBytes, uint64(1), "height 1 should be pruned")
	require.NotContains(t, s.daBlockBytes, uint64(2), "height 2 should be pruned")
	require.NotContains(t, s.daBlockBytes, uint64(3), "height 3 should be pruned")
	s.forcedInclusionMu.RUnlock()
}
