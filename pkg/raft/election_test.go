package raft

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDynamicLeaderElectionRun(t *testing.T) {
	specs := map[string]struct {
		setup   func(t *testing.T) (*DynamicLeaderElection, context.Context, context.CancelFunc)
		assertF func(t *testing.T, err error)
	}{
		"start as follower via ticker": {
			setup: func(t *testing.T) (*DynamicLeaderElection, context.Context, context.CancelFunc) {
				m := newMocksourceNode(t)
				leaderCh := make(chan bool, 1)
				m.EXPECT().leaderCh().Return((<-chan bool)(leaderCh))
				m.EXPECT().leaderID().Return("other")
				m.EXPECT().NodeID().Return("self")
				m.EXPECT().Config().Return(testCfg())
				m.EXPECT().waitForMsgsLanded(2 * time.Millisecond).Return(nil)
				m.EXPECT().GetState().Return(&RaftBlockState{})

				started := make(chan struct{})
				follower := &testRunnable{startedCh: started}
				leader := &testRunnable{}

				logger := zerolog.Nop()
				d := &DynamicLeaderElection{logger: logger, node: m,
					leaderFactory:   func() (Runnable, error) { return leader, nil },
					followerFactory: func() (Runnable, error) { return follower, nil },
				}
				ctx, cancel := context.WithCancel(t.Context())

				// Wait for follower to start via ticker path
				var once sync.Once
				go func() {
					<-started
					once.Do(cancel)
				}()
				return d, ctx, func() { once.Do(cancel) }
			},
			assertF: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, context.Canceled)
			},
		},
		"leader loss returns": {
			setup: func(t *testing.T) (*DynamicLeaderElection, context.Context, context.CancelFunc) {
				m := newMocksourceNode(t)
				leaderCh := make(chan bool, 2)
				m.EXPECT().leaderCh().Return((<-chan bool)(leaderCh))
				m.EXPECT().Config().Return(testCfg())
				m.EXPECT().waitForMsgsLanded(2 * time.Millisecond).Return(nil)

				leader := &testRunnable{runFn: func(ctx context.Context) error {
					// block until canceled by election
					<-ctx.Done()
					return ctx.Err()
				}}
				logger := zerolog.Nop()
				d := &DynamicLeaderElection{logger: logger, node: m,
					leaderFactory:   func() (Runnable, error) { return leader, nil },
					followerFactory: func() (Runnable, error) { return &testRunnable{}, nil },
				}

				ctx, cancel := context.WithCancel(t.Context())
				// Signal become leader then loss
				go func() {
					leaderCh <- true
					time.Sleep(2 * time.Millisecond)
					leaderCh <- false
				}()
				return d, ctx, cancel
			},
			assertF: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrLeadershipLost)
			},
		},
		"worker error surfaces": {
			setup: func(t *testing.T) (*DynamicLeaderElection, context.Context, context.CancelFunc) {
				m := newMocksourceNode(t)
				leaderCh := make(chan bool, 1)
				m.EXPECT().leaderCh().Return((<-chan bool)(leaderCh))
				m.EXPECT().Config().Return(testCfg())
				m.EXPECT().waitForMsgsLanded(2 * time.Millisecond).Return(nil)
				m.EXPECT().leadershipTransfer().Return(nil)

				wantErr := errors.New("boom")
				leader := &testRunnable{runFn: func(ctx context.Context) error { return wantErr }}
				logger := zerolog.Nop()
				d := &DynamicLeaderElection{logger: logger, node: m,
					leaderFactory:   func() (Runnable, error) { return leader, nil },
					followerFactory: func() (Runnable, error) { return &testRunnable{}, nil },
				}
				ctx, cancel := context.WithCancel(t.Context())
				go func() { leaderCh <- true }()
				return d, ctx, cancel
			},
			assertF: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.True(t, strings.Contains(err.Error(), "leader worker exited unexpectedly: boom"), err.Error())
			},
		},
		"leadership transfer when not synced": {
			setup: func(t *testing.T) (*DynamicLeaderElection, context.Context, context.CancelFunc) {
				m := newMocksourceNode(t)
				leaderCh := make(chan bool, 2)
				m.EXPECT().leaderCh().Return((<-chan bool)(leaderCh))
				// GetState is called during verifyState when starting as follower
				m.EXPECT().GetState().Return(&RaftBlockState{Height: 1})
				m.EXPECT().Config().Return(testCfg())
				m.EXPECT().waitForMsgsLanded(2 * time.Millisecond).Return(nil)
				m.EXPECT().NodeID().Return("self")
				m.EXPECT().leaderID().Return("self")
				// GetState is called again when transitioning follower->leader
				m.EXPECT().GetState().Return(&RaftBlockState{Height: 1})
				// When IsSynced returns -1 (local behind raft), recovery is called but no leadershipTransfer

				fStarted := make(chan struct{})
				lStarted := make(chan struct{})
				follower := &nonRecoverableRunnable{r: &testRunnable{startedCh: fStarted, isSyncedFn: func(*RaftBlockState) (int, error) { return -1, nil }}}
				// When diff=-1 (local behind), recovery is called but succeeds, then leader starts
				leader := &testRunnable{startedCh: lStarted}

				logger := zerolog.Nop()
				d := &DynamicLeaderElection{logger: logger, node: m,
					leaderFactory:   func() (Runnable, error) { return leader, nil },
					followerFactory: func() (Runnable, error) { return follower, nil },
				}
				ctx, cancel := context.WithCancel(t.Context())
				// Start as follower via false, then signal leader true
				go func() {
					leaderCh <- false
					<-fStarted // ensure follower started
					leaderCh <- true
					// Wait for leader to start
					select {
					case <-lStarted:
					case <-time.After(50 * time.Millisecond):
						t.Log("timed out waiting for leader to start")
					}
					cancel()
				}()
				return d, ctx, cancel
			},
			assertF: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, context.Canceled)
			},
		},
		"single node not synced wraps recovery": {
			setup: func(t *testing.T) (*DynamicLeaderElection, context.Context, context.CancelFunc) {
				m := newMocksourceNode(t)
				leaderCh := make(chan bool, 2)
				m.EXPECT().leaderCh().Return((<-chan bool)(leaderCh))
				// GetState is called during verifyState when starting as follower
				m.EXPECT().GetState().Return(&RaftBlockState{Height: 1})
				m.EXPECT().Config().Return(testCfg())
				m.EXPECT().waitForMsgsLanded(2 * time.Millisecond).Return(nil)
				m.EXPECT().NodeID().Return("self")
				m.EXPECT().leaderID().Return("self")
				// GetState is called again when transitioning follower->leader
				m.EXPECT().GetState().Return(&RaftBlockState{Height: 1})

				fStarted := make(chan struct{})
				lStarted := make(chan struct{})
				recoverCalled := make(chan struct{})

				follower := &testRunnable{
					startedCh:  fStarted,
					isSyncedFn: func(*RaftBlockState) (int, error) { return -1, nil },
					recoverFn: func(ctx context.Context, state *RaftBlockState) error {
						close(recoverCalled)
						return nil
					},
				}
				leader := &testRunnable{startedCh: lStarted}

				logger := zerolog.Nop()
				d := &DynamicLeaderElection{logger: logger, node: m,
					leaderFactory:   func() (Runnable, error) { return leader, nil },
					followerFactory: func() (Runnable, error) { return follower, nil },
				}
				ctx, cancel := context.WithCancel(t.Context())
				// Start as follower via false, then signal leader true
				go func() {
					leaderCh <- false
					<-fStarted // ensure follower started
					leaderCh <- true

					// Wait for recovery to be called
					select {
					case <-recoverCalled:
					case <-time.After(50 * time.Millisecond):
						t.Error("timed out waiting for recover")
					}

					// Wait for leader to start
					select {
					case <-lStarted:
					case <-time.After(50 * time.Millisecond):
						t.Error("timed out waiting for leader start")
					}

					cancel()
				}()
				return d, ctx, cancel
			},
			assertF: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, context.Canceled)
			},
		},
		"abdicate when store significantly behind raft and never catches up": {
			setup: func(t *testing.T) (*DynamicLeaderElection, context.Context, context.CancelFunc) {
				m := newMocksourceNode(t)
				leaderCh := make(chan bool, 2)
				m.EXPECT().leaderCh().Return((<-chan bool)(leaderCh))
				// GetState is called in verifyState, leader sync check, and the wait loop.
				m.EXPECT().GetState().Return(&RaftBlockState{Height: 10}).Maybe()
				// Config: once for follower waitForMsgsLanded, once for leader waitForMsgsLanded,
				// once inside waitForBlockStoreSync.
				m.EXPECT().Config().Return(testCfg()).Times(3)
				m.EXPECT().waitForMsgsLanded(2 * time.Millisecond).Return(nil)
				// NodeID + leaderID: called in leader-sync check and polled inside the wait loop.
				m.EXPECT().NodeID().Return("self").Maybe()
				m.EXPECT().leaderID().Return("self").Maybe()
				// Abdication must transfer leadership after wait times out.
				m.EXPECT().leadershipTransfer().Return(nil)

				fStarted := make(chan struct{})
				follower := &testRunnable{
					startedCh:  fStarted,
					isSyncedFn: func(*RaftBlockState) (int, error) { return -5, nil },
				}
				// Signal if leader ever starts — it must not.
				leaderStarted := make(chan struct{}, 1)
				leader := &testRunnable{runFn: func(ctx context.Context) error {
					select {
					case leaderStarted <- struct{}{}:
					default:
					}
					<-ctx.Done()
					return ctx.Err()
				}}

				logger := zerolog.Nop()
				d := &DynamicLeaderElection{logger: logger, node: m,
					leaderFactory:   func() (Runnable, error) { return leader, nil },
					followerFactory: func() (Runnable, error) { return follower, nil },
				}
				ctx, cancel := context.WithCancel(t.Context())
				go func() {
					leaderCh <- false
					<-fStarted
					leaderCh <- true
					// Wait long enough for the sync wait to time out and abdication to
					// complete, then verify the leader was never started.
					select {
					case <-leaderStarted:
						t.Error("leader should not start when store is significantly behind raft")
					case <-time.After(200 * time.Millisecond):
						// leadership transferred without starting leader — expected
					}
					cancel()
				}()
				return d, ctx, cancel
			},
			assertF: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, context.Canceled)
			},
		},
		"proceed as leader when store catches up during wait": {
			// Simulates the hot-potato scenario: all nodes behind on election,
			// but the winner's block store syncs up before the wait times out.
			setup: func(t *testing.T) (*DynamicLeaderElection, context.Context, context.CancelFunc) {
				m := newMocksourceNode(t)
				leaderCh := make(chan bool, 2)
				m.EXPECT().leaderCh().Return((<-chan bool)(leaderCh))
				m.EXPECT().GetState().Return(&RaftBlockState{Height: 10}).Maybe()
				m.EXPECT().Config().Return(testCfg()).Maybe()
				m.EXPECT().waitForMsgsLanded(2 * time.Millisecond).Return(nil)
				m.EXPECT().NodeID().Return("self").Maybe()
				m.EXPECT().leaderID().Return("self").Maybe()
				// No leadershipTransfer: the node should stay as leader.

				fStarted := make(chan struct{})
				var syncedCalls int
				follower := &testRunnable{
					startedCh: fStarted,
					isSyncedFn: func(*RaftBlockState) (int, error) {
						syncedCalls++
						if syncedCalls < 3 {
							return -5, nil // still catching up
						}
						return 0, nil // caught up
					},
				}
				leaderStarted := make(chan struct{})
				leader := &testRunnable{
					startedCh: leaderStarted,
					runFn: func(ctx context.Context) error {
						<-ctx.Done()
						return ctx.Err()
					},
				}

				logger := zerolog.Nop()
				d := &DynamicLeaderElection{logger: logger, node: m,
					leaderFactory:   func() (Runnable, error) { return leader, nil },
					followerFactory: func() (Runnable, error) { return follower, nil },
				}
				ctx, cancel := context.WithCancel(t.Context())
				go func() {
					leaderCh <- false
					<-fStarted
					leaderCh <- true
					// The leader must start once the store catches up.
					select {
					case <-leaderStarted:
						// expected: leader started after store synced
					case <-time.After(200 * time.Millisecond):
						t.Error("leader should have started once store caught up")
					}
					cancel()
				}()
				return d, ctx, cancel
			},
			assertF: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, context.Canceled)
			},
		},
		"lost leadership during sync wait": {
			setup: func(t *testing.T) (*DynamicLeaderElection, context.Context, context.CancelFunc) {
				m := newMocksourceNode(t)
				leaderCh := make(chan bool, 2)
				m.EXPECT().leaderCh().Return((<-chan bool)(leaderCh))
				// GetState is called during verifyState when starting as follower
				m.EXPECT().GetState().Return(&RaftBlockState{Height: 1})
				m.EXPECT().Config().Return(testCfg())
				m.EXPECT().waitForMsgsLanded(2 * time.Millisecond).Return(nil)
				m.EXPECT().NodeID().Return("self")
				m.EXPECT().leaderID().Return("other") // Simulate leadership lost

				fStarted := make(chan struct{})
				follower := &testRunnable{startedCh: fStarted}
				// Leader should not be started
				leader := &testRunnable{runFn: func(ctx context.Context) error {
					t.Fatal("leader should not be running")
					return nil
				}}

				logger := zerolog.Nop()
				d := &DynamicLeaderElection{logger: logger, node: m,
					leaderFactory:   func() (Runnable, error) { return leader, nil },
					followerFactory: func() (Runnable, error) { return follower, nil },
				}
				ctx, cancel := context.WithCancel(t.Context())
				go func() {
					leaderCh <- false
					<-fStarted
					leaderCh <- true
					time.Sleep(3 * time.Millisecond)
					cancel()
				}()
				return d, ctx, cancel
			},
			assertF: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, context.Canceled)
			},
		},
		"follower starts then becomes leader": {
			setup: func(t *testing.T) (*DynamicLeaderElection, context.Context, context.CancelFunc) {
				m := newMocksourceNode(t)
				leaderCh := make(chan bool, 2)
				m.EXPECT().leaderCh().Return((<-chan bool)(leaderCh))
				// GetState is called during verifyState when starting as follower
				m.EXPECT().GetState().Return(&RaftBlockState{Height: 1})
				// On leadership change to true, election will sleep SendTimeout, then check sync against state
				m.EXPECT().Config().Return(testCfg())
				m.EXPECT().waitForMsgsLanded(2 * time.Millisecond).Return(nil)
				m.EXPECT().NodeID().Return("self")
				m.EXPECT().leaderID().Return("self")
				// GetState is called again when transitioning follower->leader
				m.EXPECT().GetState().Return(&RaftBlockState{Height: 1})

				fStarted := make(chan struct{})
				lStarted := make(chan struct{})
				follower := &testRunnable{startedCh: fStarted}
				leader := &testRunnable{startedCh: lStarted}

				logger := zerolog.Nop()
				d := &DynamicLeaderElection{logger: logger, node: m,
					leaderFactory:   func() (Runnable, error) { return leader, nil },
					followerFactory: func() (Runnable, error) { return follower, nil },
				}
				ctx, cancel := context.WithCancel(t.Context())
				go func() {
					// Start as follower first
					leaderCh <- false
					<-fStarted // ensure follower started
					// Now become leader
					leaderCh <- true
					// Wait for transition to leader
					select {
					case <-lStarted:
					case <-time.After(15 * time.Millisecond):
						t.Log("timed out waiting for leader to start")
						return
					}
					// give a tiny buffer then cancel
					time.Sleep(2 * time.Millisecond)
					cancel()
				}()
				return d, ctx, cancel
			},
			assertF: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, context.Canceled)
			},
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			d, runCtx, cancel := spec.setup(t)
			defer cancel()
			err := d.Run(runCtx)
			spec.assertF(t, err)
		})
	}
}

// Helper to quickly build a Config with very short timeouts for tests
func testCfg() Config {
	return Config{SendTimeout: 2 * time.Millisecond}
}

// testRunnable is a small helper implementing Runnable for tests.
// Run behaviour is provided via runFn. IsSynced behaviour via isSyncedFn.
// When runFn is nil, it blocks until context is cancelled.
// When startedCh is non-nil, it will be closed once Run starts.
// When doneCh is non-nil, it will be closed right before Run returns.
// These channels are used only by tests to observe state.
type testRunnable struct {
	runFn      func(ctx context.Context) error
	isSyncedFn func(*RaftBlockState) (int, error)
	startedCh  chan struct{}
	doneCh     chan struct{}
	recoverFn  func(ctx context.Context, state *RaftBlockState) error
}

func (t *testRunnable) Run(ctx context.Context) error {
	if t.startedCh != nil {
		select {
		case <-t.startedCh:
		default:
			close(t.startedCh)
		}
	}
	if t.runFn != nil {
		err := t.runFn(ctx)
		if t.doneCh != nil {
			close(t.doneCh)
		}
		return err
	}
	<-ctx.Done()
	if t.doneCh != nil {
		close(t.doneCh)
	}
	return ctx.Err()
}

func (t *testRunnable) IsSynced(s *RaftBlockState) (int, error) {
	if t.isSyncedFn != nil {
		return t.isSyncedFn(s)
	}
	return 0, nil
}

func (t *testRunnable) Recover(ctx context.Context, state *RaftBlockState) error {
	if t.recoverFn != nil {
		return t.recoverFn(ctx, state)
	}
	return nil
}

type nonRecoverableRunnable struct {
	r *testRunnable
}

func (n *nonRecoverableRunnable) Run(ctx context.Context) error {
	return n.r.Run(ctx)
}

func (n *nonRecoverableRunnable) IsSynced(s *RaftBlockState) (int, error) {
	return n.r.IsSynced(s)
}

func (n *nonRecoverableRunnable) Recover(ctx context.Context, state *RaftBlockState) error {
	return n.r.Recover(ctx, state)
}
