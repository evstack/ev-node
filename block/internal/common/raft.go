package common

import (
	"context"
)

// RaftNode interface for raft consensus integration
type RaftNode interface {
	IsLeader() bool
	NodeID() string
	GetState() *RaftBlockState

	Broadcast(ctx context.Context, state *RaftBlockState) error

	SetApplyCallback(ch chan<- RaftApplyMsg)
	Shutdown() error

	AddPeer(nodeID, addr string) error
	RemovePeer(nodeID string) error
}

// todo: refactor to use proto
// RaftBlockState represents replicated block state
type RaftBlockState struct {
	Height    uint64
	Hash      []byte
	Timestamp uint64
	Header    []byte
	Data      []byte
}

// RaftApplyMsg is sent when raft applies a log entry
type RaftApplyMsg struct {
	Index uint64
	State *RaftBlockState
}
