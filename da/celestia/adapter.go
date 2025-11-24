package celestia

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/evstack/ev-node/da"
	"github.com/rs/zerolog"
)

// Adapter wraps the Celestia client to implement the da.DA interface.
// This is a temporary bridge to allow ev-node to use the native Celestia blob API
// while maintaining compatibility with the existing DA abstraction.
type Adapter struct {
	client      *Client
	logger      zerolog.Logger
	maxBlobSize uint64
}

// NewAdapter creates a new adapter that implements da.DA interface.
func NewAdapter(
	ctx context.Context,
	logger zerolog.Logger,
	addr string,
	token string,
	maxBlobSize uint64,
) (*Adapter, error) {
	client, err := NewClient(ctx, logger, addr, token, maxBlobSize)
	if err != nil {
		return nil, err
	}

	return &Adapter{
		client:      client,
		logger:      logger,
		maxBlobSize: maxBlobSize,
	}, nil
}

// Close closes the underlying client connection.
func (a *Adapter) Close() {
	a.client.Close()
}

// Submit submits blobs to Celestia and returns IDs.
func (a *Adapter) Submit(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte) ([]da.ID, error) {
	return a.SubmitWithOptions(ctx, blobs, gasPrice, namespace, nil)
}

// SubmitWithOptions submits blobs to Celestia with additional options.
func (a *Adapter) SubmitWithOptions(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte, options []byte) ([]da.ID, error) {
	if len(blobs) == 0 {
		return []da.ID{}, nil
	}

	// Validate namespace
	if err := ValidateNamespace(namespace); err != nil {
		return nil, fmt.Errorf("invalid namespace: %w", err)
	}

	// Convert blobs to Celestia format
	celestiaBlobs := make([]*Blob, len(blobs))
	for i, blob := range blobs {
		celestiaBlobs[i] = &Blob{
			Namespace: namespace,
			Data:      blob,
		}
	}

	// Parse submit options if provided
	var opts *SubmitOptions
	if len(options) > 0 {
		opts = &SubmitOptions{}
		if err := json.Unmarshal(options, opts); err != nil {
			return nil, fmt.Errorf("failed to unmarshal submit options: %w", err)
		}
		opts.Fee = gasPrice
	} else {
		opts = &SubmitOptions{Fee: gasPrice}
	}

	height, err := a.client.Submit(ctx, celestiaBlobs, opts)
	if err != nil {
		if strings.Contains(err.Error(), "timeout") {
			return nil, da.ErrTxTimedOut
		}
		if strings.Contains(err.Error(), "too large") || strings.Contains(err.Error(), "exceeds") {
			return nil, da.ErrBlobSizeOverLimit
		}
		return nil, err
	}

	// Create IDs from height and commitments
	ids := make([]da.ID, len(celestiaBlobs))
	for i, blob := range celestiaBlobs {
		ids[i] = makeID(height, blob.Commitment)
	}

	return ids, nil
}

// Get retrieves blobs by their IDs.
func (a *Adapter) Get(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Blob, error) {
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
		height, commitment, err := splitID(id)
		if err != nil {
			return nil, fmt.Errorf("invalid ID at index %d: %w", i, err)
		}
		heightGroups[height] = append(heightGroups[height], commitment)
		idToIndex[blobKey{height, string(commitment)}] = i
	}

	// Retrieve blobs for each height
	result := make([]da.Blob, len(ids))
	for height := range heightGroups {
		blobs, err := a.client.GetAll(ctx, height, []Namespace{namespace})
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

// GetIDs returns all blob IDs at the given height.
func (a *Adapter) GetIDs(ctx context.Context, height uint64, namespace []byte) (*da.GetIDsResult, error) {
	blobs, err := a.client.GetAll(ctx, height, []Namespace{namespace})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, da.ErrBlobNotFound
		}
		if strings.Contains(err.Error(), "height") && strings.Contains(err.Error(), "future") {
			return nil, da.ErrHeightFromFuture
		}
		return nil, err
	}

	if len(blobs) == 0 {
		return nil, da.ErrBlobNotFound
	}

	ids := make([]da.ID, len(blobs))
	for i, blob := range blobs {
		ids[i] = makeID(height, blob.Commitment)
	}

	return &da.GetIDsResult{
		IDs:       ids,
		Timestamp: time.Now(),
	}, nil
}

// GetProofs retrieves inclusion proofs for the given IDs.
func (a *Adapter) GetProofs(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Proof, error) {
	if len(ids) == 0 {
		return []da.Proof{}, nil
	}

	proofs := make([]da.Proof, len(ids))
	for i, id := range ids {
		height, commitment, err := splitID(id)
		if err != nil {
			return nil, fmt.Errorf("invalid ID at index %d: %w", i, err)
		}

		proof, err := a.client.GetProof(ctx, height, namespace, commitment)
		if err != nil {
			return nil, fmt.Errorf("failed to get proof for ID %d: %w", i, err)
		}

		proofs[i] = proof.Data
	}

	return proofs, nil
}

// Commit creates commitments for the given blobs.
// Note: Celestia generates commitments automatically during submission,
// so this is a no-op that returns nil commitments.
func (a *Adapter) Commit(ctx context.Context, blobs []da.Blob, namespace []byte) ([]da.Commitment, error) {
	commitments := make([]da.Commitment, len(blobs))
	for i := range blobs {
		commitments[i] = nil
	}
	return commitments, nil
}

// Validate validates commitments against proofs.
func (a *Adapter) Validate(ctx context.Context, ids []da.ID, proofs []da.Proof, namespace []byte) ([]bool, error) {
	if len(ids) != len(proofs) {
		return nil, fmt.Errorf("mismatched lengths: %d IDs vs %d proofs", len(ids), len(proofs))
	}

	results := make([]bool, len(ids))
	for i, id := range ids {
		height, commitment, err := splitID(id)
		if err != nil {
			return nil, fmt.Errorf("invalid ID at index %d: %w", i, err)
		}

		proof := &Proof{Data: proofs[i]}
		included, err := a.client.Included(ctx, height, namespace, proof, commitment)
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

// splitID splits an ID into a height and a commitment.
func splitID(id []byte) (uint64, []byte, error) {
	if len(id) <= 8 {
		return 0, nil, fmt.Errorf("invalid ID length: %d", len(id))
	}
	commitment := id[8:]
	return binary.LittleEndian.Uint64(id[:8]), commitment, nil
}
