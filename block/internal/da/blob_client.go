package da

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/celestiaorg/go-square/v3/share"
	coreda "github.com/evstack/ev-node/core/da"
	blobrpc "github.com/evstack/ev-node/da/jsonrpc/blob"
	"github.com/rs/zerolog"
)

// BlobConfig contains configuration for the blob client.
type BlobConfig struct {
	Celestia       *blobrpc.Client
	Logger         zerolog.Logger
	DefaultTimeout time.Duration
	Namespace      string
	DataNamespace  string
	MaxBlobSize    uint64
}

// BlobClient wraps the blob RPC with namespace handling and error mapping.
type BlobClient struct {
	blobAPI         *blobrpc.BlobAPI
	logger          zerolog.Logger
	defaultTimeout  time.Duration
	namespaceBz     []byte
	dataNamespaceBz []byte
	maxBlobSize     uint64
}

// NewBlobClient creates a new blob client wrapper with pre-calculated namespace bytes.
func NewBlobClient(cfg BlobConfig) *BlobClient {
	if cfg.Celestia == nil {
		return nil
	}
	if cfg.DefaultTimeout == 0 {
		cfg.DefaultTimeout = 30 * time.Second
	}
	if cfg.MaxBlobSize == 0 {
		cfg.MaxBlobSize = blobrpc.DefaultMaxBlobSize
	}

	return &BlobClient{
		blobAPI:         &cfg.Celestia.Blob,
		logger:          cfg.Logger.With().Str("component", "blob_da_client").Logger(),
		defaultTimeout:  cfg.DefaultTimeout,
		namespaceBz:     share.MustNewV0Namespace([]byte(cfg.Namespace)).Bytes(),
		dataNamespaceBz: share.MustNewV0Namespace([]byte(cfg.DataNamespace)).Bytes(),
		maxBlobSize:     cfg.MaxBlobSize,
	}
}

// Submit submits blobs to the DA layer with the specified options.
func (c *BlobClient) Submit(ctx context.Context, data [][]byte, namespace []byte, options []byte) coreda.ResultSubmit {
	// calculate blob size
	var blobSize uint64
	for _, b := range data {
		blobSize += uint64(len(b))
	}

	ns, err := share.NewNamespaceFromBytes(namespace)
	if err != nil {
		return coreda.ResultSubmit{
			BaseResult: coreda.BaseResult{
				Code:    coreda.StatusError,
				Message: fmt.Sprintf("invalid namespace: %v", err),
			},
		}
	}

	blobs := make([]*blobrpc.Blob, len(data))
	for i, raw := range data {
		if uint64(len(raw)) > c.maxBlobSize {
			return coreda.ResultSubmit{
				BaseResult: coreda.BaseResult{
					Code:    coreda.StatusTooBig,
					Message: coreda.ErrBlobSizeOverLimit.Error(),
				},
			}
		}
		blobs[i], err = blobrpc.NewBlobV0(ns, raw)
		if err != nil {
			return coreda.ResultSubmit{
				BaseResult: coreda.BaseResult{
					Code:    coreda.StatusError,
					Message: fmt.Sprintf("failed to build blob %d: %v", i, err),
				},
			}
		}
	}

	var submitOpts blobrpc.SubmitOptions
	if len(options) > 0 {
		if err := json.Unmarshal(options, &submitOpts); err != nil {
			return coreda.ResultSubmit{
				BaseResult: coreda.BaseResult{
					Code:    coreda.StatusError,
					Message: fmt.Sprintf("failed to parse submit options: %v", err),
				},
			}
		}
	}

	height, err := c.blobAPI.Submit(ctx, blobs, &submitOpts)
	if err != nil {
		code := coreda.StatusError
		switch {
		case errors.Is(err, context.Canceled):
			code = coreda.StatusContextCanceled
		case strings.Contains(err.Error(), coreda.ErrTxTimedOut.Error()):
			code = coreda.StatusNotIncludedInBlock
		case strings.Contains(err.Error(), coreda.ErrTxAlreadyInMempool.Error()):
			code = coreda.StatusAlreadyInMempool
		case strings.Contains(err.Error(), coreda.ErrTxIncorrectAccountSequence.Error()):
			code = coreda.StatusIncorrectAccountSequence
		case strings.Contains(err.Error(), coreda.ErrBlobSizeOverLimit.Error()):
			code = coreda.StatusTooBig
		case strings.Contains(err.Error(), coreda.ErrContextDeadline.Error()):
			code = coreda.StatusContextDeadline
		}
		if code == coreda.StatusTooBig {
			c.logger.Debug().Err(err).Uint64("status", uint64(code)).Msg("DA submission failed")
		} else {
			c.logger.Error().Err(err).Uint64("status", uint64(code)).Msg("DA submission failed")
		}
		return coreda.ResultSubmit{
			BaseResult: coreda.BaseResult{
				Code:           code,
				Message:        "failed to submit blobs: " + err.Error(),
				SubmittedCount: 0,
				Height:         0,
				Timestamp:      time.Now(),
				BlobSize:       blobSize,
			},
		}
	}

	if len(blobs) == 0 {
		return coreda.ResultSubmit{
			BaseResult: coreda.BaseResult{
				Code:     coreda.StatusSuccess,
				BlobSize: blobSize,
				Height:   height,
			},
		}
	}

	ids := make([]coreda.ID, len(blobs))
	for i, b := range blobs {
		ids[i] = blobrpc.MakeID(height, b.Commitment)
	}

	return coreda.ResultSubmit{
		BaseResult: coreda.BaseResult{
			Code:           coreda.StatusSuccess,
			IDs:            ids,
			SubmittedCount: uint64(len(ids)),
			Height:         height,
			BlobSize:       blobSize,
			Timestamp:      time.Now(),
		},
	}
}

// Retrieve retrieves blobs from the DA layer at the specified height and namespace.
func (c *BlobClient) Retrieve(ctx context.Context, height uint64, namespace []byte) coreda.ResultRetrieve {
	ns, err := share.NewNamespaceFromBytes(namespace)
	if err != nil {
		return coreda.ResultRetrieve{
			BaseResult: coreda.BaseResult{
				Code:    coreda.StatusError,
				Message: fmt.Sprintf("invalid namespace: %v", err),
				Height:  height,
			},
		}
	}

	getCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer cancel()

	blobs, err := c.blobAPI.GetAll(getCtx, height, []share.Namespace{ns})
	if err != nil {
		// Handle known errors by substring because RPC may wrap them.
		switch {
		case strings.Contains(err.Error(), coreda.ErrBlobNotFound.Error()):
			return coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code:      coreda.StatusNotFound,
					Message:   coreda.ErrBlobNotFound.Error(),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		case strings.Contains(err.Error(), coreda.ErrHeightFromFuture.Error()):
			return coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code:      coreda.StatusHeightFromFuture,
					Message:   coreda.ErrHeightFromFuture.Error(),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		default:
			c.logger.Error().Uint64("height", height).Err(err).Msg("failed to get blobs")
			return coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code:      coreda.StatusError,
					Message:   fmt.Sprintf("failed to get blobs: %s", err.Error()),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		}
	}

	if len(blobs) == 0 {
		c.logger.Debug().Uint64("height", height).Msg("No blobs found at height")
		return coreda.ResultRetrieve{
			BaseResult: coreda.BaseResult{
				Code:      coreda.StatusNotFound,
				Message:   coreda.ErrBlobNotFound.Error(),
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}

	out := make([][]byte, len(blobs))
	ids := make([][]byte, len(blobs))
	for i, b := range blobs {
		out[i] = b.Data()
		ids[i] = blobrpc.MakeID(height, b.Commitment)
	}

	return coreda.ResultRetrieve{
		BaseResult: coreda.BaseResult{
			Code:      coreda.StatusSuccess,
			Height:    height,
			IDs:       ids,
			Timestamp: time.Now(),
		},
		Data: out,
	}
}

// RetrieveHeaders retrieves blobs from the header namespace at the specified height.
func (c *BlobClient) RetrieveHeaders(ctx context.Context, height uint64) coreda.ResultRetrieve {
	return c.Retrieve(ctx, height, c.namespaceBz)
}

// RetrieveData retrieves blobs from the data namespace at the specified height.
func (c *BlobClient) RetrieveData(ctx context.Context, height uint64) coreda.ResultRetrieve {
	return c.Retrieve(ctx, height, c.dataNamespaceBz)
}

// GetHeaderNamespace returns the header namespace bytes.
func (c *BlobClient) GetHeaderNamespace() []byte {
	return c.namespaceBz
}

// GetDataNamespace returns the data namespace bytes.
func (c *BlobClient) GetDataNamespace() []byte {
	return c.dataNamespaceBz
}
