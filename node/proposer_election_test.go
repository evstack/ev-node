package node

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/raft"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

type proposerElectionStateReader struct {
	mu    sync.Mutex
	state types.State
	err   error
}

func (r *proposerElectionStateReader) setNextProposer(addr []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state.NextProposerAddress = append([]byte(nil), addr...)
	r.err = nil
}

func (r *proposerElectionStateReader) setErr(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.err = err
}

func (r *proposerElectionStateReader) GetState(context.Context) (types.State, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return types.State{}, r.err
	}
	return r.state, nil
}

type proposerElectionRunnable struct {
	name    string
	started chan<- string
	stopped chan<- string
}

func (r proposerElectionRunnable) Run(ctx context.Context) error {
	r.started <- r.name
	<-ctx.Done()
	r.stopped <- r.name
	return ctx.Err()
}

func (r proposerElectionRunnable) IsSynced(*raft.RaftBlockState) (int, error) {
	return 0, nil
}

func (r proposerElectionRunnable) Recover(context.Context, *raft.RaftBlockState) error {
	return nil
}

func TestDynamicProposerElectionPromotesAndDemotesFromLocalState(t *testing.T) {
	localProposer := []byte{1, 2, 3}
	otherProposer := []byte{9, 8, 7}
	stateReader := &proposerElectionStateReader{}
	stateReader.setNextProposer(otherProposer)

	started := make(chan string, 8)
	stopped := make(chan string, 8)
	election := newDynamicProposerElection(
		zerolog.Nop(),
		localProposer,
		localProposer,
		stateReader,
		func() (raft.Runnable, error) {
			return proposerElectionRunnable{name: "leader", started: started, stopped: stopped}, nil
		},
		func() (raft.Runnable, error) {
			return proposerElectionRunnable{name: "follower", started: started, stopped: stopped}, nil
		},
		time.Millisecond,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- election.Run(ctx)
	}()

	require.Equal(t, "follower", receiveProposerElectionEvent(t, started))
	require.True(t, election.IsRunning())

	stateReader.setNextProposer(localProposer)
	require.Equal(t, "follower", receiveProposerElectionEvent(t, stopped))
	require.Equal(t, "leader", receiveProposerElectionEvent(t, started))

	stateReader.setNextProposer(otherProposer)
	require.Equal(t, "leader", receiveProposerElectionEvent(t, stopped))
	require.Equal(t, "follower", receiveProposerElectionEvent(t, started))

	cancel()
	require.Equal(t, "follower", receiveProposerElectionEvent(t, stopped))
	require.ErrorIs(t, receiveProposerElectionError(t, errCh), context.Canceled)
}

func TestDynamicProposerElectionUsesInitialProposerBeforeStateExists(t *testing.T) {
	localProposer := []byte{1, 2, 3}
	otherProposer := []byte{9, 8, 7}
	stateReader := &proposerElectionStateReader{}
	stateReader.setErr(store.ErrNotFound)

	started := make(chan string, 8)
	stopped := make(chan string, 8)
	election := newDynamicProposerElection(
		zerolog.Nop(),
		localProposer,
		localProposer,
		stateReader,
		func() (raft.Runnable, error) {
			return proposerElectionRunnable{name: "leader", started: started, stopped: stopped}, nil
		},
		func() (raft.Runnable, error) {
			return proposerElectionRunnable{name: "follower", started: started, stopped: stopped}, nil
		},
		time.Millisecond,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- election.Run(ctx)
	}()

	require.Equal(t, "follower", receiveProposerElectionEvent(t, started))
	require.Equal(t, "follower", receiveProposerElectionEvent(t, stopped))
	require.Equal(t, "leader", receiveProposerElectionEvent(t, started))

	stateReader.setNextProposer(otherProposer)
	require.Equal(t, "leader", receiveProposerElectionEvent(t, stopped))
	require.Equal(t, "follower", receiveProposerElectionEvent(t, started))

	cancel()
	require.Equal(t, "follower", receiveProposerElectionEvent(t, stopped))
	require.ErrorIs(t, receiveProposerElectionError(t, errCh), context.Canceled)
}

func TestDynamicProposerElectionDoesNotUseInitialProposerForUnexpectedStateErrors(t *testing.T) {
	localProposer := []byte{1, 2, 3}
	stateReader := &proposerElectionStateReader{}
	stateReader.setErr(errors.New("state read failed"))

	election := newDynamicProposerElection(
		zerolog.Nop(),
		localProposer,
		localProposer,
		stateReader,
		func() (raft.Runnable, error) {
			t.Fatal("leader factory should not be called on unexpected state read errors")
			return nil, nil
		},
		func() (raft.Runnable, error) {
			return proposerElectionRunnable{}, nil
		},
		time.Millisecond,
	)

	shouldLead, err := election.shouldLead(context.Background())
	require.ErrorContains(t, err, "state read failed")
	require.False(t, shouldLead)
}

func receiveProposerElectionEvent(t *testing.T, ch <-chan string) string {
	t.Helper()
	select {
	case event := <-ch:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for proposer election event")
		return ""
	}
}

func receiveProposerElectionError(t *testing.T, ch <-chan error) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for proposer election error")
		return nil
	}
}
