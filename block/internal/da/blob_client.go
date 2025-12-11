package da

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/celestiaorg/go-square/v3/share"
	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
	"github.com/rs/zerolog"
)

var (
	ErrBlobNotFound               = errors.New("blob: not found")
	ErrBlobSizeOverLimit          = errors.New("blob: over size limit")
	ErrTxTimedOut                 = errors.New("timed out waiting for tx to be included in a block")
	ErrTxAlreadyInMempool         = errors.New("tx already in mempool")
	ErrTxIncorrectAccountSequence = errors.New("incorrect account sequence")
	ErrContextDeadline            = errors.New("context deadline")
	ErrHeightFromFuture           = errors.New("given height is from the future")
	ErrContextCanceled            = errors.New("context canceled")
)

// BlobConfig contains configuration for the Celestia blob client.
type BlobConfig struct {
	Logger         zerolog.Logger
	DefaultTimeout time.Duration
	Namespace      string
	DataNamespace  string
	MaxBlobSize    uint64
}

// BlobAPI captures the subset of blobrpc.BlobAPI used by BlobClient.
type BlobAPI interface {
	Submit(ctx context.Context, blobs []*blobrpc.Blob, opts *blobrpc.SubmitOptions) (uint64, error)
	GetAll(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blobrpc.Blob, error)
	GetProof(ctx context.Context, height uint64, namespace share.Namespace, commitment blobrpc.Commitment) (*blobrpc.Proof, error)
	Included(ctx context.Context, height uint64, namespace share.Namespace, proof *blobrpc.Proof, commitment blobrpc.Commitment) (bool, error)
	GetCommitmentProof(ctx context.Context, height uint64, namespace share.Namespace, shareCommitment []byte) (*blobrpc.CommitmentProof, error)
	Subscribe(ctx context.Context, namespace share.Namespace) (<-chan *blobrpc.SubscriptionResponse, error)
}

// BlobClient wraps the blob RPC with namespace handling and error mapping.
type BlobClient struct {
	blobAPI         BlobAPI
	logger          zerolog.Logger
	defaultTimeout  time.Duration
	namespaceBz     []byte
	dataNamespaceBz []byte
	maxBlobSize     uint64
}

// NewBlobClient creates a new blob client wrapper with pre-calculated namespace bytes.
func NewBlobClient(api BlobAPI, cfg BlobConfig) *BlobClient {
	if api == nil {
		return nil
	}
	if cfg.DefaultTimeout == 0 {
		cfg.DefaultTimeout = 30 * time.Second
	}
	if cfg.MaxBlobSize == 0 {
		cfg.MaxBlobSize = blobrpc.DefaultMaxBlobSize
	}

	return &BlobClient{
		blobAPI:         api,
		logger:          cfg.Logger.With().Str("component", "blob_da_client").Logger(),
		defaultTimeout:  cfg.DefaultTimeout,
		namespaceBz:     share.MustNewV0Namespace([]byte(cfg.Namespace)).Bytes(),
		dataNamespaceBz: share.MustNewV0Namespace([]byte(cfg.DataNamespace)).Bytes(),
		maxBlobSize:     cfg.MaxBlobSize,
	}
}

// Submit submits blobs to the DA layer with the specified options.
func (c *BlobClient) Submit(ctx context.Context, data [][]byte, namespace []byte, options []byte) ResultSubmit {
	// calculate blob size
	var blobSize uint64
	for _, b := range data {
		blobSize += uint64(len(b))
	}

	ns, err := share.NewNamespaceFromBytes(namespace)
	if err != nil {
		return ResultSubmit{
			BaseResult: BaseResult{
				Code:    StatusError,
				Message: fmt.Sprintf("invalid namespace: %v", err),
			},
		}
	}

	blobs := make([]*blobrpc.Blob, len(data))
	for i, raw := range data {
		if uint64(len(raw)) > c.maxBlobSize {
			return ResultSubmit{
				BaseResult: BaseResult{
					Code:    StatusTooBig,
					Message: ErrBlobSizeOverLimit.Error(),
				},
			}
		}
		blobs[i], err = blobrpc.NewBlobV0(ns, raw)
		if err != nil {
			return ResultSubmit{
				BaseResult: BaseResult{
					Code:    StatusError,
					Message: fmt.Sprintf("failed to build blob %d: %v", i, err),
				},
			}
		}
	}

	var submitOpts blobrpc.SubmitOptions
	if len(options) > 0 {
		if err := json.Unmarshal(options, &submitOpts); err != nil {
			return ResultSubmit{
				BaseResult: BaseResult{
					Code:    StatusError,
					Message: fmt.Sprintf("failed to parse submit options: %v", err),
				},
			}
		}
	}

	height, err := c.blobAPI.Submit(ctx, blobs, &submitOpts)
	if err != nil {
		code := StatusError
		switch {
		case errors.Is(err, context.Canceled):
			code = StatusContextCanceled
		case strings.Contains(err.Error(), ErrTxTimedOut.Error()):
			code = StatusNotIncludedInBlock
		case strings.Contains(err.Error(), ErrTxAlreadyInMempool.Error()):
			code = StatusAlreadyInMempool
		case strings.Contains(err.Error(), ErrTxIncorrectAccountSequence.Error()):
			code = StatusIncorrectAccountSequence
		case strings.Contains(err.Error(), ErrBlobSizeOverLimit.Error()):
			code = StatusTooBig
		case strings.Contains(err.Error(), ErrContextDeadline.Error()):
			code = StatusContextDeadline
		}
		if code == StatusTooBig {
			c.logger.Debug().Err(err).Uint64("status", uint64(code)).Msg("DA submission failed")
		} else {
			c.logger.Error().Err(err).Uint64("status", uint64(code)).Msg("DA submission failed")
		}
		return ResultSubmit{
			BaseResult: BaseResult{
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
		return ResultSubmit{
			BaseResult: BaseResult{
				Code:     StatusSuccess,
				BlobSize: blobSize,
				Height:   height,
			},
		}
	}

	ids := make([]ID, len(blobs))
	for i, b := range blobs {
		ids[i] = blobrpc.MakeID(height, b.Commitment)
	}

	return ResultSubmit{
		BaseResult: BaseResult{
			Code:           StatusSuccess,
			IDs:            ids,
			SubmittedCount: uint64(len(ids)),
			Height:         height,
			BlobSize:       blobSize,
			Timestamp:      time.Now(),
		},
	}
}

// Retrieve retrieves blobs from the DA layer at the specified height and namespace.
func (c *BlobClient) Retrieve(ctx context.Context, height uint64, namespace []byte) ResultRetrieve {
	ns, err := share.NewNamespaceFromBytes(namespace)
	if err != nil {
		return ResultRetrieve{
			BaseResult: BaseResult{
				Code:    StatusError,
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
		case strings.Contains(err.Error(), ErrBlobNotFound.Error()):
			return ResultRetrieve{
				BaseResult: BaseResult{
					Code:      StatusNotFound,
					Message:   ErrBlobNotFound.Error(),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		case strings.Contains(err.Error(), ErrHeightFromFuture.Error()):
			return ResultRetrieve{
				BaseResult: BaseResult{
					Code:      StatusHeightFromFuture,
					Message:   ErrHeightFromFuture.Error(),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		default:
			c.logger.Error().Uint64("height", height).Err(err).Msg("failed to get blobs")
			return ResultRetrieve{
				BaseResult: BaseResult{
					Code:      StatusError,
					Message:   fmt.Sprintf("failed to get blobs: %s", err.Error()),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		}
	}

	if len(blobs) == 0 {
		c.logger.Debug().Uint64("height", height).Msg("No blobs found at height")
		return ResultRetrieve{
			BaseResult: BaseResult{
				Code:      StatusNotFound,
				Message:   ErrBlobNotFound.Error(),
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

	return ResultRetrieve{
		BaseResult: BaseResult{
			Code:      StatusSuccess,
			Height:    height,
			IDs:       ids,
			Timestamp: time.Now(),
		},
		Data: out,
	}
}

// RetrieveHeaders retrieves blobs from the header namespace at the specified height.
func (c *BlobClient) RetrieveHeaders(ctx context.Context, height uint64) ResultRetrieve {
	return c.Retrieve(ctx, height, c.namespaceBz)
}

// RetrieveData retrieves blobs from the data namespace at the specified height.
func (c *BlobClient) RetrieveData(ctx context.Context, height uint64) ResultRetrieve {
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
