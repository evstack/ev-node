package single

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"

	ds "github.com/ipfs/go-datastore"
	ktds "github.com/ipfs/go-datastore/keytransform"
	"github.com/ipfs/go-datastore/query"
	"google.golang.org/protobuf/proto"

	coresequencer "github.com/evstack/ev-node/core/sequencer"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// DirectTXBatchQueue implements a persistent queue for transaction batches
type DirectTXBatchQueue struct {
	queue        []coresequencer.DirectTX
	maxQueueSize int // maximum number of batches allowed in queue (0 = unlimited)
	mu           sync.Mutex
	db           ds.Batching
}

// NewDirectTXBatchQueue creates a new DirectTXBatchQueue with the specified maximum size.
// If maxSize is 0, the queue will be unlimited.
func NewDirectTXBatchQueue(db ds.Batching, prefix string, maxSize int) *DirectTXBatchQueue {
	return &DirectTXBatchQueue{
		queue:        make([]coresequencer.DirectTX, 0),
		maxQueueSize: maxSize,
		db:           ktds.Wrap(db, ktds.PrefixTransform{Prefix: ds.NewKey(prefix)}),
	}
}

// AddBatch adds a new transaction to the queue and writes it to the WAL.
// Returns ErrQueueFull if the queue has reached its maximum size.
func (bq *DirectTXBatchQueue) AddBatch(ctx context.Context, txs ...coresequencer.DirectTX) error {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	// Check if queue is full (maxQueueSize of 0 means unlimited)
	if bq.maxQueueSize > 0 && len(bq.queue) >= bq.maxQueueSize {
		return ErrQueueFull
	}
	for _, tx := range txs {
		pbBatch := &pb.DirectTX{
			Id:              tx.ID,
			Tx:              tx.TX,
			FirstSeenHeight: tx.FirstSeenHeight,
			FirstSeenTime:   tx.FirstSeenTime,
		}

		encodedBatch, err := proto.Marshal(pbBatch)
		if err != nil {
			return err
		}

		// First write to DB for durability
		hash, err := tx.Hash()
		if err != nil {
			return err
		}
		key := hex.EncodeToString(hash)

		if err := bq.db.Put(ctx, ds.NewKey(key), encodedBatch); err != nil {
			return err
		}

		// Then add to in-memory queue
		bq.queue = append(bq.queue, tx)
	}
	return nil
}

// Peek returns the first transaction in the queue without removing it
// Returns nil and no error if queue is empty
func (bq *DirectTXBatchQueue) Peek(ctx context.Context) (*coresequencer.DirectTX, error) {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	if len(bq.queue) == 0 {
		return nil, nil
	}

	batch := bq.queue[0]
	return &batch, nil
}

// Next extracts a batch of transactions from the queue and marks it as processed in the WAL
func (bq *DirectTXBatchQueue) Next(ctx context.Context) (*coresequencer.DirectTX, error) {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	if len(bq.queue) == 0 {
		return nil, nil
	}

	batch := bq.queue[0]
	bq.queue = bq.queue[1:]

	hash, err := batch.Hash()
	if err != nil {
		return nil, err
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
func (bq *DirectTXBatchQueue) Load(ctx context.Context) error {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	// Clear the current queue
	bq.queue = make([]coresequencer.DirectTX, 0)

	q := query.Query{}
	results, err := bq.db.Query(ctx, q)
	if err != nil {
		return fmt.Errorf("error querying datastore: %w", err)
	}
	defer results.Close()

	// Load each direct tx
	for result := range results.Next() {
		if result.Error != nil {
			fmt.Printf("Error reading entry from datastore: %v\n", result.Error)
			continue
		}
		var pbBatch pb.DirectTX
		err := proto.Unmarshal(result.Value, &pbBatch)
		if err != nil {
			fmt.Printf("Error decoding batch for key '%s': %v. Skipping entry.\n", result.Key, err)
			continue
		}
		bq.queue = append(bq.queue, coresequencer.DirectTX{
			ID:              pbBatch.Id,
			TX:              pbBatch.Tx,
			FirstSeenHeight: pbBatch.FirstSeenHeight,
			FirstSeenTime:   pbBatch.FirstSeenTime,
		})
	}

	return nil
}
