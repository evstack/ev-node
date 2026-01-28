package raft

import (
	"fmt"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// RaftBlockState represents a replicated block state
type RaftBlockState = pb.RaftBlockState

// assertValid checks basic constraints but does not ensure that no gaps exist or chain continuity
func assertValid(s *RaftBlockState, next *RaftBlockState) error {
	if s.Height > next.Height {
		return fmt.Errorf("invalid height: %d > %d", s.Height, next.Height)
	}
	if s.Timestamp > next.Timestamp {
		return fmt.Errorf("invalid timestamp: %d > %d", s.Timestamp, next.Timestamp)
	}
	return nil
}

// RaftApplyMsg is sent when raft applies a log entry
type RaftApplyMsg struct {
	Index uint64
	State *RaftBlockState
}
