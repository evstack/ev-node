package da

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/celestiaorg/go-square/v3/share"
	"github.com/rs/zerolog"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/blob"
	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// Client is the interface representing the DA client.
type Client interface {
	Submit(ctx context.Context, data [][]byte, namespace []byte, options []byte) datypes.ResultSubmit
	Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve
	RetrieveHeaders(ctx context.Context, height uint64) datypes.ResultRetrieve
	RetrieveData(ctx context.Context, height uint64) datypes.ResultRetrieve

	GetHeaderNamespace() []byte
	GetDataNamespace() []byte
}

// client provides a reusable wrapper around the blob RPC with common configuration for namespaces and timeouts.
type client struct {
	blobClient      BlobAPI
	logger          zerolog.Logger
	defaultTimeout  time.Duration
	namespaceBz     []byte
	dataNamespaceBz []byte
	maxBlobSize     uint64
}

// BlobAPI captures the minimal blob RPC methods used by the node.
type BlobAPI interface {
	Submit(ctx context.Context, blobs []*blob.Blob, opts *blob.SubmitOptions) (uint64, error)
	GetAll(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blob.Blob, error)
	GetProof(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Proof, error)
	Included(ctx context.Context, height uint64, namespace share.Namespace, proof *blob.Proof, commitment blob.Commitment) (bool, error)
}

// Config contains configuration for the DA client.
type Config struct {
	BlobAPI        BlobAPI
	Logger         zerolog.Logger
	DefaultTimeout time.Duration
	Namespace      string
	DataNamespace  string
	MaxBlobSize    uint64
}

// NewClient creates a new blob client wrapper with pre-calculated namespace bytes.
func NewClient(cfg Config) *client {
	if cfg.DefaultTimeout == 0 {
		cfg.DefaultTimeout = 30 * time.Second
	}
	if cfg.MaxBlobSize == 0 {
		cfg.MaxBlobSize = blob.DefaultMaxBlobSize
	}

	return &client{
		blobClient:      cfg.BlobAPI,
		logger:          cfg.Logger.With().Str("component", "da_client").Logger(),
		defaultTimeout:  cfg.DefaultTimeout,
		namespaceBz:     coreda.NamespaceFromString(cfg.Namespace).Bytes(),
		dataNamespaceBz: coreda.NamespaceFromString(cfg.DataNamespace).Bytes(),
		maxBlobSize:     cfg.MaxBlobSize,
	}
}

// Submit submits blobs to the DA layer with the specified options.
func (c *client) Submit(ctx context.Context, data [][]byte, namespace []byte, options []byte) datypes.ResultSubmit {
	// calculate blob size
	var blobSize uint64
	for _, b := range data {
		blobSize += uint64(len(b))
	}

	ns, err := share.NewNamespaceFromBytes(namespace)
	if err != nil {
		return datypes.ResultSubmit{
			BaseResult: datypes.BaseResult{
				Code:    datypes.StatusError,
				Message: fmt.Sprintf("invalid namespace: %v", err),
			},
		}
	}

	blobs := make([]*blob.Blob, len(data))
	for i, raw := range data {
		if uint64(len(raw)) > c.maxBlobSize {
			return datypes.ResultSubmit{
				BaseResult: datypes.BaseResult{
					Code:    datypes.StatusTooBig,
					Message: datypes.ErrBlobSizeOverLimit.Error(),
				},
			}
		}
		blobs[i], err = blob.NewBlobV0(ns, raw)
		if err != nil {
			return datypes.ResultSubmit{
				BaseResult: datypes.BaseResult{
					Code:    datypes.StatusError,
					Message: fmt.Sprintf("failed to build blob %d: %v", i, err),
				},
			}
		}
	}

	var submitOpts blob.SubmitOptions
	if len(options) > 0 {
		if err := json.Unmarshal(options, &submitOpts); err != nil {
			return datypes.ResultSubmit{
				BaseResult: datypes.BaseResult{
					Code:    datypes.StatusError,
					Message: fmt.Sprintf("failed to parse submit options: %v", err),
				},
			}
		}
	}

	height, err := c.blobClient.Submit(ctx, blobs, &submitOpts)
	if err != nil {
		code := datypes.StatusError
		switch {
		case errors.Is(err, context.Canceled):
			code = datypes.StatusContextCanceled
		case strings.Contains(err.Error(), datypes.ErrTxTimedOut.Error()):
			code = datypes.StatusNotIncludedInBlock
		case strings.Contains(err.Error(), datypes.ErrTxAlreadyInMempool.Error()):
			code = datypes.StatusAlreadyInMempool
		case strings.Contains(err.Error(), datypes.ErrTxIncorrectAccountSequence.Error()):
			code = datypes.StatusIncorrectAccountSequence
		case strings.Contains(err.Error(), datypes.ErrBlobSizeOverLimit.Error()):
			code = datypes.StatusTooBig
		case strings.Contains(err.Error(), datypes.ErrContextDeadline.Error()):
			code = datypes.StatusContextDeadline
		}
		if code == datypes.StatusTooBig {
			c.logger.Debug().Err(err).Uint64("status", uint64(code)).Msg("DA submission failed")
		} else {
			c.logger.Error().Err(err).Uint64("status", uint64(code)).Msg("DA submission failed")
		}
		return datypes.ResultSubmit{
			BaseResult: datypes.BaseResult{
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
		return datypes.ResultSubmit{
			BaseResult: datypes.BaseResult{
				Code:     datypes.StatusSuccess,
				BlobSize: blobSize,
				Height:   height,
			},
		}
	}

	ids := make([]datypes.ID, len(blobs))
	for i, b := range blobs {
		ids[i] = blob.MakeID(height, b.Commitment)
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

// Retrieve retrieves blobs from the DA layer at the specified height and namespace.
func (c *client) Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	ns, err := share.NewNamespaceFromBytes(namespace)
	if err != nil {
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:    datypes.StatusError,
				Message: fmt.Sprintf("invalid namespace: %v", err),
				Height:  height,
			},
		}
	}

	getIDsCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer cancel()

	blobs, err := c.blobClient.GetAll(getIDsCtx, height, []share.Namespace{ns})
	if err != nil {
		// Handle known errors by substring because RPC may wrap them.
		switch {
		case strings.Contains(err.Error(), datypes.ErrBlobNotFound.Error()):
			return datypes.ResultRetrieve{
				BaseResult: datypes.BaseResult{
					Code:      datypes.StatusNotFound,
					Message:   datypes.ErrBlobNotFound.Error(),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		case strings.Contains(err.Error(), datypes.ErrHeightFromFuture.Error()):
			return datypes.ResultRetrieve{
				BaseResult: datypes.BaseResult{
					Code:      datypes.StatusHeightFromFuture,
					Message:   datypes.ErrHeightFromFuture.Error(),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		default:
			c.logger.Error().Uint64("height", height).Err(err).Msg("failed to get blobs")
			return datypes.ResultRetrieve{
				BaseResult: datypes.BaseResult{
					Code:      datypes.StatusError,
					Message:   fmt.Sprintf("failed to get blobs: %s", err.Error()),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		}
	}

	if len(blobs) == 0 {
		c.logger.Debug().Uint64("height", height).Msg("No blobs found at height")
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:      datypes.StatusNotFound,
				Message:   datypes.ErrBlobNotFound.Error(),
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}

	out := make([][]byte, len(blobs))
	ids := make([][]byte, len(blobs))
	for i, b := range blobs {
		out[i] = b.Data()
		ids[i] = blob.MakeID(height, b.Commitment)
	}

	return datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{
			Code:      datypes.StatusSuccess,
			Height:    height,
			IDs:       ids,
			Timestamp: time.Now(),
		},
		Data: out,
	}
}

// RetrieveHeaders retrieves blobs from the header namespace at the specified height.
func (c *client) RetrieveHeaders(ctx context.Context, height uint64) datypes.ResultRetrieve {
	return c.Retrieve(ctx, height, c.namespaceBz)
}

// RetrieveData retrieves blobs from the data namespace at the specified height.
func (c *client) RetrieveData(ctx context.Context, height uint64) datypes.ResultRetrieve {
	return c.Retrieve(ctx, height, c.dataNamespaceBz)
}

// GetHeaderNamespace returns the header namespace bytes.
func (c *client) GetHeaderNamespace() []byte {
	return c.namespaceBz
}

// GetDataNamespace returns the data namespace bytes.
func (c *client) GetDataNamespace() []byte {
	return c.dataNamespaceBz
}
