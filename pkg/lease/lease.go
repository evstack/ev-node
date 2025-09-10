package lease

import (
	"context"
	"errors"
	"time"
)

var (
	ErrLeaseNotHeld     = errors.New("lease not held")
	ErrLeaseAlreadyHeld = errors.New("lease already held by another node")
	ErrLeaseExpired     = errors.New("lease expired")
	ErrInvalidLeaseTerm = errors.New("invalid lease term")
)

// Lease represents a distributed lock/lease that can be acquired by one node at a time
type Lease interface {
	// Acquire attempts to acquire the lease for the specified duration
	// Returns true if acquired successfully, false if held by another node
	Acquire(ctx context.Context, nodeID string, duration time.Duration) (bool, error)

	// Renew extends the lease duration if currently held by this node
	Renew(ctx context.Context, nodeID string, duration time.Duration) error

	// Release releases the lease if held by this node
	Release(ctx context.Context, nodeID string) error

	// IsHeld checks if the lease is currently held by the specified node
	IsHeld(ctx context.Context, nodeID string) (bool, error)

	// GetHolder returns the current lease holder, if any
	GetHolder(ctx context.Context) (string, error)

	// GetExpiry returns when the current lease expires
	GetExpiry(ctx context.Context) (time.Time, error)
}

// LeaseInfo contains information about a lease
type LeaseInfo struct {
	Holder   string    `json:"holder"`
	Expiry   time.Time `json:"expiry"`
	Acquired time.Time `json:"acquired"`
	Version  int64     `json:"version"`
}
