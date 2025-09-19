package da

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Implemented in accordance to https://celestiaorg.github.io/celestia-app/namespace.html

const (
	// NamespaceVersionIndex is the index of the namespace version in the byte slice
	NamespaceVersionIndex = 0
	// NamespaceVersionSize is the size of the namespace version in bytes
	NamespaceVersionSize = 1
	// NamespaceIDSize is the size of the namespace ID in bytes
	NamespaceIDSize = 28
	// NamespaceSize is the total size of a namespace (version + ID) in bytes
	NamespaceSize = NamespaceVersionSize + NamespaceIDSize

	// NamespaceVersionZero is the only supported user-specifiable namespace version
	NamespaceVersionZero = uint8(0)
	// NamespaceVersionMax is the max namespace version
	NamespaceVersionMax = uint8(255)

	// NamespaceVersionZeroPrefixSize is the number of leading zero bytes required for version 0
	NamespaceVersionZeroPrefixSize = 18
	// NamespaceVersionZeroDataSize is the number of data bytes available for version 0
	NamespaceVersionZeroDataSize = 10
)

// Namespace represents a Celestia namespace
type Namespace struct {
	Version uint8
	ID      [NamespaceIDSize]byte
}

// Bytes returns the namespace as a byte slice
func (n Namespace) Bytes() []byte {
	result := make([]byte, NamespaceSize)
	result[NamespaceVersionIndex] = n.Version
	copy(result[NamespaceVersionSize:], n.ID[:])
	return result
}

// IsValidForVersion0 checks if the namespace is valid for version 0
// Version 0 requires the first 18 bytes of the ID to be zero
func (n Namespace) IsValidForVersion0() bool {
	if n.Version != NamespaceVersionZero {
		return false
	}

	for i := range NamespaceVersionZeroPrefixSize {
		if n.ID[i] != 0 {
			return false
		}
	}
	return true
}

// NewNamespaceV0 creates a new version 0 namespace from the provided data
// The data should be up to 10 bytes and will be placed in the last 10 bytes of the ID
// The first 18 bytes will be zeros as required by the specification
func NewNamespaceV0(data []byte) (*Namespace, error) {
	if len(data) > NamespaceVersionZeroDataSize {
		return nil, fmt.Errorf("data too long for version 0 namespace: got %d bytes, max %d",
			len(data), NamespaceVersionZeroDataSize)
	}

	ns := &Namespace{
		Version: NamespaceVersionZero,
	}

	// The first 18 bytes are already zero (Go zero-initializes)
	// Copy the data to the last 10 bytes
	copy(ns.ID[NamespaceVersionZeroPrefixSize:], data)

	return ns, nil
}

// NamespaceFromBytes creates a namespace from a 29-byte slice
func NamespaceFromBytes(b []byte) (*Namespace, error) {
	if len(b) != NamespaceSize {
		return nil, fmt.Errorf("invalid namespace size: expected %d, got %d", NamespaceSize, len(b))
	}

	ns := &Namespace{
		Version: b[NamespaceVersionIndex],
	}
	copy(ns.ID[:], b[NamespaceVersionSize:])

	// Validate if it's version 0
	if ns.Version == NamespaceVersionZero && !ns.IsValidForVersion0() {
		return nil, fmt.Errorf("invalid version 0 namespace: first %d bytes of ID must be zero",
			NamespaceVersionZeroPrefixSize)
	}

	return ns, nil
}

// NamespaceFromString creates a version 0 namespace from a string identifier
// The string is hashed and the first 10 bytes of the hash are used as the namespace data
func NamespaceFromString(s string) *Namespace {
	// Hash the string to get consistent bytes
	hash := sha256.Sum256([]byte(s))

	// Use the first 10 bytes of the hash for the namespace data
	ns, _ := NewNamespaceV0(hash[:NamespaceVersionZeroDataSize])
	return ns
}

// HexString returns the hex representation of the namespace
func (n Namespace) HexString() string {
	return "0x" + hex.EncodeToString(n.Bytes())
}

// ParseHexNamespace parses a hex string into a namespace
func ParseHexNamespace(hexStr string) (*Namespace, error) {
	// Remove 0x prefix if present
	hexStr = strings.TrimPrefix(hexStr, "0x")

	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid hex string: %w", err)
	}

	return NamespaceFromBytes(b)
}
