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

	"github.com/evstack/ev-node/block/internal/common"
	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// Config contains configuration for the blob DA client.
type Config struct {
	DA                       *blobrpc.Client
	Logger                   zerolog.Logger
	DefaultTimeout           time.Duration
	Namespace                string
	DataNamespace            string
	ForcedInclusionNamespace string
}

// client wraps the blob RPC with namespace handling and error mapping.
// It is unexported; callers should use the exported Client interface.
type client struct {
	blobAPI            *blobrpc.BlobAPI
	headerAPI          *blobrpc.HeaderAPI
	logger             zerolog.Logger
	defaultTimeout     time.Duration
	namespaceBz        []byte
	dataNamespaceBz    []byte
	forcedNamespaceBz  []byte
	hasForcedNamespace bool
}

// Ensure client implements the FullClient interface (Client + BlobGetter + Verifier).
var _ FullClient = (*client)(nil)

// NewClient creates a new blob client wrapper with pre-calculated namespace bytes.
func NewClient(cfg Config) FullClient {
	if cfg.DA == nil {
		return nil
	}
	if cfg.DefaultTimeout == 0 {
		cfg.DefaultTimeout = 60 * time.Second
	}

	hasForcedNamespace := cfg.ForcedInclusionNamespace != ""
	var forcedNamespaceBz []byte
	if hasForcedNamespace {
		forcedNamespaceBz = datypes.NamespaceFromString(cfg.ForcedInclusionNamespace).Bytes()
	}

	return &client{
		blobAPI:            &cfg.DA.Blob,
		headerAPI:          &cfg.DA.Header,
		logger:             cfg.Logger.With().Str("component", "da_client").Logger(),
		defaultTimeout:     cfg.DefaultTimeout,
		namespaceBz:        datypes.NamespaceFromString(cfg.Namespace).Bytes(),
		dataNamespaceBz:    datypes.NamespaceFromString(cfg.DataNamespace).Bytes(),
		forcedNamespaceBz:  forcedNamespaceBz,
		hasForcedNamespace: hasForcedNamespace,
	}
}

// Submit submits blobs to the DA layer with the specified options.
func (c *client) Submit(ctx context.Context, data [][]byte, _ float64, namespace []byte, options []byte) datypes.ResultSubmit {
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
		if uint64(len(raw)) > common.DefaultMaxBlobSize {
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
	c.logger.Debug().Int("num_ids", len(ids)).Msg("DA submission successful")

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

// getBlockTimestamp fetches the block timestamp from the DA layer header.
func (c *client) getBlockTimestamp(ctx context.Context, height uint64) (time.Time, error) {
	headerCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer cancel()

	header, err := c.headerAPI.GetByHeight(headerCtx, height)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get header timestamp for block %d: %w", height, err)
	}

	return header.Time(), nil
}

// Retrieve retrieves blobs from the DA layer at the specified height and namespace.
// It uses GetAll to fetch all blobs at once.
// The timestamp is derived from the DA block header to ensure determinism.
func (c *client) Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	ns, err := share.NewNamespaceFromBytes(namespace)
	if err != nil {
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:      datypes.StatusError,
				Message:   fmt.Sprintf("invalid namespace: %v", err),
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}

	blobCtx, blobCancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer blobCancel()

	blobs, err := c.blobAPI.GetAll(blobCtx, height, []share.Namespace{ns})
	if err != nil {
		// Handle known errors by substring because RPC may wrap them.
		switch {
		case strings.Contains(err.Error(), datypes.ErrBlobNotFound.Error()):
			c.logger.Debug().Uint64("height", height).Msg("No blobs found at height")
			// Fetch block timestamp for deterministic responses using parent context
			blockTime, err := c.getBlockTimestamp(ctx, height)
			if err != nil {
				c.logger.Error().Uint64("height", height).Err(err).Msg("failed to get block timestamp")
				blockTime = time.Now()
				// TODO: we should retry fetching the timestamp. Current time may mess block time consistency for based sequencers.
			}

			return datypes.ResultRetrieve{
				BaseResult: datypes.BaseResult{
					Code:      datypes.StatusNotFound,
					Message:   datypes.ErrBlobNotFound.Error(),
					Height:    height,
					Timestamp: blockTime,
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

	// Fetch block timestamp for deterministic responses using parent context
	blockTime, err := c.getBlockTimestamp(ctx, height)
	if err != nil {
		c.logger.Error().Uint64("height", height).Err(err).Msg("failed to get block timestamp")
		blockTime = time.Now()
		// TODO: we should retry fetching the timestamp. Current time may mess block time consistency for based sequencers.
	}

	if len(blobs) == 0 {
		c.logger.Debug().Uint64("height", height).Msg("No blobs found at height")
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:      datypes.StatusNotFound,
				Message:   datypes.ErrBlobNotFound.Error(),
				Height:    height,
				Timestamp: blockTime,
			},
		}
	}

	// Extract IDs and data from the blobs.
	ids := make([]datypes.ID, len(blobs))
	data := make([]datypes.Blob, len(blobs))
	for i, b := range blobs {
		ids[i] = blobrpc.MakeID(height, b.Commitment)
		data[i] = b.Data()
	}

	c.logger.Debug().Int("num_blobs", len(blobs)).Msg("retrieved blobs")

	return datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{
			Code:      datypes.StatusSuccess,
			Height:    height,
			IDs:       ids,
			Timestamp: blockTime,
		},
		Data: data,
	}
}

// GetLatestDAHeight returns the latest height available on the DA layer by
// querying the network head.
func (c *client) GetLatestDAHeight(ctx context.Context) (uint64, error) {
	headCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer cancel()

	header, err := c.headerAPI.NetworkHead(headCtx)
	if err != nil {
		return 0, fmt.Errorf("failed to get DA network head: %w", err)
	}
	if header == nil {
		return 0, fmt.Errorf("DA network head returned nil header")
	}

	return header.Height, nil
}

// RetrieveForcedInclusion retrieves blobs from the forced inclusion namespace at the specified height.
func (c *client) RetrieveForcedInclusion(ctx context.Context, height uint64) datypes.ResultRetrieve {
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
func (c *client) GetHeaderNamespace() []byte {
	return c.namespaceBz
}

// GetDataNamespace returns the data namespace bytes.
func (c *client) GetDataNamespace() []byte {
	return c.dataNamespaceBz
}

// GetForcedInclusionNamespace returns the forced inclusion namespace bytes.
func (c *client) GetForcedInclusionNamespace() []byte {
	return c.forcedNamespaceBz
}

// HasForcedInclusionNamespace reports whether forced inclusion namespace is configured.
func (c *client) HasForcedInclusionNamespace() bool {
	return c.hasForcedNamespace
}

// Get fetches blobs by their IDs. Used for visualization and fetching specific blobs.
func (c *client) Get(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	ns, err := share.NewNamespaceFromBytes(namespace)
	if err != nil {
		return nil, fmt.Errorf("invalid namespace: %w", err)
	}

	getCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer cancel()

	res := make([]datypes.Blob, 0, len(ids))
	for _, id := range ids {
		height, commitment := blobrpc.SplitID(id)
		if commitment == nil {
			return nil, fmt.Errorf("invalid blob id: %x", id)
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
func (c *client) GetProofs(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Proof, error) {
	if len(ids) == 0 {
		return []datypes.Proof{}, nil
	}

	ns, err := share.NewNamespaceFromBytes(namespace)
	if err != nil {
		return nil, fmt.Errorf("invalid namespace: %w", err)
	}

	getCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer cancel()

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

// Validate validates commitments against the corresponding proofs.
func (c *client) Validate(ctx context.Context, ids []datypes.ID, proofs []datypes.Proof, namespace []byte) ([]bool, error) {
	if len(ids) != len(proofs) {
		return nil, errors.New("number of IDs and proofs must match")
	}

	if len(ids) == 0 {
		return []bool{}, nil
	}

	ns, err := share.NewNamespaceFromBytes(namespace)
	if err != nil {
		return nil, fmt.Errorf("invalid namespace: %w", err)
	}

	getCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer cancel()

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

		included, err := c.blobAPI.Included(getCtx, height, ns, &proof, commitment)
		if err != nil {
			c.logger.Debug().Err(err).Uint64("height", height).Msg("blob inclusion check returned error")
		}
		results[i] = included
	}

	return results, nil
}
