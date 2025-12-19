package single

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/store"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// createTestBatch creates a batch with dummy transactions for testing
func createTestBatch(t *testing.T, txCount int) coresequencer.Batch {
	txs := make([][]byte, txCount)
	for i := 0; i < txCount; i++ {
		txs[i] = []byte{byte(i), byte(i + 1), byte(i + 2)}
	}
	return coresequencer.Batch{Transactions: txs}
}

func setupTestQueue(t *testing.T) *BatchQueue {
	// Create an in-memory thread-safe datastore
	memdb := store.NewPrefixKVStore(ds.NewMapDatastore(), "single")
	return NewBatchQueue(memdb, "batching", 0) // 0 = unlimited for existing tests
}

func TestNewBatchQueue(t *testing.T) {
	tests := []struct {
		name           string
		expectQueueLen int
	}{
		{
			name:           "queue initialization",
			expectQueueLen: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bq := setupTestQueue(t)
			if bq == nil {
				t.Fatal("expected non-nil BatchQueue")
			}
			if bq.Size() != tc.expectQueueLen {
				t.Errorf("expected queue length %d, got %d", tc.expectQueueLen, bq.Size())
			}
			if bq.db == nil {
				t.Fatal("expected non-nil db")
			}
		})
	}
}

func TestAddBatch(t *testing.T) {
	tests := []struct {
		name           string
		batchesToAdd   []int // Number of transactions in each batch
		expectQueueLen int
		expectErr      bool
	}{
		{
			name:           "add single batch",
			batchesToAdd:   []int{3},
			expectQueueLen: 1,
			expectErr:      false,
		},
		{
			name:           "add multiple batches",
			batchesToAdd:   []int{1, 2, 3},
			expectQueueLen: 3,
			expectErr:      false,
		},
		{
			name:           "add empty batch",
			batchesToAdd:   []int{0},
			expectQueueLen: 1,
			expectErr:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bq := setupTestQueue(t)
			ctx := context.Background()

			// Add batches
			for _, txCount := range tc.batchesToAdd {
				batch := createTestBatch(t, txCount)
				err := bq.AddBatch(ctx, batch)
				if tc.expectErr && err == nil {
					t.Error("expected error but got none")
				}
				if !tc.expectErr && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			// Verify queue length
			if bq.Size() != tc.expectQueueLen {
				t.Errorf("expected queue length %d, got %d", tc.expectQueueLen, bq.Size())
			}

			// Verify batches were stored in datastore
			results, err := bq.db.Query(ctx, query.Query{})
			if err != nil {
				t.Fatalf("unexpected error querying datastore: %v", err)
			}
			defer results.Close()

			count := 0
			for range results.Next() {
				count++
			}
			if count != tc.expectQueueLen {
				t.Errorf("expected datastore count %d, got %d", tc.expectQueueLen, count)
			}
		})
	}
}

func TestNextBatch(t *testing.T) {
	tests := []struct {
		name          string
		batchesToAdd  []int
		callNextCount int
		expectEmptyAt int // At which call to Next() we expect an empty batch
		expectErrors  []bool
	}{
		{
			name:          "get single batch from non-empty queue",
			batchesToAdd:  []int{3},
			callNextCount: 2, // Call Next twice (second should return empty)
			expectEmptyAt: 1, // Second call (index 1) should be empty
			expectErrors:  []bool{false, false},
		},
		{
			name:          "get batches from queue in order",
			batchesToAdd:  []int{1, 2, 3},
			callNextCount: 4, // Call Next four times
			expectEmptyAt: 3, // Fourth call (index 3) should be empty
			expectErrors:  []bool{false, false, false, false},
		},
		{
			name:          "get from empty queue",
			batchesToAdd:  []int{},
			callNextCount: 1,
			expectEmptyAt: 0, // First call should be empty
			expectErrors:  []bool{false},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bq := setupTestQueue(t)
			ctx := context.Background()

			// Add batches
			addedBatches := make([]coresequencer.Batch, 0, len(tc.batchesToAdd))
			for _, txCount := range tc.batchesToAdd {
				batch := createTestBatch(t, txCount)
				addedBatches = append(addedBatches, batch)
				err := bq.AddBatch(ctx, batch)
				if err != nil {
					t.Fatalf("unexpected error adding batch: %v", err)
				}
			}

			// Call Next the specified number of times
			for i := 0; i < tc.callNextCount; i++ {
				batch, err := bq.Next(ctx)

				// Check error as expected
				if i < len(tc.expectErrors) && tc.expectErrors[i] {
					if err == nil {
						t.Errorf("expected error on call %d but got none", i)
					}
					continue
				} else if err != nil {
					t.Errorf("unexpected error on call %d: %v", i, err)
				}

				// Check if batch should be empty at this call
				if i == tc.expectEmptyAt {
					if len(batch.Transactions) != 0 {
						t.Errorf("expected empty batch at call %d, got %d transactions", i, len(batch.Transactions))
					}
				} else if i < tc.expectEmptyAt {
					// Verify the batch matches what we added in the right order
					expectedBatch := addedBatches[i]
					if len(expectedBatch.Transactions) != len(batch.Transactions) {
						t.Errorf("expected %d transactions, got %d", len(expectedBatch.Transactions), len(batch.Transactions))
					}

					// Check each transaction
					for j, tx := range batch.Transactions {
						if !bytes.Equal(expectedBatch.Transactions[j], tx) {
							t.Errorf("transaction mismatch at index %d", j)
						}
					}
				}
			}
		})
	}
}

func TestLoad_WithMixedData(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()

	// Use a raw datastore to manually insert mixed data
	rawDB := dssync.MutexWrap(ds.NewMapDatastore())
	queuePrefix := "/batches/" // Define a specific prefix for the queue

	// Create the BatchQueue using the raw DB and the prefix
	bq := NewBatchQueue(rawDB, queuePrefix, 0) // 0 = unlimited for test
	require.NotNil(bq)

	// 1. Add valid batch data under the correct prefix
	// Use valid hex sequence keys to ensure Load parses them correctly if needed
	key1 := "8000000000000001"
	validBatch1 := createTestBatch(t, 3)
	hash1, err := validBatch1.Hash()
	require.NoError(err)
	hexHash1 := hex.EncodeToString(hash1)
	pbBatch1 := &pb.Batch{Txs: validBatch1.Transactions}
	encodedBatch1, err := proto.Marshal(pbBatch1)
	require.NoError(err)
	err = rawDB.Put(ctx, ds.NewKey(queuePrefix+key1), encodedBatch1)
	require.NoError(err)

	key2 := "8000000000000002"
	validBatch2 := createTestBatch(t, 5)
	hash2, err := validBatch2.Hash()
	require.NoError(err)
	hexHash2 := hex.EncodeToString(hash2)
	pbBatch2 := &pb.Batch{Txs: validBatch2.Transactions}
	encodedBatch2, err := proto.Marshal(pbBatch2)
	require.NoError(err)
	err = rawDB.Put(ctx, ds.NewKey(queuePrefix+key2), encodedBatch2)
	require.NoError(err)

	// 3. Add data outside the queue's prefix
	otherDataKey1 := ds.NewKey("/other/data")
	err = rawDB.Put(ctx, otherDataKey1, []byte("some other data"))
	require.NoError(err)
	otherDataKey2 := ds.NewKey("root_data") // No prefix slash
	err = rawDB.Put(ctx, otherDataKey2, []byte("more data"))
	require.NoError(err)

	// Ensure all data is initially present in the raw DB
	initialKeys := map[string]bool{
		queuePrefix + key1:     true,
		queuePrefix + key2:     true,
		otherDataKey1.String(): true,
		otherDataKey2.String(): true,
	}
	q := query.Query{}
	results, err := rawDB.Query(ctx, q)
	require.NoError(err)
	count := 0
	for res := range results.Next() {
		require.NoError(res.Error)
		_, ok := initialKeys[res.Key]
		require.True(ok, "Unexpected key found before load: %s", res.Key)
		count++
	}
	results.Close()
	require.Equal(len(initialKeys), count, "Initial data count mismatch")

	// Call Load
	loadErr := bq.Load(ctx)
	// The current implementation prints errors but continues, so we expect no error return
	require.NoError(loadErr, "Load returned an unexpected error")

	// Verify queue contains only the valid batches
	require.Equal(2, bq.Size(), "Queue should contain only the 2 valid batches")
	// Check hashes to be sure (order might vary depending on datastore query)
	loadedHashes := make(map[string]bool)
	bq.mu.Lock()
	for i := bq.head; i < len(bq.queue); i++ {
		h, _ := bq.queue[i].Batch.Hash()
		loadedHashes[hex.EncodeToString(h)] = true
	}
	bq.mu.Unlock()
	require.True(loadedHashes[hexHash1], "Valid batch 1 not found in queue")
	require.True(loadedHashes[hexHash2], "Valid batch 2 not found in queue")

	// Verify data outside the prefix remains untouched in the raw DB
	val, err := rawDB.Get(ctx, otherDataKey1)
	require.NoError(err)
	require.Equal([]byte("some other data"), val)
	val, err = rawDB.Get(ctx, otherDataKey2)
	require.NoError(err)
	require.Equal([]byte("more data"), val)
}

func TestBatchQueue_Load_SetsSequencesProperly(t *testing.T) {
	ctx := context.Background()
	db := ds.NewMapDatastore()
	prefix := "test-load-sequences"

	// Build some persisted state with both AddBatch and Prepend so we have
	// keys on both sides of the initialSeqNum.
	q1 := NewBatchQueue(db, prefix, 0)
	require.NoError(t, q1.Load(ctx))

	require.NoError(t, q1.AddBatch(ctx, coresequencer.Batch{Transactions: [][]byte{[]byte("add-1")}})) // initialSeqNum
	require.NoError(t, q1.AddBatch(ctx, coresequencer.Batch{Transactions: [][]byte{[]byte("add-2")}})) // initialSeqNum+1

	require.NoError(t, q1.Prepend(ctx, coresequencer.Batch{Transactions: [][]byte{[]byte("pre-1")}})) // initialSeqNum-1
	require.NoError(t, q1.Prepend(ctx, coresequencer.Batch{Transactions: [][]byte{[]byte("pre-2")}})) // initialSeqNum-2

	// Simulate restart.
	q2 := NewBatchQueue(db, prefix, 0)
	require.NoError(t, q2.Load(ctx))

	// After Load(), the sequencers should be positioned to avoid collisions:
	// - nextAddSeq should be (maxSeq + 1)
	// - nextPrependSeq should be (minSeq - 1)
	require.Equal(t, initialSeqNum+2, q2.nextAddSeq, "nextAddSeq should continue after the max loaded key")
	require.Equal(t, initialSeqNum-3, q2.nextPrependSeq, "nextPrependSeq should continue before the min loaded key")

	// Verify we actually use those sequences when persisting new items.
	require.NoError(t, q2.AddBatch(ctx, coresequencer.Batch{Transactions: [][]byte{[]byte("add-after-load")}}))
	_, err := q2.db.Get(ctx, ds.NewKey(seqToKey(initialSeqNum+2)))
	require.NoError(t, err, "expected AddBatch after Load to persist using nextAddSeq key")

	require.NoError(t, q2.Prepend(ctx, coresequencer.Batch{Transactions: [][]byte{[]byte("pre-after-load")}}))
	_, err = q2.db.Get(ctx, ds.NewKey(seqToKey(initialSeqNum-3)))
	require.NoError(t, err, "expected Prepend after Load to persist using nextPrependSeq key")
}

func TestConcurrency(t *testing.T) {
	bq := setupTestQueue(t)
	ctx := context.Background()

	// Number of concurrent operations
	const numOperations = 100

	// Add batches concurrently
	addWg := new(sync.WaitGroup)
	addWg.Add(numOperations)

	for i := 0; i < numOperations; i++ {
		go func(index int) {
			defer addWg.Done()
			batch := createTestBatch(t, index%10+1) // 1-10 transactions
			err := bq.AddBatch(ctx, batch)
			if err != nil {
				t.Errorf("unexpected error adding batch: %v", err)
			}
		}(i)
	}

	// Wait for all adds to complete
	addWg.Wait()

	// Verify we have expected number of batches
	if bq.Size() != numOperations {
		t.Errorf("expected %d batches, got %d", numOperations, bq.Size())
	}

	// Next operations concurrently (only half)
	nextWg := new(sync.WaitGroup)
	nextCount := numOperations / 2
	nextWg.Add(nextCount)

	for i := 0; i < nextCount; i++ {
		go func() {
			defer nextWg.Done()
			batch, err := bq.Next(ctx)
			if err != nil {
				t.Errorf("unexpected error getting batch: %v", err)
			}
			if batch == nil {
				t.Error("expected non-nil batch")
			}
		}()
	}

	// Wait for all nexts to complete
	nextWg.Wait()

	// Verify we have expected number of batches left
	if bq.Size() != numOperations-nextCount {
		t.Errorf("expected %d batches left, got %d", numOperations-nextCount, bq.Size())
	}

	// Test Load
	err := bq.Load(ctx)
	if err != nil {
		t.Errorf("unexpected error reloading: %v", err)
	}
}

func TestBatchQueue_QueueLimit(t *testing.T) {
	tests := []struct {
		name          string
		maxSize       int
		batchesToAdd  int
		expectErr     bool
		expectErrType error
	}{
		{
			name:          "unlimited queue (maxSize 0)",
			maxSize:       0,
			batchesToAdd:  100,
			expectErr:     false,
			expectErrType: nil,
		},
		{
			name:          "queue within limit",
			maxSize:       10,
			batchesToAdd:  5,
			expectErr:     false,
			expectErrType: nil,
		},
		{
			name:          "queue at exact limit",
			maxSize:       5,
			batchesToAdd:  5,
			expectErr:     false,
			expectErrType: nil,
		},
		{
			name:          "queue exceeds limit",
			maxSize:       3,
			batchesToAdd:  5,
			expectErr:     true,
			expectErrType: ErrQueueFull,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create in-memory datastore and queue with specified limit
			memdb := store.NewPrefixKVStore(ds.NewMapDatastore(), "single")
			bq := NewBatchQueue(memdb, "batching", tc.maxSize)
			ctx := context.Background()

			var lastErr error

			// Add batches up to the specified count
			for i := 0; i < tc.batchesToAdd; i++ {
				batch := createTestBatch(t, i+1)
				err := bq.AddBatch(ctx, batch)
				if err != nil {
					lastErr = err
					if !tc.expectErr {
						t.Errorf("unexpected error at batch %d: %v", i, err)
					}
					break
				}
			}

			// Check final error state
			if tc.expectErr {
				if lastErr == nil {
					t.Error("expected error but got none")
				} else if !errors.Is(lastErr, tc.expectErrType) {
					t.Errorf("expected error %v, got %v", tc.expectErrType, lastErr)
				}
			} else {
				if lastErr != nil {
					t.Errorf("unexpected error: %v", lastErr)
				}
			}

			// For limited queues, verify the queue doesn't exceed maxSize
			if tc.maxSize > 0 && bq.Size() > tc.maxSize {
				t.Errorf("queue size %d exceeds limit %d", bq.Size(), tc.maxSize)
			}

			// For unlimited queues, verify all batches were added
			if tc.maxSize == 0 && !tc.expectErr && bq.Size() != tc.batchesToAdd {
				t.Errorf("expected %d batches in unlimited queue, got %d", tc.batchesToAdd, bq.Size())
			}
		})
	}
}

func TestBatchQueue_QueueLimit_WithNext(t *testing.T) {
	// Test that removing batches with Next() allows adding more batches
	maxSize := 3
	memdb := store.NewPrefixKVStore(ds.NewMapDatastore(), "single")
	bq := NewBatchQueue(memdb, "batching", maxSize)
	ctx := context.Background()

	// Fill the queue to capacity
	for i := 0; i < maxSize; i++ {
		batch := createTestBatch(t, i+1)
		err := bq.AddBatch(ctx, batch)
		if err != nil {
			t.Fatalf("unexpected error adding batch %d: %v", i, err)
		}
	}

	// Verify queue is full
	if bq.Size() != maxSize {
		t.Errorf("expected queue size %d, got %d", maxSize, bq.Size())
	}

	// Try to add one more batch - should fail
	extraBatch := createTestBatch(t, 999)
	err := bq.AddBatch(ctx, extraBatch)
	if !errors.Is(err, ErrQueueFull) {
		t.Errorf("expected ErrQueueFull, got %v", err)
	}

	// Remove one batch using Next()
	batch, err := bq.Next(ctx)
	if err != nil {
		t.Fatalf("unexpected error in Next(): %v", err)
	}
	if batch == nil || len(batch.Transactions) == 0 {
		t.Error("expected non-empty batch from Next()")
	}

	// Verify queue size decreased
	if bq.Size() != maxSize-1 {
		t.Errorf("expected queue size %d after Next(), got %d", maxSize-1, bq.Size())
	}

	// Now adding a batch should succeed
	newBatch := createTestBatch(t, 1000)
	err = bq.AddBatch(ctx, newBatch)
	if err != nil {
		t.Errorf("unexpected error adding batch after Next(): %v", err)
	}

	// Verify queue is full again
	if bq.Size() != maxSize {
		t.Errorf("expected queue size %d after adding back, got %d", maxSize, bq.Size())
	}
}

func TestBatchQueue_QueueLimit_Concurrency(t *testing.T) {
	// Test thread safety of queue limits under concurrent access
	maxSize := 10
	memdb := dssync.MutexWrap(ds.NewMapDatastore()) // Thread-safe datastore
	bq := NewBatchQueue(memdb, "batching", maxSize)
	ctx := context.Background()

	numWorkers := 20
	batchesPerWorker := 5
	totalBatches := numWorkers * batchesPerWorker

	var wg sync.WaitGroup
	var addedCount int64
	var errorCount int64

	// Start multiple workers trying to add batches concurrently
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < batchesPerWorker; j++ {
				batch := createTestBatch(t, workerID*batchesPerWorker+j+1)
				err := bq.AddBatch(ctx, batch)
				if err != nil {
					if errors.Is(err, ErrQueueFull) {
						// Expected error when queue is full
						atomic.AddInt64(&errorCount, 1)
					} else {
						t.Errorf("unexpected error type from worker %d: %v", workerID, err)
					}
				} else {
					atomic.AddInt64(&addedCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify queue size doesn't exceed limit
	if bq.Size() > maxSize {
		t.Errorf("queue size %d exceeds limit %d", bq.Size(), maxSize)
	}

	// Verify some batches were successfully added
	if addedCount == 0 {
		t.Error("no batches were added successfully")
	}

	// Verify some batches were rejected due to queue being full
	if addedCount == int64(totalBatches) {
		t.Error("all batches were added, but queue should have been full at some point")
	}

	// Verify the sum makes sense
	if addedCount+errorCount != int64(totalBatches) {
		t.Errorf("expected %d total operations, got %d added + %d errors = %d",
			totalBatches, addedCount, errorCount, addedCount+errorCount)
	}

	t.Logf("Successfully added %d batches, rejected %d due to queue being full", addedCount, errorCount)
}

func TestBatchQueue_Prepend(t *testing.T) {
	ctx := context.Background()
	db := ds.NewMapDatastore()

	t.Run("prepend to empty queue", func(t *testing.T) {
		queue := NewBatchQueue(db, "test-prepend-empty", 0)
		err := queue.Load(ctx)
		require.NoError(t, err)

		batch := coresequencer.Batch{
			Transactions: [][]byte{[]byte("tx1"), []byte("tx2")},
		}

		err = queue.Prepend(ctx, batch)
		require.NoError(t, err)

		assert.Equal(t, 1, queue.Size())

		// Next should return the prepended batch
		nextBatch, err := queue.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, len(nextBatch.Transactions))
		assert.Equal(t, []byte("tx1"), nextBatch.Transactions[0])
	})

	t.Run("prepend to queue with items", func(t *testing.T) {
		queue := NewBatchQueue(db, "test-prepend-with-items", 0)
		err := queue.Load(ctx)
		require.NoError(t, err)

		// Add some batches first
		batch1 := coresequencer.Batch{Transactions: [][]byte{[]byte("tx1")}}
		batch2 := coresequencer.Batch{Transactions: [][]byte{[]byte("tx2")}}
		err = queue.AddBatch(ctx, batch1)
		require.NoError(t, err)
		err = queue.AddBatch(ctx, batch2)
		require.NoError(t, err)

		assert.Equal(t, 2, queue.Size())

		// Prepend a batch
		prependedBatch := coresequencer.Batch{Transactions: [][]byte{[]byte("prepended")}}
		err = queue.Prepend(ctx, prependedBatch)
		require.NoError(t, err)

		assert.Equal(t, 3, queue.Size())

		// Next should return the prepended batch first
		nextBatch, err := queue.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, len(nextBatch.Transactions))
		assert.Equal(t, []byte("prepended"), nextBatch.Transactions[0])

		// Then the original batches
		nextBatch, err = queue.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, []byte("tx1"), nextBatch.Transactions[0])

		nextBatch, err = queue.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, []byte("tx2"), nextBatch.Transactions[0])
	})

	t.Run("prepend after consuming some items", func(t *testing.T) {
		queue := NewBatchQueue(db, "test-prepend-after-consume", 0)
		err := queue.Load(ctx)
		require.NoError(t, err)

		// Add batches
		batch1 := coresequencer.Batch{Transactions: [][]byte{[]byte("tx1")}}
		batch2 := coresequencer.Batch{Transactions: [][]byte{[]byte("tx2")}}
		batch3 := coresequencer.Batch{Transactions: [][]byte{[]byte("tx3")}}
		err = queue.AddBatch(ctx, batch1)
		require.NoError(t, err)
		err = queue.AddBatch(ctx, batch2)
		require.NoError(t, err)
		err = queue.AddBatch(ctx, batch3)
		require.NoError(t, err)

		assert.Equal(t, 3, queue.Size())

		// Consume first batch
		nextBatch, err := queue.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, []byte("tx1"), nextBatch.Transactions[0])
		assert.Equal(t, 2, queue.Size())

		// Prepend - should reuse the head position
		prependedBatch := coresequencer.Batch{Transactions: [][]byte{[]byte("prepended")}}
		err = queue.Prepend(ctx, prependedBatch)
		require.NoError(t, err)

		assert.Equal(t, 3, queue.Size())

		// Should get prepended, then tx2, then tx3
		nextBatch, err = queue.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, []byte("prepended"), nextBatch.Transactions[0])

		nextBatch, err = queue.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, []byte("tx2"), nextBatch.Transactions[0])

		nextBatch, err = queue.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, []byte("tx3"), nextBatch.Transactions[0])

		assert.Equal(t, 0, queue.Size())
	})

	t.Run("multiple prepends", func(t *testing.T) {
		queue := NewBatchQueue(db, "test-multiple-prepends", 0)
		err := queue.Load(ctx)
		require.NoError(t, err)

		// Add a batch
		batch1 := coresequencer.Batch{Transactions: [][]byte{[]byte("tx1")}}
		err = queue.AddBatch(ctx, batch1)
		require.NoError(t, err)

		// Prepend multiple batches
		prepend1 := coresequencer.Batch{Transactions: [][]byte{[]byte("prepend1")}}
		prepend2 := coresequencer.Batch{Transactions: [][]byte{[]byte("prepend2")}}
		prepend3 := coresequencer.Batch{Transactions: [][]byte{[]byte("prepend3")}}

		err = queue.Prepend(ctx, prepend1)
		require.NoError(t, err)
		err = queue.Prepend(ctx, prepend2)
		require.NoError(t, err)
		err = queue.Prepend(ctx, prepend3)
		require.NoError(t, err)

		assert.Equal(t, 4, queue.Size())

		// Should get in reverse order of prepending (LIFO for prepended items)
		nextBatch, err := queue.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, []byte("prepend3"), nextBatch.Transactions[0])

		nextBatch, err = queue.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, []byte("prepend2"), nextBatch.Transactions[0])

		nextBatch, err = queue.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, []byte("prepend1"), nextBatch.Transactions[0])

		nextBatch, err = queue.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, []byte("tx1"), nextBatch.Transactions[0])
	})

	t.Run("prepend persistence across restarts", func(t *testing.T) {
		prefix := "test-prepend-persistence"
		queue := NewBatchQueue(db, prefix, 0)
		err := queue.Load(ctx)
		require.NoError(t, err)

		// Add some batches
		batch1 := coresequencer.Batch{Transactions: [][]byte{[]byte("tx1")}}
		batch2 := coresequencer.Batch{Transactions: [][]byte{[]byte("tx2")}}
		err = queue.AddBatch(ctx, batch1)
		require.NoError(t, err)
		err = queue.AddBatch(ctx, batch2)
		require.NoError(t, err)

		// Consume first batch
		_, err = queue.Next(ctx)
		require.NoError(t, err)

		// Prepend a batch (simulating transactions that couldn't fit)
		prependedBatch := coresequencer.Batch{Transactions: [][]byte{[]byte("prepended")}}
		err = queue.Prepend(ctx, prependedBatch)
		require.NoError(t, err)

		assert.Equal(t, 2, queue.Size())

		// Simulate restart by creating a new queue with same prefix
		queue2 := NewBatchQueue(db, prefix, 0)
		err = queue2.Load(ctx)
		require.NoError(t, err)

		// Should have both the prepended batch and tx2
		assert.Equal(t, 2, queue2.Size())

		// First should be prepended batch
		nextBatch, err := queue2.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, len(nextBatch.Transactions))
		assert.Contains(t, nextBatch.Transactions, []byte("prepended"))

		// Then tx2
		nextBatch, err = queue2.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, len(nextBatch.Transactions))
		assert.Contains(t, nextBatch.Transactions, []byte("tx2"))

		// Queue should be empty now
		assert.Equal(t, 0, queue2.Size())
	})
}
