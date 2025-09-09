package lease

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeaderElection(t *testing.T) {
	logger := zerolog.Nop()
	
	specs := map[string]struct {
		setup   func() (*LeaderElection, *MemoryLease)
		actions func(t *testing.T, le *LeaderElection, lease *MemoryLease)
	}{
		"single node becomes leader": {
			setup: func() (*LeaderElection, *MemoryLease) {
				lease := NewMemoryLease("test-leader")
				le := NewLeaderElection(lease, "node1", "test-leader", 1*time.Second, logger)
				return le, lease
			},
			actions: func(t *testing.T, le *LeaderElection, lease *MemoryLease) {
				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer cancel()
				
				err := le.Start(ctx)
				require.NoError(t, err)
				defer le.Stop()
				
				// Wait for leadership to be acquired
				select {
				case isLeader := <-le.LeaderChan():
					assert.True(t, isLeader)
				case <-ctx.Done():
					t.Fatal("timeout waiting for leadership")
				}
				
				assert.True(t, le.IsLeader())
				
				leader, err := le.GetLeader(ctx)
				require.NoError(t, err)
				assert.Equal(t, "node1", leader)
			},
		},
		"second node cannot become leader": {
			setup: func() (*LeaderElection, *MemoryLease) {
				lease := NewMemoryLease("test-leader")
				// First node acquires lease
				ctx := context.Background()
				_, _ = lease.Acquire(ctx, "node1", 10*time.Second)
				
				le := NewLeaderElection(lease, "node2", "test-leader", 1*time.Second, logger)
				return le, lease
			},
			actions: func(t *testing.T, le *LeaderElection, lease *MemoryLease) {
				ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
				defer cancel()
				
				err := le.Start(ctx)
				require.NoError(t, err)
				defer le.Stop()
				
				// Should not receive leadership notification quickly
				select {
				case <-le.LeaderChan():
					t.Fatal("should not become leader when lease is held")
				case <-time.After(500 * time.Millisecond):
					// Expected - no leadership acquired
				}
				
				assert.False(t, le.IsLeader())
			},
		},
		"leadership transitions on lease expiry": {
			setup: func() (*LeaderElection, *MemoryLease) {
				lease := NewMemoryLease("test-leader")
				le := NewLeaderElection(lease, "node2", "test-leader", 100*time.Millisecond, logger)
				
				// Simulate expired lease from another node
				lease.leases["test-leader"] = &LeaseInfo{
					Holder:   "node1",
					Expiry:   time.Now().Add(-1 * time.Second), // Already expired
					Acquired: time.Now().Add(-2 * time.Second),
					Version:  1,
				}
				
				return le, lease
			},
			actions: func(t *testing.T, le *LeaderElection, lease *MemoryLease) {
				ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
				defer cancel()
				
				err := le.Start(ctx)
				require.NoError(t, err)
				defer le.Stop()
				
				// Should acquire leadership since previous lease expired
				select {
				case isLeader := <-le.LeaderChan():
					assert.True(t, isLeader)
				case <-ctx.Done():
					t.Fatal("timeout waiting for leadership after expiry")
				}
				
				assert.True(t, le.IsLeader())
			},
		},
		"stop releases lease": {
			setup: func() (*LeaderElection, *MemoryLease) {
				lease := NewMemoryLease("test-leader")
				le := NewLeaderElection(lease, "node1", "test-leader", 1*time.Second, logger)
				return le, lease
			},
			actions: func(t *testing.T, le *LeaderElection, lease *MemoryLease) {
				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer cancel()
				
				err := le.Start(ctx)
				require.NoError(t, err)
				
				// Wait for leadership
				select {
				case <-le.LeaderChan():
					// Leadership acquired
				case <-ctx.Done():
					t.Fatal("timeout waiting for leadership")
				}
				
				assert.True(t, le.IsLeader())
				
				// Stop should release lease
				err = le.Stop()
				require.NoError(t, err)
				
				// Verify lease is released
				holder, err := lease.GetHolder(ctx)
				require.NoError(t, err)
				assert.Empty(t, holder)
			},
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			le, lease := spec.setup()
			spec.actions(t, le, lease)
		})
	}
}