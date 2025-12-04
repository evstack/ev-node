package datypes

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	NamespaceVersionIndex          = 0
	NamespaceVersionSize           = 1
	NamespaceIDSize                = 28
	NamespaceSize                  = NamespaceVersionSize + NamespaceIDSize
	NamespaceVersionZero           = uint8(0)
	NamespaceVersionMax            = uint8(255)
	NamespaceVersionZeroPrefixSize = 18
	NamespaceVersionZeroDataSize   = 10
)

// Namespace mirrors Celestia namespace layout (version + 28-byte ID).
type Namespace struct {
	Version uint8
	ID      [NamespaceIDSize]byte
}

// Bytes returns the namespace as a byte slice.
func (n Namespace) Bytes() []byte {
	result := make([]byte, NamespaceSize)
	result[NamespaceVersionIndex] = n.Version
	copy(result[NamespaceVersionSize:], n.ID[:])
	return result
}

// IsValidForVersion0 validates version-0 namespace rules (first 18 bytes zero).
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

// NewNamespaceV0 builds a version-0 namespace from up to 10 bytes of data.
func NewNamespaceV0(data []byte) (*Namespace, error) {
	if len(data) > NamespaceVersionZeroDataSize {
		return nil, fmt.Errorf("data too long for version 0 namespace: got %d bytes, max %d", len(data), NamespaceVersionZeroDataSize)
	}

	ns := &Namespace{Version: NamespaceVersionZero}
	copy(ns.ID[NamespaceVersionZeroPrefixSize:], data)
	return ns, nil
}

// NamespaceFromBytes parses a namespace from its byte representation.
func NamespaceFromBytes(b []byte) (*Namespace, error) {
	if len(b) != NamespaceSize {
		return nil, fmt.Errorf("invalid namespace size: expected %d, got %d", NamespaceSize, len(b))
	}

	ns := &Namespace{Version: b[NamespaceVersionIndex]}
	copy(ns.ID[:], b[NamespaceVersionSize:])

	if ns.Version == NamespaceVersionZero && !ns.IsValidForVersion0() {
		return nil, fmt.Errorf("invalid version 0 namespace: first %d bytes of ID must be zero", NamespaceVersionZeroPrefixSize)
	}

	return ns, nil
}

// NamespaceFromString deterministically builds a version-0 namespace from a string.
func NamespaceFromString(s string) *Namespace {
	hash := sha256.Sum256([]byte(s))
	ns, _ := NewNamespaceV0(hash[:NamespaceVersionZeroDataSize])
	return ns
}

// HexString returns the 0x-prefixed hex encoding of the namespace bytes.
func (n Namespace) HexString() string {
	return "0x" + hex.EncodeToString(n.Bytes())
}

// ParseHexNamespace parses a hex string (with or without 0x) into a namespace.
func ParseHexNamespace(hexStr string) (*Namespace, error) {
	hexStr = strings.TrimPrefix(hexStr, "0x")

	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid hex string: %w", err)
	}

	return NamespaceFromBytes(b)
}
