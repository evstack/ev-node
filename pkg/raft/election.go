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

	var runnable Runnable
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
				_ = d.node.leadershipTransfer()
				select {
				case errCh <- fmt.Errorf(name+" worker exited unexpectedly: %s", err):
				default: // do not block
				}
			}
		}(workerCtx)
	}
	startFollower := func() error {
		var err error
		if runnable, err = d.followerFactory(); err != nil {
			return err
		}
		// avoids validating against stale raft state.
		if err = d.node.waitForMsgsLanded(d.node.Config().SendTimeout); err != nil {
			// this wait can legitimately time out
			d.logger.Debug().Err(err).Msg("timed out waiting for raft messages before follower verification; continuing")
		}
		if err = d.verifyState(ctx, runnable); err != nil {
			return err
		}
		startWorker("follower", runnable.Run)
		return nil
	}
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	d.running.Store(true)
	defer d.running.Store(false)
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
					if diff < -1 {
						// Store is more than 1 block behind raft state.
						// RecoverFromRaft can only apply the single latest block
						// from the raft snapshot; it cannot replay a larger gap.
						//
						// Before abdicating, wait for p2p block-store sync to close
						// the gap. If all nodes restart simultaneously with lagging
						// block stores, immediate abdication causes a leadership
						// hot-potato: every elected node abdicates at once and the
						// cluster never stabilises. Waiting gives the fastest-syncing
						// peer a chance to stay as leader.
						d.logger.Warn().
							Int("store_lag_blocks", -diff).
							Uint64("raft_height", raftState.Height).
							Msg("became leader but store is significantly behind raft state; waiting for block-store sync")
						switch d.waitForBlockStoreSync(ctx, runnable) {
						case syncResultCanceled:
							return ctx.Err()
						case syncResultLostLeadership:
							d.logger.Info().Msg("lost leadership while waiting for block-store sync; skipping abdication")
							continue
						case syncResultTimeout:
							d.logger.Warn().
								Int("store_lag_blocks", -diff).
								Uint64("raft_height", raftState.Height).
								Msg("store still significantly behind raft state after wait; abdicating to prevent stalled block production")
							if tErr := d.node.leadershipTransfer(); tErr != nil {
								d.logger.Error().Err(tErr).Msg("leadership transfer failed after store-lag abdication")
								return fmt.Errorf("leadership transfer failed after store-lag abdication: %w", tErr)
							}
							continue
						case syncResultSynced:
							// Block store caught up — refresh state so the recovery
							// check below works with the latest values.
							d.logger.Info().Msg("block store caught up after wait; proceeding as leader")
							raftState = d.node.GetState()
							var syncErr error
							diff, syncErr = runnable.IsSynced(raftState)
							if syncErr != nil {
								return syncErr
							}
						}
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
				if err := startFollower(); err != nil {
					return err
				}
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
			if err := startFollower(); err != nil {
				return err
			}
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
		waitTimeout := d.node.Config().SendTimeout
		deadline := time.NewTimer(waitTimeout)
		defer deadline.Stop()
		ticker := time.NewTicker(min(50*time.Millisecond, max(waitTimeout/4, time.Millisecond)))
		defer ticker.Stop()

		for raftState == nil || raftState.Height == 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-deadline.C:
				d.logger.Info().Msg("raft state still at height 0 after wait; skipping recovery to allow normal sync")
				return nil
			case <-ticker.C:
				raftState = d.node.GetState()
			}
		}
	}
	diff, err := runnable.IsSynced(raftState)
	if err != nil {
		d.logger.Warn().Err(err).Uint64("raft_height", raftState.Height).Msg("sync check failed, attempting recovery from raft canonical state")
		if recErr := runnable.Recover(ctx, raftState); recErr != nil {
			return errors.Join(err, fmt.Errorf("recovery after sync-check failure: %w", recErr))
		}
		d.logger.Info().Msg("recovery successful after sync-check failure")
		return nil
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

type syncResult int

const (
	syncResultSynced         syncResult = iota // block store is within 1 block of raft FSM
	syncResultTimeout                          // deadline elapsed and store still lagging
	syncResultLostLeadership                   // lost leadership while waiting
	syncResultCanceled                         // context was canceled
)

// waitForBlockStoreSync polls IsSynced until the block store is within 1 block
// of the current raft FSM height, leadership is lost, or the context expires.
func (d *DynamicLeaderElection) waitForBlockStoreSync(ctx context.Context, r Runnable) syncResult {
	cfg := d.node.Config()
	timeout := cfg.ShutdownTimeout
	if timeout <= 0 {
		timeout = 5 * cfg.SendTimeout
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	pollInterval := min(100*time.Millisecond, timeout/10)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return syncResultCanceled
		case <-deadline.C:
			// Final check before giving up.
			diff, err := r.IsSynced(d.node.GetState())
			if err == nil && diff >= -1 {
				return syncResultSynced
			}
			return syncResultTimeout
		case <-ticker.C:
			if d.node.leaderID() != d.node.NodeID() {
				return syncResultLostLeadership
			}
			diff, err := r.IsSynced(d.node.GetState())
			if err == nil && diff >= -1 {
				return syncResultSynced
			}
		}
	}
}
