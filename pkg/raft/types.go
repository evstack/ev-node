package raft

import "fmt"

// RaftBlockState represents a replicated block state
type RaftBlockState struct {
	Height                      uint64 `json:"height"`
	LastSubmittedDAHeaderHeight uint64 `json:"last_submitted_da_header_height"`
	LastSubmittedDADataHeight   uint64 `json:"last_submitted_da_data_height"`
	Hash                        []byte `json:"hash"`
	Timestamp                   uint64 `json:"timestamp"`
	Header                      []byte `json:"header"`
	Data                        []byte `json:"data"`
}

// assertValid checks basic constraints but does not ensure that no gaps exist or chain continuity
func (s RaftBlockState) assertValid(next RaftBlockState) error {
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
