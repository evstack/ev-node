package lease

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryLease(t *testing.T) {
	specs := map[string]struct {
		setup    func() *MemoryLease
		actions  func(t *testing.T, lease *MemoryLease)
		expError error
	}{
		"acquire new lease": {
			setup: func() *MemoryLease {
				return NewMemoryLease("test-lease")
			},
			actions: func(t *testing.T, lease *MemoryLease) {
				ctx := context.Background()
				acquired, err := lease.Acquire(ctx, "node1", 10*time.Second)
				require.NoError(t, err)
				assert.True(t, acquired)

				holder, err := lease.GetHolder(ctx)
				require.NoError(t, err)
				assert.Equal(t, "node1", holder)
			},
		},
		"cannot acquire held lease": {
			setup: func() *MemoryLease {
				lease := NewMemoryLease("test-lease")
				ctx := context.Background()
				_, _ = lease.Acquire(ctx, "node1", 10*time.Second)
				return lease
			},
			actions: func(t *testing.T, lease *MemoryLease) {
				ctx := context.Background()
				acquired, err := lease.Acquire(ctx, "node2", 10*time.Second)
				require.NoError(t, err)
				assert.False(t, acquired)

				holder, err := lease.GetHolder(ctx)
				require.NoError(t, err)
				assert.Equal(t, "node1", holder)
			},
		},
		"renew existing lease": {
			setup: func() *MemoryLease {
				lease := NewMemoryLease("test-lease")
				ctx := context.Background()
				_, _ = lease.Acquire(ctx, "node1", 1*time.Second)
				return lease
			},
			actions: func(t *testing.T, lease *MemoryLease) {
				ctx := context.Background()
				err := lease.Renew(ctx, "node1", 10*time.Second)
				require.NoError(t, err)

				expiry, err := lease.GetExpiry(ctx)
				require.NoError(t, err)
				assert.True(t, expiry.After(time.Now().Add(5*time.Second)))
			},
		},
		"cannot renew lease held by another": {
			setup: func() *MemoryLease {
				lease := NewMemoryLease("test-lease")
				ctx := context.Background()
				_, _ = lease.Acquire(ctx, "node1", 10*time.Second)
				return lease
			},
			actions: func(t *testing.T, lease *MemoryLease) {
				ctx := context.Background()
				err := lease.Renew(ctx, "node2", 10*time.Second)
				assert.Equal(t, ErrLeaseNotHeld, err)
			},
		},
		"release lease": {
			setup: func() *MemoryLease {
				lease := NewMemoryLease("test-lease")
				ctx := context.Background()
				_, _ = lease.Acquire(ctx, "node1", 10*time.Second)
				return lease
			},
			actions: func(t *testing.T, lease *MemoryLease) {
				ctx := context.Background()
				err := lease.Release(ctx, "node1")
				require.NoError(t, err)

				holder, err := lease.GetHolder(ctx)
				require.NoError(t, err)
				assert.Empty(t, holder)
			},
		},
		"expired lease can be taken over": {
			setup: func() *MemoryLease {
				lease := NewMemoryLease("test-lease")
				ctx := context.Background()
				_, _ = lease.Acquire(ctx, "node1", 1*time.Millisecond)
				time.Sleep(2 * time.Millisecond) // Wait for expiry
				return lease
			},
			actions: func(t *testing.T, lease *MemoryLease) {
				ctx := context.Background()
				acquired, err := lease.Acquire(ctx, "node2", 10*time.Second)
				require.NoError(t, err)
				assert.True(t, acquired)

				holder, err := lease.GetHolder(ctx)
				require.NoError(t, err)
				assert.Equal(t, "node2", holder)
			},
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			lease := spec.setup()
			spec.actions(t, lease)
		})
	}
}
