package cache

import "sync"

// pendingEventsMap stores height-keyed events.
// Events are removed by getNextItem (once processed) or deleteAllForHeight (once DA-included/finalized).
type pendingEventsMap[T any] struct {
	mu    sync.Mutex
	items map[uint64]*T
}

func newPendingEventsMap[T any]() *pendingEventsMap[T] {
	return &pendingEventsMap[T]{
		items: make(map[uint64]*T),
	}
}

func (m *pendingEventsMap[T]) setItem(height uint64, item *T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[height] = item
}

func (m *pendingEventsMap[T]) getNextItem(height uint64) *T {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.items[height]
	if !ok {
		return nil
	}
	delete(m.items, height)
	return item
}

func (m *pendingEventsMap[T]) itemCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.items)
}

func (m *pendingEventsMap[T]) deleteAllForHeight(height uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, height)
}
