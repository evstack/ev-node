package executing

import (
	"context"
	"errors"
	"fmt"

	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
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
			return nil, nil, nil
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
func (e *Executor) deletePendingBlock(ctx context.Context, batch store.Batch) error {
	if err := batch.Delete(ds.NewKey(store.GetMetaKey(headerKey))); err != nil {
		return fmt.Errorf("delete pending header: %w", err)
	}

	if err := batch.Delete(ds.NewKey(store.GetMetaKey(dataKey))); err != nil {
		return fmt.Errorf("delete pending data: %w", err)
	}
	return nil
}
