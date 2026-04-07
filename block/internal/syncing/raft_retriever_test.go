package syncing

import (
	"context"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/genesis"
	pkgraft "github.com/evstack/ev-node/pkg/raft"
)

type stubRaftNode struct {
	mu        sync.Mutex
	callbacks []chan<- pkgraft.RaftApplyMsg
}

func (s *stubRaftNode) IsLeader() bool { return false }

func (s *stubRaftNode) HasQuorum() bool { return false }

func (s *stubRaftNode) GetState() *pkgraft.RaftBlockState { return nil }

func (s *stubRaftNode) Broadcast(context.Context, *pkgraft.RaftBlockState) error { return nil }

func (s *stubRaftNode) SetApplyCallback(ch chan<- pkgraft.RaftApplyMsg) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callbacks = append(s.callbacks, ch)
}

func (s *stubRaftNode) recordedCallbacks() []chan<- pkgraft.RaftApplyMsg {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]chan<- pkgraft.RaftApplyMsg, len(s.callbacks))
	copy(out, s.callbacks)
	return out
}

func TestRaftRetrieverStopClearsApplyCallback(t *testing.T) {
	t.Parallel()

	raftNode := &stubRaftNode{}
	retriever := newRaftRetriever(
		raftNode,
		genesis.Genesis{},
		zerolog.Nop(),
		nil,
		func(context.Context, *pkgraft.RaftBlockState) error { return nil },
	)

	require.NoError(t, retriever.Start(t.Context()))
	retriever.Stop()

	callbacks := raftNode.recordedCallbacks()
	require.Len(t, callbacks, 2)
	require.NotNil(t, callbacks[0])
	require.Nil(t, callbacks[1])
}
