package node

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/pkg/raft"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

type proposerStateReader interface {
	GetState(ctx context.Context) (types.State, error)
}

type proposerRole uint8

const (
	proposerRoleFollower proposerRole = iota
	proposerRoleLeader
)

type dynamicProposerElection struct {
	logger                         zerolog.Logger
	localProposer                  []byte
	initialProposer                []byte
	stateReader                    proposerStateReader
	leaderFactory, followerFactory func() (raft.Runnable, error)
	pollInterval                   time.Duration
	running                        atomic.Bool
}

func newDynamicProposerElection(
	logger zerolog.Logger,
	localProposer []byte,
	initialProposer []byte,
	stateReader proposerStateReader,
	leaderFactory func() (raft.Runnable, error),
	followerFactory func() (raft.Runnable, error),
	pollInterval time.Duration,
) *dynamicProposerElection {
	if pollInterval <= 0 {
		pollInterval = 300 * time.Millisecond
	}
	return &dynamicProposerElection{
		logger:          logger,
		localProposer:   append([]byte(nil), localProposer...),
		initialProposer: append([]byte(nil), initialProposer...),
		stateReader:     stateReader,
		leaderFactory:   leaderFactory,
		followerFactory: followerFactory,
		pollInterval:    pollInterval,
	}
}

func (d *dynamicProposerElection) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	var workerCancel context.CancelFunc = func() {}
	errCh := make(chan error, 1)
	currentRole := proposerRoleFollower

	defer func() {
		workerCancel()
		wg.Wait()
		close(errCh)
	}()
	d.running.Store(true)
	defer d.running.Store(false)

	startRole := func(role proposerRole) error {
		workerCancel()
		wg.Wait()

		var (
			runnable raft.Runnable
			err      error
			name     string
		)
		switch role {
		case proposerRoleLeader:
			name = "leader"
			runnable, err = d.leaderFactory()
		case proposerRoleFollower:
			name = "follower"
			runnable, err = d.followerFactory()
		default:
			return fmt.Errorf("unknown proposer role: %d", role)
		}
		if err != nil {
			return err
		}

		workerCtx, cancel := context.WithCancel(ctx)
		workerCancel = cancel
		currentRole = role
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := runnable.Run(workerCtx); err != nil && !errors.Is(err, context.Canceled) {
				select {
				case errCh <- fmt.Errorf("%s worker exited unexpectedly: %w", name, err):
				default:
				}
			}
		}()
		return nil
	}

	if err := startRole(proposerRoleFollower); err != nil {
		return err
	}

	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			shouldLead, err := d.shouldLead(ctx)
			if err != nil {
				d.logger.Debug().Err(err).Msg("could not read local proposer state")
				continue
			}
			if shouldLead && currentRole != proposerRoleLeader {
				d.logger.Info().Msg("local signer is next proposer, promoting to aggregator")
				if err := startRole(proposerRoleLeader); err != nil {
					return err
				}
			}
			if !shouldLead && currentRole != proposerRoleFollower {
				d.logger.Info().Msg("local signer is not next proposer, returning to sync mode")
				if err := startRole(proposerRoleFollower); err != nil {
					return err
				}
			}
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (d *dynamicProposerElection) IsRunning() bool {
	return d.running.Load()
}

func (d *dynamicProposerElection) shouldLead(ctx context.Context) (bool, error) {
	state, err := d.stateReader.GetState(ctx)
	if err != nil {
		if store.IsNotFound(err) {
			return len(d.initialProposer) > 0 && bytes.Equal(d.initialProposer, d.localProposer), nil
		}
		return false, err
	}
	expectedProposer := state.NextProposerAddress
	if len(expectedProposer) == 0 {
		expectedProposer = d.initialProposer
	}
	return len(expectedProposer) > 0 && bytes.Equal(expectedProposer, d.localProposer), nil
}
