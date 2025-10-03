package types

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"

	coreda "github.com/evstack/ev-node/core/da"
)

// SubmitWithHelpers performs blob submission using the underlying DA layer,
// handling error mapping to produce a ResultSubmit.
// It assumes blob size filtering is handled within the DA implementation's Submit.
// It mimics the logic previously found in da.DAClient.Submit.
func SubmitWithHelpers(
	ctx context.Context,
	da coreda.DA, // Use the core DA interface
	logger zerolog.Logger,
	data [][]byte,
	gasPrice float64,
	namespace []byte,
	options []byte,
) coreda.ResultSubmit { // Return core ResultSubmit type
	ids, err := da.SubmitWithOptions(ctx, data, gasPrice, namespace, options)

	// calculate blob size
	var blobSize uint64
	for _, blob := range data {
		blobSize += uint64(len(blob))
	}

	// Handle errors returned by Submit
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Debug().Msg("DA submission canceled via helper due to context cancellation")
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
			logger.Debug().Err(err).Uint64("status", uint64(status)).Msg("DA submission failed via helper")
		} else {
			logger.Error().Err(err).Uint64("status", uint64(status)).Msg("DA submission failed via helper")
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
		logger.Warn().Msg("DA submission via helper returned no IDs for non-empty input data")
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
			logger.Error().Err(err).Msg("failed to split ID")
		}
	}

	logger.Debug().Int("num_ids", len(ids)).Msg("DA submission successful via helper")
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

// RetrieveWithHelpers performs blob retrieval using the underlying DA layer,
// handling error mapping to produce a ResultRetrieve.
// It mimics the logic previously found in da.DAClient.Retrieve.
// requestTimeout defines the timeout for the each retrieval request.
func RetrieveWithHelpers(
	ctx context.Context,
	da coreda.DA,
	logger zerolog.Logger,
	dataLayerHeight uint64,
	namespace []byte,
	requestTimeout time.Duration,
) coreda.ResultRetrieve {
	// 1. Get IDs
	getIDsCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	idsResult, err := da.GetIDs(getIDsCtx, dataLayerHeight, namespace)
	if err != nil {
		// Handle specific "not found" error
		if strings.Contains(err.Error(), coreda.ErrBlobNotFound.Error()) {
			logger.Debug().Uint64("height", dataLayerHeight).Msg("Retrieve helper: Blobs not found at height")
			return coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code:      coreda.StatusNotFound,
					Message:   coreda.ErrBlobNotFound.Error(),
					Height:    dataLayerHeight,
					Timestamp: time.Now(),
				},
			}
		}
		if strings.Contains(err.Error(), coreda.ErrHeightFromFuture.Error()) {
			logger.Debug().Uint64("height", dataLayerHeight).Msg("Retrieve helper: Blobs not found at height")
			return coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code:      coreda.StatusHeightFromFuture,
					Message:   coreda.ErrHeightFromFuture.Error(),
					Height:    dataLayerHeight,
					Timestamp: time.Now(),
				},
			}
		}
		// Handle other errors during GetIDs
		logger.Error().Uint64("height", dataLayerHeight).Err(err).Msg("Retrieve helper: Failed to get IDs")
		return coreda.ResultRetrieve{
			BaseResult: coreda.BaseResult{
				Code:      coreda.StatusError,
				Message:   fmt.Sprintf("failed to get IDs: %s", err.Error()),
				Height:    dataLayerHeight,
				Timestamp: time.Now(),
			},
		}
	}

	// This check should technically be redundant if GetIDs correctly returns ErrBlobNotFound
	if idsResult == nil || len(idsResult.IDs) == 0 {
		logger.Debug().Uint64("height", dataLayerHeight).Msg("Retrieve helper: No IDs found at height")
		return coreda.ResultRetrieve{
			BaseResult: coreda.BaseResult{
				Code:      coreda.StatusNotFound,
				Message:   coreda.ErrBlobNotFound.Error(),
				Height:    dataLayerHeight,
				Timestamp: time.Now(),
			},
		}
	}
	// 2. Get Blobs using the retrieved IDs in batches
	batchSize := 100
	blobs := make([][]byte, 0, len(idsResult.IDs))
	for i := 0; i < len(idsResult.IDs); i += batchSize {
		end := min(i+batchSize, len(idsResult.IDs))

		getBlobsCtx, cancel := context.WithTimeout(ctx, requestTimeout)
		batchBlobs, err := da.Get(getBlobsCtx, idsResult.IDs[i:end], namespace)
		cancel()
		if err != nil {
			// Handle errors during Get
			logger.Error().Uint64("height", dataLayerHeight).Int("num_ids", len(idsResult.IDs)).Err(err).Msg("Retrieve helper: Failed to get blobs")
			return coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code:      coreda.StatusError,
					Message:   fmt.Sprintf("failed to get blobs for batch %d-%d: %s", i, end-1, err.Error()),
					Height:    dataLayerHeight,
					Timestamp: time.Now(),
				},
			}
		}
		blobs = append(blobs, batchBlobs...)
	}
	// Success
	logger.Debug().Uint64("height", dataLayerHeight).Int("num_blobs", len(blobs)).Msg("Retrieve helper: Successfully retrieved blobs")
	return coreda.ResultRetrieve{
		BaseResult: coreda.BaseResult{
			Code:      coreda.StatusSuccess,
			Height:    dataLayerHeight,
			IDs:       idsResult.IDs,
			Timestamp: idsResult.Timestamp,
		},
		Data: blobs,
	}
}
