package celestia

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/da"
)

// defaultRetrieveTimeout is the default timeout for DA retrieval operations
const defaultRetrieveTimeout = 10 * time.Second

// retrieveBatchSize is the number of blobs to retrieve in a single batch
const retrieveBatchSize = 100

// Client connects to celestia-node's blob API via JSON-RPC and implements the da.DA interface.
type Client struct {
	logger      zerolog.Logger
	maxBlobSize uint64
	closer      jsonrpc.ClientCloser

	Internal struct {
		Submit   func(ctx context.Context, blobs []*Blob, opts *SubmitOptions) (uint64, error)                    `perm:"write"`
		Get      func(ctx context.Context, height uint64, ns Namespace, c Commitment) (*Blob, error)              `perm:"read"`
		GetAll   func(ctx context.Context, height uint64, namespaces []Namespace) ([]*Blob, error)                `perm:"read"`
		GetProof func(ctx context.Context, height uint64, ns Namespace, c Commitment) (*Proof, error)             `perm:"read"`
		Included func(ctx context.Context, height uint64, ns Namespace, proof *Proof, c Commitment) (bool, error) `perm:"read"`
	}
}

// NewClient creates a new client connected to celestia-node that implements the da.DA interface.
// Token is obtained from: celestia light auth write
func NewClient(
	ctx context.Context,
	logger zerolog.Logger,
	addr string,
	token string,
	maxBlobSize uint64,
) (*Client, error) {
	if addr == "" {
		return nil, fmt.Errorf("address cannot be empty")
	}

	if maxBlobSize == 0 {
		return nil, fmt.Errorf("maxBlobSize must be greater than 0")
	}

	client := &Client{
		logger:      logger,
		maxBlobSize: maxBlobSize,
	}

	authHeader := http.Header{}
	if token != "" {
		authHeader.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	closer, err := jsonrpc.NewMergeClient(
		ctx,
		addr,
		"blob",
		[]interface{}{&client.Internal},
		authHeader,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create JSON-RPC client: %w", err)
	}

	client.closer = closer

	logger.Info().
		Str("address", addr).
		Uint64("max_blob_size", maxBlobSize).
		Msg("Celestia blob API client created successfully")

	return client, nil
}

// Close closes the connection. Safe to call multiple times.
func (c *Client) Close() {
	if c.closer != nil {
		c.closer()
		c.closer = nil
	}
	c.logger.Debug().Msg("Celestia client connection closed")
}

// submit is a private method that submits blobs and returns the height (used internally).
func (c *Client) submit(ctx context.Context, blobs []*Blob, opts *SubmitOptions) (uint64, error) {
	c.logger.Debug().
		Int("num_blobs", len(blobs)).
		Msg("Submitting blobs to Celestia")

	height, err := c.Internal.Submit(ctx, blobs, opts)
	if err != nil {
		c.logger.Error().
			Err(err).
			Int("num_blobs", len(blobs)).
			Msg("Failed to submit blobs")
		return 0, fmt.Errorf("failed to submit blobs: %w", err)
	}

	c.logger.Info().
		Uint64("height", height).
		Int("num_blobs", len(blobs)).
		Msg("Successfully submitted blobs")

	return height, nil
}

// get retrieves a single blob by commitment at a given height and namespace (used internally).
func (c *Client) get(ctx context.Context, height uint64, namespace Namespace, commitment Commitment) (*Blob, error) {
	c.logger.Debug().
		Uint64("height", height).
		Msg("Getting blob from Celestia")

	blob, err := c.Internal.Get(ctx, height, namespace, commitment)
	if err != nil {
		c.logger.Error().
			Err(err).
			Uint64("height", height).
			Msg("Failed to get blob")
		return nil, fmt.Errorf("failed to get blob: %w", err)
	}

	c.logger.Debug().
		Uint64("height", height).
		Int("data_size", len(blob.Data)).
		Msg("Successfully retrieved blob")

	return blob, nil
}

// getAll retrieves all blobs at a given height for the specified namespaces (used internally).
func (c *Client) getAll(ctx context.Context, height uint64, namespaces []Namespace) ([]*Blob, error) {
	c.logger.Debug().
		Uint64("height", height).
		Int("num_namespaces", len(namespaces)).
		Msg("Getting all blobs from Celestia")

	blobs, err := c.Internal.GetAll(ctx, height, namespaces)
	if err != nil {
		c.logger.Error().
			Err(err).
			Uint64("height", height).
			Int("num_namespaces", len(namespaces)).
			Msg("Failed to get blobs")
		return nil, fmt.Errorf("failed to get blobs: %w", err)
	}

	c.logger.Debug().
		Uint64("height", height).
		Int("num_blobs", len(blobs)).
		Msg("Successfully retrieved blobs")

	return blobs, nil
}

// getProof retrieves the inclusion proof for a blob (used internally).
func (c *Client) getProof(ctx context.Context, height uint64, namespace Namespace, commitment Commitment) (*Proof, error) {
	c.logger.Debug().
		Uint64("height", height).
		Msg("Getting proof from Celestia")

	proof, err := c.Internal.GetProof(ctx, height, namespace, commitment)
	if err != nil {
		c.logger.Error().
			Err(err).
			Uint64("height", height).
			Msg("Failed to get proof")
		return nil, fmt.Errorf("failed to get proof: %w", err)
	}

	c.logger.Debug().
		Uint64("height", height).
		Int("proof_size", len(proof.Data)).
		Msg("Successfully retrieved proof")

	return proof, nil
}

// included checks whether a blob is included in the Celestia block (used internally).
func (c *Client) included(ctx context.Context, height uint64, namespace Namespace, proof *Proof, commitment Commitment) (bool, error) {
	c.logger.Debug().
		Uint64("height", height).
		Msg("Checking blob inclusion in Celestia")

	included, err := c.Internal.Included(ctx, height, namespace, proof, commitment)
	if err != nil {
		c.logger.Error().
			Err(err).
			Uint64("height", height).
			Msg("Failed to check inclusion")
		return false, fmt.Errorf("failed to check inclusion: %w", err)
	}

	c.logger.Debug().
		Uint64("height", height).
		Bool("included", included).
		Msg("Inclusion check completed")

	return included, nil
}

// DA interface implementation

func (c *Client) Submit(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte) da.ResultSubmit {
	return c.SubmitWithOptions(ctx, blobs, gasPrice, namespace, nil)
}

// Get retrieves blobs by their IDs.
func (c *Client) Get(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Blob, error) {
	if len(ids) == 0 {
		return []da.Blob{}, nil
	}

	// Group IDs by height for efficient retrieval
	type blobKey struct {
		height     uint64
		commitment string
	}
	heightGroups := make(map[uint64][]Commitment)
	idToIndex := make(map[blobKey]int)

	for i, id := range ids {
		height, commitment, err := da.SplitID(id)
		if err != nil {
			return nil, fmt.Errorf("invalid ID at index %d: %w", i, err)
		}
		heightGroups[height] = append(heightGroups[height], commitment)
		idToIndex[blobKey{height, string(commitment)}] = i
	}

	// Retrieve blobs for each height
	result := make([]da.Blob, len(ids))
	for height := range heightGroups {
		blobs, err := c.getAll(ctx, height, []Namespace{namespace})
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return nil, da.ErrBlobNotFound
			}
			return nil, fmt.Errorf("failed to get blobs at height %d: %w", height, err)
		}

		// Match blobs to their original positions
		for _, blob := range blobs {
			key := blobKey{height, string(blob.Commitment)}
			if idx, ok := idToIndex[key]; ok {
				result[idx] = blob.Data
			}
		}
	}

	return result, nil
}

func (c *Client) GetIDs(ctx context.Context, height uint64, namespace []byte) (*da.GetIDsResult, error) {
	result := c.Retrieve(ctx, height, namespace)
	if result.Code != da.StatusSuccess {
		return nil, da.StatusCodeToError(result.Code, result.Message)
	}
	return &da.GetIDsResult{
		IDs:       result.IDs,
		Timestamp: result.Timestamp,
	}, nil
}

// GetProofs retrieves inclusion proofs for the given IDs.
func (c *Client) GetProofs(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Proof, error) {
	if len(ids) == 0 {
		return []da.Proof{}, nil
	}

	proofs := make([]da.Proof, len(ids))
	for i, id := range ids {
		height, commitment, err := da.SplitID(id)
		if err != nil {
			return nil, fmt.Errorf("invalid ID at index %d: %w", i, err)
		}

		proof, err := c.getProof(ctx, height, namespace, commitment)
		if err != nil {
			return nil, fmt.Errorf("failed to get proof for ID %d: %w", i, err)
		}

		proofs[i] = proof.Data
	}

	return proofs, nil
}

// Commit creates commitments for the given blobs.
// Commitments are computed locally using the same algorithm as celestia-node.
func (c *Client) Commit(ctx context.Context, blobs []da.Blob, namespace []byte) ([]da.Commitment, error) {
	commitments := make([]da.Commitment, len(blobs))
	for i, blob := range blobs {
		commitment, err := CreateCommitment(blob, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to create commitment for blob %d: %w", i, err)
		}
		commitments[i] = commitment
	}
	return commitments, nil
}

// Validate validates commitments against proofs.
func (c *Client) Validate(ctx context.Context, ids []da.ID, proofs []da.Proof, namespace []byte) ([]bool, error) {
	if len(ids) != len(proofs) {
		return nil, fmt.Errorf("mismatched lengths: %d IDs vs %d proofs", len(ids), len(proofs))
	}

	results := make([]bool, len(ids))
	for i, id := range ids {
		height, commitment, err := da.SplitID(id)
		if err != nil {
			return nil, fmt.Errorf("invalid ID at index %d: %w", i, err)
		}

		proof := &Proof{Data: proofs[i]}
		included, err := c.included(ctx, height, namespace, proof, commitment)
		if err != nil {
			return nil, fmt.Errorf("failed to validate proof %d: %w", i, err)
		}

		results[i] = included
	}

	return results, nil
}

// makeID creates an ID from a height and a commitment.
func makeID(height uint64, commitment []byte) []byte {
	id := make([]byte, len(commitment)+8)
	binary.LittleEndian.PutUint64(id, height)
	copy(id[8:], commitment)
	return id
}

func (c *Client) SubmitWithOptions(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte, options []byte) da.ResultSubmit {
	var blobSize uint64
	for _, blob := range blobs {
		blobSize += uint64(len(blob))
	}

	if len(blobs) == 0 {
		return da.ResultSubmit{
			BaseResult: da.BaseResult{
				Code:      da.StatusSuccess,
				IDs:       []da.ID{},
				Timestamp: time.Now(),
			},
		}
	}

	if err := ValidateNamespace(namespace); err != nil {
		return da.ResultSubmit{
			BaseResult: da.BaseResult{
				Code:     da.StatusError,
				Message:  fmt.Sprintf("invalid namespace: %s", err.Error()),
				BlobSize: blobSize,
			},
		}
	}

	celestiaBlobs := make([]*Blob, len(blobs))
	for i, blob := range blobs {
		commitment, err := CreateCommitment(blob, namespace)
		if err != nil {
			return da.ResultSubmit{
				BaseResult: da.BaseResult{
					Code:     da.StatusError,
					Message:  fmt.Sprintf("failed to create commitment for blob %d: %s", i, err.Error()),
					BlobSize: blobSize,
				},
			}
		}
		celestiaBlobs[i] = &Blob{
			Namespace:  namespace,
			Data:       blob,
			Commitment: commitment,
		}
	}

	var opts *SubmitOptions
	if len(options) > 0 {
		opts = &SubmitOptions{}
		if err := json.Unmarshal(options, opts); err != nil {
			return da.ResultSubmit{
				BaseResult: da.BaseResult{
					Code:     da.StatusError,
					Message:  fmt.Sprintf("failed to unmarshal submit options: %s", err.Error()),
					BlobSize: blobSize,
				},
			}
		}
		opts.Fee = gasPrice
	} else {
		opts = &SubmitOptions{Fee: gasPrice}
	}

	height, err := c.submit(ctx, celestiaBlobs, opts)
	if err != nil {
		status := da.StatusError
		errStr := err.Error()

		switch {
		case errors.Is(err, context.Canceled):
			status = da.StatusContextCanceled
		case errors.Is(err, context.DeadlineExceeded):
			status = da.StatusContextDeadline
		case strings.Contains(errStr, "timeout"):
			status = da.StatusNotIncludedInBlock
		case strings.Contains(errStr, "too large") || strings.Contains(errStr, "exceeds"):
			status = da.StatusTooBig
		case strings.Contains(errStr, "already in mempool"):
			status = da.StatusAlreadyInMempool
		case strings.Contains(errStr, "incorrect account sequence"):
			status = da.StatusIncorrectAccountSequence
		}

		if status == da.StatusTooBig {
			c.logger.Debug().Err(err).Uint64("status", uint64(status)).Msg("DA submission failed")
		} else {
			c.logger.Error().Err(err).Uint64("status", uint64(status)).Msg("DA submission failed")
		}

		return da.ResultSubmit{
			BaseResult: da.BaseResult{
				Code:      status,
				Message:   "failed to submit blobs: " + err.Error(),
				BlobSize:  blobSize,
				Timestamp: time.Now(),
			},
		}
	}

	ids := make([]da.ID, len(celestiaBlobs))
	for i, blob := range celestiaBlobs {
		ids[i] = makeID(height, blob.Commitment)
	}

	c.logger.Debug().Int("num_ids", len(ids)).Uint64("height", height).Msg("DA submission successful")
	return da.ResultSubmit{
		BaseResult: da.BaseResult{
			Code:           da.StatusSuccess,
			IDs:            ids,
			SubmittedCount: uint64(len(ids)),
			Height:         height,
			BlobSize:       blobSize,
			Timestamp:      time.Now(),
		},
	}
}

func (c *Client) Retrieve(ctx context.Context, height uint64, namespace []byte) da.ResultRetrieve {
	getCtx, cancel := context.WithTimeout(ctx, defaultRetrieveTimeout)
	defer cancel()

	blobs, err := c.getAll(getCtx, height, []Namespace{namespace})
	if err != nil {
		errStr := err.Error()

		if strings.Contains(errStr, "not found") {
			c.logger.Debug().Uint64("height", height).Msg("Blobs not found at height")
			return da.ResultRetrieve{
				BaseResult: da.BaseResult{
					Code:      da.StatusNotFound,
					Message:   da.ErrBlobNotFound.Error(),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		}
		if strings.Contains(errStr, "height") && strings.Contains(errStr, "future") {
			c.logger.Debug().Uint64("height", height).Msg("Height is from the future")
			return da.ResultRetrieve{
				BaseResult: da.BaseResult{
					Code:      da.StatusHeightFromFuture,
					Message:   da.ErrHeightFromFuture.Error(),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		}

		c.logger.Error().Uint64("height", height).Err(err).Msg("Failed to retrieve blobs")
		return da.ResultRetrieve{
			BaseResult: da.BaseResult{
				Code:      da.StatusError,
				Message:   fmt.Sprintf("failed to retrieve blobs: %s", err.Error()),
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}

	if len(blobs) == 0 {
		c.logger.Debug().Uint64("height", height).Msg("No blobs found at height")
		return da.ResultRetrieve{
			BaseResult: da.BaseResult{
				Code:      da.StatusNotFound,
				Message:   da.ErrBlobNotFound.Error(),
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}

	ids := make([]da.ID, len(blobs))
	data := make([][]byte, len(blobs))
	for i, blob := range blobs {
		ids[i] = makeID(height, blob.Commitment)
		data[i] = blob.Data
	}

	c.logger.Debug().Uint64("height", height).Int("num_blobs", len(blobs)).Msg("Successfully retrieved blobs")
	return da.ResultRetrieve{
		BaseResult: da.BaseResult{
			Code:      da.StatusSuccess,
			Height:    height,
			IDs:       ids,
			Timestamp: time.Now(),
		},
		Data: data,
	}
}
