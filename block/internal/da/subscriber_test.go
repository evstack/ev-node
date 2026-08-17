package da

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	datypes "github.com/evstack/ev-node/pkg/da/types"
	testmocks "github.com/evstack/ev-node/test/mocks"
)

// MockSubscriberHandler mocks SubscriberHandler
type MockSubscriberHandler struct {
	mock.Mock
}

type lifecycleTestClient struct {
	Client

	subscribeCalls atomic.Int32
	entered        [2]chan struct{}
	canceled       [2]chan struct{}
	release        [2]chan struct{}
	releaseOnce    [2]sync.Once
}

func newLifecycleTestClient() *lifecycleTestClient {
	client := &lifecycleTestClient{}
	for i := range 2 {
		client.entered[i] = make(chan struct{})
		client.canceled[i] = make(chan struct{})
		client.release[i] = make(chan struct{})
	}
	return client
}

func (c *lifecycleTestClient) SupportsSubscribe() bool {
	return true
}

func (c *lifecycleTestClient) Subscribe(
	ctx context.Context,
	_ []byte,
	_ bool,
) (<-chan datypes.SubscriptionEvent, error) {
	generation := int(c.subscribeCalls.Add(1) - 1)
	close(c.entered[generation])
	<-ctx.Done()
	close(c.canceled[generation])
	<-c.release[generation]
	return nil, ctx.Err()
}

func (c *lifecycleTestClient) releaseGeneration(generation int) {
	c.releaseOnce[generation].Do(func() {
		close(c.release[generation])
	})
}

type lifecycleTestHandler struct{}

func (lifecycleTestHandler) HandleEvent(context.Context, datypes.SubscriptionEvent, bool) error {
	return nil
}

func (lifecycleTestHandler) HandleCatchup(context.Context, uint64) error {
	return nil
}

func waitForLifecycleSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func (m *MockSubscriberHandler) HandleEvent(ctx context.Context, ev datypes.SubscriptionEvent, isInline bool) error {
	args := m.Called(ctx, ev, isInline)
	return args.Error(0)
}

func (m *MockSubscriberHandler) HandleCatchup(ctx context.Context, height uint64) error {
	args := m.Called(ctx, height)
	return args.Error(0)
}

func TestSubscriber_LifecycleSerializesStartAndStop(t *testing.T) {
	client := newLifecycleTestClient()
	t.Cleanup(func() {
		client.releaseGeneration(0)
		client.releaseGeneration(1)
	})

	sub := NewSubscriber(SubscriberConfig{
		Client:      client,
		Logger:      zerolog.Nop(),
		Handler:     lifecycleTestHandler{},
		Namespaces:  [][]byte{[]byte("ns")},
		DABlockTime: time.Hour,
	})

	if err := sub.Start(t.Context()); err != nil {
		t.Fatalf("start first generation: %v", err)
	}
	waitForLifecycleSignal(t, client.entered[0], "first generation to start")

	stopDone := make(chan struct{})
	go func() {
		sub.Stop()
		close(stopDone)
	}()
	waitForLifecycleSignal(t, client.canceled[0], "first generation cancellation")

	restartDone := make(chan error, 1)
	go func() {
		restartDone <- sub.Start(t.Context())
	}()
	select {
	case err := <-restartDone:
		if err != nil {
			t.Fatalf("start while stopping: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Start did not return while the previous generation was stopping")
	}
	select {
	case <-client.entered[1]:
		t.Fatal("Start launched a second generation while the first generation was stopping")
	default:
	}
	select {
	case <-stopDone:
		t.Fatal("Stop returned before the first generation exited")
	default:
	}

	concurrentStopDone := make(chan struct{})
	go func() {
		sub.Stop()
		close(concurrentStopDone)
	}()
	select {
	case <-concurrentStopDone:
		t.Fatal("concurrent Stop returned before the first generation exited")
	default:
	}

	client.releaseGeneration(0)
	waitForLifecycleSignal(t, stopDone, "first Stop to return")
	waitForLifecycleSignal(t, concurrentStopDone, "concurrent Stop to return")

	if err := sub.Start(t.Context()); err != nil {
		t.Fatalf("restart subscriber: %v", err)
	}
	waitForLifecycleSignal(t, client.entered[1], "second generation to start")

	secondStopDone := make(chan struct{})
	go func() {
		sub.Stop()
		close(secondStopDone)
	}()
	waitForLifecycleSignal(t, client.canceled[1], "second generation cancellation")
	select {
	case <-secondStopDone:
		t.Fatal("Stop returned before the second generation exited")
	default:
	}
	client.releaseGeneration(1)
	waitForLifecycleSignal(t, secondStopDone, "second Stop to return")

	// Stopping an already stopped subscriber remains safe.
	sub.Stop()
}

func TestSubscriber_RunCatchup(t *testing.T) {
	t.Run("success_sequence", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		mockHandler := new(MockSubscriberHandler)
		mockClient := testmocks.NewMockClient(t)

		sub := NewSubscriber(SubscriberConfig{
			Client:      mockClient,
			Logger:      zerolog.Nop(),
			Handler:     mockHandler,
			Namespaces:  [][]byte{[]byte("ns")},
			StartHeight: 100,
			DABlockTime: time.Millisecond,
		})

		// It should process observed heights [100..101] then stop when local passes highestSeen.
		sub.updateHighest(101)
		sub.seenSubscriptionEvent.Store(true)
		mockHandler.On("HandleCatchup", mock.Anything, uint64(100)).Return(nil).Once()
		mockHandler.On("HandleCatchup", mock.Anything, uint64(101)).Return(nil).Once()

		sub.runCatchup(ctx)

		mockHandler.AssertExpectations(t)
		assert.Equal(t, uint64(102), sub.LocalDAHeight())
		assert.True(t, sub.HasReachedHead())
	})

	t.Run("backoff_on_error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		mockHandler := new(MockSubscriberHandler)
		mockClient := testmocks.NewMockClient(t)

		sub := NewSubscriber(SubscriberConfig{
			Client:      mockClient,
			Logger:      zerolog.Nop(),
			Handler:     mockHandler,
			Namespaces:  [][]byte{[]byte("ns")},
			StartHeight: 100,
			DABlockTime: time.Millisecond,
		})

		var callCount int

		sub.updateHighest(100)
		sub.seenSubscriptionEvent.Store(true)

		mockHandler.On("HandleCatchup", mock.Anything, uint64(100)).
			Run(func(args mock.Arguments) {
				callCount++
			}).
			Return(errors.New("network failure")).Once()

		mockHandler.On("HandleCatchup", mock.Anything, uint64(100)).
			Run(func(args mock.Arguments) {
				callCount++
			}).
			Return(nil).Once()

		sub.runCatchup(ctx)

		mockHandler.AssertExpectations(t)
		assert.Equal(t, 2, callCount)
		assert.Equal(t, uint64(101), sub.LocalDAHeight())
		assert.True(t, sub.HasReachedHead())
	})
}

func TestSubscriber_RunSubscription_InlineDoesNotPrematurelyReachHead(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	mockHandler := new(MockSubscriberHandler)
	mockClient := testmocks.NewMockClient(t)

	sub := NewSubscriber(SubscriberConfig{
		Client:      mockClient,
		Logger:      zerolog.Nop(),
		Handler:     mockHandler,
		Namespaces:  [][]byte{[]byte("ns")},
		StartHeight: 100,
		DABlockTime: time.Hour,
	})

	subCh := make(chan datypes.SubscriptionEvent, 2)
	mockClient.EXPECT().
		Subscribe(mock.Anything, []byte("ns"), false).
		Return((<-chan datypes.SubscriptionEvent)(subCh), nil).
		Once()

	mockHandler.On("HandleEvent", mock.Anything, datypes.SubscriptionEvent{
		Height: 101,
		Blobs:  [][]byte{[]byte("h101")},
	}, false).Return(nil).Once()
	mockHandler.On("HandleEvent", mock.Anything, datypes.SubscriptionEvent{
		Height: 100,
		Blobs:  [][]byte{[]byte("h100")},
	}, true).Return(nil).Once()

	subCh <- datypes.SubscriptionEvent{Height: 101, Blobs: [][]byte{[]byte("h101")}}
	subCh <- datypes.SubscriptionEvent{Height: 100, Blobs: [][]byte{[]byte("h100")}}
	close(subCh)

	err := sub.runSubscription(ctx)
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "subscription channel closed")
	}
	assert.False(t, sub.HasReachedHead())
	assert.Equal(t, uint64(101), sub.LocalDAHeight())
	assert.Equal(t, uint64(101), sub.HighestSeenDAHeight())
}
