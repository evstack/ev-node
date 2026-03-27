package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/types"
)

func makeTestEvent(height uint64) *common.DAHeightEvent {
	return &common.DAHeightEvent{
		Header:   &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: height}}},
		DaHeight: height,
	}
}

func TestPendingEventsMap_BasicCRUD(t *testing.T) {
	t.Parallel()
	m := newPendingEventsMap[common.DAHeightEvent]()

	evt1 := makeTestEvent(1)
	evt3 := makeTestEvent(3)
	evt5 := makeTestEvent(5)

	m.setItem(1, evt1)
	m.setItem(3, evt3)
	m.setItem(5, evt5)

	assert.Equal(t, 3, m.itemCount())

	got1 := m.getNextItem(1)
	require.NotNil(t, got1)
	assert.Equal(t, uint64(1), got1.Header.Height())

	assert.Equal(t, 2, m.itemCount())

	got1Again := m.getNextItem(1)
	assert.Nil(t, got1Again)

	got3 := m.getNextItem(3)
	require.NotNil(t, got3)
	assert.Equal(t, uint64(3), got3.Header.Height())
}

func TestPendingEventsMap_UpdateExisting(t *testing.T) {
	t.Parallel()
	m := newPendingEventsMap[common.DAHeightEvent]()

	evt1 := makeTestEvent(1)
	m.setItem(1, evt1)
	assert.Equal(t, 1, m.itemCount())

	evt1Updated := makeTestEvent(1)
	m.setItem(1, evt1Updated)
	assert.Equal(t, 1, m.itemCount())
}

func TestPendingEventsMap_DeleteAllForHeight(t *testing.T) {
	t.Parallel()
	m := newPendingEventsMap[common.DAHeightEvent]()

	m.setItem(1, makeTestEvent(1))
	m.setItem(2, makeTestEvent(2))
	m.setItem(3, makeTestEvent(3))

	m.deleteAllForHeight(2)
	assert.Equal(t, 2, m.itemCount())

	assert.Nil(t, m.getNextItem(2))
	assert.NotNil(t, m.getNextItem(1))
	assert.NotNil(t, m.getNextItem(3))
}
