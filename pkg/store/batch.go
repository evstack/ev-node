package store

import (
	"context"
	"fmt"

	ds "github.com/ipfs/go-datastore"
	"google.golang.org/protobuf/proto"

	"github.com/evstack/ev-node/types"
)

// DefaultBatch provides a batched write interface for atomic store operations
type DefaultBatch struct {
	store *DefaultStore
	batch ds.Batch
	ctx   context.Context
}

// NewBatch creates a new batch for atomic operations
func (s *DefaultStore) NewBatch(ctx context.Context) (Batch, error) {
	batch, err := s.db.Batch(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new batch: %w", err)
	}

	return &DefaultBatch{
		store: s,
		batch: batch,
		ctx:   ctx,
	}, nil
}

// SetHeight sets the height in the batch
func (b *DefaultBatch) SetHeight(height uint64) error {
	currentHeight, err := b.store.Height(b.ctx)
	if err != nil {
		return err
	}
	if height <= currentHeight {
		return nil
	}

	heightBytes := encodeHeight(height)
	return b.batch.Put(b.ctx, ds.NewKey(getHeightKey()), heightBytes)
}

// SaveBlockData saves block data to the batch
func (b *DefaultBatch) SaveBlockData(header *types.SignedHeader, data *types.Data, signature *types.Signature) error {
	hash := header.Hash()
	height := header.Height()
	signatureHash := *signature

	headerBlob, err := header.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal Header to binary: %w", err)
	}
	dataBlob, err := data.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal Data to binary: %w", err)
	}

	if err := b.batch.Put(b.ctx, ds.NewKey(getHeaderKey(height)), headerBlob); err != nil {
		return fmt.Errorf("failed to put header blob in batch: %w", err)
	}
	if err := b.batch.Put(b.ctx, ds.NewKey(getDataKey(height)), dataBlob); err != nil {
		return fmt.Errorf("failed to put data blob in batch: %w", err)
	}
	if err := b.batch.Put(b.ctx, ds.NewKey(getSignatureKey(height)), signatureHash[:]); err != nil {
		return fmt.Errorf("failed to put signature blob in batch: %w", err)
	}

	heightBytes := encodeHeight(height)
	if err := b.batch.Put(b.ctx, ds.NewKey(getIndexKey(hash)), heightBytes); err != nil {
		return fmt.Errorf("failed to put index key in batch: %w", err)
	}

	return nil
}

// UpdateState updates the state in the batch
func (b *DefaultBatch) UpdateState(state types.State) error {
	// Save the state at the height specified in the state itself
	height := state.LastBlockHeight

	pbState, err := state.ToProto()
	if err != nil {
		return fmt.Errorf("failed to convert type state to protobuf type: %w", err)
	}
	data, err := proto.Marshal(pbState)
	if err != nil {
		return fmt.Errorf("failed to marshal state to protobuf: %w", err)
	}

	return b.batch.Put(b.ctx, ds.NewKey(getStateAtHeightKey(height)), data)
}

// Commit commits all batched operations atomically
func (b *DefaultBatch) Commit() error {
	if err := b.batch.Commit(b.ctx); err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}
	return nil
}

// Delete adds a delete operation to the batch
func (b *DefaultBatch) Delete(key ds.Key) error {
	return b.batch.Delete(b.ctx, key)
}

// Put adds a put operation to the batch
func (b *DefaultBatch) Put(key ds.Key, value []byte) error {
	return b.batch.Put(b.ctx, key, value)
}
