package cache

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	lru "github.com/hashicorp/golang-lru/v2"
	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/pkg/store"
)

// DefaultPendingCacheSize is the default size for the pending items cache.
const DefaultPendingCacheSize = 200_000

// pendingBase is a generic struct for tracking items (headers, data, etc.)
// that need to be published to the DA layer in order. It handles persistence
// of the last submitted height and provides methods for retrieving pending items.
type pendingBase[T any] struct {
	logger     zerolog.Logger
	store      store.Store
	metaKey    string
	fetch      func(ctx context.Context, store store.Store, height uint64) (T, error)
	lastHeight atomic.Uint64

	// Pending items cache to avoid re-fetching all items on every call.
	// We cache the items themselves, keyed by height.
	pendingCache *lru.Cache[uint64, T]

	mu sync.Mutex // Protects getPending logic
}

// newPendingBase constructs a new pendingBase for a given type.
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
func (pb *pendingBase[T]) getPending(ctx context.Context) ([]T, error) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	lastSubmitted := pb.lastHeight.Load()
	storeHeight, err := pb.store.Height(ctx)
	if err != nil {
		return nil, err
	}
	if lastSubmitted == storeHeight {
		return nil, nil
	}
	if lastSubmitted > storeHeight {
		return nil, fmt.Errorf("height of last submitted item (%d) is greater than height of last item (%d)", lastSubmitted, storeHeight)
	}

	// Fetch only items that are not already in cache
	for h := lastSubmitted + 1; h <= storeHeight; h++ {
		if _, ok := pb.pendingCache.Peek(h); ok {
			continue // Already cached, skip fetching
		}
		item, err := pb.fetch(ctx, pb.store, h)
		if err != nil {
			return nil, err
		}
		pb.pendingCache.Add(h, item)
	}

	// Build the result slice from cache
	pending := make([]T, 0, storeHeight-lastSubmitted)
	for h := lastSubmitted + 1; h <= storeHeight; h++ {
		if item, ok := pb.pendingCache.Get(h); ok {
			pending = append(pending, item)
		} else {
			// This shouldn't happen, but fetch if missing
			item, err := pb.fetch(ctx, pb.store, h)
			if err != nil {
				return pending, err
			}
			pb.pendingCache.Add(h, item)
			pending = append(pending, item)
		}
	}
	return pending, nil
}

func (pb *pendingBase[T]) numPending() uint64 {
	height, err := pb.store.Height(context.Background())
	if err != nil {
		pb.logger.Error().Err(err).Msg("failed to get height in numPending")
		return 0
	}
	return height - pb.lastHeight.Load()
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
		// Note: We don't explicitly clear the cache here. Instead, we use lazy invalidation
		// by checking against lastHeight. This avoids O(N) iteration over the cache on every
		// submission. The LRU will naturally evict old entries when capacity is reached.
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
