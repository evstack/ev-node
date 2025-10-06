package jsonrpc

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/core/da"
)

//go:generate mockgen -destination=mocks/api.go -package=mocks . Module
type Module interface {
	da.DA
}

// API defines the jsonrpc service module API
type API struct {
	Logger        zerolog.Logger
	MaxBlobSize   uint64
	gasPrice      float64
	gasMultiplier float64
	Internal      struct {
		Get               func(ctx context.Context, ids []da.ID, ns []byte) ([]da.Blob, error)           `perm:"read"`
		GetIDs            func(ctx context.Context, height uint64, ns []byte) (*da.GetIDsResult, error)  `perm:"read"`
		GetProofs         func(ctx context.Context, ids []da.ID, ns []byte) ([]da.Proof, error)          `perm:"read"`
		Commit            func(ctx context.Context, blobs []da.Blob, ns []byte) ([]da.Commitment, error) `perm:"read"`
		Validate          func(context.Context, []da.ID, []da.Proof, []byte) ([]bool, error)             `perm:"read"`
		Submit            func(context.Context, []da.Blob, float64, []byte) ([]da.ID, error)             `perm:"write"`
		SubmitWithOptions func(context.Context, []da.Blob, float64, []byte, []byte) ([]da.ID, error)     `perm:"write"`
		GasMultiplier     func(context.Context) (float64, error)                                         `perm:"read"`
		GasPrice          func(context.Context) (float64, error)                                         `perm:"read"`
	}
}

// Get returns Blob for each given ID, or an error.
func (api *API) Get(ctx context.Context, ids []da.ID, ns []byte) ([]da.Blob, error) {
	api.Logger.Debug().Str("method", "Get").Int("num_ids", len(ids)).Str("namespace", hex.EncodeToString(ns)).Msg("Making RPC call")
	res, err := api.Internal.Get(ctx, ids, ns)
	if err != nil {
		if strings.Contains(err.Error(), context.Canceled.Error()) {
			api.Logger.Debug().Str("method", "Get").Msg("RPC call canceled due to context cancellation")
			return res, context.Canceled
		}
		api.Logger.Error().Err(err).Str("method", "Get").Msg("RPC call failed")
		// Wrap error for context, potentially using the translated error from the RPC library
		return nil, fmt.Errorf("failed to get blobs: %w", err)
	}
	api.Logger.Debug().Str("method", "Get").Int("num_blobs_returned", len(res)).Msg("RPC call successful")
	return res, nil
}

// GetIDs returns IDs of all Blobs located in DA at given height.
func (api *API) GetIDs(ctx context.Context, height uint64, ns []byte) (*da.GetIDsResult, error) {
	api.Logger.Debug().Str("method", "GetIDs").Uint64("height", height).Str("namespace", hex.EncodeToString(ns)).Msg("Making RPC call")
	res, err := api.Internal.GetIDs(ctx, height, ns)
	if err != nil {
		// Using strings.contains since JSON RPC serialization doesn't preserve error wrapping
		// Check if the error is specifically BlobNotFound, otherwise log and return
		if strings.Contains(err.Error(), da.ErrBlobNotFound.Error()) { // Use the error variable directly
			api.Logger.Debug().Str("method", "GetIDs").Uint64("height", height).Msg("RPC call indicates blobs not found")
			return nil, err // Return the specific ErrBlobNotFound
		}
		if strings.Contains(err.Error(), da.ErrHeightFromFuture.Error()) {
			api.Logger.Debug().Str("method", "GetIDs").Uint64("height", height).Msg("RPC call indicates height from future")
			return nil, err // Return the specific ErrHeightFromFuture
		}
		if strings.Contains(err.Error(), context.Canceled.Error()) {
			api.Logger.Debug().Str("method", "GetIDs").Msg("RPC call canceled due to context cancellation")
			return res, context.Canceled
		}
		api.Logger.Error().Err(err).Str("method", "GetIDs").Msg("RPC call failed")
		return nil, err
	}

	// Handle cases where the RPC call succeeds but returns no IDs
	if res == nil || len(res.IDs) == 0 {
		api.Logger.Debug().Str("method", "GetIDs").Uint64("height", height).Msg("RPC call successful but no IDs found")
		return nil, da.ErrBlobNotFound // Return specific error for not found (use variable directly)
	}

	api.Logger.Debug().Str("method", "GetIDs").Msg("RPC call successful")
	return res, nil
}

// GetProofs returns inclusion Proofs for Blobs specified by their IDs.
func (api *API) GetProofs(ctx context.Context, ids []da.ID, ns []byte) ([]da.Proof, error) {
	api.Logger.Debug().Str("method", "GetProofs").Int("num_ids", len(ids)).Str("namespace", hex.EncodeToString(ns)).Msg("Making RPC call")
	res, err := api.Internal.GetProofs(ctx, ids, ns)
	if err != nil {
		api.Logger.Error().Err(err).Str("method", "GetProofs").Msg("RPC call failed")
	} else {
		api.Logger.Debug().Str("method", "GetProofs").Int("num_proofs_returned", len(res)).Msg("RPC call successful")
	}
	return res, err
}

// Commit creates a Commitment for each given Blob.
func (api *API) Commit(ctx context.Context, blobs []da.Blob, ns []byte) ([]da.Commitment, error) {
	api.Logger.Debug().Str("method", "Commit").Int("num_blobs", len(blobs)).Str("namespace", hex.EncodeToString(ns)).Msg("Making RPC call")
	res, err := api.Internal.Commit(ctx, blobs, ns)
	if err != nil {
		api.Logger.Error().Err(err).Str("method", "Commit").Msg("RPC call failed")
	} else {
		api.Logger.Debug().Str("method", "Commit").Int("num_commitments_returned", len(res)).Msg("RPC call successful")
	}
	return res, err
}

// Validate validates Commitments against the corresponding Proofs. This should be possible without retrieving the Blobs.
func (api *API) Validate(ctx context.Context, ids []da.ID, proofs []da.Proof, ns []byte) ([]bool, error) {
	api.Logger.Debug().Str("method", "Validate").Int("num_ids", len(ids)).Int("num_proofs", len(proofs)).Str("namespace", hex.EncodeToString(ns)).Msg("Making RPC call")
	res, err := api.Internal.Validate(ctx, ids, proofs, ns)
	if err != nil {
		api.Logger.Error().Err(err).Str("method", "Validate").Msg("RPC call failed")
	} else {
		api.Logger.Debug().Str("method", "Validate").Int("num_results_returned", len(res)).Msg("RPC call successful")
	}
	return res, err
}

// Submit submits the Blobs to Data Availability layer.
func (api *API) Submit(ctx context.Context, blobs []da.Blob, gasPrice float64, ns []byte) ([]da.ID, error) {
	api.Logger.Debug().Str("method", "Submit").Int("num_blobs", len(blobs)).Float64("gas_price", gasPrice).Str("namespace", hex.EncodeToString(ns)).Msg("Making RPC call")
	res, err := api.Internal.Submit(ctx, blobs, gasPrice, ns)
	if err != nil {
		if strings.Contains(err.Error(), context.Canceled.Error()) {
			api.Logger.Debug().Str("method", "Submit").Msg("RPC call canceled due to context cancellation")
			return res, context.Canceled
		}
		api.Logger.Error().Err(err).Str("method", "Submit").Bytes("namespace", ns).Msg("RPC call failed")
	} else {
		api.Logger.Debug().Str("method", "Submit").Int("num_ids_returned", len(res)).Msg("RPC call successful")
	}
	return res, err
}

// SubmitWithOptions submits the Blobs to Data Availability layer with additional options.
// It validates the entire batch against MaxBlobSize before submission.
// If any blob or the total batch size exceeds limits, it returns ErrBlobSizeOverLimit.
func (api *API) SubmitWithOptions(ctx context.Context, inputBlobs []da.Blob, gasPrice float64, ns []byte, options []byte) ([]da.ID, error) {
	maxBlobSize := api.MaxBlobSize

	if len(inputBlobs) == 0 {
		return []da.ID{}, nil
	}

	// Validate each blob individually and calculate total size
	var totalSize uint64
	for i, blob := range inputBlobs {
		blobLen := uint64(len(blob))
		if blobLen > maxBlobSize {
			api.Logger.Warn().Int("index", i).Uint64("blobSize", blobLen).Uint64("maxBlobSize", maxBlobSize).Msg("Individual blob exceeds MaxBlobSize")
			return nil, da.ErrBlobSizeOverLimit
		}
		totalSize += blobLen
	}

	// Validate total batch size
	if totalSize > maxBlobSize {
		return nil, da.ErrBlobSizeOverLimit
	}

	api.Logger.Debug().Str("method", "SubmitWithOptions").Int("num_blobs", len(inputBlobs)).Uint64("total_size", totalSize).Float64("gas_price", gasPrice).Str("namespace", hex.EncodeToString(ns)).Msg("Making RPC call")
	res, err := api.Internal.SubmitWithOptions(ctx, inputBlobs, gasPrice, ns, options)
	if err != nil {
		if strings.Contains(err.Error(), context.Canceled.Error()) {
			api.Logger.Debug().Str("method", "SubmitWithOptions").Msg("RPC call canceled due to context cancellation")
			return res, context.Canceled
		}
		api.Logger.Error().Err(err).Str("method", "SubmitWithOptions").Msg("RPC call failed")
	} else {
		api.Logger.Debug().Str("method", "SubmitWithOptions").Int("num_ids_returned", len(res)).Msg("RPC call successful")
	}

	return res, err
}

func (api *API) GasMultiplier(ctx context.Context) (float64, error) {
	api.Logger.Debug().Str("method", "GasMultiplier").Msg("Making RPC call")

	return api.gasMultiplier, nil
}

func (api *API) GasPrice(ctx context.Context) (float64, error) {
	api.Logger.Debug().Str("method", "GasPrice").Msg("Making RPC call")

	return api.gasPrice, nil
}

// Client is the jsonrpc client
type Client struct {
	DA     API
	closer multiClientCloser
}

// multiClientCloser is a wrapper struct to close clients across multiple namespaces.
type multiClientCloser struct {
	closers []jsonrpc.ClientCloser
}

// register adds a new closer to the multiClientCloser
func (m *multiClientCloser) register(closer jsonrpc.ClientCloser) {
	m.closers = append(m.closers, closer)
}

// closeAll closes all saved clients.
func (m *multiClientCloser) closeAll() {
	for _, closer := range m.closers {
		closer()
	}
}

// Close closes the connections to all namespaces registered on the staticClient.
func (c *Client) Close() {
	c.closer.closeAll()
}

// NewClient creates a new Client with one connection per namespace with the
// given token as the authorization token.
func NewClient(ctx context.Context, logger zerolog.Logger, addr, token string, gasPrice, gasMultiplier float64, maxBlobSize uint64) (*Client, error) {
	authHeader := http.Header{"Authorization": []string{fmt.Sprintf("Bearer %s", token)}}
	return newClient(ctx, logger, addr, authHeader, gasPrice, gasMultiplier, maxBlobSize)
}

func newClient(ctx context.Context, logger zerolog.Logger, addr string, authHeader http.Header, gasPrice, gasMultiplier float64, maxBlobSize uint64) (*Client, error) {
	var multiCloser multiClientCloser
	var client Client
	client.DA.Logger = logger
	client.DA.MaxBlobSize = maxBlobSize
	client.DA.gasPrice = gasPrice
	client.DA.gasMultiplier = gasMultiplier

	errs := getKnownErrorsMapping()
	for name, module := range moduleMap(&client) {
		closer, err := jsonrpc.NewMergeClient(ctx, addr, name, []interface{}{module}, authHeader, jsonrpc.WithErrors(errs))
		if err != nil {
			// If an error occurs, close any previously opened connections
			multiCloser.closeAll()
			return nil, err
		}
		multiCloser.register(closer)
	}

	client.closer = multiCloser // Assign the multiCloser to the client

	return &client, nil
}

func moduleMap(client *Client) map[string]interface{} {
	// TODO: this duplication of strings many times across the codebase can be avoided with issue #1176
	return map[string]interface{}{
		"da": &client.DA.Internal,
	}
}
