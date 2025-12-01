package syncing

import (
	"context"
	"testing"
	"time"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAsyncDARetriever_RequestRetrieval(t *testing.T) {
	logger := zerolog.Nop()
	mockRetriever := NewMockDARetriever(t)
	resultCh := make(chan common.DAHeightEvent, 10)

	asyncRetriever := NewAsyncDARetriever(mockRetriever, resultCh, logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	asyncRetriever.Start(ctx)
	defer asyncRetriever.Stop()

	// 1. Test successful retrieval
	height1 := uint64(100)
	mockRetriever.EXPECT().RetrieveFromDA(mock.Anything, height1).Return([]common.DAHeightEvent{{DaHeight: height1}}, nil).Once()

	asyncRetriever.RequestRetrieval(height1)

	select {
	case event := <-resultCh:
		assert.Equal(t, height1, event.DaHeight)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for result")
	}

	// 2. Test deduplication (idempotency)
	// We'll block the retriever to simulate a slow request, then send multiple requests for the same height
	height2 := uint64(200)

	// Create a channel to signal when the mock is called
	calledCh := make(chan struct{})
	// Create a channel to unblock the mock
	unblockCh := make(chan struct{})

	mockRetriever.EXPECT().RetrieveFromDA(mock.Anything, height2).RunAndReturn(func(ctx context.Context, h uint64) ([]common.DAHeightEvent, error) {
		close(calledCh)
		<-unblockCh
		return []common.DAHeightEvent{{DaHeight: h}}, nil
	}).Once() // Should be called only once despite multiple requests

	// Send first request
	asyncRetriever.RequestRetrieval(height2)

	// Wait for the worker to pick it up
	select {
	case <-calledCh:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for retriever call")
	}

	// Send duplicate requests while the first one is still in flight
	asyncRetriever.RequestRetrieval(height2)
	asyncRetriever.RequestRetrieval(height2)

	// Unblock the worker
	close(unblockCh)

	// We should receive exactly one result
	select {
	case event := <-resultCh:
		assert.Equal(t, height2, event.DaHeight)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for result")
	}

	// Ensure no more results come through
	select {
	case <-resultCh:
		t.Fatal("received duplicate result")
	default:
	}
}

func TestAsyncDARetriever_WorkerPoolLimit(t *testing.T) {
	logger := zerolog.Nop()
	mockRetriever := NewMockDARetriever(t)
	resultCh := make(chan common.DAHeightEvent, 100)

	asyncRetriever := NewAsyncDARetriever(mockRetriever, resultCh, logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	asyncRetriever.Start(ctx)
	defer asyncRetriever.Stop()

	// We have 5 workers. We'll block them all.
	unblockCh := make(chan struct{})

	// Expect 5 calls that block
	for i := 0; i < 5; i++ {
		h := uint64(1000 + i)
		mockRetriever.EXPECT().RetrieveFromDA(mock.Anything, h).RunAndReturn(func(ctx context.Context, h uint64) ([]common.DAHeightEvent, error) {
			<-unblockCh
			return []common.DAHeightEvent{{DaHeight: h}}, nil
		}).Once()
		asyncRetriever.RequestRetrieval(h)
	}

	// Give workers time to pick up tasks
	time.Sleep(100 * time.Millisecond)

	// Now send a 6th request. It should be queued but not processed yet.
	height6 := uint64(1005)
	processed6 := make(chan struct{})
	mockRetriever.EXPECT().RetrieveFromDA(mock.Anything, height6).RunAndReturn(func(ctx context.Context, h uint64) ([]common.DAHeightEvent, error) {
		close(processed6)
		return []common.DAHeightEvent{{DaHeight: h}}, nil
	}).Once()

	asyncRetriever.RequestRetrieval(height6)

	// Ensure 6th request is NOT processed yet
	select {
	case <-processed6:
		t.Fatal("6th request processed too early")
	default:
	}

	// Unblock workers
	close(unblockCh)

	// Now 6th request should be processed
	select {
	case <-processed6:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for 6th request")
	}
}
