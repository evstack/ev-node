package testclient

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/evstack/ev-node/block/internal/da"
	blob "github.com/evstack/ev-node/da/jsonrpc"
	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// client adapts the legacy datypes.DA interface to the block da.Client interface for tests.
type client struct {
	backend       datypes.DA
	namespace     []byte
	dataNamespace []byte
	forcedNs      []byte
	hasForcedNs   bool
}

// New builds a DA client suitable for tests using the provided backend and config.
func New(cfg Config) da.Client {
	hasForced := cfg.ForcedInclusionNamespace != ""
	var forced []byte
	if hasForced {
		forced = datypes.NamespaceFromString(cfg.ForcedInclusionNamespace).Bytes()
	}

	return &client{
		backend:       cfg.DA,
		namespace:     datypes.NamespaceFromString(cfg.Namespace).Bytes(),
		dataNamespace: datypes.NamespaceFromString(cfg.DataNamespace).Bytes(),
		forcedNs:      forced,
		hasForcedNs:   hasForced,
	}
}

func (c *client) Submit(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) datypes.ResultSubmit {
	var blobSize uint64
	for _, blob := range data {
		blobSize += uint64(len(blob))
	}

	if c.backend == nil {
		return datypes.ResultSubmit{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, SubmittedCount: uint64(len(data)), BlobSize: blobSize}}
	}

	ids, err := c.backend.SubmitWithOptions(ctx, data, gasPrice, namespace, options)
	if err != nil {
		status := datypes.StatusError
		switch {
		case errors.Is(err, datypes.ErrTxTimedOut):
			status = datypes.StatusNotIncludedInBlock
		case errors.Is(err, datypes.ErrTxAlreadyInMempool):
			status = datypes.StatusAlreadyInMempool
		case errors.Is(err, datypes.ErrTxIncorrectAccountSequence):
			status = datypes.StatusIncorrectAccountSequence
		case errors.Is(err, datypes.ErrBlobSizeOverLimit):
			status = datypes.StatusTooBig
		case errors.Is(err, datypes.ErrContextDeadline):
			status = datypes.StatusContextDeadline
		case errors.Is(err, context.Canceled):
			status = datypes.StatusContextCanceled
		}

		return datypes.ResultSubmit{
			BaseResult: datypes.BaseResult{
				Code:           status,
				Message:        "failed to submit blobs: " + err.Error(),
				IDs:            ids,
				SubmittedCount: uint64(len(ids)),
				Height:         0,
				BlobSize:       blobSize,
				Timestamp:      time.Now(),
			},
		}
	}

	var height uint64
	if len(ids) > 0 {
		height, _ = blob.SplitID(ids[0])
	}

	return datypes.ResultSubmit{
		BaseResult: datypes.BaseResult{
			Code:           datypes.StatusSuccess,
			IDs:            ids,
			SubmittedCount: uint64(len(ids)),
			Height:         height,
			BlobSize:       blobSize,
			Timestamp:      time.Now(),
		},
	}
}

func (c *client) Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	if c.backend == nil {
		return datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Message: datypes.ErrBlobNotFound.Error(), Height: height}}
	}

	idsRes, err := c.backend.GetIDs(ctx, height, namespace)
	if err != nil {
		status := datypes.StatusError
		msg := err.Error()
		switch {
		case strings.Contains(err.Error(), datypes.ErrBlobNotFound.Error()):
			status = datypes.StatusNotFound
			msg = datypes.ErrBlobNotFound.Error()
		case strings.Contains(err.Error(), datypes.ErrHeightFromFuture.Error()):
			status = datypes.StatusHeightFromFuture
			msg = datypes.ErrHeightFromFuture.Error()
		}
		return datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: status, Message: msg, Height: height, Timestamp: time.Now()}}
	}

	if idsRes == nil || len(idsRes.IDs) == 0 {
		return datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Message: datypes.ErrBlobNotFound.Error(), Height: height, Timestamp: time.Now()}}
	}

	blobs, err := c.backend.Get(ctx, idsRes.IDs, namespace)
	if err != nil {
		return datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusError, Message: err.Error(), Height: height, Timestamp: time.Now()}}
	}

	data := make([][]byte, len(blobs))
	copy(data, blobs)

	return datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{
			Code:      datypes.StatusSuccess,
			Height:    height,
			Timestamp: idsRes.Timestamp,
			IDs:       idsRes.IDs,
		},
		Data: data,
	}
}

func (c *client) RetrieveHeaders(ctx context.Context, height uint64) datypes.ResultRetrieve {
	return c.Retrieve(ctx, height, c.namespace)
}

func (c *client) RetrieveData(ctx context.Context, height uint64) datypes.ResultRetrieve {
	return c.Retrieve(ctx, height, c.dataNamespace)
}

func (c *client) RetrieveForcedInclusion(ctx context.Context, height uint64) datypes.ResultRetrieve {
	if !c.hasForcedNs {
		return datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusError, Message: da.ErrForceInclusionNotConfigured.Error(), Height: height}}
	}
	return c.Retrieve(ctx, height, c.forcedNs)
}

func (c *client) Get(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error) {
	if c.backend == nil {
		return nil, nil
	}
	return c.backend.Get(ctx, ids, namespace)
}

func (c *client) GetProofs(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Proof, error) {
	if c.backend == nil {
		return []datypes.Proof{}, nil
	}
	return c.backend.GetProofs(ctx, ids, namespace)
}

func (c *client) Validate(ctx context.Context, ids []datypes.ID, proofs []datypes.Proof, namespace []byte) ([]bool, error) {
	if c.backend == nil {
		return []bool{}, nil
	}
	return c.backend.Validate(ctx, ids, proofs, namespace)
}

func (c *client) GetHeaderNamespace() []byte { return c.namespace }
func (c *client) GetDataNamespace() []byte   { return c.dataNamespace }
func (c *client) GetForcedInclusionNamespace() []byte {
	return c.forcedNs
}
func (c *client) HasForcedInclusionNamespace() bool { return c.hasForcedNs }
