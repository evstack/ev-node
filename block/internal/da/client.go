// Package da provides a reusable wrapper around the core DA interface
// with common configuration for namespace handling and timeouts.
package da

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evstack/ev-node/da/jsonrpc/blob"
	"github.com/rs/zerolog"

	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// Client is the interface representing the DA client.
type Client interface {
	Submit(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) datypes.ResultSubmit
	Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve
	RetrieveHeaders(ctx context.Context, height uint64) datypes.ResultRetrieve
	RetrieveData(ctx context.Context, height uint64) datypes.ResultRetrieve
	RetrieveForcedInclusion(ctx context.Context, height uint64) datypes.ResultRetrieve

	GetHeaderNamespace() []byte
	GetDataNamespace() []byte
	GetForcedInclusionNamespace() []byte
	HasForcedInclusionNamespace() bool
	GetDA() datypes.DA
}

// client provides a reusable wrapper around the core DA interface
// with common configuration for namespace handling and timeouts.
type client struct {
	da                         datypes.DA
	logger                     zerolog.Logger
	defaultTimeout             time.Duration
	batchSize                  int
	namespaceBz                []byte
	namespaceDataBz            []byte
	namespaceForcedInclusionBz []byte
	hasForcedInclusionNs       bool
}

const (
	defaultRetrieveBatchSize = 150
)

// Config contains configuration for the DA client.
type Config struct {
	DA                       datypes.DA
	Logger                   zerolog.Logger
	DefaultTimeout           time.Duration
	RetrieveBatchSize        int
	Namespace                string
	DataNamespace            string
	ForcedInclusionNamespace string
}

// NewClient creates a new DA client with pre-calculated namespace bytes.
func NewClient(cfg Config) *client {
	if cfg.DefaultTimeout == 0 {
		cfg.DefaultTimeout = 60 * time.Second
	}
	if cfg.RetrieveBatchSize <= 0 {
		cfg.RetrieveBatchSize = defaultRetrieveBatchSize
	}

	hasForcedInclusionNs := cfg.ForcedInclusionNamespace != ""
	var namespaceForcedInclusionBz []byte
	if hasForcedInclusionNs {
		namespaceForcedInclusionBz = datypes.NamespaceFromString(cfg.ForcedInclusionNamespace).Bytes()
	}

	return &client{
		da:                         cfg.DA,
		logger:                     cfg.Logger.With().Str("component", "da_client").Logger(),
		defaultTimeout:             cfg.DefaultTimeout,
		batchSize:                  cfg.RetrieveBatchSize,
		namespaceBz:                datypes.NamespaceFromString(cfg.Namespace).Bytes(),
		namespaceDataBz:            datypes.NamespaceFromString(cfg.DataNamespace).Bytes(),
		namespaceForcedInclusionBz: namespaceForcedInclusionBz,
		hasForcedInclusionNs:       hasForcedInclusionNs,
	}
}

// Submit submits blobs to the DA layer with the specified options.
func (c *client) Submit(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) datypes.ResultSubmit {
	submitCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer cancel()

	ids, err := c.da.SubmitWithOptions(submitCtx, data, gasPrice, namespace, options)

	// calculate blob size
	var blobSize uint64
	for _, blob := range data {
		blobSize += uint64(len(blob))
	}

	// Handle errors returned by Submit
	if err != nil {
		if errors.Is(err, context.Canceled) {
			c.logger.Debug().Msg("DA submission canceled due to context cancellation")
			return datypes.ResultSubmit{
				BaseResult: datypes.BaseResult{
					Code:     datypes.StatusContextCanceled,
					Message:  "submission canceled",
					IDs:      ids,
					BlobSize: blobSize,
				},
			}
		}
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
		}

		// Use debug level for StatusTooBig as it gets handled later in submitToDA through recursive splitting
		if status == datypes.StatusTooBig {
			c.logger.Debug().Err(err).Uint64("status", uint64(status)).Msg("DA submission failed")
		} else {
			c.logger.Error().Err(err).Uint64("status", uint64(status)).Msg("DA submission failed")
		}
		return datypes.ResultSubmit{
			BaseResult: datypes.BaseResult{
				Code:           status,
				Message:        "failed to submit blobs: " + err.Error(),
				IDs:            ids,
				SubmittedCount: uint64(len(ids)),
				Height:         0,
				Timestamp:      time.Now(),
				BlobSize:       blobSize,
			},
		}
	}

	if len(ids) == 0 && len(data) > 0 {
		c.logger.Warn().Msg("DA submission returned no IDs for non-empty input data")
		return datypes.ResultSubmit{
			BaseResult: datypes.BaseResult{
				Code:    datypes.StatusError,
				Message: "failed to submit blobs: no IDs returned despite non-empty input",
			},
		}
	}

	// Get height from the first ID
	var height uint64
	if len(ids) > 0 {
		height, _ = blob.SplitID(ids[0])
		if err != nil {
			c.logger.Error().Err(err).Msg("failed to split ID")
		}
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

// Retrieve retrieves blobs from the DA layer at the specified height and namespace.
func (c *client) Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	getIDsCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer cancel()

	idsResultCore, err := c.da.GetIDs(getIDsCtx, height, namespace)
	if err != nil {
		// Handle specific "not found" error
		if strings.Contains(err.Error(), datypes.ErrBlobNotFound.Error()) {
			c.logger.Debug().Uint64("height", height).Msg("Blobs not found at height")
			return datypes.ResultRetrieve{
				BaseResult: datypes.BaseResult{
					Code:      datypes.StatusNotFound,
					Message:   datypes.ErrBlobNotFound.Error(),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		}
		if strings.Contains(err.Error(), datypes.ErrHeightFromFuture.Error()) {
			c.logger.Debug().Uint64("height", height).Msg("Blobs not found at height")
			return datypes.ResultRetrieve{
				BaseResult: datypes.BaseResult{
					Code:      datypes.StatusHeightFromFuture,
					Message:   datypes.ErrHeightFromFuture.Error(),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		}
		// Handle other errors during GetIDs
		c.logger.Error().Uint64("height", height).Err(err).Msg("Failed to get IDs")
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:      datypes.StatusError,
				Message:   fmt.Sprintf("failed to get IDs: %s", err.Error()),
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}
	var idsResult *datypes.GetIDsResult
	if idsResultCore != nil {
		idsResult = &datypes.GetIDsResult{
			IDs:       idsResultCore.IDs,
			Timestamp: idsResultCore.Timestamp,
		}
	}

	// This check should technically be redundant if GetIDs correctly returns ErrBlobNotFound
	if idsResult == nil || len(idsResult.IDs) == 0 {
		c.logger.Debug().Uint64("height", height).Msg("No IDs found at height")
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:      datypes.StatusNotFound,
				Message:   datypes.ErrBlobNotFound.Error(),
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}
	// 2. Get Blobs using the retrieved IDs in batches
	// Each batch has its own timeout while keeping the link to the parent context
	batchSize := c.batchSize
	blobs := make([][]byte, 0, len(idsResult.IDs))
	for i := 0; i < len(idsResult.IDs); i += batchSize {
		end := min(i+batchSize, len(idsResult.IDs))

		getBlobsCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
		batchBlobs, err := c.da.Get(getBlobsCtx, idsResult.IDs[i:end], namespace)
		cancel()
		if err != nil {
			// Handle errors during Get
			c.logger.Error().Uint64("height", height).Int("num_ids", len(idsResult.IDs)).Err(err).Msg("Failed to get blobs")
			return datypes.ResultRetrieve{
				BaseResult: datypes.BaseResult{
					Code:      datypes.StatusError,
					Message:   fmt.Sprintf("failed to get blobs for batch %d-%d: %s", i, end-1, err.Error()),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		}
		blobs = append(blobs, batchBlobs...)
	}
	// Success
	c.logger.Debug().Uint64("height", height).Int("num_blobs", len(blobs)).Msg("Successfully retrieved blobs")
	return datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{
			Code:      datypes.StatusSuccess,
			Height:    height,
			IDs:       idsResult.IDs,
			Timestamp: idsResult.Timestamp,
		},
		Data: blobs,
	}
}

// RetrieveHeaders retrieves blobs from the header namespace at the specified height.
func (c *client) RetrieveHeaders(ctx context.Context, height uint64) datypes.ResultRetrieve {
	return c.Retrieve(ctx, height, c.namespaceBz)
}

// RetrieveData retrieves blobs from the data namespace at the specified height.
func (c *client) RetrieveData(ctx context.Context, height uint64) datypes.ResultRetrieve {
	return c.Retrieve(ctx, height, c.namespaceDataBz)
}

// RetrieveForcedInclusion retrieves blobs from the forced inclusion namespace at the specified height.
func (c *client) RetrieveForcedInclusion(ctx context.Context, height uint64) datypes.ResultRetrieve {
	if !c.hasForcedInclusionNs {
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:    datypes.StatusError,
				Message: "forced inclusion namespace not configured",
			},
		}
	}
	return c.Retrieve(ctx, height, c.namespaceForcedInclusionBz)
}

// GetHeaderNamespace returns the header namespace bytes.
func (c *client) GetHeaderNamespace() []byte {
	return c.namespaceBz
}

// GetDataNamespace returns the data namespace bytes.
func (c *client) GetDataNamespace() []byte {
	return c.namespaceDataBz
}

// GetForcedInclusionNamespace returns the forced inclusion namespace bytes.
func (c *client) GetForcedInclusionNamespace() []byte {
	return c.namespaceForcedInclusionBz
}

// HasForcedInclusionNamespace returns whether forced inclusion namespace is configured.
func (c *client) HasForcedInclusionNamespace() bool {
	return c.hasForcedInclusionNs
}

// GetDA returns the underlying DA interface for advanced usage.
func (c *client) GetDA() datypes.DA {
	return c.da
}
