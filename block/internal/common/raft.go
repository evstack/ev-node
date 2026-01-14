package common

import (
	"context"

	"github.com/evstack/ev-node/pkg/raft"
)

// RaftNode interface for raft consensus integration
type RaftNode interface {
	IsLeader() bool
	HasQuorum() bool
	GetState() *raft.RaftBlockState

	Broadcast(ctx context.Context, state *raft.RaftBlockState) error

	SetApplyCallback(ch chan<- raft.RaftApplyMsg)
}
