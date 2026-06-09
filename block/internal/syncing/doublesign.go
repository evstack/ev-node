package syncing

import (
	"fmt"
	"sync"

	"github.com/evstack/ev-node/types"
)

// inBatchDoubleSignReporter reports two distinct, sequencer-signed headers observed at the same height
type inBatchDoubleSignReporter func(height uint64, canonical, alt *types.SignedHeader)

// doubleSignDedup collapses repeated (height, altHash) sightings so the same equivocation
// arriving from multiple batches or sources is only warned and counted once.
type doubleSignDedup struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func newDoubleSignDedup() *doubleSignDedup {
	return &doubleSignDedup{seen: make(map[string]struct{})}
}

// markSeen records (height, altHash) and returns true on first sight.
func (d *doubleSignDedup) markSeen(height uint64, altHash string) bool {
	key := fmt.Sprintf("%d/%s", height, altHash)
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.seen[key]; ok {
		return false
	}
	d.seen[key] = struct{}{}
	return true
}
