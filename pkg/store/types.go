package store

import (
	"context"

	ds "github.com/ipfs/go-datastore"

	"github.com/evstack/ev-node/types"
)

// Batch provides atomic operations for the store
type Batch interface {
	// SaveBlockData atomically saves the block header, data, and signature
	SaveBlockData(header *types.SignedHeader, data *types.Data, signature *types.Signature) error

	// SetHeight sets the height in the batch
	SetHeight(height uint64) error

	// UpdateState updates the state in the batch
	UpdateState(state types.State) error

	// Commit commits all batch operations atomically
	Commit() error

	// Put adds a put operation to the batch (used internally for rollback)
	Put(key ds.Key, value []byte) error

	// Delete adds a delete operation to the batch (used internally for rollback)
	Delete(key ds.Key) error
}

// Store is minimal interface for storing and retrieving blocks, commits and state.
type Store interface {
	Rollback
	Reader

	// SetMetadata saves arbitrary value in the store.
	//
	// This method enables evolve to safely persist any information.
	SetMetadata(ctx context.Context, key string, value []byte) error

	// Close safely closes underlying data storage, to ensure that data is actually saved.
	Close() error

	// NewBatch creates a new batch for atomic operations.
	NewBatch(ctx context.Context) (Batch, error)
}

type Reader interface {
	// Height returns height of the highest block in store.
	Height(ctx context.Context) (uint64, error)

	// GetBlockData returns block at given height, or error if it's not found in Store.
	GetBlockData(ctx context.Context, height uint64) (*types.SignedHeader, *types.Data, error)
	// GetBlockByHash returns block with given block header hash, or error if it's not found in Store.
	GetBlockByHash(ctx context.Context, hash []byte) (*types.SignedHeader, *types.Data, error)
	// GetSignature returns signature for a block at given height, or error if it's not found in Store.
	GetSignature(ctx context.Context, height uint64) (*types.Signature, error)
	// GetSignatureByHash returns signature for a block with given block header hash, or error if it's not found in Store.
	GetSignatureByHash(ctx context.Context, hash []byte) (*types.Signature, error)
	// GetHeader returns the header at given height, or error if it's not found in Store.
	GetHeader(ctx context.Context, height uint64) (*types.SignedHeader, error)

	// GetState returns last state saved with UpdateState.
	GetState(ctx context.Context) (types.State, error)
	// GetStateAtHeight returns state saved at given height, or error if it's not found in Store.
	GetStateAtHeight(ctx context.Context, height uint64) (types.State, error)

	// GetMetadata returns values stored for given key with SetMetadata.
	GetMetadata(ctx context.Context, key string) ([]byte, error)
}

type Rollback interface {
	// Rollback deletes x height from the ev-node store.
	// Aggregator is used to determine if the rollback is performed on the aggregator node.
	Rollback(ctx context.Context, height uint64, aggregator bool) error
}
