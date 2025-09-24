package cache

import (
	"context"
	"encoding/gob"
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
	m.SetHeaderSeen("h1")
	m.SetDataSeen("d1")
	assert.True(t, m.IsHeaderSeen("h1"))
	assert.True(t, m.IsDataSeen("d1"))

	m.SetHeaderDAIncluded("h1", 10)
	m.SetDataDAIncluded("d1", 11)
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
	m1.SetHeaderSeen("H2")
	m1.SetDataSeen("D2")
	m1.SetHeaderDAIncluded("H2", 100)
	m1.SetDataDAIncluded("D2", 101)
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

	// persist in store and set height
	for _, pair := range []struct {
		h *types.SignedHeader
		d *types.Data
	}{{h1, d1}, {h2, d2}, {h3, d3}} {
		err := st.SaveBlockData(ctx, pair.h, pair.d, &types.Signature{})
		require.NoError(t, err)
	}
	require.NoError(t, st.SetHeight(ctx, 3))

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
