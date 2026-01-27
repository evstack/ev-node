package evm

import (
	"bytes"
	"context"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/syndtr/goleveldb/leveldb"
)

var _ ethdb.KeyValueStore = (*wrapper)(nil)

type wrapper struct {
	ds datastore.Batching
}

// NewEVMDB creates an ethdb.KeyValueStore backed by a go-datastore.
func NewEVMDB(ds datastore.Batching) ethdb.KeyValueStore {
	return &wrapper{ds}
}

func toKey(key []byte) datastore.Key {
	return datastore.NewKey(string(key))
}

func fromKey(key string) []byte {
	if strings.HasPrefix(key, "/") {
		return []byte(key[1:])
	}
	return []byte(key)
}

func (w *wrapper) Has(key []byte) (bool, error) {
	return w.ds.Has(context.Background(), toKey(key))
}

func (w *wrapper) Get(key []byte) ([]byte, error) {
	val, err := w.ds.Get(context.Background(), toKey(key))
	if errors.Is(err, datastore.ErrNotFound) {
		return nil, leveldb.ErrNotFound
	}
	return val, err
}

func (w *wrapper) Put(key []byte, value []byte) error {
	return w.ds.Put(context.Background(), toKey(key), value)
}

func (w *wrapper) Delete(key []byte) error {
	return w.ds.Delete(context.Background(), toKey(key))
}

func (w *wrapper) DeleteRange(start, end []byte) error {
	ctx := context.Background()
	results, err := w.ds.Query(ctx, query.Query{KeysOnly: true})
	if err != nil {
		return err
	}
	defer results.Close()

	for result := range results.Next() {
		if result.Error != nil {
			return result.Error
		}
		keyBytes := fromKey(result.Key)
		if bytes.Compare(keyBytes, start) >= 0 && bytes.Compare(keyBytes, end) < 0 {
			if err := w.ds.Delete(ctx, datastore.NewKey(result.Key)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *wrapper) NewBatch() ethdb.Batch {
	return &batch{ds: w.ds}
}

func (w *wrapper) NewBatchWithSize(size int) ethdb.Batch {
	return &batch{ds: w.ds, ops: make([]batchOp, 0, size)}
}

func (w *wrapper) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return newIterator(w.ds, prefix, start)
}

func (w *wrapper) Stat() (string, error) {
	return "go-datastore wrapper", nil
}

func (w *wrapper) SyncKeyValue() error {
	return w.ds.Sync(context.Background(), datastore.NewKey("/"))
}

func (w *wrapper) Compact(start []byte, limit []byte) error {
	return nil // no-op
}

func (w *wrapper) Close() error {
	return w.ds.Close()
}

// --- Batch ---

type batchOp struct {
	key    []byte
	value  []byte
	delete bool
}

type batch struct {
	ds   datastore.Batching
	ops  []batchOp
	size int
	mu   sync.Mutex
}

func (b *batch) Put(key, value []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ops = append(b.ops, batchOp{key: append([]byte{}, key...), value: append([]byte{}, value...)})
	b.size += len(key) + len(value)
	return nil
}

func (b *batch) Delete(key []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ops = append(b.ops, batchOp{key: append([]byte{}, key...), delete: true})
	b.size += len(key)
	return nil
}

func (b *batch) DeleteRange(start, end []byte) error {
	ctx := context.Background()
	results, err := b.ds.Query(ctx, query.Query{KeysOnly: true})
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
		keyBytes := fromKey(result.Key)
		if bytes.Compare(keyBytes, start) >= 0 && bytes.Compare(keyBytes, end) < 0 {
			b.ops = append(b.ops, batchOp{key: append([]byte{}, keyBytes...), delete: true})
			b.size += len(keyBytes)
		}
	}
	return nil
}

func (b *batch) ValueSize() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.size
}

func (b *batch) Write() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	dsBatch, err := b.ds.Batch(context.Background())
	if err != nil {
		return err
	}

	ctx := context.Background()
	for _, op := range b.ops {
		if op.delete {
			if err := dsBatch.Delete(ctx, toKey(op.key)); err != nil {
				return err
			}
		} else {
			if err := dsBatch.Put(ctx, toKey(op.key), op.value); err != nil {
				return err
			}
		}
	}
	return dsBatch.Commit(ctx)
}

func (b *batch) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ops = b.ops[:0]
	b.size = 0
}

func (b *batch) Replay(w ethdb.KeyValueWriter) error {
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

// --- Iterator ---

type iterator struct {
	entries []query.Entry
	index   int
	err     error
	closed  bool
	mu      sync.Mutex
}

func newIterator(ds datastore.Batching, prefix, start []byte) *iterator {
	q := query.Query{}
	if len(prefix) > 0 {
		q.Prefix = "/" + string(prefix)
	}

	results, err := ds.Query(context.Background(), q)
	if err != nil {
		return &iterator{err: err, index: -1}
	}

	var entries []query.Entry
	for result := range results.Next() {
		if result.Error != nil {
			results.Close()
			return &iterator{err: result.Error, index: -1}
		}

		keyBytes := fromKey(result.Key)
		if len(prefix) > 0 && !bytes.HasPrefix(keyBytes, prefix) {
			continue
		}
		if len(start) > 0 && bytes.Compare(keyBytes, start) < 0 {
			continue
		}
		entries = append(entries, query.Entry{Key: result.Key, Value: result.Value})
	}
	results.Close()

	// Sort for deterministic ordering
	sort.Slice(entries, func(i, j int) bool {
		return bytes.Compare(fromKey(entries[i].Key), fromKey(entries[j].Key)) < 0
	})

	return &iterator{entries: entries, index: -1}
}

func (it *iterator) Next() bool {
	it.mu.Lock()
	defer it.mu.Unlock()

	if it.closed || it.err != nil {
		return false
	}
	it.index++
	return it.index < len(it.entries)
}

func (it *iterator) Error() error {
	it.mu.Lock()
	defer it.mu.Unlock()
	return it.err
}

func (it *iterator) Key() []byte {
	it.mu.Lock()
	defer it.mu.Unlock()

	if it.closed || it.index < 0 || it.index >= len(it.entries) {
		return nil
	}
	return fromKey(it.entries[it.index].Key)
}

func (it *iterator) Value() []byte {
	it.mu.Lock()
	defer it.mu.Unlock()

	if it.closed || it.index < 0 || it.index >= len(it.entries) {
		return nil
	}
	return it.entries[it.index].Value
}

func (it *iterator) Release() {
	it.mu.Lock()
	defer it.mu.Unlock()

	it.closed = true
	it.entries = nil
}
