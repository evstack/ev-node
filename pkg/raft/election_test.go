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
				m.EXPECT().Config().Return(testCfg())
				m.EXPECT().waitForMsgsLanded(2 * time.Millisecond).Return(nil)
				m.EXPECT().NodeID().Return("self")
				m.EXPECT().leaderID().Return("self")
				m.EXPECT().GetState().Return(&RaftBlockState{Height: 1})
				m.EXPECT().leadershipTransfer().Return(nil)

				fStarted := make(chan struct{})
				follower := &testRunnable{startedCh: fStarted, isSyncedFn: func(*RaftBlockState) bool { return false }}
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
				// Start as follower via false, then signal leader true
				go func() {
					leaderCh <- false
					<-fStarted // ensure follower started
					leaderCh <- true
					// allow time for SendTimeout sleep and transfer
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
		"lost leadership during sync wait": {
			setup: func(t *testing.T) (*DynamicLeaderElection, context.Context, context.CancelFunc) {
				m := newMocksourceNode(t)
				leaderCh := make(chan bool, 2)
				m.EXPECT().leaderCh().Return((<-chan bool)(leaderCh))
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
				// On leadership change to true, election will sleep SendTimeout, then check sync against state
				m.EXPECT().Config().Return(testCfg())
				m.EXPECT().waitForMsgsLanded(2 * time.Millisecond).Return(nil)
				m.EXPECT().NodeID().Return("self")
				m.EXPECT().leaderID().Return("self")
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
		name, spec := name, spec
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
	isSyncedFn func(*RaftBlockState) bool
	startedCh  chan struct{}
	doneCh     chan struct{}
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

func (t *testRunnable) IsSynced(s *RaftBlockState) bool {
	if t.isSyncedFn != nil {
		return t.isSyncedFn(s)
	}
	return true
}
