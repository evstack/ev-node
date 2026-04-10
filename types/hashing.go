package types

import (
	"crypto/sha256"
	"errors"
	"hash"
	"sync"
	"unsafe"

	"google.golang.org/protobuf/proto"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

var (
	leafPrefix = []byte{0}

	// sha256Pool reuses sha256 Hash instances to avoid per-block allocation.
	// sha256.New() allocates ~213 bytes (216B on 64-bit) per call. Pooling
	// eliminates this allocation entirely in the hot path.
	sha256Pool = sync.Pool{
		New: func() any {
			return sha256.New()
		},
	}
)

// HashSlim returns the SHA256 hash of the header using the slim (current) binary encoding.
func (h *Header) HashSlim() (Hash, error) {
	if h == nil {
		return nil, errors.New("header is nil")
	}

	bytes, err := h.MarshalBinary()
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256(bytes)
	return hash[:], nil
}

// HashLegacy returns the SHA256 hash of the header using the legacy binary encoding that
// includes the deprecated fields.
func (h *Header) HashLegacy() (Hash, error) {
	if h == nil {
		return nil, errors.New("header is nil")
	}

	bytes, err := h.MarshalBinaryLegacy()
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256(bytes)
	return hash[:], nil
}

// Hash returns the header hash. It reuses a memoized value if one has already
// been prepared via MemoizeHash, but it does not write to the header itself.
func (h *Header) Hash() Hash {
	if h == nil {
		return nil
	}
	if h.cachedHash != nil {
		return h.cachedHash
	}

	return h.computeHash()
}

// MemoizeHash computes the header hash and stores it on the header for future
// Hash() calls. Call this before publishing the header to shared goroutines or
// caches.
//
// If a Header struct is reused (e.g. overwritten via FromProto or field
// assignment), call InvalidateHash() first to clear the cached value before
// calling MemoizeHash again. Failure to do so will return the stale cached hash.
func (h *Header) MemoizeHash() Hash {
	if h == nil {
		return nil
	}
	if h.cachedHash != nil {
		return h.cachedHash
	}

	hash := h.computeHash()
	if hash != nil {
		h.cachedHash = hash
	}
	return hash
}

func (h *Header) computeHash() Hash {
	// Legacy hash takes precedence when legacy fields are present (backwards
	// compatibility). Slim hash is the canonical hash for all other headers.
	if h.Legacy != nil && !h.Legacy.IsZero() {
		if legacyHash, err := h.HashLegacy(); err == nil {
			return legacyHash
		}
	}

	slimHash, err := h.HashSlim()
	if err != nil {
		return nil
	}
	return slimHash
}

// InvalidateHash clears the memoized hash, forcing recomputation on the next
// Hash() call. Must be called after any mutation of Header fields.
func (h *Header) InvalidateHash() {
	if h != nil {
		h.cachedHash = nil
	}
}

// Hash returns hash of the Data
func (d *Data) Hash() Hash {
	// Ignoring the marshal error for now to satisfy the go-header interface
	// Later on the usage of Hash should be replaced with DA commitment
	dBytes, _ := d.MarshalBinary()
	s := sha256Pool.Get().(hash.Hash)
	defer sha256Pool.Put(s)
	return leafHashOpt(s, dBytes)
}

// DACommitment returns the DA commitment of the Data excluding the Metadata.
func (d *Data) DACommitment() Hash {
	pbData := pb.Data{Txs: unsafe.Slice((*[]byte)(unsafe.SliceData(d.Txs)), len(d.Txs))}
	dBytes, _ := proto.Marshal(&pbData)
	s := sha256Pool.Get().(hash.Hash)
	defer sha256Pool.Put(s)
	return leafHashOpt(s, dBytes)
}

func leafHashOpt(s hash.Hash, leaf []byte) []byte {
	s.Reset()
	s.Write(leafPrefix)
	s.Write(leaf)
	return s.Sum(nil)
}
