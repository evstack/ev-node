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
				defer le.Stop() // nolint: errcheck

				// Wait for leadership to be acquired
				select {
				case isLeader := <-le.LeaderChan():
					assert.True(t, isLeader)
				case <-ctx.Done():
					t.Fatal("timeout waiting for leadership")
				}

				isLeader, err := le.IsLeader()
				require.NoError(t, err)
				assert.True(t, isLeader)

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
				ok, err := lease.Acquire(ctx, "node1", 10*time.Second)
				require.NoError(t, err)
				require.True(t, ok)

				le := NewLeaderElection(lease, "node2", "test-leader", 1*time.Second, logger)
				return le, lease
			},
			actions: func(t *testing.T, le *LeaderElection, lease *MemoryLease) {
				ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
				defer cancel()

				err := le.Start(ctx)
				require.NoError(t, err)
				defer le.Stop() // nolint: errcheck

				// Should not receive leadership notification quickly
			Outer:
				for {
					select {
					case v := <-le.LeaderChan():
						require.False(t, v, "should not become leader when lease is held")
					case <-time.After(500 * time.Millisecond):
						// Expected - no leadership acquired
						break Outer
					}
				}

				isLeader, err := le.IsLeader()
				require.NoError(t, err)
				assert.False(t, isLeader)
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
				defer le.Stop() // nolint: errcheck

				// Should acquire leadership since previous lease expired
				select {
				case isLeader := <-le.LeaderChan():
					assert.True(t, isLeader)
				case <-ctx.Done():
					t.Fatal("timeout waiting for leadership after expiry")
				}

				isLeader, err := le.IsLeader()
				require.NoError(t, err)
				assert.True(t, isLeader)
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

				isLeader, err := le.IsLeader()
				require.NoError(t, err)
				assert.True(t, isLeader)

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

func TestSwitchAsLeader(t *testing.T) {
	logger := zerolog.Nop()

	specs := map[string]struct {
		setup   func() (*LeaderElection, *MemoryLease)
		actions func(t *testing.T, le *LeaderElection, lease *MemoryLease)
	}{
		"follower state executes follower function": {
			setup: func() (*LeaderElection, *MemoryLease) {
				lease := NewMemoryLease("test-leader")
				// Set up another node as leader first
				ctx := context.Background()
				ok, err := lease.Acquire(ctx, "other-node", 10*time.Second)
				require.NoError(t, err)
				require.True(t, ok)

				le := NewLeaderElection(lease, "node1", "test-leader", 100*time.Millisecond, logger)
				return le, lease
			},
			actions: func(t *testing.T, le *LeaderElection, lease *MemoryLease) {
				ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
				defer cancel()

				err := le.Start(ctx)
				require.NoError(t, err)
				defer le.Stop() // nolint: errcheck

				followerExecuted := make(chan struct{}, 1)
				leaderExecuted := make(chan struct{}, 1)

				followerFunc := func(ctx context.Context) {
					select {
					case followerExecuted <- struct{}{}:
					default:
					}
					// Keep running until cancelled
					<-ctx.Done()
				}

				leaderFunc := func(leaderCtx context.Context) {
					select {
					case leaderExecuted <- struct{}{}:
					default:
					}
					<-leaderCtx.Done()
				}

				// Start SwitchAsLeader BEFORE waiting for states - it needs to catch the state changes
				switchCtx, switchCancel := context.WithTimeout(ctx, 2*time.Second)
				defer switchCancel()

				go func() {
					_ = le.SwitchAsLeader(switchCtx, leaderFunc, followerFunc)
				}()

				// Verify follower function was called
				select {
				case <-followerExecuted:
					// Expected
				case <-time.After(1 * time.Second):
					t.Error("follower function should have been executed")
				}

				// Verify leader function was not called
				select {
				case <-leaderExecuted:
					t.Error("leader function should not have been executed in follower state")
				case <-time.After(100 * time.Millisecond):
					// Expected - leader function not called
				}
			},
		},
		"becomes leader transitions correctly": {
			setup: func() (*LeaderElection, *MemoryLease) {
				lease := NewMemoryLease("test-leader")
				le := NewLeaderElection(lease, "node1", "test-leader", 100*time.Millisecond, logger)
				return le, lease
			},
			actions: func(t *testing.T, le *LeaderElection, lease *MemoryLease) {
				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer cancel()

				err := le.Start(ctx)
				require.NoError(t, err)
				defer le.Stop() // nolint: errcheck

				leaderExecuted := make(chan struct{}, 1)
				followerExecuted := make(chan struct{}, 1)

				followerFunc := func(ctx context.Context) {
					select {
					case followerExecuted <- struct{}{}:
					default:
					}
					<-ctx.Done()
				}

				leaderFunc := func(leaderCtx context.Context) {
					select {
					case leaderExecuted <- struct{}{}:
					default:
					}
					// Keep running until context is cancelled
					<-leaderCtx.Done()
				}

				// Start SwitchAsLeader BEFORE waiting for states - it needs to catch the state changes
				switchCtx, switchCancel := context.WithTimeout(ctx, 3*time.Second)
				defer switchCancel()

				go func() {
					_ = le.SwitchAsLeader(switchCtx, leaderFunc, followerFunc)
				}()

				// Verify leader function was called
				select {
				case <-leaderExecuted:
					// Expected
				case <-time.After(2 * time.Second):
					t.Error("leader function should have been executed after becoming leader")
				}

				// Verify follower function was not called (since we became leader)
				select {
				case <-followerExecuted:
					t.Error("follower function should not have been executed in leader state")
				case <-time.After(100 * time.Millisecond):
					// Expected - follower function not called
				}
			},
		},
		"loses leadership returns error": {
			setup: func() (*LeaderElection, *MemoryLease) {
				lease := NewMemoryLease("test-leader")
				le := NewLeaderElection(lease, "node1", "test-leader", 50*time.Millisecond, logger)
				return le, lease
			},
			actions: func(t *testing.T, le *LeaderElection, lease *MemoryLease) {
				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer cancel()

				err := le.Start(ctx)
				require.NoError(t, err)
				defer le.Stop() // nolint: errcheck

				leaderStarted := make(chan struct{}, 1)
				leaderFunc := func(leaderCtx context.Context) {
					select {
					case leaderStarted <- struct{}{}:
					default:
					}
					// Keep running until context is cancelled
					<-leaderCtx.Done()
				}

				followerFunc := func(ctx context.Context) {
					<-ctx.Done()
				}

				// Start SwitchAsLeader BEFORE waiting for states
				switchDone := make(chan error, 1)
				go func() {
					switchDone <- le.SwitchAsLeader(ctx, leaderFunc, followerFunc)
				}()

				// Wait for leader function to start (meaning we became leader)
				select {
				case <-leaderStarted:
					// Good, leader function started
				case <-time.After(1 * time.Second):
					t.Fatal("timeout waiting for leader function to start")
				}

				// Simulate losing leadership by forcing lease renewal failure
				// Release current lease and let another node acquire it
				require.EventuallyWithT(t, func(c *assert.CollectT) {
					require.NoError(c, lease.Release(ctx, "node1"))
				}, 10*time.Second, 100*time.Millisecond, "lease should be renewed")

				ctxAcq := t.Context()
				_, err = lease.Acquire(ctxAcq, "other-node", 10*time.Second)
				require.NoError(t, err)

				// Wait for the leadership loss to be detected and propagated
				select {
				case err := <-switchDone:
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "leader lock lost")
				case <-time.After(2 * time.Second):
					t.Error("should have detected leadership loss")
				}
			},
		},
		"context cancellation returns context error": {
			setup: func() (*LeaderElection, *MemoryLease) {
				lease := NewMemoryLease("test-leader")
				le := NewLeaderElection(lease, "node1", "test-leader", 1*time.Second, logger)
				return le, lease
			},
			actions: func(t *testing.T, le *LeaderElection, lease *MemoryLease) {
				ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
				defer cancel()

				err := le.Start(ctx)
				require.NoError(t, err)
				defer le.Stop() // nolint: errcheck

				leaderFunc := func(leaderCtx context.Context) {
					<-leaderCtx.Done()
				}

				followerFunc := func(ctx context.Context) {
					<-ctx.Done()
				}

				// Cancel context quickly
				switchCtx, switchCancel := context.WithTimeout(ctx, 50*time.Millisecond)
				defer switchCancel()

				err = le.SwitchAsLeader(switchCtx, leaderFunc, followerFunc)
				assert.Error(t, err)
				assert.Equal(t, context.DeadlineExceeded, err)
			},
		},
		"leader function exit returns error": {
			setup: func() (*LeaderElection, *MemoryLease) {
				lease := NewMemoryLease("test-leader")
				le := NewLeaderElection(lease, "node1", "test-leader", 100*time.Millisecond, logger)
				return le, lease
			},
			actions: func(t *testing.T, le *LeaderElection, lease *MemoryLease) {
				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer cancel()

				err := le.Start(ctx)
				require.NoError(t, err)
				defer le.Stop() // nolint: errcheck

				leaderStarted := make(chan struct{}, 1)
				leaderFunc := func(leaderCtx context.Context) {
					select {
					case leaderStarted <- struct{}{}:
					default:
					}
				}

				followerFunc := func(ctx context.Context) {
					<-ctx.Done()
				}

				// Start SwitchAsLeader BEFORE waiting for states
				switchDone := make(chan error, 1)
				go func() {
					switchDone <- le.SwitchAsLeader(ctx, leaderFunc, followerFunc)
				}()

				// Wait for leader function to start
				select {
				case <-leaderStarted:
					// Good, leader function started
				case <-time.After(1 * time.Second):
					t.Fatal("timeout waiting for leader function to start")
				}

				// Wait for the leadership loss to be detected and propagated
				select {
				case err := <-switchDone:
					require.Error(t, err)
					assert.Contains(t, err.Error(), "leader worker exited unexpectedly")
				case <-time.After(2 * time.Second):
					t.Error("should have stopped")
				}
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
