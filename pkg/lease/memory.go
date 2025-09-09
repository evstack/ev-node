package lease

import (
	"context"
	"sync"
	"time"
)

// MemoryLease provides an in-memory implementation of the Lease interface
// This is primarily for testing and single-process scenarios
type MemoryLease struct {
	name string

	mu    sync.RWMutex
	lease *LeaseInfo
}

// NewMemoryLease creates a new memory-based lease
func NewMemoryLease(name string) *MemoryLease {
	return &MemoryLease{
		name: name,
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
	if ml.lease != nil && now.Before(ml.lease.Expiry) {
		if ml.lease.Holder == nodeID {
			// Already held by this node, extend it
			ml.lease.Expiry = now.Add(duration)
			ml.lease.Version++
			return true, nil
		}
		// Held by another node
		return false, nil
	}

	// Acquire new lease or take over expired lease
	version := uint64(1)
	if ml.lease != nil {
		version = ml.lease.Version + 1
	}
	ml.lease = &LeaseInfo{
		Holder:   nodeID,
		Expiry:   now.Add(duration),
		Acquired: now,
		Version:  version,
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

	if ml.lease == nil || ml.lease.Holder != nodeID {
		return ErrLeaseNotHeld
	}

	now := time.Now()
	if now.After(ml.lease.Expiry) {
		return ErrLeaseExpired
	}

	ml.lease.Expiry = now.Add(duration)
	ml.lease.Version++

	return nil
}

// Release releases the lease if held by this node
func (ml *MemoryLease) Release(ctx context.Context, nodeID string) error {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	if ml.lease == nil || ml.lease.Holder != nodeID {
		return ErrLeaseNotHeld
	}

	// Mark as expired to release
	ml.lease.Expiry = time.Now().Add(-1 * time.Second)
	ml.lease.Version++

	return nil
}

// IsHeld checks if the lease is currently held by the specified node
func (ml *MemoryLease) IsHeld(ctx context.Context, nodeID string) (bool, error) {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	if ml.lease == nil {
		return false, nil
	}

	now := time.Now()
	return ml.lease.Holder == nodeID && now.Before(ml.lease.Expiry), nil
}

// GetHolder returns the current lease holder, if any
func (ml *MemoryLease) GetHolder(ctx context.Context) (string, error) {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	if ml.lease == nil {
		return "", nil
	}

	now := time.Now()
	if now.After(ml.lease.Expiry) {
		return "", nil
	}

	return ml.lease.Holder, nil
}

// GetExpiry returns when the current lease expires
func (ml *MemoryLease) GetExpiry(ctx context.Context) (time.Time, error) {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	if ml.lease == nil {
		return time.Time{}, nil
	}

	return ml.lease.Expiry, nil
}

func (ml MemoryLease) Name() string {
	return ml.name
}
