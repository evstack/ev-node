package types

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog"

	coreda "github.com/evstack/ev-node/core/da"
)

var placeholder = []byte("placeholder")

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
	options []byte,
) coreda.ResultSubmit { // Return core ResultSubmit type
	ids, err := da.SubmitWithOptions(ctx, data, gasPrice, placeholder, options)

	// Handle errors returned by Submit
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Debug().Msg("DA submission canceled via helper due to context cancellation")
			return coreda.ResultSubmit{
				BaseResult: coreda.BaseResult{
					Code:    coreda.StatusContextCanceled,
					Message: "submission canceled",
					IDs:     ids,
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
		logger.Error().Err(err).Str("status", string(status)).Msg("DA submission failed via helper")
		return coreda.ResultSubmit{
			BaseResult: coreda.BaseResult{
				Code:           status,
				Message:        "failed to submit blobs: " + err.Error(),
				IDs:            ids,
				SubmittedCount: uint64(len(ids)),
				Height:         0,
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
			BlobSize:       0,
		},
	}
}

// RetrieveWithHelpers performs blob retrieval using the underlying DA layer,
// handling error mapping to produce a ResultRetrieve.
// It mimics the logic previously found in da.DAClient.Retrieve.
func RetrieveWithHelpers(
	ctx context.Context,
	da coreda.DA,
	logger zerolog.Logger,
	dataLayerHeight uint64,
	namespace []byte,
) coreda.ResultRetrieve {

	// 1. Get IDs
	idsResult, err := da.GetIDs(ctx, dataLayerHeight, namespace)
	if err != nil {
		// Handle specific "not found" error
		if strings.Contains(err.Error(), coreda.ErrBlobNotFound.Error()) {
			logger.Debug().Uint64("height", dataLayerHeight).Msg("Retrieve helper: Blobs not found at height")
			return coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code:    coreda.StatusNotFound,
					Message: coreda.ErrBlobNotFound.Error(),
					Height:  dataLayerHeight,
				},
			}
		}
		if strings.Contains(err.Error(), coreda.ErrHeightFromFuture.Error()) {
			logger.Debug().Uint64("height", dataLayerHeight).Msg("Retrieve helper: Blobs not found at height")
			return coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code:    coreda.StatusHeightFromFuture,
					Message: coreda.ErrHeightFromFuture.Error(),
					Height:  dataLayerHeight,
				},
			}
		}
		// Handle other errors during GetIDs
		logger.Error().Uint64("height", dataLayerHeight).Err(err).Msg("Retrieve helper: Failed to get IDs")
		return coreda.ResultRetrieve{
			BaseResult: coreda.BaseResult{
				Code:    coreda.StatusError,
				Message: fmt.Sprintf("failed to get IDs: %s", err.Error()),
				Height:  dataLayerHeight,
			},
		}
	}

	// This check should technically be redundant if GetIDs correctly returns ErrBlobNotFound
	if idsResult == nil || len(idsResult.IDs) == 0 {
		logger.Debug().Uint64("height", dataLayerHeight).Msg("Retrieve helper: No IDs found at height")
		return coreda.ResultRetrieve{
			BaseResult: coreda.BaseResult{
				Code:    coreda.StatusNotFound,
				Message: coreda.ErrBlobNotFound.Error(),
				Height:  dataLayerHeight,
			},
		}
	}
	// 2. Get Blobs using the retrieved IDs in batches
	batchSize := 100
	blobs := make([][]byte, 0, len(idsResult.IDs))
	for i := 0; i < len(idsResult.IDs); i += batchSize {
		end := min(i+batchSize, len(idsResult.IDs))

		batchBlobs, err := da.Get(ctx, idsResult.IDs[i:end], namespace)
		if err != nil {
			// Handle errors during Get
			logger.Error().Uint64("height", dataLayerHeight).Int("num_ids", len(idsResult.IDs)).Err(err).Msg("Retrieve helper: Failed to get blobs")
			return coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code:    coreda.StatusError,
					Message: fmt.Sprintf("failed to get blobs for batch %d-%d: %s", i, end-1, err.Error()),
					Height:  dataLayerHeight,
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
