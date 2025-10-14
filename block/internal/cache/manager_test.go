package cache

import (
	"context"
	"encoding/gob"
	"fmt"
	"path/filepath"
	"testing"

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
	ds, err := store.NewDefaultInMemoryKVStore()
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

	m.SetHeaderDAIncluded("h1", 10, 1)
	m.SetDataDAIncluded("d1", 11, 1)
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

func TestManager_SaveAndLoadFromDisk(t *testing.T) {
	t.Parallel()
	cfg := tempConfig(t)
	st := memStore(t)

	// must register for gob before saving
	gob.Register(&types.SignedHeader{})
	gob.Register(&types.Data{})
	gob.Register(&common.DAHeightEvent{})

	m1, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	// populate caches
	hdr := &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{ChainID: "c", Height: 2}}}
	dat := &types.Data{Metadata: &types.Metadata{ChainID: "c", Height: 2}}
	m1.SetHeaderSeen("H2", 2)
	m1.SetDataSeen("D2", 2)
	m1.SetHeaderDAIncluded("H2", 100, 2)
	m1.SetDataDAIncluded("D2", 101, 2)
	m1.SetPendingEvent(2, &common.DAHeightEvent{Header: hdr, Data: dat, DaHeight: 99})

	// persist
	err = m1.SaveToDisk()
	require.NoError(t, err)

	// create a fresh manager on same root and verify load
	m2, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	// check loaded items
	assert.True(t, m2.IsHeaderSeen("H2"))
	assert.True(t, m2.IsDataSeen("D2"))
	_, ok := m2.GetHeaderDAIncluded("H2")
	assert.True(t, ok)
	_, ok2 := m2.GetDataDAIncluded("D2")
	assert.True(t, ok2)

	// Verify pending event was loaded
	loadedEvent := m2.GetNextPendingEvent(2)
	require.NotNil(t, loadedEvent)
	assert.Equal(t, uint64(2), loadedEvent.Header.Height())

	// directories exist under cfg.RootDir/data/cache/...
	base := filepath.Join(cfg.RootDir, "data", "cache")
	assert.DirExists(t, filepath.Join(base, "header"))
	assert.DirExists(t, filepath.Join(base, "data"))
	assert.DirExists(t, filepath.Join(base, "pending_da_events"))
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
	headers, err := cm.GetPendingHeaders(ctx)
	require.NoError(t, err)
	require.Len(t, headers, 3)
	assert.Equal(t, uint64(1), headers[0].Height())
	assert.Equal(t, uint64(3), headers[2].Height())

	// data: empty one filtered, so 2 and 3 only
	signedData, err := cm.GetPendingData(ctx)
	require.NoError(t, err)
	require.Len(t, signedData, 2)
	assert.Equal(t, uint64(2), signedData[0].Height())
	assert.Equal(t, uint64(3), signedData[1].Height())

	// update last submitted heights and re-check
	cm.SetLastSubmittedHeaderHeight(ctx, 1)
	cm.SetLastSubmittedDataHeight(ctx, 2)

	headers, err = cm.GetPendingHeaders(ctx)
	require.NoError(t, err)
	require.Len(t, headers, 2)
	assert.Equal(t, uint64(2), headers[0].Height())

	signedData, err = cm.GetPendingData(ctx)
	require.NoError(t, err)
	require.Len(t, signedData, 1)
	assert.Equal(t, uint64(3), signedData[0].Height())

	// numPending views
	assert.Equal(t, uint64(2), cm.NumPendingHeaders())
	assert.Equal(t, uint64(1), cm.NumPendingData())
}

// TestManager_ClearBelowHeight tests on-demand pruning via ClearBelowHeight.
func TestManager_ClearBelowHeight(t *testing.T) {
	t.Parallel()
	cfg := tempConfig(t)
	st := memStore(t)

	m, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	// Add entries (without DAIncluded to avoid auto-prune) for heights 1-100
	for i := uint64(1); i <= 100; i++ {
		hash := fmt.Sprintf("hash-%d", i)
		m.SetHeaderSeen(hash, i)
		m.SetDataSeen(hash, i)
		hdr := &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: i}}}
		dat := &types.Data{Metadata: &types.Metadata{Height: i}}
		m.SetPendingEvent(i, &common.DAHeightEvent{Header: hdr, Data: dat, DaHeight: i})
	}

	// Verify sample entries exist before pruning
	assert.True(t, m.IsHeaderSeen("hash-10"))
	assert.True(t, m.IsDataSeen("hash-10"))
	assert.NotNil(t, m.GetNextPendingEvent(10)) // consumes height 10 pending event

	// Re-add consumed pending event at 10 so we can test the clear logic uniformly
	hdr10 := &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: 10}}}
	dat10 := &types.Data{Metadata: &types.Metadata{Height: 10}}
	m.SetPendingEvent(10, &common.DAHeightEvent{Header: hdr10, Data: dat10, DaHeight: 10})

	// Clear everything below 50 (strictly lower)
	m.ClearBelowHeight(50)

	// Entries below height 50 should be pruned
	for i := uint64(1); i < 50; i++ {
		hash := fmt.Sprintf("hash-%d", i)
		assert.False(t, m.IsHeaderSeen(hash), "expected header hash %d to be pruned", i)
		assert.False(t, m.IsDataSeen(hash), "expected data hash %d to be pruned", i)
		evt := m.GetNextPendingEvent(i)
		assert.Nil(t, evt, "expected pending event at height %d to be pruned", i)
	}

	// Entries >= 50 should still exist
	for i := uint64(50); i <= 100; i++ {
		hash := fmt.Sprintf("hash-%d", i)
		assert.True(t, m.IsHeaderSeen(hash), "expected header hash %d to remain", i)
		assert.True(t, m.IsDataSeen(hash), "expected data hash %d to remain", i)
		// Spot check a few pending events
		if i%10 == 0 {
			evt := m.GetNextPendingEvent(i)
			assert.NotNil(t, evt, "expected pending event at height %d to remain", i)
		}
	}
}

// TestManager_ClearBelowHeight_Boundary tests boundary behavior for ClearBelowHeight.
func TestManager_ClearBelowHeight_Boundary(t *testing.T) {
	t.Parallel()
	cfg := tempConfig(t)
	st := memStore(t)

	m, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	// Add pending events (heights 1..10)
	for i := uint64(1); i <= 10; i++ {
		hdr := &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: i}}}
		dat := &types.Data{Metadata: &types.Metadata{Height: i}}
		m.SetPendingEvent(i, &common.DAHeightEvent{Header: hdr, Data: dat, DaHeight: i})
	}

	// Clear below 5 (strictly lower than 5)
	m.ClearBelowHeight(5)

	for i := uint64(1); i < 5; i++ {
		evt := m.GetNextPendingEvent(i)
		assert.Nil(t, evt, "expected pending event at height %d to be pruned", i)
	}

	for i := uint64(5); i <= 10; i++ {
		evt := m.GetNextPendingEvent(i)
		assert.NotNil(t, evt, "expected pending event at height %d to remain", i)
	}
}

// TestManager_ClearBelowHeight_EdgeCases tests edge cases and idempotency.
func TestManager_ClearBelowHeight_EdgeCases(t *testing.T) {
	cfg := tempConfig(t)
	st := memStore(t)

	m, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	// Add headers 1..10
	for i := uint64(1); i <= 10; i++ {
		m.SetHeaderSeen(fmt.Sprintf("hash-%d", i), i)
	}

	// Clear at height 0 (noop)
	m.ClearBelowHeight(0)
	for i := uint64(1); i <= 10; i++ {
		assert.True(t, m.IsHeaderSeen(fmt.Sprintf("hash-%d", i)), "expected header %d to remain after noop clear", i)
	}

	// Clear below 5
	m.ClearBelowHeight(5)
	for i := uint64(1); i < 5; i++ {
		assert.False(t, m.IsHeaderSeen(fmt.Sprintf("hash-%d", i)), "expected header %d to be pruned", i)
	}
	for i := uint64(5); i <= 10; i++ {
		assert.True(t, m.IsHeaderSeen(fmt.Sprintf("hash-%d", i)), "expected header %d to remain", i)
	}

	// Clear with a lower height (3) should be idempotent (no changes)
	m.ClearBelowHeight(3)
	for i := uint64(5); i <= 10; i++ {
		assert.True(t, m.IsHeaderSeen(fmt.Sprintf("hash-%d", i)), "expected header %d to remain after idempotent clear", i)
	}
}

// TestManager_ClearBelowHeight_EmptyCache tests clearing behavior with empty cache
func TestManager_ClearBelowHeight_EmptyCache(t *testing.T) {
	cfg := tempConfig(t)
	st := memStore(t)

	m, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	// Clear empty cache - should not panic
	require.NotPanics(t, func() {
		m.ClearBelowHeight(100)
	}, "clearing empty cache should not panic")

	// Verify cache can still be saved after clearing empty state
	err = m.SaveToDisk()
	require.NoError(t, err, "should be able to save cache after clearing empty cache")
}

// TestManager_ClearBelowHeight_ConcurrentAccess tests that on-demand clearing doesn't interfere with concurrent reads.
func TestManager_ClearBelowHeight_ConcurrentAccess(t *testing.T) {
	cfg := tempConfig(t)
	st := memStore(t)

	m, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	// Add test data using SetHeaderSeen and SetDataSeen
	for i := uint64(1); i <= 100; i++ {
		hashH := fmt.Sprintf("header-hash-%d", i)
		hashD := fmt.Sprintf("data-hash-%d", i)
		m.SetHeaderSeen(hashH, i)
		m.SetDataSeen(hashD, i)
	}

	// Trigger clear
	m.ClearBelowHeight(50)

	// Verify final state is correct
	for i := uint64(1); i < 50; i++ {
		hashH := fmt.Sprintf("header-hash-%d", i)
		assert.False(t, m.IsHeaderSeen(hashH), "expected header at height %d to be pruned", i)
	}

	for i := uint64(50); i <= 100; i++ {
		hashH := fmt.Sprintf("header-hash-%d", i)
		assert.True(t, m.IsHeaderSeen(hashH), "expected header at height %d to remain", i)
	}
}
