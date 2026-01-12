package raft

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/raft"
	"github.com/rs/zerolog"
)

var ErrLeadershipLost = fmt.Errorf("leader lock lost")

// Runnable represents a component that can be started and performs specific operations while running.
// Run runs the main logic of the component using the provided context and returns an error if it fails.
// IsSynced checks whether the component is synced with the given RaftBlockState.
type Runnable interface {
	Run(ctx context.Context) error
	IsSynced(*RaftBlockState) bool
}

// Recoverable represents a component that can recover its state from a RaftBlockState
type Recoverable interface {
	Recover(ctx context.Context, state *RaftBlockState) error
}

type sourceNode interface {
	Config() Config
	leaderCh() <-chan bool
	leaderID() string
	NodeID() string
	GetState() *RaftBlockState
	leadershipTransfer() error
	waitForMsgsLanded(duration time.Duration) error
}

type DynamicLeaderElection struct {
	logger                         zerolog.Logger
	leaderFactory, followerFactory func() (Runnable, error)
	node                           sourceNode
	running                        atomic.Bool
}

// NewDynamicLeaderElection constructor
func NewDynamicLeaderElection(
	logger zerolog.Logger,
	leaderFactory func() (Runnable, error),
	followerFactory func() (Runnable, error),
	node *Node,
) *DynamicLeaderElection {
	return &DynamicLeaderElection{logger: logger, leaderFactory: leaderFactory, followerFactory: followerFactory, node: node}
}

// Run starts the leader election process and manages the lifecycle of leader or follower roles based on Raft events.
// This is a blocking call.
func (d *DynamicLeaderElection) Run(ctx context.Context) error {
	var isStarted, isCurrentlyLeader bool
	var workerCancel context.CancelFunc = func() {} // noop
	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	defer func() {
		workerCancel()
		wg.Wait()
		close(errCh)
	}()

	startWorker := func(name string, workerFunc func(ctx context.Context) error) {
		workerCancel()
		workerCtx, cancel := context.WithCancel(ctx)
		workerCancel = cancel
		wg.Add(1)

		// call workerFunc in a separate goroutine
		go func(childCtx context.Context) {
			defer wg.Done()
			if err := workerFunc(childCtx); err != nil && !errors.Is(err, context.Canceled) {
				select {
				case errCh <- fmt.Errorf(name+" worker exited unexpectedly: %s", err):
				default: // do not block
				}
			}
		}(workerCtx)
	}
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	d.running.Store(true)
	defer d.running.Store(false)
	var runnable Runnable
	for {
		select {
		case becameLeader := <-d.node.leaderCh():
			d.logger.Info().Msg("Raft leader changed notification")
			if becameLeader && !isCurrentlyLeader { // new leader
				if isStarted {
					d.logger.Info().Msg("became leader, stopping follower operations")
					// wait for in flight raft msgs to land, this is critical to avoid double sign on old state
					raftSynced := d.node.waitForMsgsLanded(d.node.Config().SendTimeout) == nil
					if d.node.leaderID() != d.node.NodeID() {
						d.logger.Info().Msg("lost leadership during sync wait")
						continue
					}
					// Obtain the latest state from the Raft node
					raftState := d.node.GetState()
					if !raftSynced || !runnable.IsSynced(raftState) {
						if recoverable, ok := runnable.(Recoverable); ok {
							d.logger.Info().Msg("became leader but not synced, attempting recovery")
							if err := recoverable.Recover(ctx, raftState); err != nil {
								d.logger.Error().Err(err).Msg("recovery failed")
								return err
							}
							d.logger.Info().Msg("recovery successful")
						} else {
							d.logger.Info().Uint64("raft_state_height", raftState.Height).
								Msg("became leader, but not synced. Pass on leadership")
							if err := d.node.leadershipTransfer(); err != nil && !errors.Is(err, raft.ErrNotLeader) {
								// the leadership transfer can fail due to no suitable leader. Better stop than double sign on old state
								return err
							}
							continue
						}
					}
					d.logger.Info().Msg("became leader, stopping follower operations")
					workerCancel()
					wg.Wait()
				}
				d.logger.Info().Msg("starting leader operations")
				var err error
				if runnable, err = d.leaderFactory(); err != nil {
					return err
				}
				isStarted = true
				isCurrentlyLeader = true
				startWorker("leader", runnable.Run)
			} else if !becameLeader && isCurrentlyLeader { // lost leadership
				workerCancel()
				d.logger.Info().Msg("lost leadership")
				return ErrLeadershipLost
			} else if !becameLeader && !isCurrentlyLeader && !isStarted { // start as a follower
				d.logger.Info().Msg("starting follower operations")
				isStarted = true
				var err error
				if runnable, err = d.followerFactory(); err != nil {
					return err
				}
				startWorker("follower", runnable.Run)
			}
		case <-ticker.C: // LeaderCh fires only when leader changes not on initial election
			if isStarted {
				ticker.Stop()
				ticker.C = nil
				continue
			}
			if leaderID := d.node.leaderID(); leaderID != "" && leaderID != d.node.NodeID() {
				ticker.Stop()
				ticker.C = nil
				d.logger.Info().Msg("starting follower operations")
				isStarted = true
				var err error
				if runnable, err = d.followerFactory(); err != nil {
					return err
				}
				startWorker("follower", runnable.Run)
			}
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (d *DynamicLeaderElection) IsRunning() bool {
	return d.running.Load()
}
