package syncing

import (
	"context"
	"errors"
	"testing"
	"time"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/core/execution"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

func TestHeightEvent_Structure(t *testing.T) {
	// Test that HeightEvent has all required fields
	event := common.DAHeightEvent{
		Header: &types.SignedHeader{
			Header: types.Header{
				BaseHeader: types.BaseHeader{
					ChainID: "test-chain",
					Height:  1,
				},
			},
		},
		Data: &types.Data{
			Metadata: &types.Metadata{
				ChainID: "test-chain",
				Height:  1,
			},
		},
		DaHeight:               100,
		HeaderDaIncludedHeight: 100,
	}

	assert.Equal(t, uint64(1), event.Header.Height())
	assert.Equal(t, uint64(1), event.Data.Height())
	assert.Equal(t, uint64(100), event.DaHeight)
	assert.Equal(t, uint64(100), event.HeaderDaIncludedHeight)
}

func TestCacheDAHeightEvent_Usage(t *testing.T) {
	// Test the exported DAHeightEvent type
	header := &types.SignedHeader{
		Header: types.Header{
			BaseHeader: types.BaseHeader{
				ChainID: "test-chain",
				Height:  1,
			},
		},
	}

	data := &types.Data{
		Metadata: &types.Metadata{
			ChainID: "test-chain",
			Height:  1,
		},
	}

	event := &common.DAHeightEvent{
		Header:                 header,
		Data:                   data,
		DaHeight:               100,
		HeaderDaIncludedHeight: 100,
	}

	// Verify all fields are accessible
	assert.NotNil(t, event.Header)
	assert.NotNil(t, event.Data)
	assert.Equal(t, uint64(100), event.DaHeight)
	assert.Equal(t, uint64(100), event.HeaderDaIncludedHeight)
}

func TestSyncer_isHeightFromFutureError(t *testing.T) {
	s := &Syncer{}
	// exact error
	err := common.ErrHeightFromFutureStr
	assert.True(t, s.isHeightFromFutureError(err))
	// string-wrapped error
	err = errors.New("some context: " + common.ErrHeightFromFutureStr.Error())
	assert.True(t, s.isHeightFromFutureError(err))
	// unrelated
	assert.False(t, s.isHeightFromFutureError(errors.New("boom")))
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
	cm, err := cache.NewManager(config.DefaultConfig, st, zerolog.Nop())
	require.NoError(t, err)

	// current height 1
	require.NoError(t, st.SetHeight(context.Background(), 1))

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

	remaining := cm.GetPendingEvents()
	assert.Len(t, remaining, 0)
}

func TestSyncLoopPersistState(t *testing.T) {
	ds := sync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig, st, zerolog.Nop())
	require.NoError(t, err)

	myDAHeightOffset := uint64(1)
	myFutureDAHeight := uint64(9)

	addr, pub, signer := buildSyncTestSigner(t)
	cfg := config.DefaultConfig
	cfg.DA.StartHeight = myDAHeightOffset
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	dummyExec := execution.NewDummyExecutor()

	syncerInst1 := NewSyncer(
		st,
		dummyExec,
		nil,
		cm,
		common.NopMetrics(),
		cfg,
		gen,
		nil,
		nil,
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)
	require.NoError(t, syncerInst1.initializeState())

	ctx, cancel := context.WithCancel(t.Context())
	syncerInst1.ctx = ctx
	daRtrMock, p2pHndlMock := newMockdaRetriever(t), newMockp2pHandler(t)
	syncerInst1.daRetriever, syncerInst1.p2pHandler = daRtrMock, p2pHndlMock

	// with n da blobs fetched
	for i := range myFutureDAHeight - myDAHeightOffset {
		chainHeight, daHeight := i, i+myDAHeightOffset
		_, sigHeader := makeSignedHeaderBytes(t, gen.ChainID, chainHeight, addr, pub, signer, nil)
		//_, sigData := makeSignedDataBytes(t, gen.ChainID, chainHeight, addr, pub, signer, 1)
		emptyData := types.Data{
			Metadata: &types.Metadata{
				ChainID: sigHeader.ChainID(),
				Height:  sigHeader.Height(),
				Time:    sigHeader.BaseHeader.Time,
			},
		}
		evts := []common.DAHeightEvent{{
			Header:                 sigHeader,
			Data:                   &emptyData,
			DaHeight:               daHeight,
			HeaderDaIncludedHeight: daHeight,
		}}
		daRtrMock.On("RetrieveFromDA", mock.Anything, uint64(daHeight)).Return(evts, nil)
	}

	// stop at next height
	daRtrMock.On("RetrieveFromDA", mock.Anything, uint64(myFutureDAHeight)).
		Run(func(_ mock.Arguments) {
			// wait for consumer to catch up
			require.Eventually(t, func() bool {
				return len(syncerInst1.heightInCh) == 0
			}, 1*time.Second, 10*time.Millisecond)
			cancel()
		}).
		Return(nil, coreda.ErrHeightFromFuture)

	go syncerInst1.processLoop()

	// sync from DA until stop height reached
	syncerInst1.syncLoop()
	t.Log("syncLoop on instance1 completed")

	// wait for all events consumed
	require.NoError(t, cm.SaveToDisk())
	t.Log("processLoop on instance1 completed")

	// then
	daRtrMock.AssertExpectations(t)
	p2pHndlMock.AssertExpectations(t)

	// and all processed
	assert.Len(t, syncerInst1.cache.GetPendingEvents(), 0)
	assert.Len(t, syncerInst1.heightInCh, 0)

	// and when new instance is up on restart
	cm, err = cache.NewManager(config.DefaultConfig, st, zerolog.Nop())
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
		nil,
		nil,
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
