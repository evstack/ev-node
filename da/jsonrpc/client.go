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
	"github.com/evstack/ev-node/da/compression"
	internal "github.com/evstack/ev-node/da/jsonrpc/internal"
)

//go:generate mockgen -destination=mocks/api.go -package=mocks . Module
type Module interface {
	da.DA
}

// API defines the jsonrpc service module API
type API struct {
	Logger             zerolog.Logger
	MaxBlobSize        uint64
	gasPrice           float64
	gasMultiplier      float64
	compressionEnabled bool
	compressionConfig  compression.Config
	Internal           struct {
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
	preparedNs := da.PrepareNamespace(ns)
	api.Logger.Debug().Str("method", "Get").Int("num_ids", len(ids)).Str("namespace", hex.EncodeToString(preparedNs)).Msg("Making RPC call")
	res, err := api.Internal.Get(ctx, ids, preparedNs)
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

	// Decompress blobs if compression is enabled
	if api.compressionEnabled && len(res) > 0 {
		decompressed, err := compression.DecompressBatch(res)
		if err != nil {
			api.Logger.Error().Err(err).Msg("Failed to decompress blobs")
			return nil, fmt.Errorf("failed to decompress blobs: %w", err)
		}

		// Log decompression stats
		for i, blob := range res {
			info := compression.GetCompressionInfo(blob)
			if info.IsCompressed {
				api.Logger.Debug().
					Int("blob_index", i).
					Uint64("compressed_size", info.CompressedSize).
					Uint64("original_size", info.OriginalSize).
					Float64("ratio", info.CompressionRatio).
					Msg("Blob decompression stats")
			}
		}

		return decompressed, nil
	}

	return res, nil
}

// GetIDs returns IDs of all Blobs located in DA at given height.
func (api *API) GetIDs(ctx context.Context, height uint64, ns []byte) (*da.GetIDsResult, error) {
	preparedNs := da.PrepareNamespace(ns)
	api.Logger.Debug().Str("method", "GetIDs").Uint64("height", height).Str("namespace", hex.EncodeToString(preparedNs)).Msg("Making RPC call")
	res, err := api.Internal.GetIDs(ctx, height, preparedNs)
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
	preparedNs := da.PrepareNamespace(ns)
	api.Logger.Debug().Str("method", "GetProofs").Int("num_ids", len(ids)).Str("namespace", hex.EncodeToString(preparedNs)).Msg("Making RPC call")
	res, err := api.Internal.GetProofs(ctx, ids, preparedNs)
	if err != nil {
		api.Logger.Error().Err(err).Str("method", "GetProofs").Msg("RPC call failed")
	} else {
		api.Logger.Debug().Str("method", "GetProofs").Int("num_proofs_returned", len(res)).Msg("RPC call successful")
	}
	return res, err
}

// Commit creates a Commitment for each given Blob.
func (api *API) Commit(ctx context.Context, blobs []da.Blob, ns []byte) ([]da.Commitment, error) {
	preparedNs := da.PrepareNamespace(ns)

	// Compress blobs if compression is enabled
	blobsToCommit := blobs
	if api.compressionEnabled && len(blobs) > 0 {
		compressed, err := compression.CompressBatch(blobs)
		if err != nil {
			api.Logger.Error().Err(err).Msg("Failed to compress blobs for commit")
			return nil, fmt.Errorf("failed to compress blobs: %w", err)
		}
		blobsToCommit = compressed
	}

	api.Logger.Debug().Str("method", "Commit").Int("num_blobs", len(blobsToCommit)).Str("namespace", hex.EncodeToString(preparedNs)).Msg("Making RPC call")
	res, err := api.Internal.Commit(ctx, blobsToCommit, preparedNs)
	if err != nil {
		api.Logger.Error().Err(err).Str("method", "Commit").Msg("RPC call failed")
	} else {
		api.Logger.Debug().Str("method", "Commit").Int("num_commitments_returned", len(res)).Msg("RPC call successful")
	}
	return res, err
}

// Validate validates Commitments against the corresponding Proofs. This should be possible without retrieving the Blobs.
func (api *API) Validate(ctx context.Context, ids []da.ID, proofs []da.Proof, ns []byte) ([]bool, error) {
	preparedNs := da.PrepareNamespace(ns)
	api.Logger.Debug().Str("method", "Validate").Int("num_ids", len(ids)).Int("num_proofs", len(proofs)).Str("namespace", hex.EncodeToString(preparedNs)).Msg("Making RPC call")
	res, err := api.Internal.Validate(ctx, ids, proofs, preparedNs)
	if err != nil {
		api.Logger.Error().Err(err).Str("method", "Validate").Msg("RPC call failed")
	} else {
		api.Logger.Debug().Str("method", "Validate").Int("num_results_returned", len(res)).Msg("RPC call successful")
	}
	return res, err
}

// Submit submits the Blobs to Data Availability layer.
func (api *API) Submit(ctx context.Context, blobs []da.Blob, gasPrice float64, ns []byte) ([]da.ID, error) {
	preparedNs := da.PrepareNamespace(ns)

	// Compress blobs if compression is enabled
	blobsToSubmit := blobs
	if api.compressionEnabled && len(blobs) > 0 {
		compressed, err := compression.CompressBatch(blobs)
		if err != nil {
			api.Logger.Error().Err(err).Msg("Failed to compress blobs")
			return nil, fmt.Errorf("failed to compress blobs: %w", err)
		}

		// Log compression stats
		var totalOriginal, totalCompressed uint64
		for i, blob := range compressed {
			info := compression.GetCompressionInfo(blob)
			if info.IsCompressed {
				totalOriginal += info.OriginalSize
				totalCompressed += info.CompressedSize
				api.Logger.Debug().
					Int("blob_index", i).
					Uint64("original_size", info.OriginalSize).
					Uint64("compressed_size", info.CompressedSize).
					Float64("ratio", info.CompressionRatio).
					Msg("Blob compression stats")
			}
		}

		if totalOriginal > 0 {
			savings := float64(totalOriginal-totalCompressed) / float64(totalOriginal) * 100
			api.Logger.Info().
				Uint64("total_original", totalOriginal).
				Uint64("total_compressed", totalCompressed).
				Float64("savings_percent", savings).
				Msg("Compression summary")
		}

		blobsToSubmit = compressed
	}

	api.Logger.Debug().Str("method", "Submit").Int("num_blobs", len(blobsToSubmit)).Float64("gas_price", gasPrice).Str("namespace", hex.EncodeToString(preparedNs)).Msg("Making RPC call")
	res, err := api.Internal.Submit(ctx, blobsToSubmit, gasPrice, preparedNs)
	if err != nil {
		if strings.Contains(err.Error(), context.Canceled.Error()) {
			api.Logger.Debug().Str("method", "Submit").Msg("RPC call canceled due to context cancellation")
			return res, context.Canceled
		}
		api.Logger.Error().Err(err).Str("method", "Submit").Bytes("namespace", preparedNs).Msg("RPC call failed")
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

	// Compress blobs first if compression is enabled
	blobsToSubmit := inputBlobs
	if api.compressionEnabled && len(inputBlobs) > 0 {
		compressed, err := compression.CompressBatch(inputBlobs)
		if err != nil {
			api.Logger.Error().Err(err).Msg("Failed to compress blobs")
			return nil, fmt.Errorf("failed to compress blobs: %w", err)
		}

		// Log compression stats
		var totalOriginal, totalCompressed uint64
		for i, blob := range compressed {
			info := compression.GetCompressionInfo(blob)
			if info.IsCompressed {
				totalOriginal += info.OriginalSize
				totalCompressed += info.CompressedSize
				api.Logger.Debug().
					Int("blob_index", i).
					Uint64("original_size", info.OriginalSize).
					Uint64("compressed_size", info.CompressedSize).
					Float64("ratio", info.CompressionRatio).
					Msg("Blob compression stats")
			}
		}

		if totalOriginal > 0 {
			savings := float64(totalOriginal-totalCompressed) / float64(totalOriginal) * 100
			api.Logger.Info().
				Uint64("total_original", totalOriginal).
				Uint64("total_compressed", totalCompressed).
				Float64("savings_percent", savings).
				Msg("Compression summary")
		}

		blobsToSubmit = compressed
	}

	// Validate each blob individually and calculate total size
	var totalSize uint64
	for i, blob := range blobsToSubmit {
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

	preparedNs := da.PrepareNamespace(ns)
	api.Logger.Debug().Str("method", "SubmitWithOptions").Int("num_blobs", len(blobsToSubmit)).Uint64("total_size", totalSize).Float64("gas_price", gasPrice).Str("namespace", hex.EncodeToString(preparedNs)).Msg("Making RPC call")
	res, err := api.Internal.SubmitWithOptions(ctx, blobsToSubmit, gasPrice, preparedNs, options)
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

// ClientOptions contains configuration options for the client
type ClientOptions struct {
	// Compression settings
	CompressionEnabled  bool
	CompressionLevel    int     // 1-22, default 3
	MinCompressionRatio float64 // Minimum compression ratio to store compressed, default 0.1
}

// DefaultClientOptions returns default client options with compression enabled
func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		CompressionEnabled:  true,
		CompressionLevel:    compression.DefaultZstdLevel,
		MinCompressionRatio: compression.DefaultMinCompressionRatio,
	}
}

// NewClient creates a new Client with one connection per namespace with the
// given token as the authorization token.
func NewClient(ctx context.Context, logger zerolog.Logger, addr, token string, gasPrice, gasMultiplier float64) (*Client, error) {
	authHeader := http.Header{"Authorization": []string{fmt.Sprintf("Bearer %s", token)}}
	return newClient(ctx, logger, addr, authHeader, gasPrice, gasMultiplier, DefaultClientOptions())
}

// NewClientWithOptions creates a new Client with custom options
func NewClientWithOptions(ctx context.Context, logger zerolog.Logger, addr, token string, gasPrice, gasMultiplier float64, opts ClientOptions) (*Client, error) {
	authHeader := http.Header{"Authorization": []string{fmt.Sprintf("Bearer %s", token)}}
	return newClient(ctx, logger, addr, authHeader, gasPrice, gasMultiplier, opts)
}

func newClient(ctx context.Context, logger zerolog.Logger, addr string, authHeader http.Header, gasPrice, gasMultiplier float64, opts ClientOptions) (*Client, error) {
	var multiCloser multiClientCloser
	var client Client
	client.DA.Logger = logger
	client.DA.MaxBlobSize = uint64(internal.MaxTxSize)
	client.DA.gasPrice = gasPrice
	client.DA.gasMultiplier = gasMultiplier

	// Set compression configuration
	client.DA.compressionEnabled = opts.CompressionEnabled
	client.DA.compressionConfig = compression.Config{
		Enabled:             opts.CompressionEnabled,
		ZstdLevel:           opts.CompressionLevel,
		MinCompressionRatio: opts.MinCompressionRatio,
	}

	if opts.CompressionEnabled {
		logger.Info().
			Bool("compression", opts.CompressionEnabled).
			Int("level", opts.CompressionLevel).
			Float64("min_ratio", opts.MinCompressionRatio).
			Msg("Compression enabled for JSONRPC client")
	}

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
