package based

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"

	ds "github.com/ipfs/go-datastore"
	ktds "github.com/ipfs/go-datastore/keytransform"
	"github.com/ipfs/go-datastore/query"
	"google.golang.org/protobuf/proto"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// ErrQueueFull is returned when the transaction queue has reached its maximum size
var ErrQueueFull = errors.New("transaction queue is full")

func newPrefixKV(kvStore ds.Batching, prefix string) ds.Batching {
	return ktds.Wrap(kvStore, ktds.PrefixTransform{Prefix: ds.NewKey(prefix)})
}

// TxQueue implements a persistent queue for transactions
type TxQueue struct {
	queue        [][]byte
	head         int // index of the first element in the queue
	maxQueueSize int // maximum number of transactions allowed in queue (0 = unlimited)
	mu           sync.Mutex
	db           ds.Batching
}

// NewTxQueue creates a new TxQueue with the specified maximum size.
// If maxSize is 0, the queue will be unlimited.
func NewTxQueue(db ds.Batching, prefix string, maxSize int) *TxQueue {
	return &TxQueue{
		queue:        make([][]byte, 0),
		head:         0,
		maxQueueSize: maxSize,
		db:           newPrefixKV(db, prefix),
	}
}

// Add adds a new transaction to the queue and writes it to the DB.
// Returns ErrQueueFull if the queue has reached its maximum size.
func (tq *TxQueue) Add(ctx context.Context, tx []byte) error {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	// Check if queue is full (maxQueueSize of 0 means unlimited)
	// Use effective queue size (total length minus processed head items)
	effectiveSize := len(tq.queue) - tq.head
	if tq.maxQueueSize > 0 && effectiveSize >= tq.maxQueueSize {
		return ErrQueueFull
	}

	// Generate a unique key for this transaction
	// Use a combination of queue position and transaction hash
	key := fmt.Sprintf("tx_%d_%s", len(tq.queue), hex.EncodeToString(tx[:min(32, len(tx))]))

	pbTx := &pb.Tx{
		Data: tx,
	}

	encodedTx, err := proto.Marshal(pbTx)
	if err != nil {
		return err
	}

	// First write to DB for durability
	if err := tq.db.Put(ctx, ds.NewKey(key), encodedTx); err != nil {
		return err
	}

	// Then add to in-memory queue
	tq.queue = append(tq.queue, tx)

	return nil
}

// AddBatch adds multiple transactions to the queue in a single operation
func (tq *TxQueue) AddBatch(ctx context.Context, txs [][]byte) error {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	// Check if adding these transactions would exceed the queue size
	effectiveSize := len(tq.queue) - tq.head
	if tq.maxQueueSize > 0 && effectiveSize+len(txs) > tq.maxQueueSize {
		return ErrQueueFull
	}

	// Use a batch operation for efficiency
	batch, err := tq.db.Batch(ctx)
	if err != nil {
		return fmt.Errorf("failed to create batch: %w", err)
	}

	for i, tx := range txs {
		// Generate a unique key for this transaction
		key := fmt.Sprintf("tx_%d_%s", len(tq.queue)+i, hex.EncodeToString(tx[:min(32, len(tx))]))

		pbTx := &pb.Tx{
			Data: tx,
		}

		encodedTx, err := proto.Marshal(pbTx)
		if err != nil {
			return err
		}

		if err := batch.Put(ctx, ds.NewKey(key), encodedTx); err != nil {
			return err
		}
	}

	// Commit the batch
	if err := batch.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}

	// Then add to in-memory queue
	tq.queue = append(tq.queue, txs...)

	return nil
}

// Next extracts a transaction from the queue and marks it as processed in the DB
func (tq *TxQueue) Next(ctx context.Context) ([]byte, error) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	// Check if queue is empty
	if tq.head >= len(tq.queue) {
		return nil, nil
	}

	tx := tq.queue[tq.head]
	key := fmt.Sprintf("tx_%d_%s", tq.head, hex.EncodeToString(tx[:min(32, len(tx))]))

	tq.queue[tq.head] = nil // Release memory for the dequeued element
	tq.head++

	// Compact when head gets too large to prevent memory leaks
	// Only compact when we have significant waste (more than half processed)
	// and when we have a reasonable number of processed items to avoid
	// frequent compactions on small queues
	if tq.head > len(tq.queue)/2 && tq.head > 100 {
		remaining := copy(tq.queue, tq.queue[tq.head:])
		// Zero out the rest of the slice to release memory
		for i := remaining; i < len(tq.queue); i++ {
			tq.queue[i] = nil
		}
		tq.queue = tq.queue[:remaining]
		tq.head = 0
	}

	// Delete the transaction from the DB since it's been processed
	err := tq.db.Delete(ctx, ds.NewKey(key))
	if err != nil {
		// Log the error but continue
		fmt.Printf("Error deleting processed transaction: %v\n", err)
	}

	return tx, nil
}

// Peek returns transactions from the queue without removing them
// This is useful for creating batches without committing to dequeue
func (tq *TxQueue) Peek(maxBytes uint64) [][]byte {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	if tq.head >= len(tq.queue) {
		return nil
	}

	var result [][]byte
	var totalBytes uint64

	for i := tq.head; i < len(tq.queue); i++ {
		tx := tq.queue[i]
		txSize := uint64(len(tx))

		if totalBytes+txSize > maxBytes {
			break
		}

		result = append(result, tx)
		totalBytes += txSize
	}

	return result
}

// Consume removes the first n transactions from the queue
// This should be called after successfully processing transactions returned by Peek
func (tq *TxQueue) Consume(ctx context.Context, n int) error {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	if tq.head+n > len(tq.queue) {
		return errors.New("cannot consume more transactions than available")
	}

	// Delete from DB
	for i := 0; i < n; i++ {
		tx := tq.queue[tq.head+i]
		key := fmt.Sprintf("tx_%d_%s", tq.head+i, hex.EncodeToString(tx[:min(32, len(tx))]))

		if err := tq.db.Delete(ctx, ds.NewKey(key)); err != nil {
			fmt.Printf("Error deleting consumed transaction: %v\n", err)
		}

		tq.queue[tq.head+i] = nil // Release memory
	}

	tq.head += n

	// Compact if needed
	if tq.head > len(tq.queue)/2 && tq.head > 100 {
		remaining := copy(tq.queue, tq.queue[tq.head:])
		for i := remaining; i < len(tq.queue); i++ {
			tq.queue[i] = nil
		}
		tq.queue = tq.queue[:remaining]
		tq.head = 0
	}

	return nil
}

// Load reloads all transactions from DB into the in-memory queue after a crash or restart
func (tq *TxQueue) Load(ctx context.Context) error {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	// Clear the current queue
	tq.queue = make([][]byte, 0)
	tq.head = 0

	q := query.Query{}
	results, err := tq.db.Query(ctx, q)
	if err != nil {
		return fmt.Errorf("error querying datastore: %w", err)
	}
	defer results.Close()

	// Collect all entries with their keys
	type entry struct {
		key string
		tx  []byte
	}
	var entries []entry

	// Load each transaction
	for result := range results.Next() {
		if result.Error != nil {
			fmt.Printf("Error reading entry from datastore: %v\n", result.Error)
			continue
		}
		pbTx := &pb.Tx{}
		err := proto.Unmarshal(result.Value, pbTx)
		if err != nil {
			fmt.Printf("Error decoding transaction for key '%s': %v. Skipping entry.\n", result.Key, err)
			continue
		}
		entries = append(entries, entry{key: result.Key, tx: pbTx.Data})
	}

	// Sort entries by key to maintain FIFO order
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].key < entries[j].key
	})

	// Add sorted transactions to queue
	for _, e := range entries {
		tq.queue = append(tq.queue, e.tx)
	}

	return nil
}

// Size returns the effective number of transactions in the queue
// This method is primarily for testing and monitoring purposes
func (tq *TxQueue) Size() int {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	return len(tq.queue) - tq.head
}

// Clear removes all transactions from the queue and DB
func (tq *TxQueue) Clear(ctx context.Context) error {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	// Delete all entries from DB
	q := query.Query{KeysOnly: true}
	results, err := tq.db.Query(ctx, q)
	if err != nil {
		return fmt.Errorf("error querying datastore: %w", err)
	}
	defer results.Close()

	for result := range results.Next() {
		if result.Error != nil {
			fmt.Printf("Error reading key from datastore: %v\n", result.Error)
			continue
		}
		if err := tq.db.Delete(ctx, ds.NewKey(result.Key)); err != nil {
			fmt.Printf("Error deleting key '%s': %v\n", result.Key, err)
		}
	}

	// Clear in-memory queue
	tq.queue = make([][]byte, 0)
	tq.head = 0

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
