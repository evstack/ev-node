package raft

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
