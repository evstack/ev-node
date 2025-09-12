package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/rs/zerolog"
)

// DefaultMaxBlobSize is the default max blob size
const DefaultMaxBlobSize uint64 = 64 * 64 * 481 // 1970176

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

var _ coreda.DA = &LocalDA{}

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

// GasMultiplier returns the gas multiplier.
func (d *LocalDA) GasMultiplier(ctx context.Context) (float64, error) {
	d.logger.Debug().Msg("GasMultiplier called")
	return 1.0, nil
}

// GasPrice returns the gas price.
func (d *LocalDA) GasPrice(ctx context.Context) (float64, error) {
	d.logger.Debug().Msg("GasPrice called")
	return 0.0, nil
}

// Get returns Blobs for given IDs.
func (d *LocalDA) Get(ctx context.Context, ids []coreda.ID, ns []byte) ([]coreda.Blob, error) {
	if err := validateNamespace(ns); err != nil {
		d.logger.Error().Err(err).Msg("Get: invalid namespace")
		return nil, err
	}
	d.logger.Debug().Interface("ids", ids).Msg("Get called")
	d.mu.Lock()
	defer d.mu.Unlock()
	blobs := make([]coreda.Blob, len(ids))
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
			return nil, coreda.ErrBlobNotFound
		}
	}
	d.logger.Debug().Int("count", len(blobs)).Msg("Get successful")
	return blobs, nil
}

// GetIDs returns IDs of Blobs at given DA height.
func (d *LocalDA) GetIDs(ctx context.Context, height uint64, ns []byte) (*coreda.GetIDsResult, error) {
	if err := validateNamespace(ns); err != nil {
		d.logger.Error().Err(err).Msg("GetIDs: invalid namespace")
		return nil, err
	}
	d.logger.Debug().Uint64("height", height).Msg("GetIDs called")
	d.mu.Lock()
	defer d.mu.Unlock()

	if height > d.height {
		d.logger.Error().Uint64("requested", height).Uint64("current", d.height).Msg("GetIDs: height in future")
		return nil, fmt.Errorf("height %d is in the future: %w", height, coreda.ErrHeightFromFuture)
	}

	kvps, ok := d.data[height]
	if !ok {
		d.logger.Debug().Uint64("height", height).Msg("GetIDs: no data for height")
		return nil, nil
	}

	ids := make([]coreda.ID, len(kvps))
	for i, kv := range kvps {
		ids[i] = kv.key
	}
	d.logger.Debug().Int("count", len(ids)).Msg("GetIDs successful")
	return &coreda.GetIDsResult{IDs: ids, Timestamp: d.timestamps[height]}, nil
}

// GetProofs returns inclusion Proofs for all Blobs located in DA at given height.
func (d *LocalDA) GetProofs(ctx context.Context, ids []coreda.ID, ns []byte) ([]coreda.Proof, error) {
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
	proofs := make([]coreda.Proof, len(blobs))
	for i, blob := range blobs {
		proofs[i] = d.getProof(ids[i], blob)
	}
	d.logger.Debug().Int("count", len(proofs)).Msg("GetProofs successful")
	return proofs, nil
}

// Commit returns cryptographic Commitments for given blobs.
func (d *LocalDA) Commit(ctx context.Context, blobs []coreda.Blob, ns []byte) ([]coreda.Commitment, error) {
	if err := validateNamespace(ns); err != nil {
		d.logger.Error().Err(err).Msg("Commit: invalid namespace")
		return nil, err
	}
	d.logger.Debug().Int("numBlobs", len(blobs)).Msg("Commit called")
	commits := make([]coreda.Commitment, len(blobs))
	for i, blob := range blobs {
		commits[i] = d.getHash(blob)
	}
	d.logger.Debug().Int("count", len(commits)).Msg("Commit successful")
	return commits, nil
}

// SubmitWithOptions stores blobs in DA layer (options are ignored).
func (d *LocalDA) SubmitWithOptions(ctx context.Context, blobs []coreda.Blob, gasPrice float64, ns []byte, _ []byte) ([]coreda.ID, error) {
	if err := validateNamespace(ns); err != nil {
		d.logger.Error().Err(err).Msg("SubmitWithOptions: invalid namespace")
		return nil, err
	}
	d.logger.Info().Int("numBlobs", len(blobs)).Float64("gasPrice", gasPrice).Str("namespace", hex.EncodeToString(ns)).Msg("SubmitWithOptions called")

	// Validate blob sizes before processing
	for i, blob := range blobs {
		if uint64(len(blob)) > d.maxBlobSize {
			d.logger.Error().Int("blobIndex", i).Int("blobSize", len(blob)).Uint64("maxBlobSize", d.maxBlobSize).Msg("SubmitWithOptions: blob size exceeds limit")
			return nil, coreda.ErrBlobSizeOverLimit
		}
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	ids := make([]coreda.ID, len(blobs))
	d.height += 1
	d.timestamps[d.height] = time.Now()
	for i, blob := range blobs {
		ids[i] = append(d.nextID(), d.getHash(blob)...)

		d.data[d.height] = append(d.data[d.height], kvp{ids[i], blob})
	}
	d.logger.Info().Uint64("newHeight", d.height).Int("count", len(ids)).Msg("SubmitWithOptions successful")
	return ids, nil
}

// Submit stores blobs in DA layer (options are ignored).
func (d *LocalDA) Submit(ctx context.Context, blobs []coreda.Blob, gasPrice float64, ns []byte) ([]coreda.ID, error) {
	if err := validateNamespace(ns); err != nil {
		d.logger.Error().Err(err).Msg("Submit: invalid namespace")
		return nil, err
	}
	d.logger.Info().Int("numBlobs", len(blobs)).Float64("gasPrice", gasPrice).Str("namespace", string(ns)).Msg("Submit called")

	// Validate blob sizes before processing
	for i, blob := range blobs {
		if uint64(len(blob)) > d.maxBlobSize {
			d.logger.Error().Int("blobIndex", i).Int("blobSize", len(blob)).Uint64("maxBlobSize", d.maxBlobSize).Msg("Submit: blob size exceeds limit")
			return nil, coreda.ErrBlobSizeOverLimit
		}
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	ids := make([]coreda.ID, len(blobs))
	d.height += 1
	d.timestamps[d.height] = time.Now()
	for i, blob := range blobs {
		ids[i] = append(d.nextID(), d.getHash(blob)...)

		d.data[d.height] = append(d.data[d.height], kvp{ids[i], blob})
	}
	d.logger.Info().Uint64("newHeight", d.height).Int("count", len(ids)).Msg("Submit successful")
	return ids, nil
}

// Validate checks the Proofs for given IDs.
func (d *LocalDA) Validate(ctx context.Context, ids []coreda.ID, proofs []coreda.Proof, ns []byte) ([]bool, error) {
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
