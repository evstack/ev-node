package common

import (
	"context"

	"github.com/evstack/ev-node/pkg/raft"
)

// RaftNode interface for raft consensus integration
type RaftNode interface {
	IsLeader() bool
	NodeID() string
	GetState() *raft.RaftBlockState

	Broadcast(ctx context.Context, state *raft.RaftBlockState) error

	SetApplyCallback(ch chan<- raft.RaftApplyMsg)
	Shutdown() error

	AddPeer(nodeID, addr string) error
	RemovePeer(nodeID string) error
}
