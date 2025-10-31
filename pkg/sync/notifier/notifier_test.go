package notifier

import (
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		bufferSize int
		wantSize   int
	}{
		{
			name:       "positive buffer size",
			bufferSize: 10,
			wantSize:   10,
		},
		{
			name:       "zero buffer size defaults to 1",
			bufferSize: 0,
			wantSize:   1,
		},
		{
			name:       "negative buffer size defaults to 1",
			bufferSize: -5,
			wantSize:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zerolog.Nop()
			n := New(tt.bufferSize, logger)

			require.NotNil(t, n)
			assert.Equal(t, tt.wantSize, n.size)
			assert.NotNil(t, n.subs)
			assert.Equal(t, 0, len(n.subs))
			assert.Equal(t, uint64(0), n.next)
		})
	}
}

func TestNotifier_Subscribe(t *testing.T) {
	logger := zerolog.Nop()
	n := New(5, logger)

	t.Run("single subscription", func(t *testing.T) {
		sub := n.Subscribe()
		require.NotNil(t, sub)
		assert.NotNil(t, sub.C)
		assert.Equal(t, 1, n.SubscriberCount())
		assert.Equal(t, uint64(0), sub.id)
	})

	t.Run("multiple subscriptions increment IDs", func(t *testing.T) {
		sub2 := n.Subscribe()
		sub3 := n.Subscribe()

		assert.Equal(t, 3, n.SubscriberCount())
		assert.Equal(t, uint64(1), sub2.id)
		assert.Equal(t, uint64(2), sub3.id)
	})
}

func TestNotifier_Publish_NoSubscribers(t *testing.T) {
	logger := zerolog.Nop()
	n := New(5, logger)

	evt := Event{
		Type:      EventTypeHeader,
		Height:    100,
		Hash:      "test-hash",
		Source:    SourceLocal,
		Timestamp: time.Now(),
	}

	delivered := n.Publish(evt)
	assert.False(t, delivered, "should return false when no subscribers")
}

func TestNotifier_Publish_SingleSubscriber(t *testing.T) {
	logger := zerolog.Nop()
	n := New(5, logger)

	sub := n.Subscribe()
	defer sub.Cancel()

	evt := Event{
		Type:      EventTypeHeader,
		Height:    100,
		Hash:      "test-hash",
		Source:    SourceLocal,
		Timestamp: time.Now(),
	}

	delivered := n.Publish(evt)
	assert.True(t, delivered, "should deliver to subscriber")

	select {
	case received := <-sub.C:
		assert.Equal(t, evt.Type, received.Type)
		assert.Equal(t, evt.Height, received.Height)
		assert.Equal(t, evt.Hash, received.Hash)
		assert.Equal(t, evt.Source, received.Source)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestNotifier_Publish_MultipleSubscribers(t *testing.T) {
	logger := zerolog.Nop()
	n := New(5, logger)

	sub1 := n.Subscribe()
	sub2 := n.Subscribe()
	sub3 := n.Subscribe()
	defer sub1.Cancel()
	defer sub2.Cancel()
	defer sub3.Cancel()

	evt := Event{
		Type:      EventTypeData,
		Height:    200,
		Hash:      "block-hash",
		Source:    SourceP2P,
		Timestamp: time.Now(),
	}

	delivered := n.Publish(evt)
	assert.True(t, delivered, "should deliver to at least one subscriber")

	// Verify all three subscribers receive the event
	var wg sync.WaitGroup
	wg.Add(3)

	checkSubscriber := func(sub *Subscription) {
		defer wg.Done()
		select {
		case received := <-sub.C:
			assert.Equal(t, evt.Type, received.Type)
			assert.Equal(t, evt.Height, received.Height)
		case <-time.After(time.Second):
			t.Error("timeout waiting for event")
		}
	}

	go checkSubscriber(sub1)
	go checkSubscriber(sub2)
	go checkSubscriber(sub3)

	wg.Wait()
}

func TestNotifier_Publish_FullBuffer(t *testing.T) {
	logger := zerolog.Nop()
	n := New(2, logger) // Small buffer

	sub := n.Subscribe()
	defer sub.Cancel()

	// Fill the buffer
	evt1 := Event{Type: EventTypeHeader, Height: 1, Hash: "hash1", Source: SourceLocal, Timestamp: time.Now()}
	evt2 := Event{Type: EventTypeHeader, Height: 2, Hash: "hash2", Source: SourceLocal, Timestamp: time.Now()}

	assert.True(t, n.Publish(evt1))
	assert.True(t, n.Publish(evt2))

	// This should be dropped due to full buffer
	evt3 := Event{Type: EventTypeHeader, Height: 3, Hash: "hash3", Source: SourceLocal, Timestamp: time.Now()}
	delivered := n.Publish(evt3)

	// Should still return true if at least one subscriber got it (or false if all dropped)
	// In this case, buffer is full so it should be dropped
	// The function returns true if ANY subscriber received it without dropping
	// Since we have a full buffer, this will be dropped
	assert.False(t, delivered, "should not deliver when buffer is full")
}

func TestNotifier_Publish_MixedDelivery(t *testing.T) {
	logger := zerolog.Nop()
	n := New(1, logger) // Buffer size of 1

	// Create two subscribers
	sub1 := n.Subscribe()
	sub2 := n.Subscribe()
	defer sub1.Cancel()
	defer sub2.Cancel()

	evt1 := Event{Type: EventTypeHeader, Height: 1, Hash: "hash1", Source: SourceLocal, Timestamp: time.Now()}
	evt2 := Event{Type: EventTypeHeader, Height: 2, Hash: "hash2", Source: SourceLocal, Timestamp: time.Now()}

	// Fill both buffers
	assert.True(t, n.Publish(evt1))

	// Read from sub1 only, leaving sub2 full
	select {
	case <-sub1.C:
		// sub1 buffer is now empty
	case <-time.After(time.Millisecond * 100):
		t.Fatal("timeout reading from sub1")
	}

	// Publish another event - should deliver to sub1 but drop for sub2
	delivered := n.Publish(evt2)
	assert.True(t, delivered, "should deliver to at least one subscriber (sub1)")
}

func TestSubscription_Cancel(t *testing.T) {
	logger := zerolog.Nop()
	n := New(5, logger)

	t.Run("cancel removes subscriber", func(t *testing.T) {
		sub := n.Subscribe()
		assert.Equal(t, 1, n.SubscriberCount())

		sub.Cancel()
		assert.Equal(t, 0, n.SubscriberCount())
	})

	t.Run("cancel closes channel", func(t *testing.T) {
		sub := n.Subscribe()
		sub.Cancel()

		// Channel should be closed
		_, ok := <-sub.C
		assert.False(t, ok, "channel should be closed after cancel")
	})

	t.Run("multiple cancels are safe", func(t *testing.T) {
		sub := n.Subscribe()
		sub.Cancel()
		sub.Cancel()
		sub.Cancel()

		assert.Equal(t, 0, n.SubscriberCount())
	})

	t.Run("nil subscription cancel is safe", func(t *testing.T) {
		var sub *Subscription
		assert.NotPanics(t, func() {
			sub.Cancel()
		})
	})

	t.Run("cancel prevents future deliveries", func(t *testing.T) {
		sub := n.Subscribe()
		sub.Cancel()

		evt := Event{
			Type:      EventTypeHeader,
			Height:    100,
			Hash:      "test",
			Source:    SourceLocal,
			Timestamp: time.Now(),
		}

		delivered := n.Publish(evt)
		assert.False(t, delivered, "should not deliver to cancelled subscriber")
	})
}

func TestNotifier_SubscriberCount(t *testing.T) {
	logger := zerolog.Nop()
	n := New(5, logger)

	assert.Equal(t, 0, n.SubscriberCount())

	sub1 := n.Subscribe()
	assert.Equal(t, 1, n.SubscriberCount())

	sub2 := n.Subscribe()
	assert.Equal(t, 2, n.SubscriberCount())

	sub3 := n.Subscribe()
	assert.Equal(t, 3, n.SubscriberCount())

	sub1.Cancel()
	assert.Equal(t, 2, n.SubscriberCount())

	sub2.Cancel()
	assert.Equal(t, 1, n.SubscriberCount())

	sub3.Cancel()
	assert.Equal(t, 0, n.SubscriberCount())
}

func TestEventTypes(t *testing.T) {
	tests := []struct {
		name     string
		eventType EventType
		want     string
	}{
		{
			name:     "header event type",
			eventType: EventTypeHeader,
			want:     "header",
		},
		{
			name:     "data event type",
			eventType: EventTypeData,
			want:     "data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.eventType))
		})
	}
}

func TestEventSources(t *testing.T) {
	tests := []struct {
		name   string
		source EventSource
		want   string
	}{
		{
			name:   "unknown source",
			source: SourceUnknown,
			want:   "unknown",
		},
		{
			name:   "local source",
			source: SourceLocal,
			want:   "local",
		},
		{
			name:   "p2p source",
			source: SourceP2P,
			want:   "p2p",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.source))
		})
	}
}

func TestNotifier_ConcurrentOperations(t *testing.T) {
	logger := zerolog.Nop()
	n := New(10, logger)

	const numSubscribers = 10
	const numEvents = 100

	var wg sync.WaitGroup

	// Start multiple subscribers
	subs := make([]*Subscription, numSubscribers)
	for i := 0; i < numSubscribers; i++ {
		subs[i] = n.Subscribe()
	}

	// Start publishers
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numEvents; i++ {
			evt := Event{
				Type:      EventTypeHeader,
				Height:    uint64(i),
				Hash:      "hash",
				Source:    SourceLocal,
				Timestamp: time.Now(),
			}
			n.Publish(evt)
		}
	}()

	// Start consumers
	for i := 0; i < numSubscribers; i++ {
		wg.Add(1)
		go func(sub *Subscription) {
			defer wg.Done()
			count := 0
			timeout := time.After(2 * time.Second)
			for {
				select {
				case _, ok := <-sub.C:
					if !ok {
						return
					}
					count++
					if count >= numEvents {
						return
					}
				case <-timeout:
					return
				}
			}
		}(subs[i])
	}

	// Cancel some subscribers concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond)
		for i := 0; i < numSubscribers/2; i++ {
			subs[i].Cancel()
		}
	}()

	wg.Wait()

	// Verify final state
	assert.LessOrEqual(t, n.SubscriberCount(), numSubscribers/2)
}

func TestNotifier_SubscriberCountRace(t *testing.T) {
	logger := zerolog.Nop()
	n := New(5, logger)

	var wg sync.WaitGroup
	const numGoroutines = 10

	// Concurrent Subscribe and Cancel operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sub := n.Subscribe()
			_ = n.SubscriberCount()
			sub.Cancel()
			_ = n.SubscriberCount()
		}()
	}

	wg.Wait()
	assert.Equal(t, 0, n.SubscriberCount())
}

func TestEvent_AllFields(t *testing.T) {
	now := time.Now()
	evt := Event{
		Type:      EventTypeData,
		Height:    12345,
		Hash:      "0xabcdef123456",
		Source:    SourceP2P,
		Timestamp: now,
	}

	assert.Equal(t, EventTypeData, evt.Type)
	assert.Equal(t, uint64(12345), evt.Height)
	assert.Equal(t, "0xabcdef123456", evt.Hash)
	assert.Equal(t, SourceP2P, evt.Source)
	assert.Equal(t, now, evt.Timestamp)
}

func TestSubscription_CancelIdempotency(t *testing.T) {
	logger := zerolog.Nop()
	n := New(5, logger)

	sub := n.Subscribe()
	initialCount := n.SubscriberCount()

	// Cancel multiple times
	for i := 0; i < 5; i++ {
		sub.Cancel()
	}

	// Should only decrease count once
	assert.Equal(t, initialCount-1, n.SubscriberCount())
}

func BenchmarkNotifier_Publish(b *testing.B) {
	logger := zerolog.Nop()
	n := New(100, logger)

	// Create 10 subscribers
	subs := make([]*Subscription, 10)
	for i := 0; i < 10; i++ {
		subs[i] = n.Subscribe()
		// Drain events in background
		go func(s *Subscription) {
			for range s.C {
			}
		}(subs[i])
	}

	evt := Event{
		Type:      EventTypeHeader,
		Height:    100,
		Hash:      "test-hash",
		Source:    SourceLocal,
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n.Publish(evt)
	}

	for _, sub := range subs {
		sub.Cancel()
	}
}

func BenchmarkNotifier_Subscribe(b *testing.B) {
	logger := zerolog.Nop()
	n := New(100, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sub := n.Subscribe()
		sub.Cancel()
	}
}
