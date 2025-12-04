package based

import (
	"context"
	"testing"

	ds "github.com/ipfs/go-datastore"
	"github.com/stretchr/testify/require"
)

func TestCheckpointStore_SaveAndLoad(t *testing.T) {
	ctx := context.Background()
	db := ds.NewMapDatastore()
	store := NewCheckpointStore(db)

	// Test loading when no checkpoint exists
	_, err := store.Load(ctx)
	require.ErrorIs(t, err, ErrCheckpointNotFound)

	// Test saving a checkpoint
	checkpoint := &Checkpoint{
		DAHeight: 100,
		TxIndex:  5,
	}
	err = store.Save(ctx, checkpoint)
	require.NoError(t, err)

	// Test loading the saved checkpoint
	loaded, err := store.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, checkpoint.DAHeight, loaded.DAHeight)
	require.Equal(t, checkpoint.TxIndex, loaded.TxIndex)

	// Test updating the checkpoint
	checkpoint.DAHeight = 200
	checkpoint.TxIndex = 10
	err = store.Save(ctx, checkpoint)
	require.NoError(t, err)

	loaded, err = store.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(200), loaded.DAHeight)
	require.Equal(t, uint64(10), loaded.TxIndex)
}

func TestCheckpointStore_Delete(t *testing.T) {
	ctx := context.Background()
	db := ds.NewMapDatastore()
	store := NewCheckpointStore(db)

	// Save a checkpoint
	checkpoint := &Checkpoint{
		DAHeight: 100,
		TxIndex:  5,
	}
	err := store.Save(ctx, checkpoint)
	require.NoError(t, err)

	// Delete it
	err = store.Delete(ctx)
	require.NoError(t, err)

	// Verify it's gone
	_, err = store.Load(ctx)
	require.ErrorIs(t, err, ErrCheckpointNotFound)

	// Delete again should not error
	err = store.Delete(ctx)
	require.NoError(t, err)
}

func TestCheckpointStore_CleanupLegacyQueue(t *testing.T) {
	ctx := context.Background()
	db := ds.NewMapDatastore()
	store := NewCheckpointStore(db)

	// Add some legacy queue keys (simulating old implementation)
	legacyKeys := []string{
		"/based_txs/tx_0_abc123",
		"/based_txs/tx_1_def456",
		"/based_txs/tx_2_ghi789",
	}
	for _, key := range legacyKeys {
		err := db.Put(ctx, ds.NewKey(key), []byte("dummy data"))
		require.NoError(t, err)
	}

	// Save a checkpoint (should not be cleaned up)
	checkpoint := &Checkpoint{
		DAHeight: 100,
		TxIndex:  5,
	}
	err := store.Save(ctx, checkpoint)
	require.NoError(t, err)

	// Cleanup legacy queue
	err = store.CleanupLegacyQueue(ctx)
	require.NoError(t, err)

	// Verify legacy keys are gone
	for _, key := range legacyKeys {
		has, err := db.Has(ctx, ds.NewKey(key))
		require.NoError(t, err)
		require.False(t, has, "legacy key should be deleted: %s", key)
	}

	// Verify checkpoint still exists
	loaded, err := store.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, checkpoint.DAHeight, loaded.DAHeight)
	require.Equal(t, checkpoint.TxIndex, loaded.TxIndex)
}

func TestCheckpoint_EdgeCases(t *testing.T) {
	ctx := context.Background()
	db := ds.NewMapDatastore()
	store := NewCheckpointStore(db)

	// Test with zero values
	checkpoint := &Checkpoint{
		DAHeight: 0,
		TxIndex:  0,
	}
	err := store.Save(ctx, checkpoint)
	require.NoError(t, err)

	loaded, err := store.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(0), loaded.DAHeight)
	require.Equal(t, uint64(0), loaded.TxIndex)

	// Test with max uint64 values
	checkpoint = &Checkpoint{
		DAHeight: ^uint64(0),
		TxIndex:  ^uint64(0),
	}
	err = store.Save(ctx, checkpoint)
	require.NoError(t, err)

	loaded, err = store.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, ^uint64(0), loaded.DAHeight)
	require.Equal(t, ^uint64(0), loaded.TxIndex)
}

func TestCheckpointStore_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	db := ds.NewMapDatastore()
	store := NewCheckpointStore(db)

	// Save initial checkpoint
	checkpoint := &Checkpoint{
		DAHeight: 100,
		TxIndex:  0,
	}
	err := store.Save(ctx, checkpoint)
	require.NoError(t, err)

	// Test concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			loaded, err := store.Load(ctx)
			require.NoError(t, err)
			require.NotNil(t, loaded)
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestEncodeDecodeUint64(t *testing.T) {
	testCases := []uint64{
		0,
		1,
		100,
		1000000,
		^uint64(0), // max uint64
	}

	for _, tc := range testCases {
		encoded := encodeUint64(tc)
		require.Equal(t, 8, len(encoded), "encoded length should be 8 bytes")

		decoded, err := decodeUint64(encoded)
		require.NoError(t, err)
		require.Equal(t, tc, decoded)
	}

	// Test invalid length
	_, err := decodeUint64([]byte{1, 2, 3})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid length")
}
