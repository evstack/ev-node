package lease

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// LeaderElection manages leadership election using a lease mechanism
type LeaderElection struct {
	lease         Lease
	nodeID        string
	leaseName     string
	leaseTerm     time.Duration
	renewInterval time.Duration
	logger        zerolog.Logger

	mu       sync.RWMutex
	isLeader bool
	leaderCh chan bool
	ctx      context.Context
	cancel   context.CancelFunc
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
	if le.isLeader {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := le.lease.Release(ctx, le.nodeID); err != nil {
			le.logger.Error().Err(err).Msg("failed to release lease on stop")
		}
		le.isLeader = false
	}

	le.ctx = nil
	le.cancel = nil

	le.logger.Info().Msg("leader election stopped")
	return nil
}

// IsLeader returns whether this node is currently the leader
func (le *LeaderElection) IsLeader() bool {
	le.mu.RLock()
	defer le.mu.RUnlock()
	return le.isLeader
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
			le.notifyLeadershipChange(false)
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
		le.notifyLeadershipChange(true)
	} else {
		// Log current leader for debugging
		if holder, err := le.lease.GetHolder(ctx); err == nil {
			le.logger.Debug().Str("current_leader", holder).Msg("lease held by another node")
		}
	}
}

// notifyLeadershipChange sends leadership change notification
func (le *LeaderElection) notifyLeadershipChange(isLeader bool) {
	select {
	case le.leaderCh <- isLeader:
	default:
		// Channel full, skip notification
	}
}

// DoAsLeader manages leader election and runs leader-only operations
// It's recommended that leaderFunc accepts a context.Context to allow for graceful shutdown.
func (n *LeaderElection) DoAsLeader(ctx context.Context, leaderFunc func(leaderCtx context.Context)) error {
	var isCurrentlyLeader bool
	var leaderCancel context.CancelFunc = func() {} // noop
	var wg sync.WaitGroup
	defer func() {
		leaderCancel()
		wg.Wait()
	}()
	errCh := make(chan error, 1)
	for {
		select {
		case isLeader := <-n.LeaderChan():
			if isLeader && !isCurrentlyLeader {
				n.logger.Info().Msg("became leader, starting leader operations")
				leaderCtx, cancel := context.WithCancel(ctx)
				leaderCancel = cancel
				isCurrentlyLeader = true
				wg.Add(1)
				// call leaderFunc in a separate goroutine
				go func(lCtx context.Context) {
					defer wg.Done()
					leaderFunc(lCtx)
					if lCtx.Err() == nil {
						select {
						case errCh <- fmt.Errorf("leader function exited unexpectedly"):
						default: // do not block
						}
					}
				}(leaderCtx)

			} else if !isLeader && isCurrentlyLeader {
				n.logger.Info().Msg("lost leadership")
				return fmt.Errorf("leader lock lost")
			}
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
