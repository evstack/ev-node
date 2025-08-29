package types

import (
	"crypto/sha256"
	"hash"
)

var (
	leafPrefix = []byte{0}
)

// Hash returns hash of the header
func (h *Header) Hash() Hash {
	bytes, err := h.MarshalBinary()
	if err != nil {
		return nil
	}
	hash := sha256.Sum256(bytes)
	return hash[:]
}

// Hash returns hash of the Data
func (d *Data) Hash() Hash {
	// Check for cached hash with read lock
	d.hashMutex.RLock()
	if d.cachedHash != nil {
		hash := d.cachedHash
		d.hashMutex.RUnlock()
		return hash
	}
	d.hashMutex.RUnlock()
	
	// Need to compute hash, acquire write lock
	d.hashMutex.Lock()
	defer d.hashMutex.Unlock()
	
	// Double-check in case another goroutine computed it while we were waiting
	if d.cachedHash != nil {
		return d.cachedHash
	}
	
	// Ignoring the marshal error for now to satisfy the go-header interface
	// Later on the usage of Hash should be replaced with DA commitment
	dBytes, _ := d.MarshalBinary()
	d.cachedHash = leafHashOpt(sha256.New(), dBytes)
	return d.cachedHash
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
