package cache

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"

	lru "github.com/hashicorp/golang-lru/v2"
	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/pkg/store"
)

// DefaultPendingCacheSize is the default size for the pending items cache.
const DefaultPendingCacheSize = 200_000

// inFlightClaim tracks a contiguous range of heights claimed by getPending
// for DA submission. Claims prevent concurrent getPending calls from
// returning the same items. On submission failure, resetInFlightRange
// removes the claim; if the range is below lastHeight it is added to gaps
// so the items can be re-fetched.
type inFlightClaim struct{ start, end uint64 }

// pendingBase is a generic struct for tracking items (headers, data, etc.)
// that need to be published to the DA layer in order. It handles persistence
// of the last submitted height and provides methods for retrieving pending items.
type pendingBase[T any] struct {
	logger     zerolog.Logger
	store      store.Store
	metaKey    string
	fetch      func(ctx context.Context, store store.Store, height uint64) (T, error)
	lastHeight atomic.Uint64

	// inFlightMu protects inFlightClaims and gaps.
	inFlightMu     sync.Mutex
	inFlightClaims []inFlightClaim // sorted by start
	// gaps holds ranges that failed after a later submission advanced lastHeight
	// past them. getPending checks gaps first so these items are retried.
	gaps []inFlightClaim // sorted by start

	// Pending items cache to avoid re-fetching all items on every call.
	pendingCache *lru.Cache[uint64, T]

	mu sync.Mutex // Protects getPending logic
}

func newPendingBase[T any](store store.Store, logger zerolog.Logger, metaKey string, fetch func(ctx context.Context, store store.Store, height uint64) (T, error)) (*pendingBase[T], error) {
	pendingCache, err := lru.New[uint64, T](DefaultPendingCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create pending cache: %w", err)
	}

	pb := &pendingBase[T]{
		store:        store,
		logger:       logger,
		metaKey:      metaKey,
		fetch:        fetch,
		pendingCache: pendingCache,
	}
	if err := pb.init(); err != nil {
		return nil, err
	}
	return pb, nil
}

// getPending returns a sorted slice of pending items of type T.
// It caches fetched items to avoid re-fetching on subsequent calls.
// Returned items are registered as an in-flight claim so that concurrent
// callers do not receive the same items. On failure, the caller must call
// resetInFlightRange with the same [start, end] to re-expose the items.
func (pb *pendingBase[T]) getPending(ctx context.Context) ([]T, error) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	lastSubmitted := pb.lastHeight.Load()
	storeHeight, err := pb.store.Height(ctx)
	if err != nil {
		return nil, err
	}
	if lastSubmitted > storeHeight {
		return nil, fmt.Errorf("last submitted height (%d) is greater than store height (%d)", lastSubmitted, storeHeight)
	}

	pb.inFlightMu.Lock()
	rangeStart, rangeEnd := findAvailableRange(pb.gaps, pb.inFlightClaims, lastSubmitted, storeHeight)
	pb.inFlightMu.Unlock()

	if rangeStart == 0 || rangeStart > rangeEnd {
		return nil, nil
	}

	// Cap range to cache capacity
	if rangeEnd-rangeStart+1 > uint64(DefaultPendingCacheSize) {
		rangeEnd = rangeStart + uint64(DefaultPendingCacheSize) - 1
	}

	pending := make([]T, 0, rangeEnd-rangeStart+1)
	for h := rangeStart; h <= rangeEnd; h++ {
		if item, ok := pb.pendingCache.Get(h); ok {
			pending = append(pending, item)
			continue
		}
		item, err := pb.fetch(ctx, pb.store, h)
		if err != nil {
			return pending, err
		}
		pb.pendingCache.Add(h, item)
		pending = append(pending, item)
	}

	if len(pending) > 0 {
		pb.inFlightMu.Lock()
		pb.inFlightClaims = insertClaim(pb.inFlightClaims, inFlightClaim{start: rangeStart, end: rangeEnd})
		pb.gaps = removeGapRange(pb.gaps, rangeStart, rangeEnd)
		pb.inFlightMu.Unlock()
	}

	return pending, nil
}

func (pb *pendingBase[T]) numPending() uint64 {
	height, err := pb.store.Height(context.Background())
	if err != nil {
		pb.logger.Error().Err(err).Msg("failed to get height in numPending")
		return 0
	}

	lastSubmitted := pb.lastHeight.Load()

	pb.inFlightMu.Lock()
	var count uint64
	for _, gap := range pb.gaps {
		count += countUnclaimed(gap.start, gap.end, pb.inFlightClaims)
	}
	if height > lastSubmitted {
		count += countUnclaimed(lastSubmitted+1, height, pb.inFlightClaims)
	}
	pb.inFlightMu.Unlock()

	return count
}

func (pb *pendingBase[T]) numPendingTotal() uint64 {
	height, err := pb.store.Height(context.Background())
	if err != nil {
		pb.logger.Error().Err(err).Msg("failed to get height in numPendingTotal")
		return 0
	}

	lastSubmitted := pb.lastHeight.Load()

	if height <= lastSubmitted {
		return 0
	}

	return height - lastSubmitted
}

func (pb *pendingBase[T]) getLastSubmittedHeight() uint64 {
	return pb.lastHeight.Load()
}

func (pb *pendingBase[T]) setLastSubmittedHeight(ctx context.Context, newLastSubmittedHeight uint64) {
	lsh := pb.lastHeight.Load()
	if newLastSubmittedHeight > lsh && pb.lastHeight.CompareAndSwap(lsh, newLastSubmittedHeight) {
		bz := make([]byte, 8)
		binary.LittleEndian.PutUint64(bz, newLastSubmittedHeight)
		err := pb.store.SetMetadata(ctx, pb.metaKey, bz)
		if err != nil {
			pb.logger.Error().Err(err).Msg("failed to store height of latest item submitted to DA")
		}
	}

	pb.inFlightMu.Lock()
	pb.inFlightClaims = trimClaimsBelow(pb.inFlightClaims, newLastSubmittedHeight)
	pb.gaps = trimGapsBelow(pb.gaps, newLastSubmittedHeight)
	pb.inFlightMu.Unlock()
}

// resetInFlightRange removes any in-flight claim overlapping [start, end].
// If the claim has been trimmed by setLastSubmittedHeight (partial success),
// the trimmed claim range is used for gap computation to avoid re-exposing
// already-submitted items. If no claim is found (removed by a concurrent
// setLastSubmittedHeight), the caller's range is used instead.
func (pb *pendingBase[T]) resetInFlightRange(start, end uint64) {
	pb.inFlightMu.Lock()
	defer pb.inFlightMu.Unlock()

	var removedClaim *inFlightClaim
	n := 0
	for _, c := range pb.inFlightClaims {
		if c.end < start || c.start > end {
			pb.inFlightClaims[n] = c
			n++
		} else {
			cc := c
			removedClaim = &cc
		}
	}
	pb.inFlightClaims = pb.inFlightClaims[:n]

	currentLast := pb.lastHeight.Load()

	var gapStart, gapEnd uint64
	if removedClaim != nil {
		gapStart = removedClaim.start
		gapEnd = removedClaim.end
	} else {
		gapStart = start
		gapEnd = end
	}

	if gapStart > currentLast {
		return
	}
	if gapEnd > currentLast {
		gapEnd = currentLast
	}
	if gapStart <= gapEnd {
		pb.gaps = insertClaim(pb.gaps, inFlightClaim{start: gapStart, end: gapEnd})
	}
}

func (pb *pendingBase[T]) init() error {
	raw, err := pb.store.GetMetadata(context.Background(), pb.metaKey)
	if errors.Is(err, ds.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(raw) != 8 {
		return fmt.Errorf("invalid length of last submitted height: %d, expected 8", len(raw))
	}
	lsh := binary.LittleEndian.Uint64(raw)
	if lsh == 0 {
		return nil
	}
	pb.lastHeight.CompareAndSwap(0, lsh)
	return nil
}

// ---------------------------------------------------------------------------
// Helper functions for claim / gap bookkeeping
// ---------------------------------------------------------------------------

// findAvailableRange returns the first contiguous range of heights that are
// not covered by any active claim. Gaps take priority; then items above
// lastHeight. Returns (0, 0) when nothing is available.
func findAvailableRange(gaps, claims []inFlightClaim, lastHeight, storeHeight uint64) (uint64, uint64) {
	// Check gaps first
	for _, gap := range gaps {
		s, e := firstUnclaimed(gap.start, gap.end, claims)
		if s <= e {
			return s, e
		}
	}
	// Items above lastHeight
	if lastHeight < storeHeight {
		s, e := firstUnclaimed(lastHeight+1, storeHeight, claims)
		return s, e
	}
	return 0, 0
}

// firstUnclaimed returns the first contiguous unclaimed sub-range within [lo, hi].
func firstUnclaimed(lo, hi uint64, claims []inFlightClaim) (uint64, uint64) {
	h := lo
	ci := 0
	for ci < len(claims) && claims[ci].end < h {
		ci++
	}
	// Skip past any claim that covers h
	for ci < len(claims) && h >= claims[ci].start && h <= claims[ci].end {
		h = claims[ci].end + 1
		ci++
		for ci < len(claims) && claims[ci].end < h {
			ci++
		}
	}
	if h > hi {
		return 0, 0
	}
	end := hi
	if ci < len(claims) && claims[ci].start <= hi {
		end = claims[ci].start - 1
	}
	return h, end
}

// countUnclaimed counts heights in [lo, hi] not covered by any claim.
func countUnclaimed(lo, hi uint64, claims []inFlightClaim) uint64 {
	if lo > hi {
		return 0
	}
	total := hi - lo + 1
	for _, c := range claims {
		if c.end < lo || c.start > hi {
			continue
		}
		ovStart := max(c.start, lo)
		ovEnd := min(c.end, hi)
		total -= ovEnd - ovStart + 1
	}
	return total
}

// insertClaim inserts a claim into a sorted-by-start slice, keeping it sorted.
func insertClaim(sorted []inFlightClaim, c inFlightClaim) []inFlightClaim {
	idx, _ := slices.BinarySearchFunc(sorted, c.start, func(e inFlightClaim, v uint64) int {
		if e.start < v {
			return -1
		}
		if e.start > v {
			return 1
		}
		return 0
	})
	return slices.Insert(sorted, idx, c)
}

// removeGapRange removes or trims gaps covered by [start, end].
func removeGapRange(gaps []inFlightClaim, start, end uint64) []inFlightClaim {
	var result []inFlightClaim
	for _, g := range gaps {
		if g.end < start || g.start > end {
			result = append(result, g)
			continue
		}
		// Partial overlap: keep portions outside [start, end]
		if g.start < start {
			result = append(result, inFlightClaim{start: g.start, end: start - 1})
		}
		if g.end > end {
			result = append(result, inFlightClaim{start: end + 1, end: g.end})
		}
	}
	return result
}

// trimClaimsBelow removes the portion of each claim that is <= height.
func trimClaimsBelow(claims []inFlightClaim, height uint64) []inFlightClaim {
	result := claims[:0]
	for _, c := range claims {
		if c.end <= height {
			continue
		}
		if c.start <= height {
			c.start = height + 1
		}
		result = append(result, c)
	}
	return result
}

// trimGapsBelow removes gaps fully below height and trims partial overlaps.
func trimGapsBelow(gaps []inFlightClaim, height uint64) []inFlightClaim {
	result := gaps[:0]
	for _, g := range gaps {
		if g.end <= height {
			continue
		}
		if g.start <= height {
			g.start = height + 1
		}
		result = append(result, g)
	}
	return result
}
