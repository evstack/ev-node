package da

import (
	"context"
	"errors"
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

func (m *MockSubscriberHandler) HandleEvent(ctx context.Context, ev datypes.SubscriptionEvent, isInline bool) error {
	args := m.Called(ctx, ev, isInline)
	return args.Error(0)
}

func (m *MockSubscriberHandler) HandleCatchup(ctx context.Context, height uint64) error {
	args := m.Called(ctx, height)
	return args.Error(0)
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

func TestSubscriber_RewindTo(t *testing.T) {
	t.Run("no_op_when_target_is_equal_or_higher", func(t *testing.T) {
		sub := NewSubscriber(SubscriberConfig{
			Client:      testmocks.NewMockClient(t),
			Logger:      zerolog.Nop(),
			Handler:     new(MockSubscriberHandler),
			Namespaces:  [][]byte{[]byte("ns")},
			StartHeight: 100,
			DABlockTime: time.Millisecond,
		})
		sub.localDAHeight.Store(100)

		sub.RewindTo(100)
		assert.Equal(t, uint64(100), sub.LocalDAHeight())

		sub.RewindTo(200)
		assert.Equal(t, uint64(100), sub.LocalDAHeight())
	})

	t.Run("rewinds_local_height_and_clears_head", func(t *testing.T) {
		sub := NewSubscriber(SubscriberConfig{
			Client:      testmocks.NewMockClient(t),
			Logger:      zerolog.Nop(),
			Handler:     new(MockSubscriberHandler),
			Namespaces:  [][]byte{[]byte("ns")},
			StartHeight: 100,
			DABlockTime: time.Millisecond,
		})
		sub.localDAHeight.Store(150)
		sub.headReached.Store(true)

		sub.RewindTo(120)

		assert.Equal(t, uint64(120), sub.LocalDAHeight())
		assert.False(t, sub.HasReachedHead())
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
