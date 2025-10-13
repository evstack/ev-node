package syncing

import (
	"context"
	crand "crypto/rand"
	"errors"
	"testing"
	"time"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/core/execution"
	"github.com/evstack/ev-node/pkg/genesis"
	signerpkg "github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/signer/noop"
	testmocks "github.com/evstack/ev-node/test/mocks"
	"github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/store"
	extmocks "github.com/evstack/ev-node/test/mocks/external"
	"github.com/evstack/ev-node/types"
)

// helper to create a signer, pubkey and address for tests
func buildSyncTestSigner(tb testing.TB) (addr []byte, pub crypto.PubKey, signer signerpkg.Signer) {
	tb.Helper()
	priv, _, err := crypto.GenerateEd25519Key(crand.Reader)
	require.NoError(tb, err)
	n, err := noop.NewNoopSigner(priv)
	require.NoError(tb, err)
	a, err := n.GetAddress()
	require.NoError(tb, err)
	p, err := n.GetPublic()
	require.NoError(tb, err)
	return a, p, n
}

// makeSignedHeaderBytes builds a valid SignedHeader and returns its binary encoding and the object
func makeSignedHeaderBytes(tb testing.TB, chainID string, height uint64, proposer []byte, pub crypto.PubKey, signer signerpkg.Signer, appHash []byte, data *types.Data) ([]byte, *types.SignedHeader) {
	time := uint64(time.Now().UnixNano())
	dataHash := common.DataHashForEmptyTxs
	if data != nil {
		time = uint64(data.Time().UnixNano())
		dataHash = data.DACommitment()
	}

	hdr := &types.SignedHeader{
		Header: types.Header{
			BaseHeader:      types.BaseHeader{ChainID: chainID, Height: height, Time: time},
			AppHash:         appHash,
			DataHash:        dataHash,
			ProposerAddress: proposer,
		},
		Signer: types.Signer{PubKey: pub, Address: proposer},
	}
	bz, err := types.DefaultAggregatorNodeSignatureBytesProvider(&hdr.Header)
	require.NoError(tb, err)
	sig, err := signer.Sign(bz)
	require.NoError(tb, err)
	hdr.Signature = sig
	bin, err := hdr.MarshalBinary()
	require.NoError(tb, err)
	return bin, hdr
}

func makeData(chainID string, height uint64, txs int) *types.Data {
	d := &types.Data{
		Metadata: &types.Metadata{
			ChainID: chainID,
			Height:  height,
			Time:    uint64(time.Now().Add(time.Duration(height) * time.Second).UnixNano())},
	}
	if txs > 0 {
		d.Txs = make(types.Txs, txs)
		for i := 0; i < txs; i++ {
			d.Txs[i] = types.Tx([]byte{byte(height), byte(i)})
		}
	}
	return d
}

func TestSyncer_validateBlock_DataHashMismatch(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)

	cfg := config.DefaultConfig()
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	mockExec := testmocks.NewMockExecutor(t)

	s := NewSyncer(
		st,
		mockExec,
		nil,
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

	// Create header and data with correct hash
	data := makeData(gen.ChainID, 1, 2) // non-empty
	_, header := makeSignedHeaderBytes(t, gen.ChainID, 1, addr, pub, signer, nil, data)

	err = s.validateBlock(header, data)
	require.NoError(t, err)

	// Create header and data with mismatched hash
	data = makeData(gen.ChainID, 1, 2) // non-empty
	_, header = makeSignedHeaderBytes(t, gen.ChainID, 1, addr, pub, signer, nil, nil)
	err = s.validateBlock(header, data)
	require.Error(t, err)

	// Create header and empty data
	data = makeData(gen.ChainID, 1, 0) // empty
	_, header = makeSignedHeaderBytes(t, gen.ChainID, 2, addr, pub, signer, nil, nil)
	err = s.validateBlock(header, data)
	require.Error(t, err)
}

func TestProcessHeightEvent_SyncsAndUpdatesState(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)

	cfg := config.DefaultConfig()
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	mockExec := testmocks.NewMockExecutor(t)

	s := NewSyncer(
		st,
		mockExec,
		nil,
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

	require.NoError(t, s.initializeState())
	// set a context for internal loops that expect it
	s.ctx = context.Background()
	// Create signed header & data for height 1
	lastState := s.GetLastState()
	data := makeData(gen.ChainID, 1, 0)
	_, hdr := makeSignedHeaderBytes(t, gen.ChainID, 1, addr, pub, signer, lastState.AppHash, data)

	// Expect ExecuteTxs call for height 1
	mockExec.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, uint64(1), mock.Anything, lastState.AppHash).
		Return([]byte("app1"), uint64(1024), nil).Once()

	evt := common.DAHeightEvent{Header: hdr, Data: data, DaHeight: 1}
	s.processHeightEvent(&evt)

	h, err := st.Height(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(1), h)
	st1, err := st.GetState(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(1), st1.LastBlockHeight)
}

func TestSequentialBlockSync(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	cfg := config.DefaultConfig()
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	mockExec := testmocks.NewMockExecutor(t)

	s := NewSyncer(
		st,
		mockExec,
		nil,
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
	require.NoError(t, s.initializeState())
	s.ctx = context.Background()

	// Sync two consecutive blocks via processHeightEvent so ExecuteTxs is called and state stored
	st0 := s.GetLastState()
	data1 := makeData(gen.ChainID, 1, 1) // non-empty
	_, hdr1 := makeSignedHeaderBytes(t, gen.ChainID, 1, addr, pub, signer, st0.AppHash, data1)
	// Expect ExecuteTxs call for height 1
	mockExec.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, uint64(1), mock.Anything, st0.AppHash).
		Return([]byte("app1"), uint64(1024), nil).Once()
	evt1 := common.DAHeightEvent{Header: hdr1, Data: data1, DaHeight: 10}
	s.processHeightEvent(&evt1)

	st1, _ := st.GetState(context.Background())
	data2 := makeData(gen.ChainID, 2, 0) // empty data
	_, hdr2 := makeSignedHeaderBytes(t, gen.ChainID, 2, addr, pub, signer, st1.AppHash, data2)
	// Expect ExecuteTxs call for height 2
	mockExec.EXPECT().ExecuteTxs(mock.Anything, mock.Anything, uint64(2), mock.Anything, st1.AppHash).
		Return([]byte("app2"), uint64(1024), nil).Once()
	evt2 := common.DAHeightEvent{Header: hdr2, Data: data2, DaHeight: 11}
	s.processHeightEvent(&evt2)

	// Mark DA inclusion in cache (as DA retrieval would)
	cm.SetDataDAIncluded(data1.DACommitment().String(), 10)
	cm.SetDataDAIncluded(data2.DACommitment().String(), 11) // empty data still needs cache entry
	cm.SetHeaderDAIncluded(hdr1.Header.Hash().String(), 10)
	cm.SetHeaderDAIncluded(hdr2.Header.Hash().String(), 11)

	// Verify both blocks were synced correctly
	finalState, _ := st.GetState(context.Background())
	assert.Equal(t, uint64(2), finalState.LastBlockHeight)

	// Verify DA inclusion markers are set
	_, ok := cm.GetHeaderDAIncluded(hdr1.Hash().String())
	assert.True(t, ok)
	_, ok = cm.GetHeaderDAIncluded(hdr2.Hash().String())
	assert.True(t, ok)
	_, ok = cm.GetDataDAIncluded(data1.DACommitment().String())
	assert.True(t, ok)
	_, ok = cm.GetDataDAIncluded(data2.DACommitment().String())
	assert.True(t, ok)
}

func TestSyncer_sendNonBlockingSignal(t *testing.T) {
	s := &Syncer{logger: zerolog.Nop()}
	ch := make(chan struct{}, 1)
	ch <- struct{}{}
	done := make(chan struct{})
	go func() {
		s.sendNonBlockingSignal(ch, "test")
		close(done)
	}()
	select {
	case <-done:
		// ok
	case <-time.After(200 * time.Millisecond):
		t.Fatal("sendNonBlockingSignal blocked unexpectedly")
	}
}

func TestSyncer_processPendingEvents(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	// current height 1
	batch, err := st.NewBatch(context.Background())
	require.NoError(t, err)
	require.NoError(t, batch.SetHeight(1))
	require.NoError(t, batch.Commit())

	s := &Syncer{
		store:      st,
		cache:      cm,
		ctx:        context.Background(),
		heightInCh: make(chan common.DAHeightEvent, 2),
		logger:     zerolog.Nop(),
	}

	// create two pending events, one for height 2 (> current) and one for height 1 (<= current)
	evt1 := &common.DAHeightEvent{Header: &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{ChainID: "c", Height: 1}}}, Data: &types.Data{Metadata: &types.Metadata{ChainID: "c", Height: 1}}}
	evt2 := &common.DAHeightEvent{Header: &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{ChainID: "c", Height: 2}}}, Data: &types.Data{Metadata: &types.Metadata{ChainID: "c", Height: 2}}}
	cm.SetPendingEvent(1, evt1)
	cm.SetPendingEvent(2, evt2)

	s.processPendingEvents()

	// should have forwarded height 2 and removed both
	select {
	case got := <-s.heightInCh:
		assert.Equal(t, uint64(2), got.Header.Height())
	default:
		t.Fatal("expected a forwarded pending event")
	}

	// Verify the event was removed by trying to get it again
	remaining := cm.GetNextPendingEvent(2)
	assert.Nil(t, remaining)
}

func TestSyncLoopPersistState(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	myDAHeightOffset := uint64(1)
	myFutureDAHeight := uint64(9)

	addr, pub, signer := buildSyncTestSigner(t)
	cfg := config.DefaultConfig()
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr, DAStartHeight: myDAHeightOffset}

	dummyExec := execution.NewDummyExecutor()

	// Create mock stores for P2P
	mockHeaderStore := extmocks.NewMockStore[*types.SignedHeader](t)
	mockHeaderStore.EXPECT().Height().Return(uint64(0)).Maybe()

	mockDataStore := extmocks.NewMockStore[*types.Data](t)
	mockDataStore.EXPECT().Height().Return(uint64(0)).Maybe()

	mockP2PHeaderStore := common.NewMockBroadcaster[*types.SignedHeader](t)
	mockP2PHeaderStore.EXPECT().Store().Return(mockHeaderStore).Maybe()

	mockP2PDataStore := common.NewMockBroadcaster[*types.Data](t)
	mockP2PDataStore.EXPECT().Store().Return(mockDataStore).Maybe()

	syncerInst1 := NewSyncer(
		st,
		dummyExec,
		nil,
		cm,
		common.NopMetrics(),
		cfg,
		gen,
		mockP2PHeaderStore,
		mockP2PDataStore,
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	require.NoError(t, syncerInst1.initializeState())

	ctx, cancel := context.WithCancel(t.Context())
	syncerInst1.ctx = ctx
	daRtrMock, p2pHndlMock := newMockdaRetriever(t), newMockp2pHandler(t)
	p2pHndlMock.On("ProcessHeaderRange", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	p2pHndlMock.On("ProcessDataRange", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	syncerInst1.daRetriever, syncerInst1.p2pHandler = daRtrMock, p2pHndlMock

	// with n da blobs fetched
	for i := range myFutureDAHeight - myDAHeightOffset {
		chainHeight, daHeight := i, i+myDAHeightOffset
		emptyData := &types.Data{
			Metadata: &types.Metadata{
				ChainID: gen.ChainID,
				Height:  chainHeight,
				Time:    uint64(time.Now().Add(time.Duration(chainHeight) * time.Second).UnixNano()),
			},
		}
		_, sigHeader := makeSignedHeaderBytes(t, gen.ChainID, chainHeight, addr, pub, signer, nil, emptyData)
		evts := []common.DAHeightEvent{{
			Header:   sigHeader,
			Data:     emptyData,
			DaHeight: daHeight,
		}}
		daRtrMock.On("RetrieveFromDA", mock.Anything, daHeight).Return(evts, nil)
	}

	// stop at next height
	daRtrMock.On("RetrieveFromDA", mock.Anything, myFutureDAHeight).
		Run(func(_ mock.Arguments) {
			// wait for consumer to catch up
			require.Eventually(t, func() bool {
				return len(syncerInst1.heightInCh) == 0
			}, 1*time.Second, 10*time.Millisecond)
			cancel()
		}).
		Return(nil, coreda.ErrHeightFromFuture)

	go syncerInst1.processLoop()

	// dssync from DA until stop height reached
	syncerInst1.syncLoop()
	t.Log("syncLoop on instance1 completed")

	// wait for all events consumed
	require.NoError(t, cm.SaveToDisk())
	t.Log("processLoop on instance1 completed")

	// then
	daRtrMock.AssertExpectations(t)
	p2pHndlMock.AssertExpectations(t)

	// and all processed - verify no events remain at heights we tested
	event1 := syncerInst1.cache.GetNextPendingEvent(1)
	assert.Nil(t, event1)
	event2 := syncerInst1.cache.GetNextPendingEvent(2)
	assert.Nil(t, event2)
	assert.Len(t, syncerInst1.heightInCh, 0)

	// and when new instance is up on restart
	cm, err = cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)
	require.NoError(t, cm.LoadFromDisk())

	syncerInst2 := NewSyncer(
		st,
		dummyExec,
		nil,
		cm,
		common.NopMetrics(),
		cfg,
		gen,
		mockP2PHeaderStore,
		mockP2PDataStore,
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	require.NoError(t, syncerInst2.initializeState())
	require.Equal(t, myFutureDAHeight-1, syncerInst2.GetDAHeight())

	ctx, cancel = context.WithCancel(t.Context())
	t.Cleanup(cancel)
	syncerInst2.ctx = ctx
	daRtrMock, p2pHndlMock = newMockdaRetriever(t), newMockp2pHandler(t)
	p2pHndlMock.On("ProcessHeaderRange", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	p2pHndlMock.On("ProcessDataRange", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	syncerInst2.daRetriever, syncerInst2.p2pHandler = daRtrMock, p2pHndlMock

	daRtrMock.On("RetrieveFromDA", mock.Anything, mock.Anything).
		Run(func(arg mock.Arguments) {
			cancel()
			// retrieve last one again
			assert.Equal(t, myFutureDAHeight-1, arg.Get(1).(uint64))
		}).
		Return(nil, nil)

	// when it starts, it should fetch from the last height it stopped at
	t.Log("syncLoop on instance2 started")
	syncerInst2.syncLoop()

	t.Log("syncLoop exited")
}

func TestSyncer_executeTxsWithRetry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupMock     func(*testmocks.MockExecutor)
		expectSuccess bool
		expectHash    []byte
		expectError   string
	}{
		{
			name: "success on first attempt",
			setupMock: func(exec *testmocks.MockExecutor) {
				exec.On("ExecuteTxs", mock.Anything, mock.Anything, uint64(100), mock.Anything, mock.Anything).
					Return([]byte("new-hash"), uint64(0), nil).Once()
			},
			expectSuccess: true,
			expectHash:    []byte("new-hash"),
		},
		{
			name: "success on second attempt",
			setupMock: func(exec *testmocks.MockExecutor) {
				exec.On("ExecuteTxs", mock.Anything, mock.Anything, uint64(100), mock.Anything, mock.Anything).
					Return([]byte(nil), uint64(0), errors.New("temporary failure")).Once()
				exec.On("ExecuteTxs", mock.Anything, mock.Anything, uint64(100), mock.Anything, mock.Anything).
					Return([]byte("new-hash"), uint64(0), nil).Once()
			},
			expectSuccess: true,
			expectHash:    []byte("new-hash"),
		},
		{
			name: "success on third attempt",
			setupMock: func(exec *testmocks.MockExecutor) {
				exec.On("ExecuteTxs", mock.Anything, mock.Anything, uint64(100), mock.Anything, mock.Anything).
					Return([]byte(nil), uint64(0), errors.New("temporary failure")).Times(2)
				exec.On("ExecuteTxs", mock.Anything, mock.Anything, uint64(100), mock.Anything, mock.Anything).
					Return([]byte("new-hash"), uint64(0), nil).Once()
			},
			expectSuccess: true,
			expectHash:    []byte("new-hash"),
		},
		{
			name: "failure after max retries",
			setupMock: func(exec *testmocks.MockExecutor) {
				exec.On("ExecuteTxs", mock.Anything, mock.Anything, uint64(100), mock.Anything, mock.Anything).
					Return([]byte(nil), uint64(0), errors.New("persistent failure")).Times(common.MaxRetriesBeforeHalt)
			},
			expectSuccess: false,
			expectError:   "failed to execute transactions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			exec := testmocks.NewMockExecutor(t)
			tt.setupMock(exec)

			s := &Syncer{
				exec:   exec,
				ctx:    ctx,
				logger: zerolog.Nop(),
			}

			rawTxs := [][]byte{[]byte("tx1"), []byte("tx2")}
			header := types.Header{
				BaseHeader: types.BaseHeader{Height: 100, Time: uint64(time.Now().UnixNano())},
			}
			currentState := types.State{AppHash: []byte("current-hash")}

			result, err := s.executeTxsWithRetry(ctx, rawTxs, header, currentState)

			if tt.expectSuccess {
				require.NoError(t, err)
				assert.Equal(t, tt.expectHash, result)
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
