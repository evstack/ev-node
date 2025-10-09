package raft

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrLeadershipLost = fmt.Errorf("leader lock lost")

// RunWithElection manages leader election and runs leader-only operations.
// It's recommended that leaderFunc accepts a context.Context to allow for graceful shutdown.
// This is a blocking operation
func (n *Node) RunWithElection(ctx context.Context, leaderFunc, followerFunc func(leaderCtx context.Context) error) error {
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

	for {
		select {
		case isLeader := <-n.raft.LeaderCh():
			n.logger.Info().Msg("Raft leader changed notification")
			if isLeader && !isCurrentlyLeader { // new leader
				if isStarted {
					n.logger.Info().Msg("became leader, stopping follower operations")
					workerCancel()
					wg.Wait()
				}
				n.logger.Info().Msg("starting leader operations")
				isCurrentlyLeader, isStarted = true, true
				startWorker("leader", leaderFunc)
			} else if !isLeader && isCurrentlyLeader { // lost leadership
				workerCancel()
				n.logger.Info().Msg("lost leadership")
				return ErrLeadershipLost
			} else if !isLeader && !isCurrentlyLeader && !isStarted { // start as follower
				n.logger.Info().Msg("starting follower operations")
				isStarted = true
				startWorker("follower", followerFunc)
			}
		case <-ticker.C: // LeaderCh fires only when leader changes not on initial election
			if isStarted {
				ticker.Stop()
				continue
			}
			_, nodeID := n.raft.LeaderWithID()
			if nodeID != "" && string(nodeID) != n.config.NodeID {
				ticker.Stop()
				n.logger.Info().Msg("starting follower operations")
				isStarted = true
				startWorker("follower", followerFunc)
			}
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
