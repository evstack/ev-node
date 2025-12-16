package common

import (
	"context"
	"errors"
	"fmt"

	ds "github.com/ipfs/go-datastore"
	"google.golang.org/protobuf/proto"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// ErrCheckpointNotFound is returned when no checkpoint exists in the datastore
var ErrCheckpointNotFound = errors.New("checkpoint not found")

// Checkpoint tracks the position in the DA where we last processed transactions
type Checkpoint struct {
	// DAHeight is the DA block height we're currently processing or have just finished
	DAHeight uint64
	// TxIndex is the index of the next transaction to process within the DA block's forced inclusion batch
	// If TxIndex == 0, it means we've finished processing the previous DA block and should fetch the next one
	TxIndex uint64
}

// CheckpointStore manages persistence of the checkpoint
type CheckpointStore struct {
	db            ds.Batching
	checkpointKey ds.Key
}

// NewCheckpointStore creates a new checkpoint store
func NewCheckpointStore(db ds.Batching, checkpointkey ds.Key) *CheckpointStore {
	return &CheckpointStore{
		db:            db,
		checkpointKey: checkpointkey,
	}
}

// Load loads the checkpoint from the datastore
// Returns ErrCheckpointNotFound if no checkpoint exists
func (cs *CheckpointStore) Load(ctx context.Context) (*Checkpoint, error) {
	data, err := cs.db.Get(ctx, cs.checkpointKey)
	if err != nil {
		if errors.Is(err, ds.ErrNotFound) {
			return nil, ErrCheckpointNotFound
		}
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	pbCheckpoint := &pb.SequencerDACheckpoint{}
	if err := proto.Unmarshal(data, pbCheckpoint); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}

	return &Checkpoint{
		DAHeight: pbCheckpoint.DaHeight,
		TxIndex:  pbCheckpoint.TxIndex,
	}, nil
}

// Save persists the checkpoint to the datastore
func (cs *CheckpointStore) Save(ctx context.Context, checkpoint *Checkpoint) error {
	pbCheckpoint := &pb.SequencerDACheckpoint{
		DaHeight: checkpoint.DAHeight,
		TxIndex:  checkpoint.TxIndex,
	}

	data, err := proto.Marshal(pbCheckpoint)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	if err := cs.db.Put(ctx, cs.checkpointKey, data); err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	return nil
}

// Delete removes the checkpoint from the datastore
func (cs *CheckpointStore) Delete(ctx context.Context) error {
	if err := cs.db.Delete(ctx, cs.checkpointKey); err != nil {
		if errors.Is(err, ds.ErrNotFound) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}
	return nil
}
