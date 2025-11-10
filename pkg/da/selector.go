package da

import (
	"sync/atomic"
)

// AddressSelector defines the interface for selecting a signing address from a list.
type AddressSelector interface {
	// Next returns the next address to use for signing.
	// Implementations may return empty string (NoOpSelector) or panic (RoundRobinSelector with no addresses).
	Next() string
}

// RoundRobinSelector implements round-robin selection of signing addresses.
// This helps prevent sequence mismatches in Cosmos SDK when submitting
// multiple transactions concurrently.
type RoundRobinSelector struct {
	addresses []string
	counter   atomic.Uint64
}

// NewRoundRobinSelector creates a new round-robin address selector.
// Panics if addresses is empty - use NewNoOpSelector instead.
func NewRoundRobinSelector(addresses []string) *RoundRobinSelector {
	if len(addresses) == 0 {
		panic("NewRoundRobinSelector: addresses slice is empty; use NewNoOpSelector instead")
	}
	return &RoundRobinSelector{
		addresses: addresses,
	}
}

// Next returns the next address in round-robin fashion.
// Thread-safe for concurrent access.
// Panics if no addresses are configured - this indicates a programming error.
func (s *RoundRobinSelector) Next() string {
	if len(s.addresses) == 0 {
		panic("RoundRobinSelector.Next: no addresses configured; use NewNoOpSelector instead")
	}

	if len(s.addresses) == 1 {
		return s.addresses[0]
	}

	// Atomically increment and get the previous value for this call
	index := s.counter.Add(1) - 1
	return s.addresses[index%uint64(len(s.addresses))]
}

// NoOpSelector always returns an empty string.
// Used when no signing addresses are configured.
type NoOpSelector struct{}

// NewNoOpSelector creates a selector that returns no address.
func NewNoOpSelector() *NoOpSelector {
	return &NoOpSelector{}
}

// Next returns an empty string.
func (s *NoOpSelector) Next() string {
	return ""
}
