package datypes

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"
)

// DA defines the data-availability interface using the shared datypes types.
type DA interface {
	Get(ctx context.Context, ids []ID, namespace []byte) ([]Blob, error)
	GetIDs(ctx context.Context, height uint64, namespace []byte) (*GetIDsResult, error)
	GetProofs(ctx context.Context, ids []ID, namespace []byte) ([]Proof, error)
	Commit(ctx context.Context, blobs []Blob, namespace []byte) ([]Commitment, error)
	Submit(ctx context.Context, blobs []Blob, gasPrice float64, namespace []byte) ([]ID, error)
	SubmitWithOptions(ctx context.Context, blobs []Blob, gasPrice float64, namespace []byte, options []byte) ([]ID, error)
	Validate(ctx context.Context, ids []ID, proofs []Proof, namespace []byte) ([]bool, error)
}

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

var (
	_ DA = (*DummyDA)(nil)

	errHeightFromFutureStr = fmt.Errorf("%w", ErrHeightFromFuture)
)

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
			return nil, ErrBlobNotFound
		}
		blobs = append(blobs, blob)
	}
	return blobs, nil
}

// GetIDs returns IDs of all blobs at the given height.
func (d *DummyDA) GetIDs(ctx context.Context, height uint64, namespace []byte) (*GetIDsResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if height > d.currentHeight {
		return nil, fmt.Errorf("%w: requested %d, current %d", errHeightFromFutureStr, height, d.currentHeight)
	}

	ids, exists := d.blobsByHeight[height]
	if !exists {
		return &GetIDsResult{IDs: []ID{}, Timestamp: time.Now()}, nil
	}

	filteredIDs := make([]ID, 0, len(ids))
	for _, id := range ids {
		idStr := string(id)
		if ns, ok := d.namespaceByID[idStr]; ok && bytes.Equal(ns, namespace) {
			filteredIDs = append(filteredIDs, id)
		}
	}

	return &GetIDsResult{IDs: filteredIDs, Timestamp: d.timestampsByHeight[height]}, nil
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
		commitments = append(commitments, blob)
	}
	return commitments, nil
}

// Submit submits blobs to the DA layer.
func (d *DummyDA) Submit(ctx context.Context, blobs []Blob, gasPrice float64, namespace []byte) ([]ID, error) {
	return d.SubmitWithOptions(ctx, blobs, gasPrice, namespace, nil)
}

// SetSubmitFailure simulates DA layer going down by making Submit calls fail
func (d *DummyDA) SetSubmitFailure(shouldFail bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.submitShouldFail = shouldFail
}

// SubmitWithOptions submits blobs to the DA layer with additional options.
func (d *DummyDA) SubmitWithOptions(ctx context.Context, blobs []Blob, gasPrice float64, namespace []byte, options []byte) ([]ID, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.submitShouldFail {
		return nil, errors.New("simulated DA layer failure")
	}

	height := d.currentHeight + 1
	ids := make([]ID, 0, len(blobs))
	var currentSize uint64

	for _, blob := range blobs {
		blobLen := uint64(len(blob))
		if blobLen > d.maxBlobSize {
			return nil, ErrBlobSizeOverLimit
		}

		if currentSize+blobLen > d.maxBlobSize {
			break
		}
		currentSize += blobLen

		bz := sha256.Sum256(blob)
		commitment := bz[:]

		id := makeID(height, commitment)
		idStr := string(id)

		d.blobs[idStr] = blob
		d.commitments[idStr] = commitment
		d.proofs[idStr] = commitment
		d.namespaceByID[idStr] = namespace

		ids = append(ids, id)
	}

	if existingIDs, exists := d.blobsByHeight[height]; exists {
		d.blobsByHeight[height] = append(existingIDs, ids...)
	} else {
		d.blobsByHeight[height] = ids
	}
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

// makeID creates an ID from a height and a commitment.
func makeID(height uint64, commitment []byte) []byte {
	id := make([]byte, len(commitment)+8)
	binary.LittleEndian.PutUint64(id, height)
	copy(id[8:], commitment)
	return id
}
