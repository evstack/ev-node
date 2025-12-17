package single

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"google.golang.org/protobuf/proto"

	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/store"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// ErrQueueFull is returned when the batch queue has reached its maximum size
var ErrQueueFull = errors.New("batch queue is full")

// BatchQueue implements a persistent queue for transaction batches
type BatchQueue struct {
	queue        []coresequencer.Batch
	head         int // index of the first element in the queue
	maxQueueSize int // maximum number of batches allowed in queue (0 = unlimited)
	mu           sync.Mutex
	db           ds.Batching
}

// NewBatchQueue creates a new BatchQueue with the specified maximum size.
// If maxSize is 0, the queue will be unlimited.
func NewBatchQueue(db ds.Batching, prefix string, maxSize int) *BatchQueue {
	return &BatchQueue{
		queue:        make([]coresequencer.Batch, 0),
		head:         0,
		maxQueueSize: maxSize,
		db:           store.NewPrefixKVStore(db, prefix),
	}
}

// AddBatch adds a new transaction to the queue and writes it to the WAL.
// Returns ErrQueueFull if the queue has reached its maximum size.
func (bq *BatchQueue) AddBatch(ctx context.Context, batch coresequencer.Batch) error {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	// Check if queue is full (maxQueueSize of 0 means unlimited)
	// Use effective queue size (total length minus processed head items)
	effectiveSize := len(bq.queue) - bq.head
	if bq.maxQueueSize > 0 && effectiveSize >= bq.maxQueueSize {
		return ErrQueueFull
	}

	if err := bq.persistBatch(ctx, batch); err != nil {
		return err
	}

	// Then add to in-memory queue
	bq.queue = append(bq.queue, batch)

	return nil
}

// Prepend adds a batch to the front of the queue (before head position).
// This is used to return transactions that couldn't fit in the current batch.
// The batch is persisted to the DB to ensure durability in case of crashes.
//
// NOTE: Prepend intentionally bypasses the maxQueueSize limit to ensure high-priority
// transactions can always be re-queued. This means the effective queue size may temporarily
// exceed the configured maximum when Prepend is used. This is by design to prevent loss
// of transactions that have already been accepted but couldn't fit in the current batch.
func (bq *BatchQueue) Prepend(ctx context.Context, batch coresequencer.Batch) error {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	if err := bq.persistBatch(ctx, batch); err != nil {
		return err
	}

	// Then add to in-memory queue
	// If we have room before head, use it
	if bq.head > 0 {
		bq.head--
		bq.queue[bq.head] = batch
	} else {
		// Need to expand the queue at the front
		bq.queue = append([]coresequencer.Batch{batch}, bq.queue...)
	}

	return nil
}

// Next extracts a batch of transactions from the queue and marks it as processed in the WAL
func (bq *BatchQueue) Next(ctx context.Context) (*coresequencer.Batch, error) {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	// Check if queue is empty
	if bq.head >= len(bq.queue) {
		return &coresequencer.Batch{Transactions: nil}, nil
	}

	batch := bq.queue[bq.head]
	bq.queue[bq.head] = coresequencer.Batch{} // Release memory for the dequeued element
	bq.head++

	// Compact when head gets too large to prevent memory leaks
	// Only compact when we have significant waste (more than half processed)
	// and when we have a reasonable number of processed items to avoid
	// frequent compactions on small queues
	if bq.head > len(bq.queue)/2 && bq.head > 100 {
		remaining := copy(bq.queue, bq.queue[bq.head:])
		// Zero out the rest of the slice to release memory
		for i := remaining; i < len(bq.queue); i++ {
			bq.queue[i] = coresequencer.Batch{}
		}
		bq.queue = bq.queue[:remaining]
		bq.head = 0
	}

	hash, err := batch.Hash()
	if err != nil {
		return &coresequencer.Batch{Transactions: nil}, err
	}
	key := hex.EncodeToString(hash)

	// Delete the batch from the WAL since it's been processed
	err = bq.db.Delete(ctx, ds.NewKey(key))
	if err != nil {
		// Log the error but continue
		fmt.Printf("Error deleting processed batch: %v\n", err)
	}

	return &batch, nil
}

// Load reloads all batches from WAL file into the in-memory queue after a crash or restart
func (bq *BatchQueue) Load(ctx context.Context) error {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	// Clear the current queue
	bq.queue = make([]coresequencer.Batch, 0)
	bq.head = 0

	q := query.Query{}
	results, err := bq.db.Query(ctx, q)
	if err != nil {
		return fmt.Errorf("error querying datastore: %w", err)
	}
	defer results.Close()

	// Load each batch
	for result := range results.Next() {
		if result.Error != nil {
			fmt.Printf("Error reading entry from datastore: %v\n", result.Error)
			continue
		}
		pbBatch := &pb.Batch{}
		err := proto.Unmarshal(result.Value, pbBatch)
		if err != nil {
			fmt.Printf("Error decoding batch for key '%s': %v. Skipping entry.\n", result.Key, err)
			continue
		}
		bq.queue = append(bq.queue, coresequencer.Batch{Transactions: pbBatch.Txs})
	}

	return nil
}

// Size returns the effective number of batches in the queue
// This method is primarily for testing and monitoring purposes
func (bq *BatchQueue) Size() int {
	bq.mu.Lock()
	defer bq.mu.Unlock()
	return len(bq.queue) - bq.head
}

// persistBatch persists a batch to the datastore
func (bq *BatchQueue) persistBatch(ctx context.Context, batch coresequencer.Batch) error {
	hash, err := batch.Hash()
	if err != nil {
		return err
	}
	key := hex.EncodeToString(hash)

	pbBatch := &pb.Batch{
		Txs: batch.Transactions,
	}

	encodedBatch, err := proto.Marshal(pbBatch)
	if err != nil {
		return err
	}

	// First write to DB for durability
	if err := bq.db.Put(ctx, ds.NewKey(key), encodedBatch); err != nil {
		return err
	}

	return nil
}
