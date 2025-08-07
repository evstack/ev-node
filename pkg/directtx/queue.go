package directtx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	ds "github.com/ipfs/go-datastore"
	ktds "github.com/ipfs/go-datastore/keytransform"
	"github.com/ipfs/go-datastore/query"
	"google.golang.org/protobuf/proto"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

var ErrQueueFull = errors.New("direct TX queue is full")

// TXQueue implements a persistent queue for direct transactions.
// It is backed by a datastore for durability.
type TXQueue struct {
	maxQueueSize int
	db           ds.Batching
	mu           sync.Mutex
	queue        []DirectTX
}

// NewTXQueue creates a new TXQueue with the specified maximum size.
func NewTXQueue(db ds.Batching, prefix string, maxSize int) *TXQueue {
	if maxSize <= 0 {
		panic("maxSize must be greater than 0")
	}
	return &TXQueue{
		queue:        make([]DirectTX, 0),
		maxQueueSize: maxSize,
		db:           ktds.Wrap(db, ktds.PrefixTransform{Prefix: ds.NewKey(prefix)}),
	}
}

// AddBatch adds a new transaction to the queue and writes it to the WAL.
// Returns ErrQueueFull if the queue has reached its maximum size.
func (bq *TXQueue) AddBatch(ctx context.Context, txs ...DirectTX) error {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	// Check if queue is full (maxQueueSize of 0 means unlimited)
	if bq.maxQueueSize > 0 && len(bq.queue) >= bq.maxQueueSize {
		return ErrQueueFull
	}
	for _, tx := range txs {
		pbTX := &pb.DirectTX{
			Id:              tx.ID,
			Tx:              tx.TX,
			FirstSeenHeight: tx.FirstSeenHeight,
			FirstSeenTime:   tx.FirstSeenTime,
		}

		bz, err := proto.Marshal(pbTX)
		if err != nil {
			return err
		}

		// First write to DB for durability
		if err := bq.db.Put(ctx, buildStoreKey(tx), bz); err != nil {
			return err
		}

		// Then add to in-memory queue
		bq.queue = append(bq.queue, tx)
	}
	return nil
}

// Peek returns the first transaction in the queue without removing it
// Returns nil and no error if queue is empty
func (bq *TXQueue) Peek(_ context.Context) (*DirectTX, error) {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	if len(bq.queue) == 0 {
		return nil, nil
	}
	return &bq.queue[0], nil
}

// Next extracts a TX from the queue and marks it as processed in the store
func (bq *TXQueue) Next(ctx context.Context) (*DirectTX, error) {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	if len(bq.queue) == 0 {
		return nil, nil
	}

	tx := bq.queue[0]
	bq.queue = bq.queue[1:]

	if err := bq.deleteInDB(ctx, tx); err != nil {
		return nil, err
	}

	return &tx, nil
}

func (bq *TXQueue) delete(ctx context.Context, srcTx []byte) error {
	bq.mu.Lock()
	defer bq.mu.Unlock()
	for i, d := range bq.queue {
		if bytes.Equal(d.TX, srcTx) {
			if err := bq.deleteInDB(ctx, d); err != nil {
				return err
			}
			bq.queue = append(bq.queue[:i], bq.queue[i+1:]...)
			return nil // delete oldest only
		}
	}
	return nil

}

func (bq *TXQueue) deleteInDB(ctx context.Context, dTX DirectTX) error {
	err := bq.db.Delete(ctx, buildStoreKey(dTX))
	if err != nil {
		return fmt.Errorf("deleting processed direct tx: %w", err)
	}
	return nil
}

// Load reloads all direct tx from WAL file into the in-memory queue after a crash or restart
func (bq *TXQueue) Load(ctx context.Context, callbacks ...func(tx DirectTX)) error {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	// Clear the current queue
	bq.queue = make([]DirectTX, 0)

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
		var pbTX pb.DirectTX
		err := proto.Unmarshal(result.Value, &pbTX)
		if err != nil {
			fmt.Printf("Error decoding tx for key '%s': %v. Skipping entry.\n", result.Key, err)
			continue
		}
		tx := DirectTX{
			ID:              pbTX.Id,
			TX:              pbTX.Tx,
			FirstSeenHeight: pbTX.FirstSeenHeight,
			FirstSeenTime:   pbTX.FirstSeenTime,
		}
		for _, f := range callbacks {
			f(tx)
		}
		bq.queue = append(bq.queue, tx)
	}

	return nil
}

// Size returns the current number of items in the queue
func (bq *TXQueue) Size() int {
	bq.mu.Lock()
	defer bq.mu.Unlock()
	return len(bq.queue)
}
