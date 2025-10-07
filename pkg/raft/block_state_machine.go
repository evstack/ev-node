package raft

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/evstack/ev-node/types"
)

// BlockApplier is called when a new block is committed via raft
type BlockApplier func(header *types.SignedHeader, data *types.Data) error

// BlockStateMachine implements the StateMachine interface for block processing state
type BlockStateMachine struct {
	mu                  sync.RWMutex
	lastProcessedHeight uint64
	lastProcessedHash   string
	lastProcessedTime   int64
	lastProcessedHeader *types.SignedHeader
	lastProcessedData   *types.Data
	blockApplier        BlockApplier
}

// BlockState represents the state stored in raft
type BlockState struct {
	Height     uint64 `json:"height"`
	Hash       string `json:"hash"`
	Time       int64  `json:"time"`
	HeaderData []byte `json:"header_data,omitempty"`
	BlockData  []byte `json:"block_data,omitempty"`
}

// NewBlockStateMachine creates a new block state machine
func NewBlockStateMachine(blockApplier BlockApplier) *BlockStateMachine {
	return &BlockStateMachine{
		blockApplier: blockApplier,
	}
}

// Apply applies a committed entry to the state machine
func (sm *BlockStateMachine) Apply(data []byte) error {
	var state BlockState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("unmarshal block state: %w", err)
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.lastProcessedHeight = state.Height
	sm.lastProcessedHash = state.Hash
	sm.lastProcessedTime = state.Time

	// Deserialize full block data if present
	var header *types.SignedHeader
	var blockData *types.Data

	if len(state.HeaderData) > 0 {
		header = &types.SignedHeader{}
		if err := header.UnmarshalBinary(state.HeaderData); err != nil {
			return fmt.Errorf("unmarshal header: %w", err)
		}
		sm.lastProcessedHeader = header
	}

	if len(state.BlockData) > 0 {
		blockData = &types.Data{}
		if err := blockData.UnmarshalBinary(state.BlockData); err != nil {
			return fmt.Errorf("unmarshal block data: %w", err)
		}
		sm.lastProcessedData = blockData
	}

	// Invoke callback if full block data is available
	if sm.blockApplier != nil && header != nil && blockData != nil {
		if err := sm.blockApplier(header, blockData); err != nil {
			return fmt.Errorf("apply block: %w", err)
		}
	}

	return nil
}

// Snapshot creates a snapshot of the current state
func (sm *BlockStateMachine) Snapshot() ([]byte, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state := BlockState{
		Height: sm.lastProcessedHeight,
		Hash:   sm.lastProcessedHash,
		Time:   sm.lastProcessedTime,
	}

	// Serialize full block data if available
	if sm.lastProcessedHeader != nil {
		headerData, err := sm.lastProcessedHeader.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("marshal header: %w", err)
		}
		state.HeaderData = headerData
	}

	if sm.lastProcessedData != nil {
		blockData, err := sm.lastProcessedData.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("marshal block data: %w", err)
		}
		state.BlockData = blockData
	}

	data, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("marshal snapshot: %w", err)
	}

	return data, nil
}

// Restore restores the state from a snapshot
func (sm *BlockStateMachine) Restore(snapshot []byte) error {
	if len(snapshot) == 0 {
		return nil
	}

	var state BlockState
	if err := json.Unmarshal(snapshot, &state); err != nil {
		return fmt.Errorf("unmarshal snapshot: %w", err)
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.lastProcessedHeight = state.Height
	sm.lastProcessedHash = state.Hash
	sm.lastProcessedTime = state.Time

	// Deserialize full block data if present
	if len(state.HeaderData) > 0 {
		header := &types.SignedHeader{}
		if err := header.UnmarshalBinary(state.HeaderData); err != nil {
			return fmt.Errorf("unmarshal header: %w", err)
		}
		sm.lastProcessedHeader = header
	} else {
		sm.lastProcessedHeader = nil
	}

	if len(state.BlockData) > 0 {
		blockData := &types.Data{}
		if err := blockData.UnmarshalBinary(state.BlockData); err != nil {
			return fmt.Errorf("unmarshal block data: %w", err)
		}
		sm.lastProcessedData = blockData
	} else {
		sm.lastProcessedData = nil
	}

	return nil
}

// GetLastProcessed returns the last processed block information
func (sm *BlockStateMachine) GetLastProcessed() (height uint64, hash string, time int64) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.lastProcessedHeight, sm.lastProcessedHash, sm.lastProcessedTime
}

// GetLastBlock returns the last processed block header and data
func (sm *BlockStateMachine) GetLastBlock() (*types.SignedHeader, *types.Data) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.lastProcessedHeader, sm.lastProcessedData
}

// SetLastProcessed updates the last processed block (for local use, not via raft)
func (sm *BlockStateMachine) SetLastProcessed(height uint64, hash string, time int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.lastProcessedHeight = height
	sm.lastProcessedHash = hash
	sm.lastProcessedTime = time
}

// CreateBlockStateData creates the data to propose to raft with full block data
func CreateBlockStateData(header *types.SignedHeader, data *types.Data) ([]byte, error) {
	state := BlockState{
		Height: header.Height(),
		Hash:   header.Hash().String(),
		Time:   header.Time().UnixNano(),
	}

	// Serialize header
	headerData, err := header.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal header: %w", err)
	}
	state.HeaderData = headerData

	// Serialize block data
	blockData, err := data.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal block data: %w", err)
	}
	state.BlockData = blockData

	stateData, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("marshal block state: %w", err)
	}

	return stateData, nil
}
