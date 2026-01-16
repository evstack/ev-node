package evm

import (
	"bytes"
	"context"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/syndtr/goleveldb/leveldb"
)

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
func (w *wrapper) DeleteRange(start []byte, end []byte) error {
	// Query all keys and delete those in range
	q := query.Query{KeysOnly: true}
	results, err := w.ds.Query(context.Background(), q)
	if err != nil {
		return err
	}
	defer results.Close()

	for result := range results.Next() {
		if result.Error != nil {
			return result.Error
		}
		keyBytes := datastoreKeyToBytes(result.Entry.Key)
		if bytes.Compare(keyBytes, start) >= 0 && bytes.Compare(keyBytes, end) < 0 {
			if err := w.ds.Delete(context.Background(), datastore.NewKey(result.Entry.Key)); err != nil {
				return err
			}
		}
	}
	return nil
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
	return newIterator(w.ds, prefix, start)
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
	// Query all keys and mark those in range for deletion
	q := query.Query{KeysOnly: true}
	results, err := b.ds.Query(context.Background(), q)
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
type iteratorWrapper struct {
	results query.Results
	current query.Entry
	prefix  []byte
	start   []byte
	err     error
	started bool
	closed  bool
	mu      sync.Mutex
}

var _ ethdb.Iterator = &iteratorWrapper{}

func newIterator(ds datastore.Batching, prefix []byte, start []byte) *iteratorWrapper {
	q := query.Query{
		KeysOnly: false,
	}

	if len(prefix) > 0 {
		q.Prefix = "/" + string(prefix)
	}

	results, err := ds.Query(context.Background(), q)

	return &iteratorWrapper{
		results: results,
		prefix:  prefix,
		start:   start,
		err:     err,
		started: false,
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

	for {
		result, ok := it.results.NextSync()
		if !ok {
			return false
		}
		if result.Error != nil {
			it.err = result.Error
			return false
		}

		keyBytes := datastoreKeyToBytes(result.Entry.Key)

		// Check if key matches prefix (if prefix is set)
		if len(it.prefix) > 0 && !bytes.HasPrefix(keyBytes, it.prefix) {
			continue
		}

		// Check if key is >= start (if start is set)
		if len(it.start) > 0 && bytes.Compare(keyBytes, it.start) < 0 {
			continue
		}

		it.current = result.Entry
		it.started = true
		return true
	}
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

	if !it.started || it.closed {
		return nil
	}
	return datastoreKeyToBytes(it.current.Key)
}

// Value implements ethdb.Iterator.
func (it *iteratorWrapper) Value() []byte {
	it.mu.Lock()
	defer it.mu.Unlock()

	if !it.started || it.closed {
		return nil
	}
	return it.current.Value
}

// Release implements ethdb.Iterator.
func (it *iteratorWrapper) Release() {
	it.mu.Lock()
	defer it.mu.Unlock()

	if it.closed {
		return
	}
	it.closed = true
	if it.results != nil {
		it.results.Close()
	}
}
