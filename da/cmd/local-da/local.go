package main

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/celestiaorg/go-square/v3/share"
	"github.com/rs/zerolog"

	proxy "github.com/evstack/ev-node/da/jsonrpc"
	"github.com/evstack/ev-node/pkg/blob"
)

// DefaultMaxBlobSize is the default max blob size
const DefaultMaxBlobSize uint64 = 2 * 1024 * 1024 // 2MB

// LocalDA is a simple in-memory implementation of the blob API used for testing.
type LocalDA struct {
	mu          *sync.RWMutex
	blobs       map[uint64][]*blob.Blob
	timestamps  map[uint64]time.Time
	maxBlobSize uint64
	height      uint64
	logger      zerolog.Logger
}

// NewLocalDA creates a new LocalDA instance.
func NewLocalDA(logger zerolog.Logger, opts ...func(*LocalDA) *LocalDA) *LocalDA {
	da := &LocalDA{
		mu:          new(sync.RWMutex),
		blobs:       make(map[uint64][]*blob.Blob),
		timestamps:  make(map[uint64]time.Time),
		maxBlobSize: DefaultMaxBlobSize,
		logger:      logger,
	}
	for _, f := range opts {
		da = f(da)
	}
	da.logger.Info().Msg("NewLocalDA: initialized LocalDA")
	return da
}

// Ensure LocalDA satisfies the blob API used by the JSON-RPC server.
var _ proxy.BlobAPI = (*LocalDA)(nil)

// Submit stores blobs at a new height and returns that height.
func (d *LocalDA) Submit(ctx context.Context, blobs []*blob.Blob, opts *blob.SubmitOptions) (uint64, error) {
	for i, b := range blobs {
		if uint64(len(b.Data())) > d.maxBlobSize {
			d.logger.Error().Int("blobIndex", i).Int("blobSize", len(b.Data())).Uint64("maxBlobSize", d.maxBlobSize).Msg("Submit: blob size exceeds limit")
			return 0, fmt.Errorf("blob size exceeds limit: %d > %d", len(b.Data()), d.maxBlobSize)
		}
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.height++
	height := d.height
	d.timestamps[height] = time.Now()
	for _, b := range blobs {
		d.blobs[height] = append(d.blobs[height], cloneBlob(b))
	}
	d.logger.Info().Uint64("height", height).Int("count", len(blobs)).Msg("Submit successful")
	return height, nil
}

// GetAll returns blobs stored at the provided height filtered by namespaces.
func (d *LocalDA) GetAll(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blob.Blob, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Return error if requesting a height from the future
	if height > d.height {
		return nil, fmt.Errorf("height %d is in the future (current: %d): given height is from the future", height, d.height)
	}

	entries, ok := d.blobs[height]
	if !ok {
		return nil, nil
	}

	if len(namespaces) == 0 {
		return cloneBlobs(entries), nil
	}

	var filtered []*blob.Blob
	for _, b := range entries {
		if matchesNamespace(b.Namespace(), namespaces) {
			filtered = append(filtered, cloneBlob(b))
		}
	}
	return filtered, nil
}

// GetProof returns a placeholder proof if the blob exists at the height.
func (d *LocalDA) GetProof(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Proof, error) {
	if !d.hasBlob(height, namespace, commitment) {
		return nil, fmt.Errorf("blob not found")
	}
	proof := blob.Proof{}
	return &proof, nil
}

// Included verifies that the blob exists at the height. The proof parameter is ignored.
func (d *LocalDA) Included(ctx context.Context, height uint64, namespace share.Namespace, proof *blob.Proof, commitment blob.Commitment) (bool, error) {
	return d.hasBlob(height, namespace, commitment), nil
}

func (d *LocalDA) hasBlob(height uint64, namespace share.Namespace, commitment blob.Commitment) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	entries, ok := d.blobs[height]
	if !ok {
		return false
	}

	for _, b := range entries {
		if bytes.Equal(b.Namespace().Bytes(), namespace.Bytes()) && bytes.Equal(b.Commitment, commitment) {
			return true
		}
	}
	return false
}

func matchesNamespace(ns share.Namespace, targets []share.Namespace) bool {
	for _, target := range targets {
		if bytes.Equal(ns.Bytes(), target.Bytes()) {
			return true
		}
	}
	return false
}

func cloneBlobs(blobs []*blob.Blob) []*blob.Blob {
	out := make([]*blob.Blob, len(blobs))
	for i, b := range blobs {
		out[i] = cloneBlob(b)
	}
	return out
}

func cloneBlob(b *blob.Blob) *blob.Blob {
	if b == nil {
		return nil
	}
	clone := *b
	return &clone
}

// WithMaxBlobSize returns a function that sets the max blob size of LocalDA
func WithMaxBlobSize(maxBlobSize uint64) func(*LocalDA) *LocalDA {
	return func(da *LocalDA) *LocalDA {
		da.maxBlobSize = maxBlobSize
		return da
	}
}
