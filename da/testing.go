package da

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"
)

var _ DA = (*DummyDA)(nil)

// DummyDA is a simple in-memory implementation of the DA interface for testing purposes.
type DummyDA struct {
	mu                 sync.RWMutex
	blobs              map[string]Blob
	commitments        map[string]Commitment
	proofs             map[string]Proof
	blobsByHeight      map[uint64][]ID
	timestampsByHeight map[uint64]time.Time
	namespaceByID      map[string][]byte // Track namespace for each blob ID
	maxBlobSize        uint64

	// DA height simulation
	currentHeight uint64
	blockTime     time.Duration
	stopCh        chan struct{}

	// Simulated failure support
	submitShouldFail bool
}

var ErrHeightFromFutureStr = fmt.Errorf("given height is from the future")

// NewDummyDA creates a new instance of DummyDA with the specified maximum blob size and block time.
func NewDummyDA(maxBlobSize uint64, blockTime time.Duration) *DummyDA {
	return &DummyDA{
		blobs:              make(map[string]Blob),
		commitments:        make(map[string]Commitment),
		proofs:             make(map[string]Proof),
		blobsByHeight:      make(map[uint64][]ID),
		timestampsByHeight: make(map[uint64]time.Time),
		namespaceByID:      make(map[string][]byte),
		maxBlobSize:        maxBlobSize,
		blockTime:          blockTime,
		stopCh:             make(chan struct{}),
		currentHeight:      0,
	}
}

// StartHeightTicker starts a goroutine that increments currentHeight every blockTime.
func (d *DummyDA) StartHeightTicker() {
	go func() {
		ticker := time.NewTicker(d.blockTime)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				d.mu.Lock()
				d.currentHeight++
				d.mu.Unlock()
			case <-d.stopCh:
				return
			}
		}
	}()
}

// StopHeightTicker stops the height ticker goroutine.
func (d *DummyDA) StopHeightTicker() {
	close(d.stopCh)
}

// Get returns blobs for the given IDs.
func (d *DummyDA) Get(ctx context.Context, ids []ID, namespace []byte) ([]Blob, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	blobs := make([]Blob, 0, len(ids))
	for _, id := range ids {
		blob, exists := d.blobs[string(id)]
		if !exists {
			return nil, ErrBlobNotFound // Use the specific error type
		}
		blobs = append(blobs, blob)
	}
	return blobs, nil
}

// GetIDs returns IDs of all blobs at the given height.
// Delegates to Retrieve.
func (d *DummyDA) GetIDs(ctx context.Context, height uint64, namespace []byte) (*GetIDsResult, error) {
	result := d.Retrieve(ctx, height, namespace)
	if result.Code != StatusSuccess {
		return nil, StatusCodeToError(result.Code, result.Message)
	}
	return &GetIDsResult{
		IDs:       result.IDs,
		Timestamp: result.Timestamp,
	}, nil
}

// GetProofs returns proofs for the given IDs.
func (d *DummyDA) GetProofs(ctx context.Context, ids []ID, namespace []byte) ([]Proof, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	proofs := make([]Proof, 0, len(ids))
	for _, id := range ids {
		proof, exists := d.proofs[string(id)]
		if !exists {
			return nil, errors.New("proof not found")
		}
		proofs = append(proofs, proof)
	}
	return proofs, nil
}

// Commit creates commitments for the given blobs.
func (d *DummyDA) Commit(ctx context.Context, blobs []Blob, namespace []byte) ([]Commitment, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	commitments := make([]Commitment, 0, len(blobs))
	for _, blob := range blobs {
		// For simplicity, we use the blob itself as the commitment
		commitment := blob
		commitments = append(commitments, commitment)
	}
	return commitments, nil
}

// Submit submits blobs to the DA layer and returns a structured result.
// Delegates to SubmitWithOptions with nil options.
func (d *DummyDA) Submit(ctx context.Context, blobs []Blob, gasPrice float64, namespace []byte) ResultSubmit {
	return d.SubmitWithOptions(ctx, blobs, gasPrice, namespace, nil)
}

// SetSubmitFailure simulates DA layer going down by making Submit calls fail
func (d *DummyDA) SetSubmitFailure(shouldFail bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.submitShouldFail = shouldFail
}

// Validate validates commitments against proofs.
func (d *DummyDA) Validate(ctx context.Context, ids []ID, proofs []Proof, namespace []byte) ([]bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(ids) != len(proofs) {
		return nil, errors.New("number of IDs and proofs must match")
	}

	results := make([]bool, len(ids))
	for i, id := range ids {
		_, exists := d.blobs[string(id)]
		results[i] = exists
	}

	return results, nil
}

// SubmitWithOptions submits blobs to the DA layer with additional options and returns a structured result.
// This is the primary implementation - Submit delegates to this method.
func (d *DummyDA) SubmitWithOptions(ctx context.Context, blobs []Blob, gasPrice float64, namespace []byte, options []byte) ResultSubmit {
	// Calculate blob size upfront
	var blobSize uint64
	for _, blob := range blobs {
		blobSize += uint64(len(blob))
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if we should simulate failure
	if d.submitShouldFail {
		return ResultSubmit{
			BaseResult: BaseResult{
				Code:     StatusError,
				Message:  "simulated DA layer failure",
				BlobSize: blobSize,
			},
		}
	}

	height := d.currentHeight + 1
	ids := make([]ID, 0, len(blobs))
	var currentSize uint64

	for _, blob := range blobs {
		blobLen := uint64(len(blob))
		// Check individual blob size first
		if blobLen > d.maxBlobSize {
			return ResultSubmit{
				BaseResult: BaseResult{
					Code:     StatusTooBig,
					Message:  "failed to submit blobs: " + ErrBlobSizeOverLimit.Error(),
					BlobSize: blobSize,
				},
			}
		}

		// Check cumulative batch size
		if currentSize+blobLen > d.maxBlobSize {
			// Stop processing blobs for this batch, return IDs collected so far
			break
		}
		currentSize += blobLen

		// Create a commitment using SHA-256 hash
		bz := sha256.Sum256(blob)
		commitment := bz[:]

		// Create ID from height and commitment
		id := makeID(height, commitment)
		idStr := string(id)

		d.blobs[idStr] = blob
		d.commitments[idStr] = commitment
		d.proofs[idStr] = commitment       // Simple proof
		d.namespaceByID[idStr] = namespace // Store namespace for this blob

		ids = append(ids, id)
	}

	// Add the IDs to the blobsByHeight map if they don't already exist
	if existingIDs, exists := d.blobsByHeight[height]; exists {
		d.blobsByHeight[height] = append(existingIDs, ids...)
	} else {
		d.blobsByHeight[height] = ids
	}
	d.timestampsByHeight[height] = time.Now()

	return ResultSubmit{
		BaseResult: BaseResult{
			Code:           StatusSuccess,
			IDs:            ids,
			SubmittedCount: uint64(len(ids)),
			Height:         height,
			BlobSize:       blobSize,
			Timestamp:      time.Now(),
		},
	}
}

// Retrieve retrieves all blobs at the given height and returns a structured result.
// This is the primary implementation - GetIDs delegates to this method.
func (d *DummyDA) Retrieve(ctx context.Context, height uint64, namespace []byte) ResultRetrieve {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Check height bounds
	if height > d.currentHeight {
		return ResultRetrieve{
			BaseResult: BaseResult{
				Code:      StatusHeightFromFuture,
				Message:   ErrHeightFromFuture.Error(),
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}

	// Get IDs at height
	ids, exists := d.blobsByHeight[height]
	if !exists {
		return ResultRetrieve{
			BaseResult: BaseResult{
				Code:      StatusNotFound,
				Message:   ErrBlobNotFound.Error(),
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}

	// Filter IDs by namespace and collect blobs
	filteredIDs := make([]ID, 0)
	blobs := make([]Blob, 0)
	for _, id := range ids {
		if ns, nsExists := d.namespaceByID[string(id)]; nsExists && bytes.Equal(ns, namespace) {
			filteredIDs = append(filteredIDs, id)
			if blob, blobExists := d.blobs[string(id)]; blobExists {
				blobs = append(blobs, blob)
			}
		}
	}

	// Handle empty result after namespace filtering
	if len(filteredIDs) == 0 {
		return ResultRetrieve{
			BaseResult: BaseResult{
				Code:      StatusNotFound,
				Message:   ErrBlobNotFound.Error(),
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}

	return ResultRetrieve{
		BaseResult: BaseResult{
			Code:      StatusSuccess,
			Height:    height,
			IDs:       filteredIDs,
			Timestamp: d.timestampsByHeight[height],
		},
		Data: blobs,
	}
}
