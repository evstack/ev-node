package da

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundRobinSelector_EmptyList(t *testing.T) {
	selector := NewRoundRobinSelector([]string{})

	addr := selector.Next()
	assert.Empty(t, addr, "should return empty string for empty address list")

	// Multiple calls should still return empty
	addr = selector.Next()
	assert.Empty(t, addr)
}

func TestRoundRobinSelector_SingleAddress(t *testing.T) {
	addresses := []string{"celestia1abc123"}
	selector := NewRoundRobinSelector(addresses)

	// All calls should return the same address
	for i := 0; i < 10; i++ {
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
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			for j := 0; j < numCallsPerGoroutine; j++ {
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
	for i := 0; i < 1000; i++ {
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
	for i := 0; i < 10; i++ {
		addr := selector.Next()
		assert.Empty(t, addr, "NoOpSelector should always return empty string")
	}
}

func TestNoOpSelector_Concurrent(t *testing.T) {
	selector := NewNoOpSelector()

	const numGoroutines = 50
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				addr := selector.Next()
				assert.Empty(t, addr)
			}
		}()
	}

	wg.Wait()
}
