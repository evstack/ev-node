package raft

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

var ErrLeadershipLost = fmt.Errorf("leader lock lost")

// Runnable represents a component that can be started and performs specific operations while running.
type Runnable interface {
	// Run runs the main logic of the component using the provided context and returns an error if it fails.
	Run(ctx context.Context) error
	// IsSynced checks whether the component is synced with the given RaftBlockState.
	// -1 means raft is ahead by 1, 0 equal and a positive number the blocks that the local state is ahead of the raft state.
	IsSynced(*RaftBlockState) (int, error)
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
		wg.Wait() // Ensure previous worker is fully stopped
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
				// Critical Safety Check: Wait for FSM to apply all committed logs up to LastIndex.
				// If we start leader operations with stale FSM, we risk Double Signing (proposing old blocks).
				if err := d.node.waitForMsgsLanded(d.node.Config().SendTimeout); err != nil {
					d.logger.Error().Err(err).Msg("failed to wait for messages to land - FSM lagging, abdicating to prevent safety violation")
					if tErr := d.node.leadershipTransfer(); tErr != nil {
						d.logger.Error().Err(tErr).Msg("failed to transfer leadership")
					}
					// Do not start worker.
					continue
				}

				if isStarted {
					d.logger.Info().Msg("became leader, stopping follower operations")
					if d.node.leaderID() != d.node.NodeID() {
						d.logger.Info().Msg("lost leadership during sync wait")
						continue
					}
					// Obtain the latest state from the Raft node
					raftState := d.node.GetState()
					diff, err := runnable.IsSynced(raftState)
					if err != nil {
						return err
					}
					if diff != 0 {
						d.logger.Info().Msg("became leader but not synced, attempting recovery")
						if err := runnable.Recover(ctx, raftState); err != nil {
							d.logger.Error().Err(err).Msg("recovery failed")
							_ = d.node.leadershipTransfer()
							return err
						}
						d.logger.Info().Msg("recovery successful")
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

				if err = d.verifyState(ctx, runnable); err != nil {
					return err
				}
				startWorker("follower", runnable.Run)
			}
			// LeaderCh fires only when leader changes not on initial election
		case <-ticker.C:
			leaderID := d.node.leaderID()
			if leaderID == "" {
				continue
			}
			ticker.Stop()
			ticker.C = nil
			if isStarted {
				continue
			}
			if leaderID == d.node.NodeID() {
				continue // let the leaderCh fire
			}

			d.logger.Info().Msg("starting follower operations")
			isStarted = true
			var err error
			if runnable, err = d.followerFactory(); err != nil {
				return err
			}
			startWorker("follower", runnable.Run)
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// verify follower state
func (d *DynamicLeaderElection) verifyState(ctx context.Context, runnable Runnable) error {
	// Verify sync state before starting follower operations
	raftState := d.node.GetState()
	if raftState == nil || raftState.Height == 0 {
		// Initial/empty raft state - skip recovery and let normal sync handle it.
		// This can happen during rolling restarts when the Raft FSM hasn't replayed logs yet.
		d.logger.Info().Msg("raft state at height 0, skipping recovery to allow normal sync")
		return nil
	}
	diff, err := runnable.IsSynced(raftState)
	if err != nil {
		return err
	}
	if diff == 0 {
		return nil
	}
	// If local state is behind raft state, we don't need to force recover (rollback).
	// Instead, we should let the normal sync mechanism catch up.
	if diff < 0 {
		// calculate local height from raft height and difference
		// diff = local - raft  =>  local = raft + diff
		localHeight := uint64(int64(raftState.Height) + int64(diff))
		d.logger.Info().
			Uint64("local_height", localHeight).
			Uint64("raft_height", raftState.Height).
			Int("diff", diff).
			Msg("local state behind raft state, skipping recovery to allow catchup")
		return nil
	}

	d.logger.Info().Uint64("raft_height", raftState.Height).Int("diff", diff).Msg("follower not synced, attempting recovery before start")
	if err := runnable.Recover(ctx, raftState); err != nil {
		d.logger.Error().Err(err).Msg("follower recovery failed")
		return err
	}
	d.logger.Info().Msg("follower recovery successful")
	return nil
}

func (d *DynamicLeaderElection) IsRunning() bool {
	return d.running.Load()
}
