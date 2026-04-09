package executor

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"maps"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	dssync "github.com/ipfs/go-datastore/sync"

	"github.com/evstack/ev-node/core/execution"
)

var (
	genesisInitializedKey = ds.NewKey("/genesis/initialized")
	genesisStateRootKey   = ds.NewKey("/genesis/stateroot")
	heightKeyPrefix       = ds.NewKey("/height")
	finalizedHeightKey    = ds.NewKey("/finalizedHeight")
	reservedKeys          = map[ds.Key]bool{
		genesisInitializedKey: true,
		genesisStateRootKey:   true,
		finalizedHeightKey:    true,
	}
)

const shardCount = 64

type txShard struct {
	mu    sync.Mutex
	buf   [][]byte
	flush chan [][]byte
}

type KVExecutor struct {
	db               ds.Batching
	blocksProduced   atomic.Uint64
	totalExecutedTxs atomic.Uint64

	stateMu    sync.RWMutex
	stateMap   map[string]string
	sortedKeys []string
	cachedRoot []byte
	rootDirty  bool

	shards   [shardCount]txShard
	shardIdx atomic.Uint64

	dbWriteCh chan dbWriteReq
}

type dbWriteReq struct {
	height uint64
	kvs    []kvEntry
}

type kvEntry struct {
	key   string
	value string
}

type ExecutorStats struct {
	BlocksProduced   uint64
	TotalExecutedTxs uint64
}

func (k *KVExecutor) GetStats() ExecutorStats {
	return ExecutorStats{
		BlocksProduced:   k.blocksProduced.Load(),
		TotalExecutedTxs: k.totalExecutedTxs.Load(),
	}
}

func NewKVExecutor() *KVExecutor {
	k := &KVExecutor{
		db:        dssync.MutexWrap(ds.NewMapDatastore()),
		stateMap:  make(map[string]string),
		dbWriteCh: make(chan dbWriteReq, 4096),
	}

	for i := range k.shards {
		k.shards[i].buf = make([][]byte, 0, 256)
		k.shards[i].flush = make(chan [][]byte, 64)
	}

	go k.dbWriterLoop()

	return k
}

func (k *KVExecutor) dbWriterLoop() {
	for req := range k.dbWriteCh {
		if len(req.kvs) == 0 {
			continue
		}
		ctx := context.Background()
		batch, err := k.db.Batch(ctx)
		if err != nil {
			continue
		}
		for _, kv := range req.kvs {
			dsKey := getTxKey(req.height, kv.key)
			_ = batch.Put(ctx, dsKey, bytesFromString(kv.value))
		}
		_ = batch.Commit(ctx)
	}
}

func (k *KVExecutor) GetStoreValue(_ context.Context, key string) (string, bool) {
	k.stateMu.RLock()
	val := k.stateMap[key]
	k.stateMu.RUnlock()
	return val, val != ""
}

func (k *KVExecutor) GetAllEntries() map[string]string {
	k.stateMu.RLock()
	m := make(map[string]string, len(k.stateMap))
	maps.Copy(m, k.stateMap)
	k.stateMu.RUnlock()
	return m
}

func (k *KVExecutor) insertSorted(key string) {
	n := len(k.sortedKeys)
	i := sort.SearchStrings(k.sortedKeys, key)
	if i < n && k.sortedKeys[i] == key {
		return
	}
	k.sortedKeys = append(k.sortedKeys, "")
	copy(k.sortedKeys[i+1:], k.sortedKeys[i:])
	k.sortedKeys[i] = key
}

func (k *KVExecutor) computeRootFromStateLocked() []byte {
	if !k.rootDirty && k.cachedRoot != nil {
		return k.cachedRoot
	}

	h := fnv.New128a()
	for _, key := range k.sortedKeys {
		h.Write(bytesFromString(key))
		h.Write([]byte{0})
		h.Write(bytesFromString(k.stateMap[key]))
		h.Write([]byte{0})
	}

	k.cachedRoot = h.Sum(nil)
	k.rootDirty = false
	return k.cachedRoot
}

func (k *KVExecutor) computeStateRoot(_ context.Context) ([]byte, error) {
	k.stateMu.Lock()
	root := k.computeRootFromStateLocked()
	k.stateMu.Unlock()
	return root, nil
}

func (k *KVExecutor) InitChain(ctx context.Context, genesisTime time.Time, initialHeight uint64, chainID string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	initialized, err := k.db.Has(ctx, genesisInitializedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check genesis initialization status: %w", err)
	}

	if initialized {
		genesisRoot, err := k.db.Get(ctx, genesisStateRootKey)
		if err != nil {
			return nil, fmt.Errorf("genesis initialized but failed to retrieve state root: %w", err)
		}
		return genesisRoot, nil
	}

	stateRoot, err := k.computeStateRoot(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to compute initial state root for genesis: %w", err)
	}

	batch, err := k.db.Batch(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create batch for genesis persistence: %w", err)
	}
	if err := batch.Put(ctx, genesisStateRootKey, stateRoot); err != nil {
		return nil, fmt.Errorf("failed to put genesis state root in batch: %w", err)
	}
	if err := batch.Put(ctx, genesisInitializedKey, []byte("true")); err != nil {
		return nil, fmt.Errorf("failed to put genesis initialized flag in batch: %w", err)
	}
	if err := batch.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit genesis persistence batch: %w", err)
	}

	return stateRoot, nil
}

func (k *KVExecutor) GetTxs(ctx context.Context) ([][]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	var all [][]byte
	for i := range k.shards {
		s := &k.shards[i]
		s.mu.Lock()
		if len(s.buf) > 0 {
			batch := s.buf
			s.buf = make([][]byte, 0, cap(batch))
			s.mu.Unlock()
			all = append(all, batch...)
		} else {
			s.mu.Unlock()
		}
		select {
		case batch := <-s.flush:
			all = append(all, batch...)
		default:
		}
	}

	if len(all) == 0 {
		return nil, nil
	}
	return all, nil
}

func (k *KVExecutor) ExecuteTxs(ctx context.Context, txs [][]byte, blockHeight uint64, timestamp time.Time, prevStateRoot []byte) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	type parsedTx struct {
		key   string
		value string
	}

	parsed := make([]parsedTx, 0, len(txs))

	for _, tx := range txs {
		if len(tx) == 0 {
			continue
		}

		s := stringUnsafe(tx)
		before, after, ok := strings.Cut(s, "=")
		if !ok {
			continue
		}

		key := strings.TrimSpace(before)
		if key == "" {
			continue
		}

		value := strings.TrimSpace(after)
		dsKey := getTxKey(blockHeight, key)
		if reservedKeys[dsKey] {
			continue
		}

		parsed = append(parsed, parsedTx{key: key, value: value})
	}

	k.stateMu.Lock()
	for _, p := range parsed {
		if _, exists := k.stateMap[p.key]; !exists {
			k.insertSorted(p.key)
		}
		k.stateMap[p.key] = p.value
	}
	k.rootDirty = true
	root := k.computeRootFromStateLocked()
	k.stateMu.Unlock()

	if len(parsed) > 0 {
		kvs := make([]kvEntry, len(parsed))
		for i, p := range parsed {
			kvs[i] = kvEntry{key: p.key, value: p.value}
		}
		select {
		case k.dbWriteCh <- dbWriteReq{height: blockHeight, kvs: kvs}:
		default:
		}
	}

	k.blocksProduced.Add(1)
	k.totalExecutedTxs.Add(uint64(len(parsed)))

	return root, nil
}

func (k *KVExecutor) SetFinal(ctx context.Context, blockHeight uint64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if blockHeight == 0 {
		return errors.New("invalid blockHeight: cannot be zero")
	}

	return k.db.Put(ctx, finalizedHeightKey, strconv.AppendUint(nil, blockHeight, 10))
}

func (k *KVExecutor) InjectTx(tx []byte) {
	s := &k.shards[(k.shardIdx.Add(1))%shardCount]
	s.mu.Lock()
	s.buf = append(s.buf, tx)
	s.mu.Unlock()
}

func (k *KVExecutor) InjectTxs(txs [][]byte) int {
	accepted := 0
	for _, tx := range txs {
		s := &k.shards[(k.shardIdx.Add(1))%shardCount]
		s.mu.Lock()
		s.buf = append(s.buf, tx)
		s.mu.Unlock()
		accepted++
	}
	return accepted
}

func (k *KVExecutor) Rollback(ctx context.Context, height uint64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if height == 0 {
		return fmt.Errorf("cannot rollback to height 0: invalid height")
	}

	batch, err := k.db.Batch(ctx)
	if err != nil {
		return fmt.Errorf("failed to create batch for rollback: %w", err)
	}

	q := query.Query{}
	results, err := k.db.Query(ctx, q)
	if err != nil {
		return fmt.Errorf("failed to query keys for rollback: %w", err)
	}
	defer results.Close()

	keysToDelete := make([]ds.Key, 0)
	heightPrefix := heightKeyPrefix.String()

	for result := range results.Next() {
		if result.Error != nil {
			return fmt.Errorf("error iterating query results during rollback: %w", result.Error)
		}

		key := result.Key
		if after, ok := strings.CutPrefix(key, heightPrefix+"/"); ok {
			parts := strings.Split(after, "/")
			if len(parts) > 0 {
				var keyHeight uint64
				if _, err := fmt.Sscanf(parts[0], "%d", &keyHeight); err == nil {
					if keyHeight > height {
						keysToDelete = append(keysToDelete, ds.NewKey(key))
					}
				}
			}
		}
	}

	for _, key := range keysToDelete {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := batch.Delete(ctx, key); err != nil {
			return fmt.Errorf("failed to stage delete operation for key '%s' during rollback: %w", key.String(), err)
		}
	}

	if finalizedHeightBytes, err := k.db.Get(ctx, finalizedHeightKey); err == nil {
		var finalizedHeight uint64
		if _, err := fmt.Sscanf(string(finalizedHeightBytes), "%d", &finalizedHeight); err == nil {
			if finalizedHeight > height {
				if err := batch.Put(ctx, finalizedHeightKey, strconv.AppendUint(nil, height, 10)); err != nil {
					return fmt.Errorf("failed to update finalized height during rollback: %w", err)
				}
			}
		}
	}

	if err := batch.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit rollback batch: %w", err)
	}

	k.stateMu.Lock()
	k.stateMap = make(map[string]string)
	k.rebuildStateFromDB(ctx)
	k.sortedKeys = k.sortedKeys[:0]
	for key := range k.stateMap {
		k.sortedKeys = append(k.sortedKeys, key)
	}
	sort.Strings(k.sortedKeys)
	k.rootDirty = true
	k.stateMu.Unlock()

	return nil
}

func (k *KVExecutor) rebuildStateFromDB(ctx context.Context) {
	q := query.Query{}
	results, err := k.db.Query(ctx, q)
	if err != nil {
		return
	}
	defer results.Close()

	heightPrefix := heightKeyPrefix.String()
	type entry struct {
		height uint64
		value  string
	}
	latest := make(map[string]entry)

	for result := range results.Next() {
		if result.Error != nil {
			return
		}
		if after, ok := strings.CutPrefix(result.Key, heightPrefix+"/"); ok {
			parts := strings.Split(after, "/")
			if len(parts) >= 2 {
				var keyHeight uint64
				if _, err := fmt.Sscanf(parts[0], "%d", &keyHeight); err == nil {
					actualKey := strings.Join(parts[1:], "/")
					if e, ok := latest[actualKey]; !ok || keyHeight > e.height {
						latest[actualKey] = entry{height: keyHeight, value: string(result.Value)}
					}
				}
			}
		}
	}

	for key, e := range latest {
		k.stateMap[key] = e.value
	}
}

func getTxKey(height uint64, txKey string) ds.Key {
	return heightKeyPrefix.ChildString(strconv.FormatUint(height, 10) + "/" + txKey)
}

func (k *KVExecutor) GetExecutionInfo(_ context.Context) (execution.ExecutionInfo, error) {
	return execution.ExecutionInfo{MaxGas: 0}, nil
}

func (k *KVExecutor) FilterTxs(_ context.Context, txs [][]byte, maxBytes, maxGas uint64, hasForceIncludedTransaction bool) ([]execution.FilterStatus, error) {
	result := make([]execution.FilterStatus, len(txs))

	var cumulativeBytes uint64
	limitReached := false

	for i, tx := range txs {
		if len(tx) == 0 {
			result[i] = execution.FilterRemove
			continue
		}

		txBytes := uint64(len(tx))

		if hasForceIncludedTransaction {
			eqIdx := indexByte(tx, '=')
			if eqIdx < 0 {
				result[i] = execution.FilterRemove
				continue
			}
			if isAllSpace(tx[:eqIdx]) {
				result[i] = execution.FilterRemove
				continue
			}
		}

		if maxBytes > 0 && txBytes > maxBytes {
			result[i] = execution.FilterRemove
			continue
		}

		if limitReached {
			result[i] = execution.FilterPostpone
			continue
		}

		if maxBytes > 0 && cumulativeBytes+txBytes > maxBytes {
			limitReached = true
			result[i] = execution.FilterPostpone
			continue
		}

		cumulativeBytes += txBytes
		result[i] = execution.FilterOK
	}

	return result, nil
}

func indexByte(s []byte, c byte) int {
	for i, b := range s {
		if b == c {
			return i
		}
	}
	return -1
}

func isAllSpace(s []byte) bool {
	allSpace := true
	for _, b := range s {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			allSpace = false
			break
		}
	}
	return allSpace && len(s) == 0
}

func stringUnsafe(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func bytesFromString(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			int
		}{s, len(s)},
	))
}
