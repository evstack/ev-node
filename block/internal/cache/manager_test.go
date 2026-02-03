package cache

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

// helper to make a temp config rooted at t.TempDir()
func tempConfig(t *testing.T) config.Config {
	cfg := config.DefaultConfig()
	cfg.RootDir = t.TempDir()
	return cfg
}

// helper to make an in-memory store
func memStore(t *testing.T) store.Store {
	ds, err := store.NewTestInMemoryKVStore()
	require.NoError(t, err)
	return store.New(ds)
}

func TestManager_HeaderDataOperations(t *testing.T) {
	t.Parallel()
	cfg := tempConfig(t)
	st := memStore(t)

	m, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	// seen & DA included flags
	m.SetHeaderSeen("h1", 1)
	m.SetDataSeen("d1", 1)
	assert.True(t, m.IsHeaderSeen("h1"))
	assert.True(t, m.IsDataSeen("d1"))

	m.SetHeaderDAIncluded("h1", 10, 2)
	m.SetDataDAIncluded("d1", 11, 2)
	_, ok := m.GetHeaderDAIncluded("h1")
	assert.True(t, ok)
	_, ok = m.GetDataDAIncluded("d1")
	assert.True(t, ok)
}

func TestManager_PendingEventsCRUD(t *testing.T) {
	t.Parallel()
	cfg := tempConfig(t)
	st := memStore(t)

	m, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	evt1 := &common.DAHeightEvent{Header: &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: 1}}}, DaHeight: 5}
	evt3 := &common.DAHeightEvent{Header: &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: 3}}}, DaHeight: 7}
	evt5 := &common.DAHeightEvent{Data: &types.Data{Metadata: &types.Metadata{Height: 5}}, DaHeight: 9}

	m.SetPendingEvent(5, evt5)
	m.SetPendingEvent(1, evt1)
	m.SetPendingEvent(3, evt3)

	// Test getting specific events
	got1 := m.GetNextPendingEvent(1)
	require.NotNil(t, got1)
	assert.Equal(t, evt1.DaHeight, got1.DaHeight)

	got3 := m.GetNextPendingEvent(3)
	require.NotNil(t, got3)
	assert.Equal(t, evt3.DaHeight, got3.DaHeight)

	got5 := m.GetNextPendingEvent(5)
	require.NotNil(t, got5)
	assert.Equal(t, evt5.DaHeight, got5.DaHeight)

	// Events should be removed after GetNextPendingEvent
	got1Again := m.GetNextPendingEvent(1)
	assert.Nil(t, got1Again)
}

func TestManager_SaveAndRestoreFromStore(t *testing.T) {
	t.Parallel()
	cfg := tempConfig(t)
	st := memStore(t)
	ctx := context.Background()

	// First, we need to save some block data to the store so RestoreFromStore can find the hashes
	h1, d1 := types.GetRandomBlock(1, 1, "test-chain")
	h2, d2 := types.GetRandomBlock(2, 1, "test-chain")

	batch1, err := st.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch1.SaveBlockData(h1, d1, &types.Signature{}))
	require.NoError(t, batch1.SetHeight(1))
	require.NoError(t, batch1.Commit())

	batch2, err := st.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch2.SaveBlockData(h2, d2, &types.Signature{}))
	require.NoError(t, batch2.SetHeight(2))
	require.NoError(t, batch2.Commit())

	m1, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	// Set DA inclusion for the blocks
	m1.SetHeaderDAIncluded(h1.Hash().String(), 100, 1)
	m1.SetDataDAIncluded(d1.DACommitment().String(), 100, 1)
	m1.SetHeaderDAIncluded(h2.Hash().String(), 101, 2)
	m1.SetDataDAIncluded(d2.DACommitment().String(), 101, 2)

	// Persist to store
	err = m1.SaveToStore()
	require.NoError(t, err)

	// Create a fresh manager on same store and verify restore
	m2, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	// Check DA inclusion was restored
	daHeight, ok := m2.GetHeaderDAIncluded(h1.Hash().String())
	assert.True(t, ok)
	assert.Equal(t, uint64(100), daHeight)

	daHeight, ok = m2.GetDataDAIncluded(d1.DACommitment().String())
	assert.True(t, ok)
	assert.Equal(t, uint64(100), daHeight)

	daHeight, ok = m2.GetHeaderDAIncluded(h2.Hash().String())
	assert.True(t, ok)
	assert.Equal(t, uint64(101), daHeight)

	daHeight, ok = m2.GetDataDAIncluded(d2.DACommitment().String())
	assert.True(t, ok)
	assert.Equal(t, uint64(101), daHeight)
}

func TestManager_GetNextPendingEvent_NonExistent(t *testing.T) {
	t.Parallel()
	cfg := tempConfig(t)
	st := memStore(t)

	m, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	// Try to get non-existent event
	event := m.GetNextPendingEvent(999)
	assert.Nil(t, event)
}

func TestPendingHeadersAndData_Flow(t *testing.T) {
	t.Parallel()
	st := memStore(t)
	ctx := context.Background()
	logger := zerolog.Nop()

	// create 3 blocks, with varying number of txs to test data filtering
	chainID := "chain-pending"
	h1, d1 := types.GetRandomBlock(1, 0, chainID) // empty data -> should be filtered out
	h2, d2 := types.GetRandomBlock(2, 1, chainID)
	h3, d3 := types.GetRandomBlock(3, 2, chainID)

	// persist in store and set height using batch
	for i, pair := range []struct {
		h *types.SignedHeader
		d *types.Data
	}{{h1, d1}, {h2, d2}, {h3, d3}} {
		batch, err := st.NewBatch(ctx)
		require.NoError(t, err)
		err = batch.SaveBlockData(pair.h, pair.d, &types.Signature{})
		require.NoError(t, err)
		err = batch.SetHeight(uint64(i + 1))
		require.NoError(t, err)
		err = batch.Commit()
		require.NoError(t, err)
	}

	// construct manager which brings up pending managers
	cfg := tempConfig(t)
	cm, err := NewManager(cfg, st, logger)
	require.NoError(t, err)

	// headers: all 3 should be pending initially
	headers, _, err := cm.GetPendingHeaders(ctx)
	require.NoError(t, err)
	require.Len(t, headers, 3)
	assert.Equal(t, uint64(1), headers[0].Height())
	assert.Equal(t, uint64(3), headers[2].Height())

	// data: empty one filtered, so 2 and 3 only
	signedData, _, err := cm.GetPendingData(ctx)
	require.NoError(t, err)
	require.Len(t, signedData, 2)
	assert.Equal(t, uint64(2), signedData[0].Height())
	assert.Equal(t, uint64(3), signedData[1].Height())

	// update last submitted heights and re-check
	cm.SetLastSubmittedHeaderHeight(ctx, 1)
	cm.SetLastSubmittedDataHeight(ctx, 2)

	headers, _, err = cm.GetPendingHeaders(ctx)
	require.NoError(t, err)
	require.Len(t, headers, 2)
	assert.Equal(t, uint64(2), headers[0].Height())

	signedData, _, err = cm.GetPendingData(ctx)
	require.NoError(t, err)
	require.Len(t, signedData, 1)
	assert.Equal(t, uint64(3), signedData[0].Height())

	// numPending views
	assert.Equal(t, uint64(2), cm.NumPendingHeaders())
	assert.Equal(t, uint64(1), cm.NumPendingData())
}

func TestManager_TxOperations(t *testing.T) {
	t.Parallel()
	cfg := tempConfig(t)
	st := memStore(t)

	m, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	// Initially not seen
	assert.False(t, m.IsTxSeen("tx1"))
	assert.False(t, m.IsTxSeen("tx2"))

	// Mark as seen
	m.SetTxSeen("tx1")
	m.SetTxSeen("tx2")

	// Should now be seen
	assert.True(t, m.IsTxSeen("tx1"))
	assert.True(t, m.IsTxSeen("tx2"))
}

func TestManager_CleanupOldTxs(t *testing.T) {
	t.Parallel()
	cfg := tempConfig(t)
	st := memStore(t)

	m, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	// Add some transactions
	m.SetTxSeen("tx1")
	m.SetTxSeen("tx2")
	m.SetTxSeen("tx3")

	// Verify all are seen
	assert.True(t, m.IsTxSeen("tx1"))
	assert.True(t, m.IsTxSeen("tx2"))
	assert.True(t, m.IsTxSeen("tx3"))

	// Cleanup with a very short duration (should remove all)
	removed := m.CleanupOldTxs(1 * time.Nanosecond)
	assert.Equal(t, 3, removed)

	// All should now be gone
	assert.False(t, m.IsTxSeen("tx1"))
	assert.False(t, m.IsTxSeen("tx2"))
	assert.False(t, m.IsTxSeen("tx3"))
}

func TestManager_CleanupOldTxs_SelectiveRemoval(t *testing.T) {
	t.Parallel()
	cfg := tempConfig(t)
	st := memStore(t)

	m, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	impl := m.(*implementation)

	// Add old transactions with backdated timestamps
	oldTime := time.Now().Add(-48 * time.Hour)
	impl.txCache.setSeen("old-tx1", 0)
	impl.txTimestamps.Store("old-tx1", oldTime)
	impl.txCache.setSeen("old-tx2", 0)
	impl.txTimestamps.Store("old-tx2", oldTime)

	// Add recent transactions
	m.SetTxSeen("new-tx1")
	m.SetTxSeen("new-tx2")

	// Cleanup transactions older than 24 hours
	removed := m.CleanupOldTxs(24 * time.Hour)
	assert.Equal(t, 2, removed)

	// Old transactions should be gone
	assert.False(t, m.IsTxSeen("old-tx1"))
	assert.False(t, m.IsTxSeen("old-tx2"))

	// New transactions should still be present
	assert.True(t, m.IsTxSeen("new-tx1"))
	assert.True(t, m.IsTxSeen("new-tx2"))
}

func TestManager_CleanupOldTxs_DefaultDuration(t *testing.T) {
	t.Parallel()
	cfg := tempConfig(t)
	st := memStore(t)

	m, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	impl := m.(*implementation)

	// Add transactions with specific old timestamp (older than default 24h)
	veryOldTime := time.Now().Add(-25 * time.Hour)
	impl.txCache.setSeen("very-old-tx", 0)
	impl.txTimestamps.Store("very-old-tx", veryOldTime)

	// Add recent transaction
	m.SetTxSeen("recent-tx")

	// Call cleanup with 0 duration (should use default)
	removed := m.CleanupOldTxs(0)
	assert.Equal(t, 1, removed)

	// Very old transaction should be gone
	assert.False(t, m.IsTxSeen("very-old-tx"))

	// Recent transaction should still be present
	assert.True(t, m.IsTxSeen("recent-tx"))
}

func TestManager_CleanupOldTxs_NoTransactions(t *testing.T) {
	t.Parallel()
	cfg := tempConfig(t)
	st := memStore(t)

	m, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	// Cleanup on empty cache should return 0
	removed := m.CleanupOldTxs(24 * time.Hour)
	assert.Equal(t, 0, removed)
}

func TestManager_TxCache_NotPersistedToStore(t *testing.T) {
	t.Parallel()
	cfg := tempConfig(t)
	st := memStore(t)

	// Create first manager and add transactions
	m1, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	m1.SetTxSeen("persistent-tx1")
	m1.SetTxSeen("persistent-tx2")

	assert.True(t, m1.IsTxSeen("persistent-tx1"))
	assert.True(t, m1.IsTxSeen("persistent-tx2"))

	// Save to store
	err = m1.SaveToStore()
	require.NoError(t, err)

	// Create new manager - tx cache should be empty (not persisted)
	m2, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	// TX cache is ephemeral and not persisted
	assert.False(t, m2.IsTxSeen("persistent-tx1"))
	assert.False(t, m2.IsTxSeen("persistent-tx2"))
}

func TestManager_DeleteHeight_PreservesTxCache(t *testing.T) {
	t.Parallel()
	cfg := tempConfig(t)
	st := memStore(t)

	m, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	// Add items to various caches at height 5
	m.SetHeaderSeen("header-5", 5)
	m.SetDataSeen("data-5", 5)
	m.SetTxSeen("tx-persistent")

	// Verify all are present
	assert.True(t, m.IsHeaderSeen("header-5"))
	assert.True(t, m.IsDataSeen("data-5"))
	assert.True(t, m.IsTxSeen("tx-persistent"))

	// Delete height 5
	m.DeleteHeight(5)

	// Header and data should be gone
	assert.False(t, m.IsHeaderSeen("header-5"))
	assert.False(t, m.IsDataSeen("data-5"))

	// Transaction should still be present (height-independent)
	assert.True(t, m.IsTxSeen("tx-persistent"))
}

func TestManager_DAInclusionPersistence(t *testing.T) {
	t.Parallel()
	cfg := tempConfig(t)
	st := memStore(t)
	ctx := context.Background()

	// Create blocks and save to store
	h1, d1 := types.GetRandomBlock(1, 1, "test-chain")

	batch, err := st.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SaveBlockData(h1, d1, &types.Signature{}))
	require.NoError(t, batch.SetHeight(1))
	require.NoError(t, batch.Commit())

	// Create manager and set DA inclusion
	m1, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	headerHash := h1.Hash().String()
	dataHash := d1.DACommitment().String()

	m1.SetHeaderDAIncluded(headerHash, 100, 1)
	m1.SetDataDAIncluded(dataHash, 101, 1)

	// Verify DA height is tracked
	assert.Equal(t, uint64(101), m1.DaHeight())

	// Create new manager - DA inclusion should be restored
	m2, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	// DA inclusion should be restored from store
	daHeight, ok := m2.GetHeaderDAIncluded(headerHash)
	assert.True(t, ok)
	assert.Equal(t, uint64(100), daHeight)

	daHeight, ok = m2.GetDataDAIncluded(dataHash)
	assert.True(t, ok)
	assert.Equal(t, uint64(101), daHeight)

	// Max DA height should also be restored
	assert.Equal(t, uint64(101), m2.DaHeight())
}

func TestCacheManager_Creation(t *testing.T) {
	t.Parallel()
	cfg := tempConfig(t)
	st := memStore(t)

	cm, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)
	require.NotNil(t, cm)

	// Test basic operations
	cm.SetHeaderSeen("h1", 1)
	assert.True(t, cm.IsHeaderSeen("h1"))

	cm.SetHeaderDAIncluded("h1", 100, 1)
	daHeight, ok := cm.GetHeaderDAIncluded("h1")
	assert.True(t, ok)
	assert.Equal(t, uint64(100), daHeight)
}
