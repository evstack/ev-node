package store

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	ds "github.com/ipfs/go-datastore"
	"google.golang.org/protobuf/proto"

	"github.com/evstack/ev-node/types"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// DefaultStore is a default store implementation.
type DefaultStore struct {
	db ds.Batching
}

var _ Store = &DefaultStore{}

// New returns new, default store.
func New(ds ds.Batching) Store {
	return &DefaultStore{
		db: ds,
	}
}

// Close safely closes underlying data storage, to ensure that data is actually saved.
func (s *DefaultStore) Close() error {
	return s.db.Close()
}

// Height returns height of the highest block saved in the Store.
func (s *DefaultStore) Height(ctx context.Context) (uint64, error) {
	heightBytes, err := s.db.Get(ctx, ds.NewKey(getHeightKey()))
	if errors.Is(err, ds.ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		// Since we can't return an error due to interface constraints,
		// we log by returning 0 when there's an error reading from disk
		return 0, err
	}

	height, err := decodeHeight(heightBytes)
	if err != nil {
		return 0, err
	}
	return height, nil
}

// GetBlockData returns block header and data at given height, or error if it's not found in Store.
func (s *DefaultStore) GetBlockData(ctx context.Context, height uint64) (*types.SignedHeader, *types.Data, error) {
	header, err := s.GetHeader(ctx, height)
	if err != nil {
		return nil, nil, err
	}
	dataBlob, err := s.db.Get(ctx, ds.NewKey(getDataKey(height)))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load block data: %w", err)
	}
	data := new(types.Data)
	err = data.UnmarshalBinary(dataBlob)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal block data: %w", err)
	}
	return header, data, nil
}

// GetBlockByHash returns block with given block header hash, or error if it's not found in Store.
func (s *DefaultStore) GetBlockByHash(ctx context.Context, hash []byte) (*types.SignedHeader, *types.Data, error) {
	height, err := s.getHeightByHash(ctx, hash)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load height from index %w", err)
	}
	return s.GetBlockData(ctx, height)
}

// getHeightByHash returns height for a block with given block header hash.
func (s *DefaultStore) getHeightByHash(ctx context.Context, hash []byte) (uint64, error) {
	heightBytes, err := s.db.Get(ctx, ds.NewKey(getIndexKey(hash)))
	if err != nil {
		return 0, fmt.Errorf("failed to get height for hash %v: %w", hash, err)
	}
	height, err := decodeHeight(heightBytes)
	if err != nil {
		return 0, fmt.Errorf("failed to decode height: %w", err)
	}
	return height, nil
}

// GetSignatureByHash returns signature for a block at given height, or error if it's not found in Store.
func (s *DefaultStore) GetSignatureByHash(ctx context.Context, hash []byte) (*types.Signature, error) {
	height, err := s.getHeightByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to load hash from index: %w", err)
	}
	return s.GetSignature(ctx, height)
}

// GetHeader returns the header at the given height or error if it's not found in Store.
func (s *DefaultStore) GetHeader(ctx context.Context, height uint64) (*types.SignedHeader, error) {
	headerBlob, err := s.db.Get(ctx, ds.NewKey(getHeaderKey(height)))
	if err != nil {
		return nil, fmt.Errorf("load block header: %w", err)
	}
	header := new(types.SignedHeader)
	if err = header.UnmarshalBinary(headerBlob); err != nil {
		return nil, fmt.Errorf("unmarshal block header: %w", err)
	}
	return header, nil
}

// GetSignature returns signature for a block with given block header hash, or error if it's not found in Store.
func (s *DefaultStore) GetSignature(ctx context.Context, height uint64) (*types.Signature, error) {
	signatureData, err := s.db.Get(ctx, ds.NewKey(getSignatureKey(height)))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve signature from height %v: %w", height, err)
	}
	signature := types.Signature(signatureData)
	return &signature, nil
}

// GetState returns last state saved with UpdateState.
func (s *DefaultStore) GetState(ctx context.Context) (types.State, error) {
	currentHeight, err := s.Height(ctx)
	if err != nil {
		return types.State{}, fmt.Errorf("failed to get current height: %w", err)
	}

	blob, err := s.db.Get(ctx, ds.NewKey(getStateAtHeightKey(currentHeight)))
	if err != nil {
		return types.State{}, fmt.Errorf("failed to retrieve state: %w", err)
	}
	var pbState pb.State
	err = proto.Unmarshal(blob, &pbState)
	if err != nil {
		return types.State{}, fmt.Errorf("failed to unmarshal state from protobuf: %w", err)
	}

	var state types.State
	err = state.FromProto(&pbState)
	return state, err
}

// GetStateAtHeight returns the state at the given height.
// If no state is stored at that height, it returns an error.
func (s *DefaultStore) GetStateAtHeight(ctx context.Context, height uint64) (types.State, error) {
	blob, err := s.db.Get(ctx, ds.NewKey(getStateAtHeightKey(height)))
	if err != nil {
		if errors.Is(err, ds.ErrNotFound) {
			return types.State{}, fmt.Errorf("no state found at height %d", height)
		}
		return types.State{}, fmt.Errorf("failed to retrieve state at height %d: %w", height, err)
	}

	var pbState pb.State
	if err := proto.Unmarshal(blob, &pbState); err != nil {
		return types.State{}, fmt.Errorf("failed to unmarshal state from protobuf at height %d: %w", height, err)
	}

	var state types.State
	if err := state.FromProto(&pbState); err != nil {
		return types.State{}, fmt.Errorf("failed to convert protobuf to state at height %d: %w", height, err)
	}

	return state, nil
}

// SetMetadata saves arbitrary value in the store.
//
// Metadata is separated from other data by using prefix in KV.
func (s *DefaultStore) SetMetadata(ctx context.Context, key string, value []byte) error {
	err := s.db.Put(ctx, ds.NewKey(getMetaKey(key)), value)
	if err != nil {
		return fmt.Errorf("failed to set metadata for key '%s': %w", key, err)
	}
	return nil
}

// GetMetadata returns values stored for given key with SetMetadata.
func (s *DefaultStore) GetMetadata(ctx context.Context, key string) ([]byte, error) {
	data, err := s.db.Get(ctx, ds.NewKey(getMetaKey(key)))
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata for key '%s': %w", key, err)
	}
	return data, nil
}

// Rollback rolls back block data until the given height from the store.
// When aggregator is true, it will check the latest data included height and prevent rollback further than that.
// NOTE: this function does not rollback metadata. Those should be handled separately if required.
// Other stores are not rolled back either.
func (s *DefaultStore) Rollback(ctx context.Context, height uint64, aggregator bool) error {
	batch, err := s.db.Batch(ctx)
	if err != nil {
		return fmt.Errorf("failed to create a new batch: %w", err)
	}

	currentHeight, err := s.Height(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current height: %w", err)
	}

	if currentHeight <= height {
		return fmt.Errorf("current height %d is already less than or equal to rollback height %d", currentHeight, height)
	}

	daIncludedHeightBz, err := s.GetMetadata(ctx, DAIncludedHeightKey)
	if err != nil && !errors.Is(err, ds.ErrNotFound) {
		return fmt.Errorf("failed to get DA included height: %w", err)
	} else if len(daIncludedHeightBz) == 8 { // valid height stored, so able to check
		daIncludedHeight := binary.LittleEndian.Uint64(daIncludedHeightBz)
		if daIncludedHeight > height {
			// an aggregator must not rollback a finalized height, DA is the source of truth
			if aggregator {
				return fmt.Errorf("DA included height is greater than the rollback height: cannot rollback a finalized height")
			} else { // in case of syncing issues, rollback the included height is OK.
				bz := make([]byte, 8)
				binary.LittleEndian.PutUint64(bz, height)
				if err := batch.Put(ctx, ds.NewKey(getMetaKey(DAIncludedHeightKey)), bz); err != nil {
					return fmt.Errorf("failed to update DA included height: %w", err)
				}
			}
		}
	}

	for currentHeight > height {
		header, err := s.GetHeader(ctx, currentHeight)
		if err != nil {
			return fmt.Errorf("failed to get header at height %d: %w", currentHeight, err)
		}

		if err := batch.Delete(ctx, ds.NewKey(getHeaderKey(currentHeight))); err != nil {
			return fmt.Errorf("failed to delete header blob in batch: %w", err)
		}

		if err := batch.Delete(ctx, ds.NewKey(getDataKey(currentHeight))); err != nil {
			return fmt.Errorf("failed to delete data blob in batch: %w", err)
		}

		if err := batch.Delete(ctx, ds.NewKey(getSignatureKey(currentHeight))); err != nil {
			return fmt.Errorf("failed to delete signature of block blob in batch: %w", err)
		}

		hash := header.Hash()
		if err := batch.Delete(ctx, ds.NewKey(getIndexKey(hash))); err != nil {
			return fmt.Errorf("failed to delete index key in batch: %w", err)
		}

		if err := batch.Delete(ctx, ds.NewKey(getStateAtHeightKey(currentHeight))); err != nil {
			return fmt.Errorf("failed to delete state in batch: %w", err)
		}

		currentHeight--
	}

	// set height -- using set height checks the current height
	// so we cannot use that
	heightBytes := encodeHeight(height)
	if err := batch.Put(ctx, ds.NewKey(getHeightKey()), heightBytes); err != nil {
		return fmt.Errorf("failed to set height: %w", err)
	}

	if err := batch.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}

	return nil
}

const heightLength = 8

func encodeHeight(height uint64) []byte {
	heightBytes := make([]byte, heightLength)
	binary.LittleEndian.PutUint64(heightBytes, height)
	return heightBytes
}

func decodeHeight(heightBytes []byte) (uint64, error) {
	if len(heightBytes) != heightLength {
		return 0, fmt.Errorf("invalid height length: %d (expected %d)", len(heightBytes), heightLength)
	}
	return binary.LittleEndian.Uint64(heightBytes), nil
}
