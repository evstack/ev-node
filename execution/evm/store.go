package evm

import (
	"context"
	"encoding/binary"
	"fmt"

	ds "github.com/ipfs/go-datastore"
	"google.golang.org/protobuf/proto"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// Store prefix for execution/evm data - keeps it isolated from other ev-node data
const evmStorePrefix = "evm/"

// ExecMeta stages
const (
	ExecStageStarted   = "started"
	ExecStageBuilt     = "built"
	ExecStageSubmitted = "submitted"
	ExecStagePromoted  = "promoted"
)

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

// EVMStore wraps a ds.Batching datastore with a prefix for EVM execution data.
// This keeps EVM-specific data isolated from other ev-node data.
type EVMStore struct {
	db ds.Batching
}

// NewEVMStore creates a new EVMStore wrapping the given datastore.
func NewEVMStore(db ds.Batching) *EVMStore {
	return &EVMStore{db: db}
}

// execMetaKey returns the datastore key for ExecMeta at a given height.
func execMetaKey(height uint64) ds.Key {
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, height)
	return ds.NewKey(evmStorePrefix + "execmeta/" + string(heightBytes))
}

// GetExecMeta retrieves execution metadata for the given height.
// Returns nil, nil if not found.
func (s *EVMStore) GetExecMeta(ctx context.Context, height uint64) (*ExecMeta, error) {
	key := execMetaKey(height)
	data, err := s.db.Get(ctx, key)
	if err != nil {
		if err == ds.ErrNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get exec meta: %w", err)
	}

	var pbMeta pb.ExecMeta
	if err := proto.Unmarshal(data, &pbMeta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal exec meta: %w", err)
	}

	meta := &ExecMeta{}
	if err := meta.FromProto(&pbMeta); err != nil {
		return nil, fmt.Errorf("failed to convert exec meta from proto: %w", err)
	}

	return meta, nil
}

// SaveExecMeta persists execution metadata for the given height.
func (s *EVMStore) SaveExecMeta(ctx context.Context, meta *ExecMeta) error {
	key := execMetaKey(meta.Height)
	data, err := proto.Marshal(meta.ToProto())
	if err != nil {
		return fmt.Errorf("failed to marshal exec meta: %w", err)
	}

	if err := s.db.Put(ctx, key, data); err != nil {
		return fmt.Errorf("failed to save exec meta: %w", err)
	}

	return nil
}

// Sync ensures all pending writes are flushed to disk.
func (s *EVMStore) Sync(ctx context.Context) error {
	return s.db.Sync(ctx, ds.NewKey(evmStorePrefix))
}
