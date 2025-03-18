package da

import (
	"context"
	"crypto/sha256"
	"errors"
	"sync"
	"time"
)

// DummyDA is a simple in-memory implementation of the DA interface for testing purposes.
type DummyDA struct {
	mu                 sync.RWMutex
	blobs              map[string]Blob
	commitments        map[string]Commitment
	proofs             map[string]Proof
	blobsByHeight      map[uint64][]ID
	timestampsByHeight map[uint64]time.Time
	maxBlobSize        uint64
	gasPrice           float64
	gasMultiplier      float64
}

// NewDummyDA creates a new instance of DummyDA with the specified maximum blob size.
func NewDummyDA(maxBlobSize uint64, gasPrice float64, gasMultiplier float64) *DummyDA {
	return &DummyDA{
		blobs:              make(map[string]Blob),
		commitments:        make(map[string]Commitment),
		proofs:             make(map[string]Proof),
		blobsByHeight:      make(map[uint64][]ID),
		timestampsByHeight: make(map[uint64]time.Time),
		maxBlobSize:        maxBlobSize,
		gasPrice:           gasPrice,
		gasMultiplier:      gasMultiplier,
	}
}

// MaxBlobSize returns the maximum blob size.
func (d *DummyDA) MaxBlobSize(ctx context.Context) (uint64, error) {
	return d.maxBlobSize, nil
}

// GasPrice returns the gas price for the DA layer.
func (d *DummyDA) GasPrice(ctx context.Context) (float64, error) {
	return d.gasPrice, nil
}

// GasMultiplier returns the gas multiplier for the DA layer.
func (d *DummyDA) GasMultiplier(ctx context.Context) (float64, error) {
	return d.gasMultiplier, nil
}

// Get returns blobs for the given IDs.
func (d *DummyDA) Get(ctx context.Context, ids []ID, namespace []byte) ([]Blob, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	blobs := make([]Blob, 0, len(ids))
	for _, id := range ids {
		blob, exists := d.blobs[string(id)]
		if !exists {
			return nil, errors.New("blob not found")
		}
		blobs = append(blobs, blob)
	}
	return blobs, nil
}

// GetIDs returns IDs of all blobs at the given height.
func (d *DummyDA) GetIDs(ctx context.Context, height uint64, namespace []byte) (*GetIDsResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ids, exists := d.blobsByHeight[height]
	if !exists {
		return &GetIDsResult{
			IDs:       []ID{},
			Timestamp: time.Now(),
		}, nil
	}

	return &GetIDsResult{
		IDs:       ids,
		Timestamp: d.timestampsByHeight[height],
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

// Submit submits blobs to the DA layer with additional options.
func (d *DummyDA) Submit(ctx context.Context, blobs []Blob, gasPrice float64, namespace []byte, options []byte) ([]ID, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	height := uint64(len(d.blobsByHeight))
	ids := make([]ID, 0, len(blobs))

	for _, blob := range blobs {
		if uint64(len(blob)) > d.maxBlobSize {
			return nil, errors.New("blob size exceeds maximum")
		}

		// Create a commitment using SHA-256 hash
		bz := sha256.Sum256(blob)
		commitment := bz[:]

		// Create ID from height and commitment
		id := makeID(height, commitment)
		idStr := string(id)

		d.blobs[idStr] = blob
		d.commitments[idStr] = commitment
		d.proofs[idStr] = commitment // Simple proof

		ids = append(ids, id)
	}

	d.blobsByHeight[height] = ids
	d.timestampsByHeight[height] = time.Now()

	return ids, nil
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
