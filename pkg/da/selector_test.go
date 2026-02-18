package da

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundRobinSelector_EmptyList(t *testing.T) {
	// Should panic when creating selector with empty address list
	assert.Panics(t, func() {
		NewRoundRobinSelector([]string{})
	}, "should panic when creating RoundRobinSelector with empty address list")
}

func TestRoundRobinSelector_NextWithoutAddresses(t *testing.T) {
	// Should panic if Next is called on a selector with no addresses
	// (e.g., if someone creates the struct directly without using the constructor)
	selector := &RoundRobinSelector{addresses: []string{}}
	assert.Panics(t, func() {
		selector.Next()
	}, "should panic when calling Next with no addresses configured")
}

func TestRoundRobinSelector_SingleAddress(t *testing.T) {
	addresses := []string{"celestia1abc123"}
	selector := NewRoundRobinSelector(addresses)

	// All calls should return the same address
	for range 10 {
		addr := selector.Next()
		assert.Equal(t, "celestia1abc123", addr, "should always return the single address")
	}
}

func TestRoundRobinSelector_MultipleAddresses(t *testing.T) {
	addresses := []string{
		"celestia1abc123",
		"celestia1def456",
		"celestia1ghi789",
	}
	selector := NewRoundRobinSelector(addresses)

	// First round
	assert.Equal(t, "celestia1abc123", selector.Next())
	assert.Equal(t, "celestia1def456", selector.Next())
	assert.Equal(t, "celestia1ghi789", selector.Next())

	// Second round - should cycle back
	assert.Equal(t, "celestia1abc123", selector.Next())
	assert.Equal(t, "celestia1def456", selector.Next())
	assert.Equal(t, "celestia1ghi789", selector.Next())
}

func TestRoundRobinSelector_Concurrent(t *testing.T) {
	addresses := []string{
		"celestia1abc123",
		"celestia1def456",
		"celestia1ghi789",
	}
	selector := NewRoundRobinSelector(addresses)

	const numGoroutines = 100
	const numCallsPerGoroutine = 100

	results := make([]string, numGoroutines*numCallsPerGoroutine)
	var wg sync.WaitGroup

	// Launch concurrent goroutines
	for i := range numGoroutines {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			for j := range numCallsPerGoroutine {
				addr := selector.Next()
				results[start+j] = addr
			}
		}(i * numCallsPerGoroutine)
	}

	wg.Wait()

	// Verify all results are valid addresses
	for _, addr := range results {
		require.Contains(t, addresses, addr, "all returned addresses should be from the configured list")
	}

	// Count occurrences of each address
	counts := make(map[string]int)
	for _, addr := range results {
		counts[addr]++
	}

	// Each address should be used approximately equally (within 10% tolerance)
	expectedCount := len(results) / len(addresses)
	tolerance := expectedCount / 10

	for _, addr := range addresses {
		count := counts[addr]
		assert.InDelta(t, expectedCount, count, float64(tolerance),
			"address %s should be used approximately %d times, got %d", addr, expectedCount, count)
	}
}

func TestRoundRobinSelector_WrapAround(t *testing.T) {
	addresses := []string{"addr1", "addr2"}
	selector := NewRoundRobinSelector(addresses)

	// Test wrap around behavior with large number of calls
	seen := make(map[string]int)
	for range 1000 {
		addr := selector.Next()
		seen[addr]++
	}

	// Both addresses should be used 500 times each
	assert.Equal(t, 500, seen["addr1"])
	assert.Equal(t, 500, seen["addr2"])
}

func TestNoOpSelector(t *testing.T) {
	selector := NewNoOpSelector()

	// Should always return empty string
	for range 10 {
		addr := selector.Next()
		assert.Empty(t, addr, "NoOpSelector should always return empty string")
	}
}

func TestNoOpSelector_Concurrent(t *testing.T) {
	selector := NewNoOpSelector()

	const numGoroutines = 50
	var wg sync.WaitGroup

	for range numGoroutines {
		wg.Go(func() {
			for range 100 {
				addr := selector.Next()
				assert.Empty(t, addr)
			}
		})
	}

	wg.Wait()
}
