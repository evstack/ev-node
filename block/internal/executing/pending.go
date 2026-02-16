package executing

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
	"github.com/ipfs/go-datastore"
	ds "github.com/ipfs/go-datastore"
)

const (
	headerKey = "pending_header"
	dataKey   = "pending_data"
)

// getPendingBlock retrieves the pending block from metadata if it exists
func (e *Executor) getPendingBlock(ctx context.Context) (*types.SignedHeader, *types.Data, error) {
	headerBytes, err := e.store.GetMetadata(ctx, headerKey)
	if err != nil {
		if errors.Is(err, ds.ErrNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	dataBytes, err := e.store.GetMetadata(ctx, dataKey)
	if err != nil {
		if errors.Is(err, ds.ErrNotFound) {
			return nil, nil, fmt.Errorf("pending header exists but data is missing: corrupt state")
		}
		return nil, nil, err
	}

	header := new(types.SignedHeader)
	if err := header.UnmarshalBinary(headerBytes); err != nil {
		return nil, nil, fmt.Errorf("unmarshal pending header: %w", err)
	}

	data := new(types.Data)
	if err := data.UnmarshalBinary(dataBytes); err != nil {
		return nil, nil, fmt.Errorf("unmarshal pending data: %w", err)
	}
	return header, data, nil
}

// savePendingBlock saves a block to metadata as pending
func (e *Executor) savePendingBlock(ctx context.Context, header *types.SignedHeader, data *types.Data) error {
	headerBytes, err := header.MarshalBinary()
	if err != nil {
		return fmt.Errorf("marshal header: %w", err)
	}

	dataBytes, err := data.MarshalBinary()
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}

	batch, err := e.store.NewBatch(ctx)
	if err != nil {
		return fmt.Errorf("create batch for early save: %w", err)
	}

	if err := batch.Put(ds.NewKey(store.GetMetaKey(headerKey)), headerBytes); err != nil {
		return fmt.Errorf("save pending header: %w", err)
	}

	if err := batch.Put(ds.NewKey(store.GetMetaKey(dataKey)), dataBytes); err != nil {
		return fmt.Errorf("save pending data: %w", err)
	}

	if err := batch.Commit(); err != nil {
		return fmt.Errorf("commit pending block: %w", err)
	}
	return nil
}

// deletePendingBlock removes pending block metadata
func (e *Executor) deletePendingBlock(batch store.Batch) error {
	if err := batch.Delete(ds.NewKey(store.GetMetaKey(headerKey))); err != nil {
		return fmt.Errorf("delete pending header: %w", err)
	}

	if err := batch.Delete(ds.NewKey(store.GetMetaKey(dataKey))); err != nil {
		return fmt.Errorf("delete pending data: %w", err)
	}
	return nil
}

// migrateLegacyPendingBlock detects old-style pending blocks that were stored
// at height N+1 via SaveBlockData with an empty signature (pre-upgrade format)
// and migrates them to the new metadata-key format (m/pending_header, m/pending_data).
//
// This prevents double-signing when a node is upgraded: without migration the
// new code would not find the pending block and would create+sign a new one at
// the same height.
func (e *Executor) migrateLegacyPendingBlock(ctx context.Context) error {
	candidateHeight := e.getLastState().LastBlockHeight + 1
	pendingHeader, pendingData, err := e.store.GetBlockData(ctx, candidateHeight)
	if err != nil {
		if !errors.Is(err, datastore.ErrNotFound) {
			return fmt.Errorf("get block data: %w", err)
		}
		return nil
	}
	if len(pendingHeader.Signature) != 0 {
		return errors.New("pending block with signatures found")
	}
	// Migrate: write header+data to the new metadata keys.
	if err := e.savePendingBlock(ctx, pendingHeader, pendingData); err != nil {
		return fmt.Errorf("save migrated pending block: %w", err)
	}

	// Clean up old-style keys.
	batch, err := e.store.NewBatch(ctx)
	if err != nil {
		return fmt.Errorf("create cleanup batch: %w", err)
	}

	headerBytes, err := pendingHeader.MarshalBinary()
	if err != nil {
		return fmt.Errorf("marshal header for hash: %w", err)
	}
	headerHash := sha256.Sum256(headerBytes)

	for _, key := range []string{
		store.GetHeaderKey(candidateHeight),
		store.GetDataKey(candidateHeight),
		store.GetSignatureKey(candidateHeight),
		store.GetIndexKey(headerHash[:]),
	} {
		if err := batch.Delete(ds.NewKey(key)); err != nil && !errors.Is(err, ds.ErrNotFound) {
			return fmt.Errorf("delete legacy key %s: %w", key, err)
		}
	}

	if err := batch.Commit(); err != nil {
		return fmt.Errorf("commit cleanup batch: %w", err)
	}

	e.logger.Info().
		Uint64("height", candidateHeight).
		Msg("migrated legacy pending block to metadata format")
	return nil
}
