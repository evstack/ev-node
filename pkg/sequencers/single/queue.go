package single

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/sequencers/common"
	"github.com/evstack/ev-node/pkg/store"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

var ErrQueueFull = common.ErrQueueFull

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

	// inFlight holds items returned by Drain that haven't been acked yet.
	// A subsequent Drain rolls these back to the front of the queue.
	inFlight []queuedItem

	// inFlightPostponed holds txs that should be requeued on Ack.
	// Set via SetPostponed between Drain and Ack. Cleared only on successful Ack.
	inFlightPostponed [][]byte
	// postponedItem holds a postponed batch persisted to the WAL during Ack.
	// It is only prepended to the in-memory queue once Ack fully succeeds, so
	// a direct Ack retry does not persist a duplicate entry. If a Drain rolls
	// the in-flight state back instead, the entry is discarded again because
	// its txs are still covered by the rolled-back WAL entries.
	postponedItem *queuedItem

	// txSeen is an in-memory dedup set keyed by sha256 hash of each tx.
	// hashes are added in AddBatch and removed on successful Ack.
	// prevents the reaper from enqueuing the same tx multiple scrape cycles.
	txSeen map[[32]byte]struct{}

	// totalEnqueued counts batches ever enqueued via AddBatch. Monotonic,
	// never decremented, so callers can detect new enqueues race-free.
	totalEnqueued uint64

	// Sequence numbers for generating new keys
	nextAddSeq     uint64
	nextPrependSeq uint64

	mu     sync.Mutex
	db     ds.Batching
	logger zerolog.Logger
}

// NewBatchQueue creates a new BatchQueue with the specified maximum size.
// If maxSize is 0, the queue will be unlimited.
func NewBatchQueue(db ds.Batching, prefix string, maxSize int, logger zerolog.Logger) *BatchQueue {
	return &BatchQueue{
		queue:          make([]queuedItem, 0),
		head:           0,
		maxQueueSize:   maxSize,
		txSeen:         make(map[[32]byte]struct{}),
		db:             store.NewPrefixKVStore(db, prefix),
		nextAddSeq:     initialSeqNum,
		nextPrependSeq: initialSeqNum - 1,
		logger:         logger,
	}
}

// seqToKey converts a sequence number to a hex-encoded big-endian key.
// We use big-endian so that lexicographical sort order matches numeric order.
func seqToKey(seq uint64) string {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, seq)
	return hex.EncodeToString(b)
}

// txHash returns the sha256 of a transaction.
func txHash(tx []byte) [32]byte {
	return sha256.Sum256(tx)
}

// AddBatch adds a new transaction batch to the queue and writes it to the WAL.
// Duplicate transactions (by hash) already in the queue or in-flight are silently skipped.
// Returns ErrQueueFull if the queue has reached its maximum size.
func (bq *BatchQueue) AddBatch(ctx context.Context, batch coresequencer.Batch) error {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	// dedup: skip txs already queued or in-flight, track accepted hashes immediately
	unique := make([][]byte, 0, len(batch.Transactions))
	hashes := make([][32]byte, 0, len(batch.Transactions))
	for _, tx := range batch.Transactions {
		h := txHash(tx)
		if _, dup := bq.txSeen[h]; dup {
			continue
		}
		unique = append(unique, tx)
		hashes = append(hashes, h)
		bq.txSeen[h] = struct{}{}
	}
	if len(unique) == 0 {
		return nil
	}
	batch = coresequencer.Batch{Transactions: unique}

	// Check if queue is full (maxQueueSize of 0 means unlimited)
	// effective size includes both queued and drained-but-unacked entries
	effectiveSize := len(bq.queue) - bq.head + len(bq.inFlight)
	if bq.maxQueueSize > 0 && effectiveSize >= bq.maxQueueSize {
		bq.rollbackSeenLocked(hashes)
		return ErrQueueFull
	}

	key := seqToKey(bq.nextAddSeq)
	if err := bq.persistBatch(ctx, batch, key); err != nil {
		bq.rollbackSeenLocked(hashes)
		return err
	}
	bq.nextAddSeq++

	bq.queue = append(bq.queue, queuedItem{Batch: batch, Key: key})
	bq.totalEnqueued++

	return nil
}

// rollbackSeenLocked removes the given hashes from the dedup set.
// Must be called with bq.mu held.
func (bq *BatchQueue) rollbackSeenLocked(hashes [][32]byte) {
	for _, h := range hashes {
		delete(bq.txSeen, h)
	}
}

// Drain merges multiple queue entries into a single batch up to maxBytes.
// If maxBytes is 0, all available entries are drained.
// Any previously un-acked inFlight entries are rolled back to the front first.
// Drained entries move to inFlight state; WAL entries are NOT deleted until Ack.
func (bq *BatchQueue) Drain(ctx context.Context, maxBytes uint64) (*coresequencer.Batch, error) {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	bq.rollbackInFlightLocked(ctx)

	if bq.head >= len(bq.queue) {
		return &coresequencer.Batch{Transactions: nil}, nil
	}

	var totalBytes uint64
	var allTxs [][]byte

	for bq.head < len(bq.queue) {
		item := bq.queue[bq.head]

		var entryBytes uint64
		for _, tx := range item.Batch.Transactions {
			entryBytes += uint64(len(tx))
		}

		if maxBytes > 0 && totalBytes+entryBytes > maxBytes && len(allTxs) > 0 {
			break
		}

		allTxs = append(allTxs, item.Batch.Transactions...)
		totalBytes += entryBytes
		bq.inFlight = append(bq.inFlight, item)
		bq.queue[bq.head] = queuedItem{}
		bq.head++
	}

	bq.compactLocked()

	if len(allTxs) == 0 {
		return &coresequencer.Batch{Transactions: nil}, nil
	}

	return &coresequencer.Batch{Transactions: allTxs}, nil
}

// SetPostponed records txs that should be requeued on the next Ack.
// Must be called between Drain and Ack. The queue owns this state so
// it is only cleared on successful Ack — no data loss on failure.
func (bq *BatchQueue) SetPostponed(txs [][]byte) {
	bq.mu.Lock()
	defer bq.mu.Unlock()
	if bq.postponedItem != nil {
		return
	}
	bq.inFlightPostponed = txs
}

// Ack commits the current inFlight entries: durably requeues any postponed
// transactions first, then deletes committed WAL entries. On failure neither
// inFlight nor inFlightPostponed is cleared, so the next Drain will roll
// entries back and a retry is safe.
func (bq *BatchQueue) Ack(ctx context.Context) error {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	// persist postponed txs BEFORE deleting source WAL entries.
	// if this fails the original entries still exist — no data loss.
	// the item is only prepended to the in-memory queue after the WAL
	// deletes succeed, so a rollback never sees its txs twice.
	if len(bq.inFlightPostponed) > 0 && bq.postponedItem == nil {
		batch := coresequencer.Batch{Transactions: bq.inFlightPostponed}
		key := seqToKey(bq.nextPrependSeq)
		if err := bq.persistBatch(ctx, batch, key); err != nil {
			return fmt.Errorf("failed to persist postponed txs: %w", err)
		}
		bq.nextPrependSeq--
		bq.postponedItem = &queuedItem{Batch: batch, Key: key}
	}

	// delete WAL entries for committed inFlight items in one batch.
	// on failure, return error WITHOUT clearing state so next Drain
	// rolls them back and they can be retried.
	if len(bq.inFlight) > 0 {
		b, err := bq.db.Batch(ctx)
		if err != nil {
			return fmt.Errorf("failed to create WAL delete batch: %w", err)
		}
		for _, item := range bq.inFlight {
			if err := b.Delete(ctx, ds.NewKey(item.Key)); err != nil {
				return fmt.Errorf("failed to delete committed WAL entry %s: %w", item.Key, err)
			}
		}
		if err := b.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit WAL deletes: %w", err)
		}
	}

	// success — remove committed tx hashes from dedup set.
	// postponed txs are a subset of inFlight but stay in txSeen
	// since they're re-queued via the prepended item.
	var postponed map[[32]byte]struct{}
	if len(bq.inFlightPostponed) > 0 {
		postponed = make(map[[32]byte]struct{}, len(bq.inFlightPostponed))
		for _, tx := range bq.inFlightPostponed {
			postponed[txHash(tx)] = struct{}{}
		}
	}
	for _, item := range bq.inFlight {
		for _, tx := range item.Batch.Transactions {
			h := txHash(tx)
			if _, ok := postponed[h]; ok {
				continue
			}
			delete(bq.txSeen, h)
		}
	}

	// requeue the persisted postponed entry now that the commit is durable
	if bq.postponedItem != nil {
		bq.prependItemLocked(*bq.postponedItem)
	}

	clear(bq.inFlight)
	bq.inFlight = bq.inFlight[:0]
	bq.inFlightPostponed = nil
	bq.postponedItem = nil

	return nil
}

// prependItemLocked inserts an item at the front of the queue.
// Must be called with bq.mu held.
func (bq *BatchQueue) prependItemLocked(item queuedItem) {
	if bq.head > 0 {
		bq.head--
		bq.queue[bq.head] = item
	} else {
		bq.queue = append([]queuedItem{item}, bq.queue...)
	}
}

// rollbackInFlightLocked moves un-acked inFlight items back to the front of the queue.
// Postponed state is discarded: the postponed txs are still covered by the
// rolled-back WAL entries, so a persisted postponed entry would duplicate
// them and is deleted (best-effort; Load dedups any leftover on restart).
// The caller is expected to make a fresh SetPostponed decision after the
// next Drain. Must be called with bq.mu held.
func (bq *BatchQueue) rollbackInFlightLocked(ctx context.Context) {
	if len(bq.inFlight) == 0 {
		return
	}

	if bq.postponedItem != nil {
		if err := bq.db.Delete(ctx, ds.NewKey(bq.postponedItem.Key)); err != nil {
			bq.logger.Warn().Err(err).Str("key", bq.postponedItem.Key).
				Msg("failed to delete rolled-back postponed WAL entry")
		}
		bq.postponedItem = nil
	}
	bq.inFlightPostponed = nil

	if bq.head >= len(bq.inFlight) {
		// enough head slots — fill them directly
		for i := len(bq.inFlight) - 1; i >= 0; i-- {
			bq.head--
			bq.queue[bq.head] = bq.inFlight[i]
		}
	} else {
		// not enough head slots — single bulk prepend O(n)
		tail := bq.queue[bq.head:]
		newQueue := make([]queuedItem, 0, len(bq.inFlight)+len(tail))
		newQueue = append(newQueue, bq.inFlight...)
		newQueue = append(newQueue, tail...)
		bq.queue = newQueue
		bq.head = 0
	}

	clear(bq.inFlight)
	bq.inFlight = bq.inFlight[:0]
}

// compactLocked compacts the queue when head gets too large.
// Must be called with bq.mu held.
func (bq *BatchQueue) compactLocked() {
	if bq.head > len(bq.queue)/2 && bq.head > 100 {
		remaining := copy(bq.queue, bq.queue[bq.head:])
		for i := remaining; i < len(bq.queue); i++ {
			bq.queue[i] = queuedItem{}
		}
		bq.queue = bq.queue[:remaining]
		bq.head = 0
	}
}

// dedupAndEnqueueLocked filters duplicate txs from batch and enqueues the remainder.
// The WAL is kept in sync: fully-duplicate entries are deleted and partially-duplicate
// entries are rewritten, so stale duplicate txs cannot be resurrected by a later reload.
// Cleanup failures are non-fatal — the filtered in-memory state stays authoritative.
// Must be called with bq.mu held.
func (bq *BatchQueue) dedupAndEnqueueLocked(ctx context.Context, batch coresequencer.Batch, key string) {
	filtered := make([][]byte, 0, len(batch.Transactions))
	for _, tx := range batch.Transactions {
		h := txHash(tx)
		if _, dup := bq.txSeen[h]; dup {
			continue
		}
		filtered = append(filtered, tx)
		bq.txSeen[h] = struct{}{}
	}

	switch {
	case len(filtered) == 0:
		if err := bq.db.Delete(ctx, ds.NewKey(key)); err != nil {
			bq.logger.Error().Err(err).Str("key", key).Msg("failed to delete duplicate WAL entry")
		}
		return
	case len(filtered) < len(batch.Transactions):
		batch = coresequencer.Batch{Transactions: filtered}
		if err := bq.persistBatch(ctx, batch, key); err != nil {
			bq.logger.Error().Err(err).Str("key", key).Msg("failed to rewrite partially duplicate WAL entry")
		}
	}

	bq.queue = append(bq.queue, queuedItem{Batch: batch, Key: key})
}

// DropIncluded removes the given transactions from queued entries and the WAL.
// It reconciles a crash between block commit and Ack: after a restart the WAL
// may still hold entries whose txs were already committed in the last block.
// Entries are rewritten in place (or deleted when emptied) so a subsequent
// reload stays consistent. Returns the number of dropped transactions.
// It must be called on a freshly loaded queue: only queued entries are
// scanned, so any in-flight entries would be missed.
func (bq *BatchQueue) DropIncluded(ctx context.Context, included [][]byte) (int, error) {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	includedSet := make(map[[32]byte]struct{}, len(included))
	for _, tx := range included {
		includedSet[txHash(tx)] = struct{}{}
	}

	var dropped int
	kept := bq.queue[:bq.head]
	for _, item := range bq.queue[bq.head:] {
		remaining := make([][]byte, 0, len(item.Batch.Transactions))
		for _, tx := range item.Batch.Transactions {
			h := txHash(tx)
			if _, ok := includedSet[h]; ok {
				delete(bq.txSeen, h)
				dropped++
				continue
			}
			remaining = append(remaining, tx)
		}

		switch {
		case len(remaining) == len(item.Batch.Transactions):
			// nothing dropped, keep as is
		case len(remaining) == 0:
			if err := bq.db.Delete(ctx, ds.NewKey(item.Key)); err != nil {
				return dropped, fmt.Errorf("failed to delete included WAL entry %s: %w", item.Key, err)
			}
			continue
		default:
			item.Batch = coresequencer.Batch{Transactions: remaining}
			if err := bq.persistBatch(ctx, item.Batch, item.Key); err != nil {
				return dropped, fmt.Errorf("failed to rewrite WAL entry %s: %w", item.Key, err)
			}
		}
		kept = append(kept, item)
	}
	bq.queue = kept

	return dropped, nil
}

// Load reloads all batches from WAL file into the in-memory queue after a crash or restart
func (bq *BatchQueue) Load(ctx context.Context) error {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	// Clear the current queue, dedup set, and reset sequences
	bq.queue = make([]queuedItem, 0)
	bq.head = 0
	bq.txSeen = make(map[[32]byte]struct{})
	bq.inFlight = nil
	bq.inFlightPostponed = nil
	bq.postponedItem = nil
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
			bq.logger.Error().Err(result.Error).Msg("failed to read entry from datastore")
			continue
		}
		// We care about the last part of the key (the sequence number)
		// ds.Key usually has a leading slash.
		keyName := ds.NewKey(result.Key).Name()

		var pbBatch pb.Batch
		err := proto.Unmarshal(result.Value, &pbBatch)
		if err != nil {
			bq.logger.Error().Err(err).Str("key", keyName).Msg("failed to decode batch, skipping entry")
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
			bq.dedupAndEnqueueLocked(ctx, batch, keyName)
		} else {
			legacyItems = append(legacyItems, queuedItem{Batch: batch, Key: result.Key})
		}
	}
	if len(legacyItems) == 0 {
		return nil
	}
	bq.logger.Info().Int("count", len(legacyItems)).Msg("found legacy items to migrate")

	for _, item := range legacyItems {
		newKeyName := seqToKey(bq.nextAddSeq)

		if err := bq.persistBatch(ctx, item.Batch, newKeyName); err != nil {
			bq.logger.Error().Err(err).Str("key", item.Key).Msg("failed to migrate legacy item")
			continue
		}

		if err := bq.db.Delete(ctx, ds.NewKey(item.Key)); err != nil {
			bq.logger.Error().Err(err).Str("key", item.Key).Msg("failed to delete legacy key after migration")
		}

		bq.dedupAndEnqueueLocked(ctx, item.Batch, newKeyName)
		bq.nextAddSeq++
	}

	return nil
}

// Size returns the total number of pending batches (queued + drained-but-unacked).
func (bq *BatchQueue) Size() int {
	bq.mu.Lock()
	defer bq.mu.Unlock()
	return len(bq.queue) - bq.head + len(bq.inFlight)
}

// TotalEnqueued returns a monotonic count of batches ever enqueued via
// AddBatch. Unlike Size it never decreases, so comparing two snapshots
// reliably detects whether new batches were enqueued in between.
func (bq *BatchQueue) TotalEnqueued() uint64 {
	bq.mu.Lock()
	defer bq.mu.Unlock()
	return bq.totalEnqueued
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
