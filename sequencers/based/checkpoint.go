package based

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"google.golang.org/protobuf/proto"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

var (
	// checkpointKey is the datastore key for persisting the checkpoint
	checkpointKey = ds.NewKey("/based/checkpoint")

	// ErrCheckpointNotFound is returned when no checkpoint exists in the datastore
	ErrCheckpointNotFound = errors.New("checkpoint not found")
)

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
	db ds.Batching
}

// NewCheckpointStore creates a new checkpoint store
func NewCheckpointStore(db ds.Batching) *CheckpointStore {
	return &CheckpointStore{
		db: db,
	}
}

// Load loads the checkpoint from the datastore
// Returns ErrCheckpointNotFound if no checkpoint exists
func (cs *CheckpointStore) Load(ctx context.Context) (*Checkpoint, error) {
	data, err := cs.db.Get(ctx, checkpointKey)
	if err != nil {
		if errors.Is(err, ds.ErrNotFound) {
			return nil, ErrCheckpointNotFound
		}
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	pbCheckpoint := &pb.BasedCheckpoint{}
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
	pbCheckpoint := &pb.BasedCheckpoint{
		DaHeight: checkpoint.DAHeight,
		TxIndex:  checkpoint.TxIndex,
	}

	data, err := proto.Marshal(pbCheckpoint)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	if err := cs.db.Put(ctx, checkpointKey, data); err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	return nil
}

// Delete removes the checkpoint from the datastore
func (cs *CheckpointStore) Delete(ctx context.Context) error {
	if err := cs.db.Delete(ctx, checkpointKey); err != nil {
		if errors.Is(err, ds.ErrNotFound) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}
	return nil
}

// Legacy key format for migration from old queue-based implementation
// This allows us to detect and clean up old queue data
func isLegacyQueueKey(key ds.Key) bool {
	// Old queue keys had format "/based_txs/tx_..."
	return key.String() != checkpointKey.String() &&
		len(key.String()) > 0
}

// CleanupLegacyQueue removes all legacy queue entries from the datastore
// This should be called during migration from the old queue-based implementation
func (cs *CheckpointStore) CleanupLegacyQueue(ctx context.Context) error {
	// Query all keys in the datastore
	results, err := cs.db.Query(ctx, query.Query{KeysOnly: true})
	if err != nil {
		return fmt.Errorf("failed to query datastore: %w", err)
	}
	defer results.Close()

	deletedCount := 0
	for result := range results.Next() {
		if result.Error != nil {
			continue
		}

		key := ds.NewKey(result.Key)
		// Only delete keys that are not the checkpoint
		if key.String() != checkpointKey.String() {
			if err := cs.db.Delete(ctx, key); err != nil {
				// Log but continue - best effort cleanup
				continue
			}
			deletedCount++
		}
	}

	return nil
}

// Helper function to encode uint64 to bytes for potential future use
func encodeUint64(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

// Helper function to decode bytes to uint64 for potential future use
func decodeUint64(b []byte) (uint64, error) {
	if len(b) != 8 {
		return 0, fmt.Errorf("invalid length for uint64: got %d, expected 8", len(b))
	}
	return binary.BigEndian.Uint64(b), nil
}
