// Package da provides a reusable wrapper around the core DA interface
// with common configuration for namespace handling and timeouts.
package da

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"

	coreda "github.com/evstack/ev-node/core/da"
)

// Client is the interface representing the DA client.
type Client interface {
	Submit(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) coreda.ResultSubmit
	Retrieve(ctx context.Context, height uint64, namespace []byte) coreda.ResultRetrieve
	RetrieveHeaders(ctx context.Context, height uint64) coreda.ResultRetrieve
	RetrieveData(ctx context.Context, height uint64) coreda.ResultRetrieve

	GetHeaderNamespace() []byte
	GetDataNamespace() []byte
	GetDA() coreda.DA
}

// client provides a reusable wrapper around the core DA interface
// with common configuration for namespace handling and timeouts.
type client struct {
	da              coreda.DA
	logger          zerolog.Logger
	defaultTimeout  time.Duration
	batchSize       int
	namespaceBz     []byte
	namespaceDataBz []byte
}

const (
	defaultRetrieveBatchSize = 150
)

// Config contains configuration for the DA client.
type Config struct {
	DA                coreda.DA
	Logger            zerolog.Logger
	DefaultTimeout    time.Duration
	Namespace         string
	DataNamespace     string
	RetrieveBatchSize int
}

// NewClient creates a new DA client with pre-calculated namespace bytes.
func NewClient(cfg Config) *client {
	if cfg.DefaultTimeout == 0 {
		cfg.DefaultTimeout = 60 * time.Second
	}
	if cfg.RetrieveBatchSize <= 0 {
		cfg.RetrieveBatchSize = defaultRetrieveBatchSize
	}

	return &client{
		da:              cfg.DA,
		logger:          cfg.Logger.With().Str("component", "da_client").Logger(),
		defaultTimeout:  cfg.DefaultTimeout,
		batchSize:       cfg.RetrieveBatchSize,
		namespaceBz:     coreda.NamespaceFromString(cfg.Namespace).Bytes(),
		namespaceDataBz: coreda.NamespaceFromString(cfg.DataNamespace).Bytes(),
	}
}

// Submit submits blobs to the DA layer with the specified options.
func (c *client) Submit(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) coreda.ResultSubmit {
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
			return coreda.ResultSubmit{
				BaseResult: coreda.BaseResult{
					Code:     coreda.StatusContextCanceled,
					Message:  "submission canceled",
					IDs:      ids,
					BlobSize: blobSize,
				},
			}
		}
		status := coreda.StatusError
		switch {
		case errors.Is(err, coreda.ErrTxTimedOut):
			status = coreda.StatusNotIncludedInBlock
		case errors.Is(err, coreda.ErrTxAlreadyInMempool):
			status = coreda.StatusAlreadyInMempool
		case errors.Is(err, coreda.ErrTxIncorrectAccountSequence):
			status = coreda.StatusIncorrectAccountSequence
		case errors.Is(err, coreda.ErrBlobSizeOverLimit):
			status = coreda.StatusTooBig
		case errors.Is(err, coreda.ErrContextDeadline):
			status = coreda.StatusContextDeadline
		}

		// Use debug level for StatusTooBig as it gets handled later in submitToDA through recursive splitting
		if status == coreda.StatusTooBig {
			c.logger.Debug().Err(err).Uint64("status", uint64(status)).Msg("DA submission failed")
		} else {
			c.logger.Error().Err(err).Uint64("status", uint64(status)).Msg("DA submission failed")
		}
		return coreda.ResultSubmit{
			BaseResult: coreda.BaseResult{
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
		return coreda.ResultSubmit{
			BaseResult: coreda.BaseResult{
				Code:    coreda.StatusError,
				Message: "failed to submit blobs: no IDs returned despite non-empty input",
			},
		}
	}

	// Get height from the first ID
	var height uint64
	if len(ids) > 0 {
		height, _, err = coreda.SplitID(ids[0])
		if err != nil {
			c.logger.Error().Err(err).Msg("failed to split ID")
		}
	}

	c.logger.Debug().Int("num_ids", len(ids)).Msg("DA submission successful")
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
func (c *client) Retrieve(ctx context.Context, height uint64, namespace []byte) coreda.ResultRetrieve {
	getIDsCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer cancel()

	idsResult, err := c.da.GetIDs(getIDsCtx, height, namespace)
	if err != nil {
		// Handle specific "not found" error
		if strings.Contains(err.Error(), coreda.ErrBlobNotFound.Error()) {
			c.logger.Debug().Uint64("height", height).Msg("Blobs not found at height")
			return coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code:      coreda.StatusNotFound,
					Message:   coreda.ErrBlobNotFound.Error(),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		}
		if strings.Contains(err.Error(), coreda.ErrHeightFromFuture.Error()) {
			c.logger.Debug().Uint64("height", height).Msg("Blobs not found at height")
			return coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code:      coreda.StatusHeightFromFuture,
					Message:   coreda.ErrHeightFromFuture.Error(),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		}
		// Handle other errors during GetIDs
		c.logger.Error().Uint64("height", height).Err(err).Msg("Failed to get IDs")
		return coreda.ResultRetrieve{
			BaseResult: coreda.BaseResult{
				Code:      coreda.StatusError,
				Message:   fmt.Sprintf("failed to get IDs: %s", err.Error()),
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}

	// This check should technically be redundant if GetIDs correctly returns ErrBlobNotFound
	if idsResult == nil || len(idsResult.IDs) == 0 {
		c.logger.Debug().Uint64("height", height).Msg("No IDs found at height")
		return coreda.ResultRetrieve{
			BaseResult: coreda.BaseResult{
				Code:      coreda.StatusNotFound,
				Message:   coreda.ErrBlobNotFound.Error(),
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
			return coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code:      coreda.StatusError,
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
	return coreda.ResultRetrieve{
		BaseResult: coreda.BaseResult{
			Code:      coreda.StatusSuccess,
			Height:    height,
			IDs:       idsResult.IDs,
			Timestamp: idsResult.Timestamp,
		},
		Data: blobs,
	}
}

// RetrieveHeaders retrieves blobs from the header namespace at the specified height.
func (c *client) RetrieveHeaders(ctx context.Context, height uint64) coreda.ResultRetrieve {
	return c.Retrieve(ctx, height, c.namespaceBz)
}

// RetrieveData retrieves blobs from the data namespace at the specified height.
func (c *client) RetrieveData(ctx context.Context, height uint64) coreda.ResultRetrieve {
	return c.Retrieve(ctx, height, c.namespaceDataBz)
}

// GetHeaderNamespace returns the header namespace bytes.
func (c *client) GetHeaderNamespace() []byte {
	return c.namespaceBz
}

// GetDataNamespace returns the data namespace bytes.
func (c *client) GetDataNamespace() []byte {
	return c.namespaceDataBz
}

// GetDA returns the underlying DA interface for advanced usage.
func (c *client) GetDA() coreda.DA {
	return c.da
}
