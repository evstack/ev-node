package evm

import (
	"bytes"
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/syndtr/goleveldb/leveldb"
)

// deleteRangeBatchSize is the number of keys to delete in a single batch
// to avoid holding locks for too long.
const deleteRangeBatchSize = 1000

var _ ethdb.KeyValueStore = &wrapper{}

type wrapper struct {
	ds datastore.Batching
}

func keyToDatastoreKey(key []byte) datastore.Key {
	return datastore.NewKey(string(key))
}

func datastoreKeyToBytes(key string) []byte {
	// datastore keys have a leading slash, remove it
	if strings.HasPrefix(key, "/") {
		return []byte(key[1:])
	}
	return []byte(key)
}

// Close implements ethdb.KeyValueStore.
func (w *wrapper) Close() error {
	return w.ds.Close()
}

// Compact implements ethdb.KeyValueStore.
func (w *wrapper) Compact(start []byte, limit []byte) error {
	// Compaction is not supported by go-datastore, this is a no-op
	return nil
}

// Delete implements ethdb.KeyValueStore.
func (w *wrapper) Delete(key []byte) error {
	return w.ds.Delete(context.Background(), keyToDatastoreKey(key))
}

// DeleteRange implements ethdb.KeyValueStore.
// Optimized to use prefix-based querying when possible and batch deletions.
func (w *wrapper) DeleteRange(start []byte, end []byte) error {
	ctx := context.Background()

	// Find common prefix between start and end to narrow the query
	prefix := commonPrefix(start, end)

	q := query.Query{
		KeysOnly: true,
	}
	if len(prefix) > 0 {
		q.Prefix = "/" + string(prefix)
	}

	results, err := w.ds.Query(ctx, q)
	if err != nil {
		return err
	}
	defer results.Close()

	// Collect keys to delete in batches
	var keysToDelete []datastore.Key
	for result := range results.Next() {
		if result.Error != nil {
			return result.Error
		}
		keyBytes := datastoreKeyToBytes(result.Entry.Key)
		if bytes.Compare(keyBytes, start) >= 0 && bytes.Compare(keyBytes, end) < 0 {
			keysToDelete = append(keysToDelete, datastore.NewKey(result.Entry.Key))

			// Process in batches to avoid holding too many keys in memory
			if len(keysToDelete) >= deleteRangeBatchSize {
				if err := w.deleteBatch(ctx, keysToDelete); err != nil {
					return err
				}
				keysToDelete = keysToDelete[:0]
			}
		}
	}

	// Delete remaining keys
	if len(keysToDelete) > 0 {
		return w.deleteBatch(ctx, keysToDelete)
	}
	return nil
}

// deleteBatch deletes a batch of keys using a batched operation.
func (w *wrapper) deleteBatch(ctx context.Context, keys []datastore.Key) error {
	batch, err := w.ds.Batch(ctx)
	if err != nil {
		// Fallback to individual deletes if batching not supported
		for _, key := range keys {
			if err := w.ds.Delete(ctx, key); err != nil {
				return err
			}
		}
		return nil
	}

	for _, key := range keys {
		if err := batch.Delete(ctx, key); err != nil {
			return err
		}
	}
	return batch.Commit(ctx)
}

// commonPrefix returns the common prefix between two byte slices.
func commonPrefix(a, b []byte) []byte {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	var i int
	for i = 0; i < minLen; i++ {
		if a[i] != b[i] {
			break
		}
	}
	return a[:i]
}

// Get implements ethdb.KeyValueStore.
func (w *wrapper) Get(key []byte) ([]byte, error) {
	val, err := w.ds.Get(context.Background(), keyToDatastoreKey(key))
	if err == datastore.ErrNotFound {
		return nil, leveldb.ErrNotFound
	}
	return val, err
}

// Has implements ethdb.KeyValueStore.
func (w *wrapper) Has(key []byte) (bool, error) {
	return w.ds.Has(context.Background(), keyToDatastoreKey(key))
}

// NewBatch implements ethdb.KeyValueStore.
func (w *wrapper) NewBatch() ethdb.Batch {
	return &batchWrapper{
		ds:   w.ds,
		ops:  nil,
		size: 0,
	}
}

// NewBatchWithSize implements ethdb.KeyValueStore.
func (w *wrapper) NewBatchWithSize(size int) ethdb.Batch {
	return &batchWrapper{
		ds:   w.ds,
		ops:  make([]batchOp, 0, size),
		size: 0,
	}
}

// NewIterator implements ethdb.KeyValueStore.
func (w *wrapper) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return newIterator(w.ds, prefix, start, 0)
}

// NewIteratorWithSize creates an iterator with an estimated size hint for buffer allocation.
func (w *wrapper) NewIteratorWithSize(prefix []byte, start []byte, sizeHint int) ethdb.Iterator {
	return newIterator(w.ds, prefix, start, sizeHint)
}

// Put implements ethdb.KeyValueStore.
func (w *wrapper) Put(key []byte, value []byte) error {
	return w.ds.Put(context.Background(), keyToDatastoreKey(key), value)
}

// Stat implements ethdb.KeyValueStore.
func (w *wrapper) Stat() (string, error) {
	return "go-datastore wrapper", nil
}

// SyncKeyValue implements ethdb.KeyValueStore.
func (w *wrapper) SyncKeyValue() error {
	return w.ds.Sync(context.Background(), datastore.NewKey("/"))
}

func NewEVMDB(ds datastore.Batching) ethdb.KeyValueStore {
	return &wrapper{ds}
}

// batchOp represents a single batch operation
type batchOp struct {
	key    []byte
	value  []byte // nil means delete
	delete bool
}

// batchWrapper implements ethdb.Batch
type batchWrapper struct {
	ds   datastore.Batching
	ops  []batchOp
	size int
	mu   sync.Mutex
}

var _ ethdb.Batch = &batchWrapper{}

// Put implements ethdb.Batch.
func (b *batchWrapper) Put(key []byte, value []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.ops = append(b.ops, batchOp{
		key:    append([]byte{}, key...),
		value:  append([]byte{}, value...),
		delete: false,
	})
	b.size += len(key) + len(value)
	return nil
}

// Delete implements ethdb.Batch.
func (b *batchWrapper) Delete(key []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.ops = append(b.ops, batchOp{
		key:    append([]byte{}, key...),
		value:  nil,
		delete: true,
	})
	b.size += len(key)
	return nil
}

// DeleteRange implements ethdb.Batch.
func (b *batchWrapper) DeleteRange(start []byte, end []byte) error {
	ctx := context.Background()

	// Use common prefix to narrow query
	prefix := commonPrefix(start, end)

	q := query.Query{KeysOnly: true}
	if len(prefix) > 0 {
		q.Prefix = "/" + string(prefix)
	}

	results, err := b.ds.Query(ctx, q)
	if err != nil {
		return err
	}
	defer results.Close()

	b.mu.Lock()
	defer b.mu.Unlock()

	for result := range results.Next() {
		if result.Error != nil {
			return result.Error
		}
		keyBytes := datastoreKeyToBytes(result.Entry.Key)
		if bytes.Compare(keyBytes, start) >= 0 && bytes.Compare(keyBytes, end) < 0 {
			b.ops = append(b.ops, batchOp{
				key:    append([]byte{}, keyBytes...),
				value:  nil,
				delete: true,
			})
			b.size += len(keyBytes)
		}
	}
	return nil
}

// ValueSize implements ethdb.Batch.
func (b *batchWrapper) ValueSize() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.size
}

// Write implements ethdb.Batch.
func (b *batchWrapper) Write() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	batch, err := b.ds.Batch(context.Background())
	if err != nil {
		return err
	}

	for _, op := range b.ops {
		if op.delete {
			if err := batch.Delete(context.Background(), keyToDatastoreKey(op.key)); err != nil {
				return err
			}
		} else {
			if err := batch.Put(context.Background(), keyToDatastoreKey(op.key), op.value); err != nil {
				return err
			}
		}
	}

	return batch.Commit(context.Background())
}

// Reset implements ethdb.Batch.
func (b *batchWrapper) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ops = b.ops[:0]
	b.size = 0
}

// Replay implements ethdb.Batch.
func (b *batchWrapper) Replay(w ethdb.KeyValueWriter) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, op := range b.ops {
		if op.delete {
			if err := w.Delete(op.key); err != nil {
				return err
			}
		} else {
			if err := w.Put(op.key, op.value); err != nil {
				return err
			}
		}
	}
	return nil
}

// iteratorWrapper implements ethdb.Iterator
// It provides sorted iteration over keys with optional prefix and start position.
type iteratorWrapper struct {
	entries []query.Entry // Pre-sorted entries for deterministic iteration
	index   int
	prefix  []byte
	start   []byte
	err     error
	closed  bool
	mu      sync.Mutex
}

var _ ethdb.Iterator = &iteratorWrapper{}

func newIterator(ds datastore.Batching, prefix []byte, start []byte, sizeHint int) *iteratorWrapper {
	q := query.Query{
		KeysOnly: false,
	}

	if len(prefix) > 0 {
		q.Prefix = "/" + string(prefix)
	}

	results, err := ds.Query(context.Background(), q)
	if err != nil {
		return &iteratorWrapper{
			err:    err,
			closed: false,
			index:  -1,
		}
	}

	// Collect and sort all entries for deterministic ordering
	// go-datastore doesn't guarantee order, but ethdb.Iterator expects sorted keys
	var entries []query.Entry
	if sizeHint > 0 {
		entries = make([]query.Entry, 0, sizeHint)
	}

	for result := range results.Next() {
		if result.Error != nil {
			results.Close()
			return &iteratorWrapper{
				err:    result.Error,
				closed: false,
				index:  -1,
			}
		}

		keyBytes := datastoreKeyToBytes(result.Entry.Key)

		// Filter by prefix
		if len(prefix) > 0 && !bytes.HasPrefix(keyBytes, prefix) {
			continue
		}

		// Filter by start position
		if len(start) > 0 && bytes.Compare(keyBytes, start) < 0 {
			continue
		}

		entries = append(entries, result.Entry)
	}
	results.Close()

	// Sort entries by key for deterministic iteration order
	sort.Slice(entries, func(i, j int) bool {
		keyI := datastoreKeyToBytes(entries[i].Key)
		keyJ := datastoreKeyToBytes(entries[j].Key)
		return bytes.Compare(keyI, keyJ) < 0
	})

	return &iteratorWrapper{
		entries: entries,
		prefix:  prefix,
		start:   start,
		index:   -1,
		closed:  false,
	}
}

// Next implements ethdb.Iterator.
func (it *iteratorWrapper) Next() bool {
	it.mu.Lock()
	defer it.mu.Unlock()

	if it.closed || it.err != nil {
		return false
	}

	it.index++
	return it.index < len(it.entries)
}

// Error implements ethdb.Iterator.
func (it *iteratorWrapper) Error() error {
	it.mu.Lock()
	defer it.mu.Unlock()
	return it.err
}

// Key implements ethdb.Iterator.
func (it *iteratorWrapper) Key() []byte {
	it.mu.Lock()
	defer it.mu.Unlock()

	if it.closed || it.index < 0 || it.index >= len(it.entries) {
		return nil
	}
	return datastoreKeyToBytes(it.entries[it.index].Key)
}

// Value implements ethdb.Iterator.
func (it *iteratorWrapper) Value() []byte {
	it.mu.Lock()
	defer it.mu.Unlock()

	if it.closed || it.index < 0 || it.index >= len(it.entries) {
		return nil
	}
	return it.entries[it.index].Value
}

// Release implements ethdb.Iterator.
func (it *iteratorWrapper) Release() {
	it.mu.Lock()
	defer it.mu.Unlock()

	if it.closed {
		return
	}
	it.closed = true
	// Clear entries to allow GC
	it.entries = nil
}
