package lease

import (
	"context"
	"sync"
	"time"
)

// MemoryLease provides an in-memory implementation of the Lease interface
// This is primarily for testing and single-process scenarios
type MemoryLease struct {
	mu       sync.RWMutex
	leases   map[string]*LeaseInfo
	name     string
}

// NewMemoryLease creates a new memory-based lease
func NewMemoryLease(name string) *MemoryLease {
	return &MemoryLease{
		leases: make(map[string]*LeaseInfo),
		name:   name,
	}
}

// Acquire attempts to acquire the lease for the specified duration
func (ml *MemoryLease) Acquire(ctx context.Context, nodeID string, duration time.Duration) (bool, error) {
	if duration <= 0 {
		return false, ErrInvalidLeaseTerm
	}

	ml.mu.Lock()
	defer ml.mu.Unlock()

	now := time.Now()
	lease, exists := ml.leases[ml.name]

	// Check if lease exists and is still valid
	if exists && now.Before(lease.Expiry) {
		if lease.Holder == nodeID {
			// Already held by this node, extend it
			lease.Expiry = now.Add(duration)
			lease.Version++
			return true, nil
		}
		// Held by another node
		return false, nil
	}

	// Acquire new lease or take over expired lease
	ml.leases[ml.name] = &LeaseInfo{
		Holder:   nodeID,
		Expiry:   now.Add(duration),
		Acquired: now,
		Version:  1,
	}
	if exists {
		ml.leases[ml.name].Version = lease.Version + 1
	}

	return true, nil
}

// Renew extends the lease duration if currently held by this node
func (ml *MemoryLease) Renew(ctx context.Context, nodeID string, duration time.Duration) error {
	if duration <= 0 {
		return ErrInvalidLeaseTerm
	}

	ml.mu.Lock()
	defer ml.mu.Unlock()

	lease, exists := ml.leases[ml.name]
	if !exists {
		return ErrLeaseNotHeld
	}

	now := time.Now()
	if now.After(lease.Expiry) {
		return ErrLeaseExpired
	}

	if lease.Holder != nodeID {
		return ErrLeaseNotHeld
	}

	lease.Expiry = now.Add(duration)
	lease.Version++
	
	return nil
}

// Release releases the lease if held by this node
func (ml *MemoryLease) Release(ctx context.Context, nodeID string) error {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	lease, exists := ml.leases[ml.name]
	if !exists {
		return ErrLeaseNotHeld
	}

	if lease.Holder != nodeID {
		return ErrLeaseNotHeld
	}

	// Mark as expired to release
	lease.Expiry = time.Now().Add(-1 * time.Second)
	lease.Version++

	return nil
}

// IsHeld checks if the lease is currently held by the specified node
func (ml *MemoryLease) IsHeld(ctx context.Context, nodeID string) (bool, error) {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	lease, exists := ml.leases[ml.name]
	if !exists {
		return false, nil
	}

	now := time.Now()
	return lease.Holder == nodeID && now.Before(lease.Expiry), nil
}

// GetHolder returns the current lease holder, if any
func (ml *MemoryLease) GetHolder(ctx context.Context) (string, error) {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	lease, exists := ml.leases[ml.name]
	if !exists {
		return "", nil
	}

	now := time.Now()
	if now.After(lease.Expiry) {
		return "", nil
	}

	return lease.Holder, nil
}

// GetExpiry returns when the current lease expires
func (ml *MemoryLease) GetExpiry(ctx context.Context) (time.Time, error) {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	lease, exists := ml.leases[ml.name]
	if !exists {
		return time.Time{}, nil
	}

	return lease.Expiry, nil
}