package based

import (
	"context"
	"testing"

	ds "github.com/ipfs/go-datastore"
	syncds "github.com/ipfs/go-datastore/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTxQueue_AddAndNext(t *testing.T) {
	db := syncds.MutexWrap(ds.NewMapDatastore())
	queue := NewTxQueue(db, "test", 0)

	ctx := context.Background()

	// Add transactions
	err := queue.Add(ctx, []byte("tx1"))
	require.NoError(t, err)
	err = queue.Add(ctx, []byte("tx2"))
	require.NoError(t, err)

	assert.Equal(t, 2, queue.Size())

	// Get next transaction
	tx, err := queue.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, []byte("tx1"), tx)
	assert.Equal(t, 1, queue.Size())

	// Get second transaction
	tx, err = queue.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, []byte("tx2"), tx)
	assert.Equal(t, 0, queue.Size())

	// Queue should be empty
	tx, err = queue.Next(ctx)
	require.NoError(t, err)
	assert.Nil(t, tx)
}

func TestTxQueue_AddBatch(t *testing.T) {
	db := syncds.MutexWrap(ds.NewMapDatastore())
	queue := NewTxQueue(db, "test", 0)

	ctx := context.Background()

	txs := [][]byte{
		[]byte("tx1"),
		[]byte("tx2"),
		[]byte("tx3"),
	}

	err := queue.AddBatch(ctx, txs)
	require.NoError(t, err)

	assert.Equal(t, 3, queue.Size())

	// Verify all transactions
	for i, expectedTx := range txs {
		tx, err := queue.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, expectedTx, tx, "transaction %d should match", i)
	}

	assert.Equal(t, 0, queue.Size())
}

func TestTxQueue_MaxSize(t *testing.T) {
	db := syncds.MutexWrap(ds.NewMapDatastore())
	queue := NewTxQueue(db, "test", 2) // Max 2 transactions

	ctx := context.Background()

	// Add first transaction
	err := queue.Add(ctx, []byte("tx1"))
	require.NoError(t, err)

	// Add second transaction
	err = queue.Add(ctx, []byte("tx2"))
	require.NoError(t, err)

	// Third transaction should fail
	err = queue.Add(ctx, []byte("tx3"))
	assert.ErrorIs(t, err, ErrQueueFull)

	// Size should still be 2
	assert.Equal(t, 2, queue.Size())

	// After removing one, we should be able to add again
	_, err = queue.Next(ctx)
	require.NoError(t, err)

	err = queue.Add(ctx, []byte("tx3"))
	require.NoError(t, err)
	assert.Equal(t, 2, queue.Size())
}

func TestTxQueue_AddBatchMaxSize(t *testing.T) {
	db := syncds.MutexWrap(ds.NewMapDatastore())
	queue := NewTxQueue(db, "test", 3)

	ctx := context.Background()

	// Add one transaction
	err := queue.Add(ctx, []byte("tx1"))
	require.NoError(t, err)

	// Try to add 3 more (would exceed limit)
	txs := [][]byte{
		[]byte("tx2"),
		[]byte("tx3"),
		[]byte("tx4"),
	}
	err = queue.AddBatch(ctx, txs)
	assert.ErrorIs(t, err, ErrQueueFull)

	// Size should still be 1
	assert.Equal(t, 1, queue.Size())
}

func TestTxQueue_Persistence(t *testing.T) {
	db := syncds.MutexWrap(ds.NewMapDatastore())
	queue := NewTxQueue(db, "test", 0)

	ctx := context.Background()

	// Add some transactions
	err := queue.Add(ctx, []byte("tx1"))
	require.NoError(t, err)
	err = queue.Add(ctx, []byte("tx2"))
	require.NoError(t, err)
	err = queue.Add(ctx, []byte("tx3"))
	require.NoError(t, err)

	assert.Equal(t, 3, queue.Size())

	// Create a new queue with the same datastore
	queue2 := NewTxQueue(db, "test", 0)

	// Load from persistence
	err = queue2.Load(ctx)
	require.NoError(t, err)

	// Should have all transactions
	assert.Equal(t, 3, queue2.Size())

	// Verify transactions are in order
	tx, err := queue2.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, []byte("tx1"), tx)

	tx, err = queue2.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, []byte("tx2"), tx)

	tx, err = queue2.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, []byte("tx3"), tx)

	assert.Equal(t, 0, queue2.Size())
}

func TestTxQueue_PersistenceAfterPartialConsumption(t *testing.T) {
	db := syncds.MutexWrap(ds.NewMapDatastore())
	queue := NewTxQueue(db, "test", 0)

	ctx := context.Background()

	// Add transactions
	err := queue.Add(ctx, []byte("tx1"))
	require.NoError(t, err)
	err = queue.Add(ctx, []byte("tx2"))
	require.NoError(t, err)
	err = queue.Add(ctx, []byte("tx3"))
	require.NoError(t, err)

	// Consume first transaction
	tx, err := queue.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, []byte("tx1"), tx)

	// Create new queue and load
	queue2 := NewTxQueue(db, "test", 0)
	err = queue2.Load(ctx)
	require.NoError(t, err)

	// Should only have remaining transactions
	assert.Equal(t, 2, queue2.Size())

	tx, err = queue2.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, []byte("tx2"), tx)

	tx, err = queue2.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, []byte("tx3"), tx)
}

func TestTxQueue_Peek(t *testing.T) {
	db := syncds.MutexWrap(ds.NewMapDatastore())
	queue := NewTxQueue(db, "test", 0)

	ctx := context.Background()

	// Add transactions of different sizes
	err := queue.Add(ctx, make([]byte, 50)) // 50 bytes
	require.NoError(t, err)
	err = queue.Add(ctx, make([]byte, 60)) // 60 bytes
	require.NoError(t, err)
	err = queue.Add(ctx, make([]byte, 100)) // 100 bytes
	require.NoError(t, err)

	// Peek with 100 bytes limit - should get first tx only
	txs := queue.Peek(100)
	assert.Equal(t, 1, len(txs))
	assert.Equal(t, 50, len(txs[0]))

	// Queue size should not change
	assert.Equal(t, 3, queue.Size())

	// Peek with 120 bytes limit - should get first two txs
	txs = queue.Peek(120)
	assert.Equal(t, 2, len(txs))
	assert.Equal(t, 50, len(txs[0]))
	assert.Equal(t, 60, len(txs[1]))

	// Queue size should still not change
	assert.Equal(t, 3, queue.Size())

	// Peek with 300 bytes limit - should get all txs
	txs = queue.Peek(300)
	assert.Equal(t, 3, len(txs))
}

func TestTxQueue_Consume(t *testing.T) {
	db := syncds.MutexWrap(ds.NewMapDatastore())
	queue := NewTxQueue(db, "test", 0)

	ctx := context.Background()

	// Add transactions
	err := queue.Add(ctx, []byte("tx1"))
	require.NoError(t, err)
	err = queue.Add(ctx, []byte("tx2"))
	require.NoError(t, err)
	err = queue.Add(ctx, []byte("tx3"))
	require.NoError(t, err)

	assert.Equal(t, 3, queue.Size())

	// Consume first 2 transactions
	err = queue.Consume(ctx, 2)
	require.NoError(t, err)

	assert.Equal(t, 1, queue.Size())

	// Next transaction should be tx3
	tx, err := queue.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, []byte("tx3"), tx)
}

func TestTxQueue_PeekAndConsume(t *testing.T) {
	db := syncds.MutexWrap(ds.NewMapDatastore())
	queue := NewTxQueue(db, "test", 0)

	ctx := context.Background()

	// Add transactions
	err := queue.Add(ctx, make([]byte, 50))
	require.NoError(t, err)
	err = queue.Add(ctx, make([]byte, 60))
	require.NoError(t, err)
	err = queue.Add(ctx, make([]byte, 100))
	require.NoError(t, err)

	// Peek to see what fits in 120 bytes
	txs := queue.Peek(120)
	assert.Equal(t, 2, len(txs))

	// Consume those transactions
	err = queue.Consume(ctx, len(txs))
	require.NoError(t, err)

	// Should have 1 transaction left
	assert.Equal(t, 1, queue.Size())

	// Next transaction should be the 100-byte one
	tx, err := queue.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, 100, len(tx))
}

func TestTxQueue_ConsumeMoreThanAvailable(t *testing.T) {
	db := syncds.MutexWrap(ds.NewMapDatastore())
	queue := NewTxQueue(db, "test", 0)

	ctx := context.Background()

	// Add 2 transactions
	err := queue.Add(ctx, []byte("tx1"))
	require.NoError(t, err)
	err = queue.Add(ctx, []byte("tx2"))
	require.NoError(t, err)

	// Try to consume 3 transactions
	err = queue.Consume(ctx, 3)
	assert.Error(t, err)

	// Size should be unchanged
	assert.Equal(t, 2, queue.Size())
}

func TestTxQueue_Clear(t *testing.T) {
	db := syncds.MutexWrap(ds.NewMapDatastore())
	queue := NewTxQueue(db, "test", 0)

	ctx := context.Background()

	// Add transactions
	err := queue.Add(ctx, []byte("tx1"))
	require.NoError(t, err)
	err = queue.Add(ctx, []byte("tx2"))
	require.NoError(t, err)
	err = queue.Add(ctx, []byte("tx3"))
	require.NoError(t, err)

	assert.Equal(t, 3, queue.Size())

	// Clear the queue
	err = queue.Clear(ctx)
	require.NoError(t, err)

	assert.Equal(t, 0, queue.Size())

	// Queue should be empty
	tx, err := queue.Next(ctx)
	require.NoError(t, err)
	assert.Nil(t, tx)

	// Verify persistence is also cleared
	queue2 := NewTxQueue(db, "test", 0)
	err = queue2.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, queue2.Size())
}

func TestTxQueue_PrefixIsolation(t *testing.T) {
	db := syncds.MutexWrap(ds.NewMapDatastore())

	ctx := context.Background()

	// Create two queues with different prefixes
	queue1 := NewTxQueue(db, "queue1", 0)
	queue2 := NewTxQueue(db, "queue2", 0)

	// Add different transactions to each
	err := queue1.Add(ctx, []byte("tx1"))
	require.NoError(t, err)
	err = queue2.Add(ctx, []byte("tx2"))
	require.NoError(t, err)

	assert.Equal(t, 1, queue1.Size())
	assert.Equal(t, 1, queue2.Size())

	// Load each queue separately
	queue1New := NewTxQueue(db, "queue1", 0)
	err = queue1New.Load(ctx)
	require.NoError(t, err)

	queue2New := NewTxQueue(db, "queue2", 0)
	err = queue2New.Load(ctx)
	require.NoError(t, err)

	// Each should have its own transaction
	assert.Equal(t, 1, queue1New.Size())
	assert.Equal(t, 1, queue2New.Size())

	tx, err := queue1New.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, []byte("tx1"), tx)

	tx, err = queue2New.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, []byte("tx2"), tx)
}

func TestTxQueue_MemoryCompaction(t *testing.T) {
	db := syncds.MutexWrap(ds.NewMapDatastore())
	queue := NewTxQueue(db, "test", 0)

	ctx := context.Background()

	// Add more than 100 transactions to trigger compaction
	for i := 0; i < 150; i++ {
		err := queue.Add(ctx, []byte{byte(i)})
		require.NoError(t, err)
	}

	// Consume 100 transactions to trigger compaction
	for i := 0; i < 100; i++ {
		_, err := queue.Next(ctx)
		require.NoError(t, err)
	}

	// Size should be 50
	assert.Equal(t, 50, queue.Size())

	// Remaining transactions should be correct
	for i := 100; i < 150; i++ {
		tx, err := queue.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, []byte{byte(i)}, tx)
	}
}
