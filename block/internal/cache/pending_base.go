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

const (
	// DefaultMarshalledCacheSize is the default size for the marshalled bytes cache.
	// This bounds memory usage while still providing good cache hit rates.
	// Each marshalled header is ~1-2KB, so 1M entries ≈ 1-2GB max.
	DefaultMarshalledCacheSize = 1_000_000
)

// pendingBase is a generic struct for tracking items (headers, data, etc.)
// that need to be published to the DA layer in order. It handles persistence
// of the last submitted height and provides methods for retrieving pending items.
type pendingBase[T any] struct {
	logger     zerolog.Logger
	store      store.Store
	metaKey    string
	fetch      func(ctx context.Context, store store.Store, height uint64) (T, error)
	lastHeight atomic.Uint64

	// Marshalling cache to avoid redundant marshalling - now bounded with LRU
	marshalledCache   *lru.Cache[uint64, []byte]
	marshalledCacheMu sync.RWMutex
}

// newPendingBase constructs a new pendingBase for a given type.
func newPendingBase[T any](store store.Store, logger zerolog.Logger, metaKey string, fetch func(ctx context.Context, store store.Store, height uint64) (T, error)) (*pendingBase[T], error) {
	cache, err := lru.New[uint64, []byte](DefaultMarshalledCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create marshalled cache: %w", err)
	}

	pb := &pendingBase[T]{
		store:           store,
		logger:          logger,
		metaKey:         metaKey,
		fetch:           fetch,
		marshalledCache: cache,
	}
	if err := pb.init(); err != nil {
		return nil, err
	}
	return pb, nil
}

// getPending returns a sorted slice of pending items of type T.
func (pb *pendingBase[T]) getPending(ctx context.Context) ([]T, error) {
	lastSubmitted := pb.lastHeight.Load()
	height, err := pb.store.Height(ctx)
	if err != nil {
		return nil, err
	}
	if lastSubmitted == height {
		return nil, nil
	}
	if lastSubmitted > height {
		return nil, fmt.Errorf("height of last submitted item (%d) is greater than height of last item (%d)", lastSubmitted, height)
	}
	pending := make([]T, 0, height-lastSubmitted)
	for i := lastSubmitted + 1; i <= height; i++ {
		item, err := pb.fetch(ctx, pb.store, i)
		if err != nil {
			return pending, err
		}
		pending = append(pending, item)
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

		// Clear marshalled cache for submitted heights
		pb.clearMarshalledCacheUpTo(newLastSubmittedHeight)
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

// getMarshalledForHeight returns cached marshalled bytes for a height, or nil if not cached
func (pb *pendingBase[T]) getMarshalledForHeight(height uint64) []byte {
	pb.marshalledCacheMu.RLock()
	defer pb.marshalledCacheMu.RUnlock()

	if val, ok := pb.marshalledCache.Get(height); ok {
		return val
	}
	return nil
}

// setMarshalledForHeight caches marshalled bytes for a height
func (pb *pendingBase[T]) setMarshalledForHeight(height uint64, marshalled []byte) {
	pb.marshalledCacheMu.Lock()
	defer pb.marshalledCacheMu.Unlock()

	pb.marshalledCache.Add(height, marshalled)
}

// clearMarshalledCacheUpTo removes cached marshalled bytes up to and including the given height.
// With LRU cache, we iterate through keys and remove those <= height.
func (pb *pendingBase[T]) clearMarshalledCacheUpTo(height uint64) {
	pb.marshalledCacheMu.Lock()
	defer pb.marshalledCacheMu.Unlock()

	// Get all keys and remove those that are <= height
	keys := pb.marshalledCache.Keys()
	for _, h := range keys {
		if h <= height {
			pb.marshalledCache.Remove(h)
		}
	}
}
