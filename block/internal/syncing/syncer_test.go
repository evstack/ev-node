package syncing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
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
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
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
