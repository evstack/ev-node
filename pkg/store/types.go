package store

import (
	"context"

	ds "github.com/ipfs/go-datastore"

	"github.com/evstack/ev-node/types"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// Batch provides atomic operations for the store
type Batch interface {
	// SaveBlockData atomically saves the block header, data, and signature
	SaveBlockData(header *types.SignedHeader, data *types.Data, signature *types.Signature) error

	// SetHeight sets the height in the batch
	SetHeight(height uint64) error

	// UpdateState updates the state in the batch
	UpdateState(state types.State) error

	// SaveExecMeta saves execution metadata in the batch
	SaveExecMeta(meta *ExecMeta) error

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

	// GetExecMeta returns execution metadata for a given height, or error if not found.
	GetExecMeta(ctx context.Context, height uint64) (*ExecMeta, error)
}

type Rollback interface {
	// Rollback deletes x height from the ev-node store.
	// Aggregator is used to determine if the rollback is performed on the aggregator node.
	Rollback(ctx context.Context, height uint64, aggregator bool) error
}

// ExecMeta tracks execution state per height for idempotent execution.
// This enables crash recovery and prevents sibling block creation on retries.
type ExecMeta struct {
	// Height is the block height this execution metadata is for
	Height uint64
	// ParentHash is the EL parent block hash used for this execution
	ParentHash []byte
	// PayloadID is the Engine API payload ID (optional, set after forkchoiceUpdatedV3)
	PayloadID []byte
	// BlockHash is the EL block hash once built (set after getPayloadV4)
	BlockHash []byte
	// StateRoot is the state root from the execution payload
	StateRoot []byte
	// TxHash is the hash of the transaction list for sanity checks
	TxHash []byte
	// Timestamp is the block timestamp
	Timestamp int64
	// Stage indicates the current execution stage:
	// "started" - forkchoiceUpdatedV3 called, payloadID obtained
	// "built" - getPayloadV4 called, payload retrieved
	// "submitted" - newPayloadV4 called, payload marked VALID
	// "promoted" - final forkchoiceUpdatedV3 called, block is head
	Stage string
	// UpdatedAtUnix is the Unix timestamp when this metadata was last updated
	UpdatedAtUnix int64
}

// ExecMeta stages
const (
	ExecStageStarted   = "started"
	ExecStageBuilt     = "built"
	ExecStageSubmitted = "submitted"
	ExecStagePromoted  = "promoted"
)

// ToProto converts ExecMeta into protobuf representation.
func (em *ExecMeta) ToProto() *pb.ExecMeta {
	return &pb.ExecMeta{
		Height:        em.Height,
		ParentHash:    em.ParentHash,
		PayloadId:     em.PayloadID,
		BlockHash:     em.BlockHash,
		StateRoot:     em.StateRoot,
		TxHash:        em.TxHash,
		Timestamp:     em.Timestamp,
		Stage:         em.Stage,
		UpdatedAtUnix: em.UpdatedAtUnix,
	}
}

// FromProto fills ExecMeta with data from its protobuf representation.
func (em *ExecMeta) FromProto(other *pb.ExecMeta) error {
	if other == nil {
		return nil
	}
	em.Height = other.Height
	em.ParentHash = append([]byte(nil), other.ParentHash...)
	em.PayloadID = append([]byte(nil), other.PayloadId...)
	em.BlockHash = append([]byte(nil), other.BlockHash...)
	em.StateRoot = append([]byte(nil), other.StateRoot...)
	em.TxHash = append([]byte(nil), other.TxHash...)
	em.Timestamp = other.Timestamp
	em.Stage = other.Stage
	em.UpdatedAtUnix = other.UpdatedAtUnix
	return nil
}
