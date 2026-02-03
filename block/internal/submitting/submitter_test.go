package submitting

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/rpc/server"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/store"
	testmocks "github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
	"github.com/libp2p/go-libp2p/core/crypto"
)

func TestSubmitter_setFinalWithRetry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupMock      func(*testmocks.MockExecutor)
		expectSuccess  bool
		expectAttempts int
		expectError    string
	}{
		{
			name: "success on first attempt",
			setupMock: func(exec *testmocks.MockExecutor) {
				exec.On("SetFinal", mock.Anything, uint64(100)).Return(nil).Once()
			},
			expectSuccess:  true,
			expectAttempts: 1,
		},
		{
			name: "success on second attempt",
			setupMock: func(exec *testmocks.MockExecutor) {
				exec.On("SetFinal", mock.Anything, uint64(100)).Return(errors.New("temporary failure")).Once()
				exec.On("SetFinal", mock.Anything, uint64(100)).Return(nil).Once()
			},
			expectSuccess:  true,
			expectAttempts: 2,
		},
		{
			name: "success on third attempt",
			setupMock: func(exec *testmocks.MockExecutor) {
				exec.On("SetFinal", mock.Anything, uint64(100)).Return(errors.New("temporary failure")).Times(2)
				exec.On("SetFinal", mock.Anything, uint64(100)).Return(nil).Once()
			},
			expectSuccess:  true,
			expectAttempts: 3,
		},
		{
			name: "failure after max retries",
			setupMock: func(exec *testmocks.MockExecutor) {
				exec.On("SetFinal", mock.Anything, uint64(100)).Return(errors.New("persistent failure")).Times(common.MaxRetriesBeforeHalt)
			},
			expectSuccess: false,
			expectError:   "failed to set final height after 3 attempts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			exec := testmocks.NewMockExecutor(t)
			tt.setupMock(exec)

			s := &Submitter{
				exec:   exec,
				ctx:    ctx,
				logger: zerolog.Nop(),
			}

			err := s.setFinalWithRetry(100)

			if tt.expectSuccess {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				if tt.expectError != "" {
					assert.Contains(t, err.Error(), tt.expectError)
				}
			}

			exec.AssertExpectations(t)
		})
	}
}

func TestSubmitter_IsHeightDAIncluded(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	cm, st := newTestCacheAndStore(t)
	batch, err := st.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SetHeight(5))
	require.NoError(t, batch.Commit())

	s := &Submitter{store: st, cache: cm, logger: zerolog.Nop()}
	s.ctx = ctx

	h1, d1 := newHeaderAndData("chain", 3, true)
	h2, d2 := newHeaderAndData("chain", 4, true)

	cm.SetHeaderDAIncluded(h1.Hash().String(), 100, 2)
	cm.SetDataDAIncluded(d1.DACommitment().String(), 100, 2)
	cm.SetHeaderDAIncluded(h2.Hash().String(), 101, 4)
	// no data for h2

	specs := map[string]struct {
		height uint64
		header *types.SignedHeader
		data   *types.Data
		exp    bool
		expErr bool
	}{
		"below store height and cached": {height: 3, header: h1, data: d1, exp: true},
		"above store height":            {height: 6, header: h2, data: d2, exp: false},
		"data missing":                  {height: 4, header: h2, data: d2, exp: false},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			included, err := s.IsHeightDAIncluded(spec.height, spec.header, spec.data)
			if spec.expErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, spec.exp, included)
		})
	}
}

func TestSubmitter_setSequencerHeightToDAHeight(t *testing.T) {
	ctx := t.Context()
	cm, _ := newTestCacheAndStore(t)

	// Use a mock store to validate metadata writes
	mockStore := testmocks.NewMockStore(t)

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "test-ns"
	cfg.DA.DataNamespace = "test-data-ns"
	metrics := common.NopMetrics()
	daClient := testmocks.NewMockClient(t)
	// Namespace getters may be called implicitly; allow optional returns
	daClient.On("GetHeaderNamespace").Return([]byte(cfg.DA.Namespace)).Maybe()
	daClient.On("GetDataNamespace").Return([]byte(cfg.DA.DataNamespace)).Maybe()
	daClient.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()
	daClient.On("HasForcedInclusionNamespace").Return(false).Maybe()
	daSub := NewDASubmitter(daClient, cfg, genesis.Genesis{}, common.BlockOptions{}, metrics, zerolog.Nop(), nil, nil)
	s := NewSubmitter(mockStore, nil, cm, metrics, cfg, genesis.Genesis{}, daSub, nil, nil, zerolog.Nop(), nil)
	s.ctx = ctx

	h, d := newHeaderAndData("chain", 1, true)

	// set DA included heights in cache
	cm.SetHeaderDAIncluded(h.Hash().String(), 100, 1)
	cm.SetDataDAIncluded(d.DACommitment().String(), 90, 1)

	headerKey := store.GetHeightToDAHeightHeaderKey(1)
	dataKey := store.GetHeightToDAHeightDataKey(1)

	hBz := make([]byte, 8)
	binary.LittleEndian.PutUint64(hBz, 100)
	dBz := make([]byte, 8)
	binary.LittleEndian.PutUint64(dBz, 90)
	gBz := make([]byte, 8)
	binary.LittleEndian.PutUint64(gBz, 90) // min(header=100, data=90)

	mockStore.On("SetMetadata", mock.Anything, headerKey, hBz).Return(nil).Once()
	mockStore.On("SetMetadata", mock.Anything, dataKey, dBz).Return(nil).Once()
	mockStore.On("SetMetadata", mock.Anything, store.GenesisDAHeightKey, gBz).Return(nil).Once()

	require.NoError(t, s.setNodeHeightToDAHeight(ctx, 1, h, d, true))
}

func TestSubmitter_setNodeHeightToDAHeight_Errors(t *testing.T) {
	ctx := t.Context()
	cm, st := newTestCacheAndStore(t)

	s := &Submitter{store: st, cache: cm, logger: zerolog.Nop()}

	h, d := newHeaderAndData("chain", 1, true)

	// No cache entries -> expect error on missing header
	_, ok := cm.GetHeaderDAIncluded(h.Hash().String())
	assert.False(t, ok)
	assert.Error(t, s.setNodeHeightToDAHeight(ctx, 1, h, d, false))

	// Add header, missing data
	cm.SetHeaderDAIncluded(h.Hash().String(), 10, 1)
	assert.Error(t, s.setNodeHeightToDAHeight(ctx, 1, h, d, false))
}

func TestSubmitter_initializeDAIncludedHeight(t *testing.T) {
	ctx := t.Context()
	_, st := newTestCacheAndStore(t)

	// write DAIncludedHeightKey
	bz := make([]byte, 8)
	binary.LittleEndian.PutUint64(bz, 7)
	require.NoError(t, st.SetMetadata(ctx, store.DAIncludedHeightKey, bz))

	s := &Submitter{store: st, daIncludedHeight: &atomic.Uint64{}, logger: zerolog.Nop()}
	require.NoError(t, s.initializeDAIncludedHeight(ctx))
	assert.Equal(t, uint64(7), s.GetDAIncludedHeight())
}

func TestSubmitter_processDAInclusionLoop_advances(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Clean up any existing visualization server
	defer server.SetDAVisualizationServer(nil)
	server.SetDAVisualizationServer(nil)

	cm, st := newTestCacheAndStore(t)

	// small block time to tick quickly
	cfg := config.DefaultConfig()
	cfg.DA.BlockTime.Duration = 5 * time.Millisecond
	cfg.RPC.EnableDAVisualization = false // Ensure visualization is disabled
	metrics := common.PrometheusMetrics("test")

	exec := testmocks.NewMockExecutor(t)
	exec.On("SetFinal", mock.Anything, uint64(1)).Return(nil).Once()
	exec.On("SetFinal", mock.Anything, uint64(2)).Return(nil).Once()

	daClient := testmocks.NewMockClient(t)
	daClient.On("GetHeaderNamespace").Return([]byte(cfg.DA.Namespace)).Maybe()
	daClient.On("GetDataNamespace").Return([]byte(cfg.DA.DataNamespace)).Maybe()
	daClient.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()
	daClient.On("HasForcedInclusionNamespace").Return(false).Maybe()
	daSub := NewDASubmitter(daClient, cfg, genesis.Genesis{}, common.BlockOptions{}, metrics, zerolog.Nop(), nil, nil)
	s := NewSubmitter(st, exec, cm, metrics, cfg, genesis.Genesis{}, daSub, nil, nil, zerolog.Nop(), nil)

	// prepare two consecutive blocks in store with DA included in cache
	h1, d1 := newHeaderAndData("chain", 1, true)
	h2, d2 := newHeaderAndData("chain", 2, true)
	require.NotEqual(t, h1.Hash(), h2.Hash())
	require.NotEqual(t, d1.DACommitment(), d2.DACommitment())

	sig := types.Signature([]byte("sig"))

	// Save block 1
	batch1, err := st.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch1.SaveBlockData(h1, d1, &sig))
	require.NoError(t, batch1.SetHeight(1))
	require.NoError(t, batch1.Commit())

	// Save block 2
	batch2, err := st.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch2.SaveBlockData(h2, d2, &sig))
	require.NoError(t, batch2.SetHeight(2))
	require.NoError(t, batch2.Commit())

	cm.SetHeaderDAIncluded(h1.Hash().String(), 100, 1)
	cm.SetDataDAIncluded(d1.DACommitment().String(), 100, 1)
	cm.SetHeaderDAIncluded(h2.Hash().String(), 101, 2)
	cm.SetDataDAIncluded(d2.DACommitment().String(), 101, 2)

	s.ctx, s.cancel = ctx, cancel
	require.NoError(t, s.initializeDAIncludedHeight(ctx))
	require.Equal(t, uint64(0), s.GetDAIncludedHeight())

	// when
	require.NoError(t, s.Start(ctx))

	require.Eventually(t, func() bool {
		return s.GetDAIncludedHeight() == 2
	}, 1*time.Second, 10*time.Millisecond)
	require.NoError(t, s.Stop())

	// verify metadata mapping persisted in store
	for i := range 2 {
		hBz, err := st.GetMetadata(ctx, store.GetHeightToDAHeightHeaderKey(uint64(i+1)))
		require.NoError(t, err)
		assert.Equal(t, uint64(100+i), binary.LittleEndian.Uint64(hBz))

		dBz, err := st.GetMetadata(ctx, store.GetHeightToDAHeightDataKey(uint64(i+1)))
		require.NoError(t, err)
		assert.Equal(t, uint64(100+i), binary.LittleEndian.Uint64(dBz))
	}

	localHeightBz, err := s.store.GetMetadata(ctx, store.DAIncludedHeightKey)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), binary.LittleEndian.Uint64(localHeightBz))

}

// helper to create a minimal header and data for tests
func newHeaderAndData(chainID string, height uint64, nonEmpty bool) (*types.SignedHeader, *types.Data) {
	now := time.Now()
	h := &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{ChainID: chainID, Height: height, Time: uint64(now.UnixNano())}, ProposerAddress: []byte{1}}}
	d := &types.Data{Metadata: &types.Metadata{ChainID: chainID, Height: height, Time: uint64(now.UnixNano())}}
	if nonEmpty {
		d.Txs = types.Txs{types.Tx(fmt.Sprintf("any-unique-tx-%d-%d", height, now.UnixNano()))}
	}
	return h, d
}

func newTestCacheAndStore(t *testing.T) (cache.Manager, store.Store) {
	st := store.New(dssync.MutexWrap(datastore.NewMapDatastore()))
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)
	return cm, st
}

// TestSubmitter_daSubmissionLoop ensures that when there are pending headers/data,
// the submitter invokes the DA submitter methods periodically.
func TestSubmitter_daSubmissionLoop(t *testing.T) {
	ctx := t.Context()

	cm, st := newTestCacheAndStore(t)

	// Set a small block time so the ticker fires quickly
	cfg := config.DefaultConfig()
	cfg.DA.BlockTime.Duration = 5 * time.Millisecond
	// Use immediate batching strategy so submissions happen right away
	cfg.DA.BatchingStrategy = "immediate"
	metrics := common.NopMetrics()

	// Prepare fake DA submitter capturing calls
	fakeDA := &fakeDASubmitter{
		chHdr:  make(chan struct{}, 1),
		chData: make(chan struct{}, 1),
	}

	// Provide a non-nil executor; it won't be used because DA inclusion won't advance
	exec := testmocks.NewMockExecutor(t)

	batchingStrategy, err := NewBatchingStrategy(cfg.DA)
	require.NoError(t, err)

	s := &Submitter{
		store:            st,
		exec:             exec,
		cache:            cm,
		metrics:          metrics,
		config:           cfg,
		genesis:          genesis.Genesis{},
		daSubmitter:      fakeDA,
		signer:           &fakeSigner{},
		daIncludedHeight: &atomic.Uint64{},
		batchingStrategy: batchingStrategy,
		logger:           zerolog.Nop(),
	}

	// Set last submit times far in past so strategy allows submission
	pastTime := time.Now().Add(-time.Hour).UnixNano()
	s.lastHeaderSubmit.Store(pastTime)
	s.lastDataSubmit.Store(pastTime)

	// Make there be pending headers and data by setting store height > last submitted
	h1, d1 := newHeaderAndData("test-chain", 1, true)
	h2, d2 := newHeaderAndData("test-chain", 2, true)

	// Store the blocks
	batch, err := st.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SaveBlockData(h1, d1, &types.Signature{}))
	require.NoError(t, batch.SaveBlockData(h2, d2, &types.Signature{}))
	require.NoError(t, batch.SetHeight(2))
	require.NoError(t, batch.Commit())

	// Start and wait for calls
	require.NoError(t, s.Start(ctx))
	t.Cleanup(func() { _ = s.Stop() })

	// Both should be invoked eventually
	require.Eventually(t, func() bool {
		select {
		case <-fakeDA.chHdr:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		select {
		case <-fakeDA.chData:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
}

// fakeDASubmitter is a lightweight test double for the DASubmitter used by the loop.
type fakeDASubmitter struct {
	chHdr  chan struct{}
	chData chan struct{}
}

func (f *fakeDASubmitter) SubmitHeaders(ctx context.Context, _ []*types.SignedHeader, _ [][]byte, _ cache.Manager, _ signer.Signer) error {
	select {
	case f.chHdr <- struct{}{}:
	default:
	}
	return nil
}

func (f *fakeDASubmitter) SubmitData(ctx context.Context, _ []*types.SignedData, _ [][]byte, _ cache.Manager, _ signer.Signer, _ genesis.Genesis) error {
	select {
	case f.chData <- struct{}{}:
	default:
	}
	return nil
}

// fakeSigner implements signer.Signer with deterministic behavior for tests.
type fakeSigner struct{}

func (f *fakeSigner) Sign(msg []byte) ([]byte, error)   { return append([]byte(nil), msg...), nil }
func (f *fakeSigner) GetPublic() (crypto.PubKey, error) { return nil, nil }
func (f *fakeSigner) GetAddress() ([]byte, error)       { return []byte("addr"), nil }

func TestSubmitter_CacheClearedOnHeightInclusion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cm, st := newTestCacheAndStore(t)

	cfg := config.DefaultConfig()
	cfg.DA.BlockTime.Duration = 5 * time.Millisecond
	metrics := common.NopMetrics()

	exec := testmocks.NewMockExecutor(t)
	exec.On("SetFinal", mock.Anything, uint64(1)).Return(nil).Once()
	exec.On("SetFinal", mock.Anything, uint64(2)).Return(nil).Once()

	daClient := testmocks.NewMockClient(t)
	daClient.On("GetHeaderNamespace").Return([]byte(cfg.DA.Namespace)).Maybe()
	daClient.On("GetDataNamespace").Return([]byte(cfg.DA.DataNamespace)).Maybe()
	daClient.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()
	daClient.On("HasForcedInclusionNamespace").Return(false).Maybe()
	daSub := NewDASubmitter(daClient, cfg, genesis.Genesis{}, common.BlockOptions{}, metrics, zerolog.Nop(), nil, nil)
	s := NewSubmitter(st, exec, cm, metrics, cfg, genesis.Genesis{}, daSub, nil, nil, zerolog.Nop(), nil)

	// Create test blocks
	h1, d1 := newHeaderAndData("chain", 1, true)
	h2, d2 := newHeaderAndData("chain", 2, true)
	h3, d3 := newHeaderAndData("chain", 3, true)

	sig := types.Signature([]byte("sig"))

	// Save blocks to store
	blocks := []struct {
		header *types.SignedHeader
		data   *types.Data
		height uint64
	}{
		{h1, d1, 1},
		{h2, d2, 2},
		{h3, d3, 3},
	}

	for _, block := range blocks {
		batch, err := st.NewBatch(ctx)
		require.NoError(t, err)
		require.NoError(t, batch.SaveBlockData(block.header, block.data, &sig))
		require.NoError(t, batch.SetHeight(block.height))
		require.NoError(t, batch.Commit())
	}

	// Set up cache with headers and data seen for all heights
	cm.SetHeaderSeen(h1.Hash().String(), 1)
	cm.SetDataSeen(d1.DACommitment().String(), 1)
	cm.SetHeaderSeen(h2.Hash().String(), 2)
	cm.SetDataSeen(d2.DACommitment().String(), 2)
	cm.SetHeaderSeen(h3.Hash().String(), 3)
	cm.SetDataSeen(d3.DACommitment().String(), 3)

	// Verify items are seen in cache before processing
	assert.True(t, cm.IsHeaderSeen(h1.Hash().String()))
	assert.True(t, cm.IsDataSeen(d1.DACommitment().String()))
	assert.True(t, cm.IsHeaderSeen(h2.Hash().String()))
	assert.True(t, cm.IsDataSeen(d2.DACommitment().String()))
	assert.True(t, cm.IsHeaderSeen(h3.Hash().String()))
	assert.True(t, cm.IsDataSeen(d3.DACommitment().String()))

	// Set DA inclusion for heights 1 and 2 only (height 3 will remain unprocessed)
	cm.SetHeaderDAIncluded(h1.Hash().String(), 100, 1)
	cm.SetDataDAIncluded(d1.DACommitment().String(), 100, 1)
	cm.SetHeaderDAIncluded(h2.Hash().String(), 101, 2)
	cm.SetDataDAIncluded(d2.DACommitment().String(), 101, 2)

	s.ctx, s.cancel = ctx, cancel
	require.NoError(t, s.initializeDAIncludedHeight(ctx))
	require.Equal(t, uint64(0), s.GetDAIncludedHeight())

	// Start submitter to process DA inclusions
	require.NoError(t, s.Start(ctx))

	// Wait for heights 1 and 2 to be processed
	require.Eventually(t, func() bool {
		return s.GetDAIncludedHeight() == 2
	}, 1*time.Second, 10*time.Millisecond)

	require.NoError(t, s.Stop())

	// Verify cache is cleared for processed heights (1 and 2)
	assert.False(t, cm.IsHeaderSeen(h1.Hash().String()), "height 1 header should be cleared from cache")
	assert.False(t, cm.IsDataSeen(d1.DACommitment().String()), "height 1 data should be cleared from cache")
	assert.False(t, cm.IsHeaderSeen(h2.Hash().String()), "height 2 header should be cleared from cache")
	assert.False(t, cm.IsDataSeen(d2.DACommitment().String()), "height 2 data should be cleared from cache")

	// Verify DA inclusion status remains for processed heights
	_, h1DAIncluded := cm.GetHeaderDAIncluded(h1.Hash().String())
	_, d1DAIncluded := cm.GetDataDAIncluded(d1.DACommitment().String())
	_, h2DAIncluded := cm.GetHeaderDAIncluded(h2.Hash().String())
	_, d2DAIncluded := cm.GetDataDAIncluded(d2.DACommitment().String())
	assert.True(t, h1DAIncluded, "height 1 header DA inclusion status should remain")
	assert.True(t, d1DAIncluded, "height 1 data DA inclusion status should remain")
	assert.True(t, h2DAIncluded, "height 2 header DA inclusion status should remain")
	assert.True(t, d2DAIncluded, "height 2 data DA inclusion status should remain")

	// Verify unprocessed height 3 cache remains intact
	assert.True(t, cm.IsHeaderSeen(h3.Hash().String()), "height 3 header should remain in cache")
	assert.True(t, cm.IsDataSeen(d3.DACommitment().String()), "height 3 data should remain in cache")

	// Verify height 3 has no DA inclusion status since it wasn't processed
	_, h3DAIncluded := cm.GetHeaderDAIncluded(h3.Hash().String())
	_, d3DAIncluded := cm.GetDataDAIncluded(d3.DACommitment().String())
	assert.False(t, h3DAIncluded, "height 3 header should not have DA inclusion status")
	assert.False(t, d3DAIncluded, "height 3 data should not have DA inclusion status")
}
