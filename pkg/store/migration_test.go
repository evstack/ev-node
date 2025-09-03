package store

import (
	"context"
	"testing"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/evstack/ev-node/types"
)

// TestStateMigrationFromLegacyFormat tests the lazy migration from old state format to new format
func TestStateMigrationFromLegacyFormat(t *testing.T) {
	ctx := context.Background()

	// Create a new store
	db, err := NewDefaultKVStore("", t.TempDir(), "migration_test")
	require.NoError(t, err)
	s := New(db)

	// Create a test state
	testState := types.State{
		LastBlockHeight: 100,
		LastBlockTime:   time.Now(),
		// Add other necessary fields based on your State structure
	}

	// Manually store state in the legacy format (key "s")
	pbState, err := testState.ToProto()
	require.NoError(t, err)
	data, err := proto.Marshal(pbState)
	require.NoError(t, err)

	// Store using legacy key directly
	legacyKey := ds.NewKey("s")
	err = db.Put(ctx, legacyKey, data)
	require.NoError(t, err)

	// Set a height in the store
	err = s.SetHeight(ctx, 100)
	require.NoError(t, err)

	// Verify legacy state exists
	has, err := db.Has(ctx, legacyKey)
	require.NoError(t, err)
	assert.True(t, has, "legacy state should exist before migration")

	// Call GetState which should trigger migration
	retrievedState, err := s.GetState(ctx)
	require.NoError(t, err)
	assert.Equal(t, testState.LastBlockHeight, retrievedState.LastBlockHeight)
	assert.True(t, testState.LastBlockTime.Equal(retrievedState.LastBlockTime), "times should be equal")

	// Verify legacy state is deleted
	has, err = db.Has(ctx, legacyKey)
	require.NoError(t, err)
	assert.False(t, has, "legacy state should be deleted after migration")

	// Verify new format state exists
	newKey := ds.NewKey(getStateAtHeightKey(100))
	has, err = db.Has(ctx, newKey)
	require.NoError(t, err)
	assert.True(t, has, "state should exist in new format after migration")

	// Call GetState again to ensure it works with new format
	retrievedState2, err := s.GetState(ctx)
	require.NoError(t, err)
	assert.Equal(t, testState.LastBlockHeight, retrievedState2.LastBlockHeight)
}

// TestStateMigrationWithUpdateState tests migration cleanup during UpdateState
func TestStateMigrationWithUpdateState(t *testing.T) {
	ctx := context.Background()

	// Create a new store
	db, err := NewDefaultKVStore("", t.TempDir(), "migration_test2")
	require.NoError(t, err)
	s := New(db)

	// Create a test state for legacy format
	legacyState := types.State{
		LastBlockHeight: 50,
		LastBlockTime:   time.Now(),
	}

	// Manually store state in the legacy format
	pbState, err := legacyState.ToProto()
	require.NoError(t, err)
	data, err := proto.Marshal(pbState)
	require.NoError(t, err)

	legacyKey := ds.NewKey("s")
	err = db.Put(ctx, legacyKey, data)
	require.NoError(t, err)

	// Set initial height
	err = s.SetHeight(ctx, 50)
	require.NoError(t, err)

	// Create a new state to update
	newState := types.State{
		LastBlockHeight: 51,
		LastBlockTime:   time.Now(),
	}

	// UpdateState should clean up legacy state
	err = s.UpdateState(ctx, newState)
	require.NoError(t, err)

	// Verify legacy state is deleted
	has, err := db.Has(ctx, legacyKey)
	require.NoError(t, err)
	assert.False(t, has, "legacy state should be deleted after UpdateState")

	// Verify new state is stored in new format
	retrievedState, err := s.GetState(ctx)
	require.NoError(t, err)
	assert.Equal(t, newState.LastBlockHeight, retrievedState.LastBlockHeight)
}

// TestGetStateAtHeightWithLegacyMigration tests GetStateAtHeight with legacy migration
func TestGetStateAtHeightWithLegacyMigration(t *testing.T) {
	ctx := context.Background()

	// Create a new store
	db, err := NewDefaultKVStore("", t.TempDir(), "migration_test3")
	require.NoError(t, err)
	s := New(db)

	// Create a test state for legacy format
	testState := types.State{
		LastBlockHeight: 75,
		LastBlockTime:   time.Now(),
	}

	// Store state in legacy format
	pbState, err := testState.ToProto()
	require.NoError(t, err)
	data, err := proto.Marshal(pbState)
	require.NoError(t, err)

	legacyKey := ds.NewKey("s")
	err = db.Put(ctx, legacyKey, data)
	require.NoError(t, err)

	// Set height
	err = s.SetHeight(ctx, 75)
	require.NoError(t, err)

	// GetStateAtHeight for current height should trigger migration
	retrievedState, err := s.GetStateAtHeight(ctx, 75)
	require.NoError(t, err)
	assert.Equal(t, testState.LastBlockHeight, retrievedState.LastBlockHeight)

	// Verify migration occurred
	has, err := db.Has(ctx, legacyKey)
	require.NoError(t, err)
	assert.False(t, has, "legacy state should be deleted after migration")

	// Verify new format exists
	newKey := ds.NewKey(getStateAtHeightKey(75))
	has, err = db.Has(ctx, newKey)
	require.NoError(t, err)
	assert.True(t, has, "state should exist in new format")
}

// TestNoMigrationWhenNewFormatExists tests that migration doesn't happen if new format already exists
func TestNoMigrationWhenNewFormatExists(t *testing.T) {
	ctx := context.Background()

	// Create a new store
	db, err := NewDefaultKVStore("", t.TempDir(), "migration_test4")
	require.NoError(t, err)
	s := New(db)

	// Create and store state in new format
	newState := types.State{
		LastBlockHeight: 200,
		LastBlockTime:   time.Now(),
	}

	// Set height
	err = s.SetHeight(ctx, 200)
	require.NoError(t, err)

	// Store state using new format
	err = s.UpdateState(ctx, newState)
	require.NoError(t, err)

	// Also create a legacy state (this simulates a partially migrated state)
	legacyState := types.State{
		LastBlockHeight: 100, // Different from new state
		LastBlockTime:   time.Now(),
	}
	pbState, err := legacyState.ToProto()
	require.NoError(t, err)
	data, err := proto.Marshal(pbState)
	require.NoError(t, err)

	legacyKey := ds.NewKey("s")
	err = db.Put(ctx, legacyKey, data)
	require.NoError(t, err)

	// GetState should return new format state, not legacy
	retrievedState, err := s.GetState(ctx)
	require.NoError(t, err)
	assert.Equal(t, newState.LastBlockHeight, retrievedState.LastBlockHeight)
	assert.NotEqual(t, legacyState.LastBlockHeight, retrievedState.LastBlockHeight)

	// Legacy state might still exist since GetState found new format first
	// But UpdateState will clean it up
	err = s.UpdateState(ctx, newState)
	require.NoError(t, err)

	// Now legacy should be gone
	has, err := db.Has(ctx, legacyKey)
	require.NoError(t, err)
	assert.False(t, has, "legacy state should be cleaned up by UpdateState")
}
