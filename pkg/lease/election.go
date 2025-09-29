package lease

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type LeaderElector interface {
	Start(ctx context.Context) error
	Stop() error
	RunWithElection(ctx context.Context, leaderFunc, followerFunc func(leaderCtx context.Context) error) error
}

// LeaderElection manages leadership election using a lease mechanism
type LeaderElection struct {
	lease         Lease
	nodeID        string
	leaseName     string
	leaseTerm     time.Duration
	renewInterval time.Duration
	logger        zerolog.Logger

	mu                 sync.RWMutex
	isLockAcquiredOnce bool
	isLeader           bool
	leaderCh           chan bool
	ctx                context.Context
	cancel             context.CancelFunc
}

// NewLeaderElection creates a new leader election instance
func NewLeaderElection(lease Lease, nodeID, leaseName string, leaseTerm time.Duration, logger zerolog.Logger) *LeaderElection {
	return &LeaderElection{
		lease:         lease,
		nodeID:        nodeID,
		leaseName:     leaseName,
		leaseTerm:     leaseTerm,
		renewInterval: leaseTerm / 3, // Renew at 1/3 of lease term
		logger:        logger.With().Str("component", "leader-election").Str("node", nodeID).Logger(),
		leaderCh:      make(chan bool, 1),
	}
}

// Start begins the leader election process
func (le *LeaderElection) Start(ctx context.Context) error {
	le.mu.Lock()
	defer le.mu.Unlock()

	if le.ctx != nil {
		return fmt.Errorf("leader election already started")
	}

	le.ctx, le.cancel = context.WithCancel(ctx)

	go le.electionLoop()

	le.logger.Info().Msg("leader election started")
	return nil
}

// Stop stops the leader election and releases any held lease
func (le *LeaderElection) Stop() error {
	le.mu.Lock()
	defer le.mu.Unlock()

	if le.cancel == nil {
		return nil
	}

	le.cancel()

	// Release lease if we hold it
	if le.isLockAcquiredOnce && le.isLeader {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := le.lease.Release(ctx, le.nodeID); err != nil {
			le.logger.Error().Err(err).Msg("failed to release lease on stop")
		}
		le.isLeader = false
	}

	le.cancel = nil
	le.logger.Info().Msg("leader election stopped")
	return nil
}

// IsLeader returns whether this node is currently the leader
func (le *LeaderElection) IsLeader() (bool, error) {
	le.mu.RLock()
	defer le.mu.RUnlock()
	if !le.isLockAcquiredOnce {
		return false, errors.New("undefined state")
	}
	return le.isLeader, nil
}

// LeaderChan returns a channel that receives leadership status changes
func (le *LeaderElection) LeaderChan() <-chan bool {
	return le.leaderCh
}

// GetLeader returns the current leader node ID
func (le *LeaderElection) GetLeader(ctx context.Context) (string, error) {
	return le.lease.GetHolder(ctx)
}

// electionLoop runs the main election logic
func (le *LeaderElection) electionLoop() {
	ticker := time.NewTicker(le.renewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-le.ctx.Done():
			return
		case <-ticker.C:
			le.tryAcquireOrRenewLease()
		}
	}
}

// tryAcquireOrRenewLease attempts to acquire or renew the leadership lease
func (le *LeaderElection) tryAcquireOrRenewLease() {
	ctx, cancel := context.WithTimeout(le.ctx, 10*time.Second)
	defer cancel()

	le.mu.Lock()
	defer le.mu.Unlock()

	if le.isLeader {
		// Try to renew existing lease
		if err := le.lease.Renew(ctx, le.nodeID, le.leaseTerm); err != nil {
			le.logger.Error().Err(err).Msg("failed to renew lease, stepping down")
			le.isLeader = false
			le.notifyLeadershipUpdate(false)
			return
		}

		le.logger.Debug().Msg("lease renewed successfully")
		return
	}

	// Try to acquire lease
	acquired, err := le.lease.Acquire(ctx, le.nodeID, le.leaseTerm)
	if err != nil {
		le.logger.Error().Err(err).Msg("failed to acquire lease")
		return
	}
	if acquired {
		le.logger.Info().Msg("acquired leadership lease")
		le.isLeader = true
		le.isLockAcquiredOnce = true
		le.notifyLeadershipUpdate(true)
	} else {
		if !le.isLockAcquiredOnce {
			le.isLockAcquiredOnce = true
			le.notifyLeadershipUpdate(false)
		}
		// Log current leader for debugging
		if holder, err := le.lease.GetHolder(ctx); err == nil {
			le.logger.Debug().Str("current_leader", holder).Msg("lease held by another node")
		}
	}
}

// notifyLeadershipUpdate sends leadership change notification
func (le *LeaderElection) notifyLeadershipUpdate(isLeader bool) {
	select {
	case le.leaderCh <- isLeader:
	default:
		// Channel full, skip notification
	}
}

var ErrLeadershipLost = fmt.Errorf("leader lock lost")

// RunWithElection manages leader election and runs leader-only operations.
// It's recommended that leaderFunc accepts a context.Context to allow for graceful shutdown.
// This is a blocking operation
func (le *LeaderElection) RunWithElection(ctx context.Context, leaderFunc, followerFunc func(leaderCtx context.Context) error) error {
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
				case errCh <- errors.New(name + " worker exited unexpectedly"):
				default: // do not block
				}
			}
		}(workerCtx)
	}

	for {
		select {
		case isLeader := <-le.LeaderChan():
			if isLeader && !isCurrentlyLeader { // new leader
				if isStarted {
					le.logger.Info().Msg("became leader, stopping follower operations")
					workerCancel()
					wg.Wait()
				}
				le.logger.Info().Msg("starting leader operations")
				isCurrentlyLeader, isStarted = true, true
				startWorker("leader", leaderFunc)
			} else if !isLeader && isCurrentlyLeader { // lost leadership
				workerCancel()
				le.logger.Info().Msg("lost leadership")
				return ErrLeadershipLost
			} else if !isLeader && !isCurrentlyLeader && !isStarted { // start as follower
				le.logger.Info().Msg("starting follower operations")
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

var _ LeaderElector = (*AlwaysLeaderElection)(nil)

type AlwaysLeaderElection struct {
}

func (n AlwaysLeaderElection) Start(ctx context.Context) error {
	return nil
}

func (n AlwaysLeaderElection) Stop() error {
	return nil
}

func (n AlwaysLeaderElection) RunWithElection(ctx context.Context, leaderFunc, _ func(leaderCtx context.Context) error) error {
	return leaderFunc(ctx)
}
