package single

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
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

// initialSeqNum is the starting sequence number for new queues.
// It is set to the middle of the uint64 range to allow for both
// appending (incrementing) and prepending (decrementing) transactions.
const initialSeqNum = uint64(0x8000000000000000)

// queuedItem holds a batch and its associated persistence key
type queuedItem struct {
	Batch coresequencer.Batch
	Key   string
}

// BatchQueue implements a persistent queue for transaction batches
type BatchQueue struct {
	queue        []queuedItem
	head         int // index of the first element in the queue
	maxQueueSize int // maximum number of batches allowed in queue (0 = unlimited)

	// Sequence numbers for generating new keys
	nextAddSeq     uint64
	nextPrependSeq uint64

	mu sync.Mutex
	db ds.Batching
}

// NewBatchQueue creates a new BatchQueue with the specified maximum size.
// If maxSize is 0, the queue will be unlimited.
func NewBatchQueue(db ds.Batching, prefix string, maxSize int) *BatchQueue {
	return &BatchQueue{
		queue:          make([]queuedItem, 0),
		head:           0,
		maxQueueSize:   maxSize,
		db:             store.NewPrefixKVStore(db, prefix),
		nextAddSeq:     initialSeqNum,
		nextPrependSeq: initialSeqNum - 1,
	}
}

// seqToKey converts a sequence number to a hex-encoded big-endian key.
// We use big-endian so that lexicographical sort order matches numeric order.
func seqToKey(seq uint64) string {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, seq)
	return hex.EncodeToString(b)
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

	key := seqToKey(bq.nextAddSeq)
	if err := bq.persistBatch(ctx, batch, key); err != nil {
		return err
	}
	bq.nextAddSeq++

	// Then add to in-memory queue
	bq.queue = append(bq.queue, queuedItem{Batch: batch, Key: key})

	return nil
}

// Prepend adds a batch to the front of the queue (before head position).
// This is used to return transactions that couldn't fit in the current batch.
// The batch is persisted to the DB to ensure durability in case of crashes.
//
// NOTE: Prepend intentionally bypasses the maxQueueSize limit to ensure high-priority
// transactions can always be re-queued.
func (bq *BatchQueue) Prepend(ctx context.Context, batch coresequencer.Batch) error {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	key := seqToKey(bq.nextPrependSeq)
	if err := bq.persistBatch(ctx, batch, key); err != nil {
		return err
	}
	bq.nextPrependSeq--

	item := queuedItem{Batch: batch, Key: key}

	// Then add to in-memory queue
	// If we have room before head, use it
	if bq.head > 0 {
		bq.head--
		bq.queue[bq.head] = item
	} else {
		// Need to expand the queue at the front
		bq.queue = append([]queuedItem{item}, bq.queue...)
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

	item := bq.queue[bq.head]
	// Release memory for the dequeued element
	bq.queue[bq.head] = queuedItem{}
	bq.head++

	// Compact when head gets too large to prevent memory leaks
	// Only compact when we have significant waste (more than half processed)
	// and when we have a reasonable number of processed items to avoid
	// frequent compactions on small queues
	if bq.head > len(bq.queue)/2 && bq.head > 100 {
		remaining := copy(bq.queue, bq.queue[bq.head:])
		// Zero out the rest of the slice
		for i := remaining; i < len(bq.queue); i++ {
			bq.queue[i] = queuedItem{}
		}
		bq.queue = bq.queue[:remaining]
		bq.head = 0
	}

	// Delete the batch from the WAL since it's been processed
	// Use the stored key directly
	if err := bq.db.Delete(ctx, ds.NewKey(item.Key)); err != nil {
		// Log the error but continue
		fmt.Printf("Error deleting processed batch: %v\n", err)
	}

	return &item.Batch, nil
}

// Load reloads all batches from WAL file into the in-memory queue after a crash or restart
func (bq *BatchQueue) Load(ctx context.Context) error {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	// Clear the current queue and reset sequences
	bq.queue = make([]queuedItem, 0)
	bq.head = 0
	bq.nextAddSeq = initialSeqNum
	bq.nextPrependSeq = initialSeqNum - 1

	q := query.Query{
		Orders: []query.Order{query.OrderByKey{}},
	}
	results, err := bq.db.Query(ctx, q)
	if err != nil {
		return fmt.Errorf("error querying datastore: %w", err)
	}
	defer results.Close()

	var legacyItems []queuedItem
	for result := range results.Next() {
		if result.Error != nil {
			fmt.Printf("Error reading entry from datastore: %v\n", result.Error)
			continue
		}
		// We care about the last part of the key (the sequence number)
		// ds.Key usually has a leading slash.
		keyName := ds.NewKey(result.Key).Name()

		var pbBatch pb.Batch
		err := proto.Unmarshal(result.Value, &pbBatch)
		if err != nil {
			fmt.Printf("Error decoding batch for key '%s': %v. Skipping entry.\n", keyName, err)
			continue
		}

		batch := coresequencer.Batch{Transactions: pbBatch.Txs}

		// Check if key is valid hex sequence number (16 hex chars)
		// We use strict 16 check because seqToKey always produces 16 hex chars.
		isValid := false
		if len(keyName) == 16 {
			if seq, err := strconv.ParseUint(keyName, 16, 64); err == nil {
				isValid = true
				if seq >= bq.nextAddSeq {
					bq.nextAddSeq = seq + 1
				}
				if seq <= bq.nextPrependSeq {
					bq.nextPrependSeq = seq - 1
				}
			}
		}
		if isValid {
			bq.queue = append(bq.queue, queuedItem{Batch: batch, Key: keyName})
		} else {
			legacyItems = append(legacyItems, queuedItem{Batch: batch, Key: result.Key})
		}
	}
	if len(legacyItems) == 0 {
		return nil
	}
	fmt.Printf("Found %d legacy items to migrate...\n", len(legacyItems))

	for _, item := range legacyItems {
		newKeyName := seqToKey(bq.nextAddSeq)

		if err := bq.persistBatch(ctx, item.Batch, newKeyName); err != nil {
			fmt.Printf("Failed to migrate legacy item %s: %v\n", item.Key, err)
			continue
		}

		if err := bq.db.Delete(ctx, ds.NewKey(item.Key)); err != nil {
			fmt.Printf("Failed to delete legacy key %s after migration: %v\n", item.Key, err)
		}

		bq.queue = append(bq.queue, queuedItem{Batch: item.Batch, Key: newKeyName})
		bq.nextAddSeq++
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

// persistBatch persists a batch to the datastore with the given key
func (bq *BatchQueue) persistBatch(ctx context.Context, batch coresequencer.Batch, key string) error {
	pbBatch := &pb.Batch{
		Txs: batch.Transactions,
	}

	encodedBatch, err := proto.Marshal(pbBatch)
	if err != nil {
		return err
	}

	// Write to DB
	if err := bq.db.Put(ctx, ds.NewKey(key), encodedBatch); err != nil {
		return err
	}

	return nil
}
