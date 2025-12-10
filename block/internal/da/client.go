package da

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/celestiaorg/go-square/v3/share"
	blobrpc "github.com/evstack/ev-node/da/jsonrpc"
	"github.com/rs/zerolog"

	datypes "github.com/evstack/ev-node/pkg/da/types"
)

var (
	ErrBlobNotFound               = datypes.ErrBlobNotFound
	ErrBlobSizeOverLimit          = datypes.ErrBlobSizeOverLimit
	ErrTxTimedOut                 = datypes.ErrTxTimedOut
	ErrTxAlreadyInMempool         = datypes.ErrTxAlreadyInMempool
	ErrTxIncorrectAccountSequence = datypes.ErrTxIncorrectAccountSequence
	ErrContextDeadline            = datypes.ErrContextDeadline
	ErrHeightFromFuture           = datypes.ErrHeightFromFuture
	ErrContextCanceled            = datypes.ErrContextCanceled
)

// Config contains configuration for the Celestia blob client.
type Config struct {
	Client                   *blobrpc.Client
	Logger                   zerolog.Logger
	DefaultTimeout           time.Duration
	Namespace                string
	DataNamespace            string
	ForcedInclusionNamespace string
	MaxBlobSize              uint64
}

// Client wraps the blob RPC with namespace handling and error mapping.
type Client struct {
	blobAPI            *blobrpc.BlobAPI
	logger             zerolog.Logger
	defaultTimeout     time.Duration
	namespaceBz        []byte
	dataNamespaceBz    []byte
	forcedNamespaceBz  []byte
	hasForcedNamespace bool
	maxBlobSize        uint64
}

// Ensure Client implements the block DA client interface.
var _ Interface = (*Client)(nil)

// NewClient creates a new blob client wrapper with pre-calculated namespace bytes.
func NewClient(cfg Config) *Client {
	if cfg.Client == nil {
		return nil
	}
	if cfg.DefaultTimeout == 0 {
		cfg.DefaultTimeout = 30 * time.Second
	}
	if cfg.MaxBlobSize == 0 {
		cfg.MaxBlobSize = blobrpc.DefaultMaxBlobSize
	}

	return &Client{
		blobAPI:         &cfg.Client.Blob,
		logger:          cfg.Logger.With().Str("component", "blob_da_client").Logger(),
		defaultTimeout:  cfg.DefaultTimeout,
		namespaceBz:     share.MustNewV0Namespace([]byte(cfg.Namespace)).Bytes(),
		dataNamespaceBz: share.MustNewV0Namespace([]byte(cfg.DataNamespace)).Bytes(),
		forcedNamespaceBz: func() []byte {
			if cfg.ForcedInclusionNamespace == "" {
				return nil
			}
			return share.MustNewV0Namespace([]byte(cfg.ForcedInclusionNamespace)).Bytes()
		}(),
		hasForcedNamespace: cfg.ForcedInclusionNamespace != "",
		maxBlobSize:        cfg.MaxBlobSize,
	}
}

// Submit submits blobs to the DA layer with the specified options.
func (c *Client) Submit(ctx context.Context, data [][]byte, _ float64, namespace []byte, options []byte) datypes.ResultSubmit {
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

	blobs := make([]*blobrpc.Blob, len(data))
	for i, raw := range data {
		if uint64(len(raw)) > c.maxBlobSize {
			return datypes.ResultSubmit{
				BaseResult: datypes.BaseResult{
					Code:    datypes.StatusTooBig,
					Message: datypes.ErrBlobSizeOverLimit.Error(),
				},
			}
		}
		blobs[i], err = blobrpc.NewBlobV0(ns, raw)
		if err != nil {
			return datypes.ResultSubmit{
				BaseResult: datypes.BaseResult{
					Code:    datypes.StatusError,
					Message: fmt.Sprintf("failed to build blob %d: %v", i, err),
				},
			}
		}
	}

	var submitOpts blobrpc.SubmitOptions
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

	height, err := c.blobAPI.Submit(ctx, blobs, &submitOpts)
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
		ids[i] = blobrpc.MakeID(height, b.Commitment)
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
func (c *Client) Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
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

	getCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer cancel()

	blobs, err := c.blobAPI.GetAll(getCtx, height, []share.Namespace{ns})
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
		ids[i] = blobrpc.MakeID(height, b.Commitment)
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
func (c *Client) RetrieveHeaders(ctx context.Context, height uint64) datypes.ResultRetrieve {
	return c.Retrieve(ctx, height, c.namespaceBz)
}

// RetrieveData retrieves blobs from the data namespace at the specified height.
func (c *Client) RetrieveData(ctx context.Context, height uint64) datypes.ResultRetrieve {
	return c.Retrieve(ctx, height, c.dataNamespaceBz)
}

// RetrieveForcedInclusion retrieves blobs from the forced inclusion namespace at the specified height.
func (c *Client) RetrieveForcedInclusion(ctx context.Context, height uint64) datypes.ResultRetrieve {
	if !c.hasForcedNamespace {
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:    datypes.StatusError,
				Message: "forced inclusion namespace not configured",
				Height:  height,
			},
		}
	}
	return c.Retrieve(ctx, height, c.forcedNamespaceBz)
}

// GetHeaderNamespace returns the header namespace bytes.
func (c *Client) GetHeaderNamespace() []byte {
	return c.namespaceBz
}

// GetDataNamespace returns the data namespace bytes.
func (c *Client) GetDataNamespace() []byte {
	return c.dataNamespaceBz
}

// GetForcedInclusionNamespace returns the forced inclusion namespace bytes.
func (c *Client) GetForcedInclusionNamespace() []byte {
	return c.forcedNamespaceBz
}

// HasForcedInclusionNamespace reports whether forced inclusion namespace is configured.
func (c *Client) HasForcedInclusionNamespace() bool {
	return c.hasForcedNamespace
}

// Get implements a minimal DA surface used by visualization: fetch blobs by IDs.
func (c *Client) Get(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	getCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer cancel()

	res := make([]datypes.Blob, 0, len(ids))
	for _, id := range ids {
		height, commitment := blobrpc.SplitID(id)
		if commitment == nil {
			return nil, fmt.Errorf("invalid blob id: %x", id)
		}

		ns, err := share.NewNamespaceFromBytes(namespace)
		if err != nil {
			return nil, fmt.Errorf("invalid namespace: %w", err)
		}

		b, err := c.blobAPI.Get(getCtx, height, ns, commitment)
		if err != nil {
			return nil, err
		}
		if b == nil {
			continue
		}
		res = append(res, b.Data())
	}

	return res, nil
}

// GetProofs returns inclusion proofs for the provided IDs.
func (c *Client) GetProofs(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Proof, error) {
	if len(ids) == 0 {
		return []datypes.Proof{}, nil
	}

	getCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer cancel()

	ns, err := share.NewNamespaceFromBytes(namespace)
	if err != nil {
		return nil, fmt.Errorf("invalid namespace: %w", err)
	}

	proofs := make([]datypes.Proof, len(ids))
	for i, id := range ids {
		height, commitment := blobrpc.SplitID(id)
		if commitment == nil {
			return nil, fmt.Errorf("invalid blob id: %x", id)
		}

		proof, err := c.blobAPI.GetProof(getCtx, height, ns, commitment)
		if err != nil {
			return nil, err
		}

		bz, err := json.Marshal(proof)
		if err != nil {
			return nil, err
		}
		proofs[i] = bz
	}

	return proofs, nil
}

// Validate mirrors the deprecated DA server logic: it unmarshals proofs and calls Included.
func (c *Client) Validate(ctx context.Context, ids []datypes.ID, proofs []datypes.Proof, namespace []byte) ([]bool, error) {
	if len(ids) != len(proofs) {
		return nil, errors.New("number of IDs and proofs must match")
	}

	validateCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer cancel()

	ns, err := share.NewNamespaceFromBytes(namespace)
	if err != nil {
		return nil, fmt.Errorf("invalid namespace: %w", err)
	}

	results := make([]bool, len(ids))
	for i, id := range ids {
		var proof blobrpc.Proof
		if err := json.Unmarshal(proofs[i], &proof); err != nil {
			return nil, err
		}

		height, commitment := blobrpc.SplitID(id)
		if commitment == nil {
			return nil, fmt.Errorf("invalid blob id: %x", id)
		}

		included, err := c.blobAPI.Included(validateCtx, height, ns, &proof, commitment)
		if err != nil {
			c.logger.Debug().Err(err).Uint64("height", height).Msg("blob inclusion check returned error")
		}
		results[i] = included
	}

	return results, nil
}
