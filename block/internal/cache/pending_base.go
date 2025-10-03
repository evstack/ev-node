package cache

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync/atomic"

	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/pkg/store"
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
}

// newPendingBase constructs a new pendingBase for a given type.
func newPendingBase[T any](store store.Store, logger zerolog.Logger, metaKey string, fetch func(ctx context.Context, store store.Store, height uint64) (T, error)) (*pendingBase[T], error) {
	pb := &pendingBase[T]{
		store:   store,
		logger:  logger,
		metaKey: metaKey,
		fetch:   fetch,
	}
	if err := pb.init(); err != nil {
		return nil, err
	}
	return pb, nil
}

// getPending returns a sorted slice of pending items of type T.
func (pb *pendingBase[T]) getPending(ctx context.Context) ([]T, error) {
	pending := make([]T, 0)
	item, err := pb.fetch(ctx, pb.store, 2402427)
	if err != nil {
		return pending, err
	}
	pending = append(pending, item)
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
