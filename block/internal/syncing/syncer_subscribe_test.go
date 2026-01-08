package syncing

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/types"
	"github.com/rs/zerolog"
)

func TestSyncer_DAWorker_CatchupThenFollow(t *testing.T) {
	// Setup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockDARetriever := NewMockDARetriever(t)

	c, err := cache.NewCacheManager(config.DefaultConfig(), zerolog.Nop())
	require.NoError(t, err)

	// Create Syncer (simplified)
	syncer := &Syncer{
		logger:            zerolog.Nop(),
		daRetriever:       mockDARetriever,
		ctx:               ctx,
		daRetrieverHeight: &atomic.Uint64{}, // Starts at 0
		cache:             c,
		config:            config.DefaultConfig(),
		heightInCh:        make(chan common.DAHeightEvent, 100),
		wg:                sync.WaitGroup{},
	}
	syncer.daRetrieverHeight.Store(1)

	// Defines

	// Expectations:
	// 1. fetchDAUntilCaughtUp called initially.
	//    It calls RetrieveFromDA. We Simulate 1 block retrieved, then caught up.

	// Event 1 (Catchup)
	evt1 := common.DAHeightEvent{
		Header:   &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: 1}}},
		DaHeight: 1,
	}
	// Event 2 (Follow via subscription)
	evt2 := common.DAHeightEvent{
		Header:   &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: 2}}},
		DaHeight: 2,
	}

	// 1. Initial Catchup
	// Retrieve height 1 -> succeed
	mockDARetriever.On("RetrieveFromDA", mock.Anything, uint64(1)).Return([]common.DAHeightEvent{evt1}, nil).Once()
	// Retrieve height 2 -> Future/Caught up -> return HeightFromFuture error to signal caught up
	mockDARetriever.On("RetrieveFromDA", mock.Anything, uint64(2)).Return(nil, datypes.ErrHeightFromFuture).Once()

	// 2. Subscribe
	subCh := make(chan common.DAHeightEvent, 10)
	mockDARetriever.On("Subscribe", mock.Anything).Return((<-chan common.DAHeightEvent)(subCh), nil)

	// Run daWorkerLoop in goroutine
	syncer.wg.Add(1)
	go syncer.daWorkerLoop()

	// Verify Catchup Event received
	select {
	case e := <-syncer.heightInCh:
		assert.Equal(t, uint64(1), e.DaHeight)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for catchup event")
	}

	// Now we should be in Follow mode (Subscribe called).
	// Feed event 2 to subscription
	subCh <- evt2

	// Verify Follow Event received
	select {
	case e := <-syncer.heightInCh:
		assert.Equal(t, uint64(2), e.DaHeight)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for follow event")
	}

	// Cleanup
	cancel()
	syncer.wg.Wait()
}

func TestSyncer_DAWorker_GapDetection(t *testing.T) {
	// Setup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockDARetriever := NewMockDARetriever(t)

	c, err := cache.NewCacheManager(config.DefaultConfig(), zerolog.Nop())
	require.NoError(t, err)

	syncer := &Syncer{
		logger:            zerolog.Nop(),
		daRetriever:       mockDARetriever,
		ctx:               ctx,
		daRetrieverHeight: &atomic.Uint64{},
		cache:             c,
		config:            config.DefaultConfig(),
		heightInCh:        make(chan common.DAHeightEvent, 100),
		wg:                sync.WaitGroup{},
	}
	syncer.daRetrieverHeight.Store(10) // Assume we are at height 10

	// Expectations:
	// 1. Initial Catchup -> immediately caught up
	mockDARetriever.On("RetrieveFromDA", mock.Anything, uint64(10)).Return(nil, datypes.ErrHeightFromFuture).Once()

	// 2. Subscribe
	subCh := make(chan common.DAHeightEvent, 10)
	mockDARetriever.On("Subscribe", mock.Anything).Return((<-chan common.DAHeightEvent)(subCh), nil)

	// 3. Gap handling
	// We send event with DA height 15 (expected 10). Should trigger catchup.
	// Catchup should call RetrieveFromDA(10), then 11...
	// We simulate RetrieveFromDA(10) returning an event, then caught up again.

	evtGap := common.DAHeightEvent{
		Header:   &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: 15}}},
		DaHeight: 15,
	}

	evtFill := common.DAHeightEvent{
		Header:   &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: 10}}},
		DaHeight: 10,
	}

	// Catchup logic when gap is detected:
	// 1. RetrieveFromDA(10) -> returns evtFill
	// 2. RetrieveFromDA(11) -> returns HeightFromFuture (caught up again)
	mockDARetriever.On("RetrieveFromDA", mock.Anything, uint64(10)).Return([]common.DAHeightEvent{evtFill}, nil).Once()
	mockDARetriever.On("RetrieveFromDA", mock.Anything, uint64(11)).Return(nil, datypes.ErrHeightFromFuture).Once()

	// Run daWorkerLoop
	syncer.wg.Add(1)
	go syncer.daWorkerLoop()

	// Wait for initial catchup (nothing to receive)

	// Trigger Gap by sending event 15
	subCh <- evtGap

	// Verify that we received the "Fill" event (height 10)
	// Note: The gap event (15) is DROPPED/Ignored in the current logic?
	// Logic:
	// if event.DaHeight > nextExpectedHeight {
	//    return nil // break follow loop, go to catchup
	// }
	// So 15 is dropped. Catchup starts at 10. Fetch 10.

	select {
	case e := <-syncer.heightInCh:
		assert.Equal(t, uint64(10), e.DaHeight)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for fill event")
	}

	// Cleanup
	cancel()
	syncer.wg.Wait()
}
