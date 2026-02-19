package types

import (
	"crypto/sha256"
	"errors"
	"hash"
)

var (
	leafPrefix = []byte{0}
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

// Hash returns hash of the header
func (h *Header) Hash() Hash {
	if h == nil {
		return nil
	}

	slimHash, err := h.HashSlim()
	if err != nil {
		return nil
	}

	if h.Legacy != nil && !h.Legacy.IsZero() {
		legacyHash, err := h.HashLegacy()
		if err == nil {
			return legacyHash
		}
	}

	return slimHash
}

// Hash returns hash of the Data
func (d *Data) Hash() Hash {
	// Ignoring the marshal error for now to satisfy the go-header interface
	// Later on the usage of Hash should be replaced with DA commitment
	dBytes, _ := d.MarshalBinary()
	return leafHashOpt(sha256.New(), dBytes)
}

// DACommitment returns the DA commitment of the Data excluding the Metadata
func (d *Data) DACommitment() Hash {
	// Prune the Data to only include the Txs
	prunedData := &Data{
		Txs: d.Txs,
	}
	dBytes, _ := prunedData.MarshalBinary()
	return leafHashOpt(sha256.New(), dBytes)
}

func leafHashOpt(s hash.Hash, leaf []byte) []byte {
	s.Reset()
	s.Write(leafPrefix)
	s.Write(leaf)
	return s.Sum(nil)
}
