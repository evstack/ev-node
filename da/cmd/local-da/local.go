package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	da "github.com/evstack/ev-node/da"
	"github.com/rs/zerolog"
)

// DefaultMaxBlobSize is the default max blob size
const DefaultMaxBlobSize uint64 = 2 * 1024 * 1024 // 2MB

// LocalDA is a simple implementation of in-memory DA. Not production ready! Intended only for testing!
//
// Data is stored in a map, where key is a serialized sequence number. This key is returned as ID.
// Commitments are simply hashes, and proofs are ED25519 signatures.
type LocalDA struct {
	mu          *sync.Mutex // protects data and height
	data        map[uint64][]kvp
	timestamps  map[uint64]time.Time
	maxBlobSize uint64
	height      uint64
	privKey     ed25519.PrivateKey
	pubKey      ed25519.PublicKey

	logger zerolog.Logger
}

type kvp struct {
	key, value []byte
}

// NewLocalDA create new instance of DummyDA
func NewLocalDA(logger zerolog.Logger, opts ...func(*LocalDA) *LocalDA) *LocalDA {
	da := &LocalDA{
		mu:          new(sync.Mutex),
		data:        make(map[uint64][]kvp),
		timestamps:  make(map[uint64]time.Time),
		maxBlobSize: DefaultMaxBlobSize,
		logger:      logger,
	}
	for _, f := range opts {
		da = f(da)
	}
	da.pubKey, da.privKey, _ = ed25519.GenerateKey(rand.Reader)
	da.logger.Info().Msg("NewLocalDA: initialized LocalDA")
	return da
}

var _ da.DA = &LocalDA{}

// validateNamespace checks that namespace is exactly 29 bytes
func validateNamespace(ns []byte) error {
	if len(ns) != 29 {
		return fmt.Errorf("namespace must be exactly 29 bytes, got %d", len(ns))
	}
	return nil
}

// MaxBlobSize returns the max blob size in bytes.
func (d *LocalDA) MaxBlobSize(ctx context.Context) (uint64, error) {
	d.logger.Debug().Uint64("maxBlobSize", d.maxBlobSize).Msg("MaxBlobSize called")
	return d.maxBlobSize, nil
}

// Get returns Blobs for given IDs.
func (d *LocalDA) Get(ctx context.Context, ids []da.ID, ns []byte) ([]da.Blob, error) {
	if err := validateNamespace(ns); err != nil {
		d.logger.Error().Err(err).Msg("Get: invalid namespace")
		return nil, err
	}
	d.logger.Debug().Interface("ids", ids).Msg("Get called")
	d.mu.Lock()
	defer d.mu.Unlock()
	blobs := make([]da.Blob, len(ids))
	for i, id := range ids {
		if len(id) < 8 {
			d.logger.Error().Interface("id", id).Msg("Get: invalid ID length")
			return nil, errors.New("invalid ID")
		}
		height := binary.LittleEndian.Uint64(id)
		found := false
		for j := 0; !found && j < len(d.data[height]); j++ {
			if bytes.Equal(d.data[height][j].key, id) {
				blobs[i] = d.data[height][j].value
				found = true
			}
		}
		if !found {
			d.logger.Warn().Interface("id", id).Uint64("height", height).Msg("Get: blob not found")
			return nil, da.ErrBlobNotFound
		}
	}
	d.logger.Debug().Int("count", len(blobs)).Msg("Get successful")
	return blobs, nil
}

// GetIDs returns IDs of Blobs at given DA height.
// Delegates to Retrieve.
func (d *LocalDA) GetIDs(ctx context.Context, height uint64, ns []byte) (*da.GetIDsResult, error) {
	result := d.Retrieve(ctx, height, ns)
	if result.Code != da.StatusSuccess {
		return nil, da.StatusCodeToError(result.Code, result.Message)
	}
	return &da.GetIDsResult{
		IDs:       result.IDs,
		Timestamp: result.Timestamp,
	}, nil
}

// GetProofs returns inclusion Proofs for all Blobs located in DA at given height.
func (d *LocalDA) GetProofs(ctx context.Context, ids []da.ID, ns []byte) ([]da.Proof, error) {
	if err := validateNamespace(ns); err != nil {
		d.logger.Error().Err(err).Msg("GetProofs: invalid namespace")
		return nil, err
	}
	d.logger.Debug().Interface("ids", ids).Msg("GetProofs called")
	blobs, err := d.Get(ctx, ids, ns)
	if err != nil {
		d.logger.Error().Err(err).Msg("GetProofs: failed to get blobs")
		return nil, err
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	proofs := make([]da.Proof, len(blobs))
	for i, blob := range blobs {
		proofs[i] = d.getProof(ids[i], blob)
	}
	d.logger.Debug().Int("count", len(proofs)).Msg("GetProofs successful")
	return proofs, nil
}

// Commit returns cryptographic Commitments for given blobs.
func (d *LocalDA) Commit(ctx context.Context, blobs []da.Blob, ns []byte) ([]da.Commitment, error) {
	if err := validateNamespace(ns); err != nil {
		d.logger.Error().Err(err).Msg("Commit: invalid namespace")
		return nil, err
	}
	d.logger.Debug().Int("numBlobs", len(blobs)).Msg("Commit called")
	commits := make([]da.Commitment, len(blobs))
	for i, blob := range blobs {
		commits[i] = d.getHash(blob)
	}
	d.logger.Debug().Int("count", len(commits)).Msg("Commit successful")
	return commits, nil
}

// Submit stores blobs in DA layer and returns a structured result.
// Delegates to SubmitWithOptions with nil options.
func (d *LocalDA) Submit(ctx context.Context, blobs []da.Blob, gasPrice float64, ns []byte) da.ResultSubmit {
	return d.SubmitWithOptions(ctx, blobs, gasPrice, ns, nil)
}

// Validate checks the Proofs for given IDs.
func (d *LocalDA) Validate(ctx context.Context, ids []da.ID, proofs []da.Proof, ns []byte) ([]bool, error) {
	if err := validateNamespace(ns); err != nil {
		d.logger.Error().Err(err).Msg("Validate: invalid namespace")
		return nil, err
	}
	d.logger.Debug().Int("numIDs", len(ids)).Int("numProofs", len(proofs)).Msg("Validate called")
	if len(ids) != len(proofs) {
		d.logger.Error().Int("ids", len(ids)).Int("proofs", len(proofs)).Msg("Validate: id/proof count mismatch")
		return nil, errors.New("number of IDs doesn't equal to number of proofs")
	}
	results := make([]bool, len(ids))
	for i := 0; i < len(ids); i++ {
		results[i] = ed25519.Verify(d.pubKey, ids[i][8:], proofs[i])
		d.logger.Debug().Interface("id", ids[i]).Bool("result", results[i]).Msg("Validate result")
	}
	d.logger.Debug().Interface("results", results).Msg("Validate finished")
	return results, nil
}

func (d *LocalDA) nextID() []byte {
	return d.getID(d.height)
}

func (d *LocalDA) getID(cnt uint64) []byte {
	id := make([]byte, 8)
	binary.LittleEndian.PutUint64(id, cnt)
	return id
}

func (d *LocalDA) getHash(blob []byte) []byte {
	sha := sha256.Sum256(blob)
	return sha[:]
}

func (d *LocalDA) getProof(id, blob []byte) []byte {
	sign := ed25519.Sign(d.privKey, d.getHash(blob))
	return sign
}

// WithMaxBlobSize returns a function that sets the max blob size of LocalDA
func WithMaxBlobSize(maxBlobSize uint64) func(*LocalDA) *LocalDA {
	return func(da *LocalDA) *LocalDA {
		da.maxBlobSize = maxBlobSize
		return da
	}
}

// SubmitWithOptions stores blobs in DA layer with additional options and returns a structured result.
// This is the primary implementation - Submit delegates to this method.
func (d *LocalDA) SubmitWithOptions(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte, options []byte) da.ResultSubmit {
	// Calculate blob size upfront
	var blobSize uint64
	for _, blob := range blobs {
		blobSize += uint64(len(blob))
	}

	// Validate namespace
	if err := validateNamespace(namespace); err != nil {
		d.logger.Error().Err(err).Msg("SubmitWithResult: invalid namespace")
		return da.ResultSubmit{
			BaseResult: da.BaseResult{
				Code:     da.StatusError,
				Message:  err.Error(),
				BlobSize: blobSize,
			},
		}
	}

	// Validate blob sizes before processing
	for i, blob := range blobs {
		if uint64(len(blob)) > d.maxBlobSize {
			d.logger.Error().Int("blobIndex", i).Int("blobSize", len(blob)).Uint64("maxBlobSize", d.maxBlobSize).Msg("SubmitWithResult: blob size exceeds limit")
			return da.ResultSubmit{
				BaseResult: da.BaseResult{
					Code:     da.StatusTooBig,
					Message:  "failed to submit blobs: " + da.ErrBlobSizeOverLimit.Error(),
					BlobSize: blobSize,
				},
			}
		}
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	ids := make([]da.ID, len(blobs))
	d.height++
	d.timestamps[d.height] = time.Now()
	for i, blob := range blobs {
		ids[i] = append(d.nextID(), d.getHash(blob)...)
		d.data[d.height] = append(d.data[d.height], kvp{ids[i], blob})
	}

	d.logger.Debug().Int("num_ids", len(ids)).Uint64("height", d.height).Msg("DA submission successful")
	return da.ResultSubmit{
		BaseResult: da.BaseResult{
			Code:           da.StatusSuccess,
			IDs:            ids,
			SubmittedCount: uint64(len(ids)),
			Height:         d.height,
			BlobSize:       blobSize,
			Timestamp:      time.Now(),
		},
	}
}

// Retrieve retrieves all blobs at the given height and returns a structured result.
// This is the primary implementation - GetIDs delegates to this method.
func (d *LocalDA) Retrieve(ctx context.Context, height uint64, namespace []byte) da.ResultRetrieve {
	// Validate namespace
	if err := validateNamespace(namespace); err != nil {
		d.logger.Error().Err(err).Msg("Retrieve: invalid namespace")
		return da.ResultRetrieve{
			BaseResult: da.BaseResult{
				Code:      da.StatusError,
				Message:   err.Error(),
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Check height bounds
	if height > d.height {
		d.logger.Error().Uint64("requested", height).Uint64("current", d.height).Msg("Retrieve: height in future")
		return da.ResultRetrieve{
			BaseResult: da.BaseResult{
				Code:      da.StatusHeightFromFuture,
				Message:   da.ErrHeightFromFuture.Error(),
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}

	// Get data at height
	kvps, ok := d.data[height]
	if !ok || len(kvps) == 0 {
		d.logger.Debug().Uint64("height", height).Msg("Retrieve: no data for height")
		return da.ResultRetrieve{
			BaseResult: da.BaseResult{
				Code:      da.StatusNotFound,
				Message:   da.ErrBlobNotFound.Error(),
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}

	// Extract IDs and blobs
	ids := make([]da.ID, len(kvps))
	blobs := make([][]byte, len(kvps))
	for i, kv := range kvps {
		ids[i] = kv.key
		blobs[i] = kv.value
	}

	d.logger.Debug().Uint64("height", height).Int("num_blobs", len(blobs)).Msg("Successfully retrieved blobs")
	return da.ResultRetrieve{
		BaseResult: da.BaseResult{
			Code:      da.StatusSuccess,
			Height:    height,
			IDs:       ids,
			Timestamp: d.timestamps[height],
		},
		Data: blobs,
	}
}
