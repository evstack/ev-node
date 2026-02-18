package common

import (
	"context"
	"testing"

	ds "github.com/ipfs/go-datastore"
	"github.com/stretchr/testify/require"
)

var (
	checkpointKey = ds.NewKey("/checkpoint")
)

func TestCheckpointStore_SaveAndLoad(t *testing.T) {
	ctx := context.Background()
	db := ds.NewMapDatastore()
	store := NewCheckpointStore(db, checkpointKey)

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
	store := NewCheckpointStore(db, checkpointKey)

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

func TestCheckpoint_EdgeCases(t *testing.T) {
	ctx := context.Background()
	db := ds.NewMapDatastore()
	store := NewCheckpointStore(db, checkpointKey)

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
	store := NewCheckpointStore(db, checkpointKey)

	// Save initial checkpoint
	checkpoint := &Checkpoint{
		DAHeight: 100,
		TxIndex:  0,
	}
	err := store.Save(ctx, checkpoint)
	require.NoError(t, err)

	// Test concurrent reads
	done := make(chan bool, 10)
	for range 10 {
		go func() {
			defer func() { done <- true }()
			loaded, err := store.Load(ctx)
			require.NoError(t, err)
			require.NotNil(t, loaded)
		}()
	}

	for range 10 {
		<-done
	}
}
