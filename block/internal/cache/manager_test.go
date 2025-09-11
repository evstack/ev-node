package cache

import (
	"encoding/gob"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

// helper to make a temp config rooted at t.TempDir()
func tempConfig(t *testing.T) config.Config {
	cfg := config.DefaultConfig
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

	// construct simple header and data
	header := &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{ChainID: "test", Height: 1}}}
	data := &types.Data{Metadata: &types.Metadata{ChainID: "test", Height: 1}}

	// set and get by height
	m.SetHeader(1, header)
	m.SetData(1, data)
	assert.Equal(t, header, m.GetHeader(1))
	assert.Equal(t, data, m.GetData(1))

	// seen & DA included flags
	m.SetHeaderSeen("h1")
	m.SetDataSeen("d1")
	assert.True(t, m.IsHeaderSeen("h1"))
	assert.True(t, m.IsDataSeen("d1"))

	m.SetHeaderDAIncluded("h1", 10)
	m.SetDataDAIncluded("d1", 11)
	assert.True(t, m.IsHeaderDAIncluded("h1"))
	assert.True(t, m.IsDataDAIncluded("d1"))

	// cleanup of processed
	m.ClearProcessedHeader(1)
	m.ClearProcessedData(1)
	assert.Nil(t, m.GetHeader(1))
	assert.Nil(t, m.GetData(1))
}

func TestManager_PendingEventsCRUD(t *testing.T) {
	t.Parallel()
	cfg := tempConfig(t)
	st := memStore(t)

	m, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	evt1 := &DAHeightEvent{Header: &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: 1}}}, DaHeight: 5}
	evt3 := &DAHeightEvent{Header: &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: 3}}}, DaHeight: 7}
	evt5 := &DAHeightEvent{Data: &types.Data{Metadata: &types.Metadata{Height: 5}}, DaHeight: 9}

	m.SetPendingEvent(5, evt5)
	m.SetPendingEvent(1, evt1)
	m.SetPendingEvent(3, evt3)

	got := m.GetPendingEvents()
	require.Len(t, got, 3)
	assert.Equal(t, evt1, got[1])
	assert.Equal(t, evt3, got[3])
	assert.Equal(t, evt5, got[5])

	// delete and re-check
	m.DeletePendingEvent(3)
	got = m.GetPendingEvents()
	require.Len(t, got, 2)
	_, ok := got[3]
	assert.False(t, ok)
}

func TestManager_SaveAndLoadFromDisk(t *testing.T) {
	t.Parallel()
	cfg := tempConfig(t)
	st := memStore(t)

	// must register for gob before saving
	gob.Register(&types.SignedHeader{})
	gob.Register(&types.Data{})
	gob.Register(&DAHeightEvent{})

	m1, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	// populate caches
	hdr := &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{ChainID: "c", Height: 2}}}
	dat := &types.Data{Metadata: &types.Metadata{ChainID: "c", Height: 2}}
	m1.SetHeader(2, hdr)
	m1.SetData(2, dat)
	m1.SetHeaderSeen("H2")
	m1.SetDataSeen("D2")
	m1.SetHeaderDAIncluded("H2", 100)
	m1.SetDataDAIncluded("D2", 101)
	m1.SetPendingEvent(2, &DAHeightEvent{Header: hdr, Data: dat, DaHeight: 99})

	// persist
	err = m1.SaveToDisk()
	require.NoError(t, err)

	// create a fresh manager on same root and verify load
	m2, err := NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	// check loaded items
	assert.NotNil(t, m2.GetHeader(2))
	assert.NotNil(t, m2.GetData(2))
	assert.True(t, m2.IsHeaderSeen("H2"))
	assert.True(t, m2.IsDataSeen("D2"))
	assert.True(t, m2.IsHeaderDAIncluded("H2"))
	assert.True(t, m2.IsDataDAIncluded("D2"))

	events := m2.GetPendingEvents()
	require.Len(t, events, 1)
	assert.Equal(t, uint64(2), events[2].Header.Height())

	// directories exist under cfg.RootDir/data/cache/...
	base := filepath.Join(cfg.RootDir, "data", "cache")
	assert.DirExists(t, filepath.Join(base, "header"))
	assert.DirExists(t, filepath.Join(base, "data"))
	assert.DirExists(t, filepath.Join(base, "pending_da_events"))
}
