package da

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamespaceV0Creation(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectError bool
		description string
	}{
		{
			name:        "valid 10 byte data",
			data:        []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			expectError: false,
			description: "Should create valid namespace with 10 bytes of data",
		},
		{
			name:        "valid 5 byte data",
			data:        []byte{1, 2, 3, 4, 5},
			expectError: false,
			description: "Should create valid namespace with 5 bytes of data (padded with zeros)",
		},
		{
			name:        "empty data",
			data:        []byte{},
			expectError: false,
			description: "Should create valid namespace with empty data (all zeros)",
		},
		{
			name:        "data too long",
			data:        []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
			expectError: true,
			description: "Should fail with data longer than 10 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns, err := NewNamespaceV0(tt.data)
			
			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Nil(t, ns)
			} else {
				require.NoError(t, err, tt.description)
				require.NotNil(t, ns)
				
				// Verify version is 0
				assert.Equal(t, NamespaceVersionZero, ns.Version, "Version should be 0")
				
				// Verify first 18 bytes of ID are zeros
				for i := 0; i < NamespaceVersionZeroPrefixSize; i++ {
					assert.Equal(t, byte(0), ns.ID[i], "First 18 bytes should be zero")
				}
				
				// Verify data is in the last 10 bytes
				expectedData := make([]byte, NamespaceVersionZeroDataSize)
				copy(expectedData, tt.data)
				actualData := ns.ID[NamespaceVersionZeroPrefixSize:]
				assert.Equal(t, expectedData, actualData, "Data should match in last 10 bytes")
				
				// Verify total size
				assert.Equal(t, NamespaceSize, len(ns.Bytes()), "Total namespace size should be 29 bytes")
				
				// Verify it's valid for version 0
				assert.True(t, ns.IsValidForVersion0(), "Should be valid for version 0")
			}
		})
	}
}

func TestNamespaceFromBytes(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expectError bool
		description string
	}{
		{
			name:        "valid version 0 namespace",
			input:       append([]byte{0}, append(make([]byte, 18), []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}...)...),
			expectError: false,
			description: "Should parse valid version 0 namespace",
		},
		{
			name:        "invalid size - too short",
			input:       []byte{0, 0, 0},
			expectError: true,
			description: "Should fail with input shorter than 29 bytes",
		},
		{
			name:        "invalid size - too long",
			input:       make([]byte, 30),
			expectError: true,
			description: "Should fail with input longer than 29 bytes",
		},
		{
			name:        "invalid version 0 - non-zero prefix",
			input:       append([]byte{0}, append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}...)...),
			expectError: true,
			description: "Should fail when version 0 namespace has non-zero bytes in first 18 bytes of ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns, err := NamespaceFromBytes(tt.input)
			
			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Nil(t, ns)
			} else {
				require.NoError(t, err, tt.description)
				require.NotNil(t, ns)
				assert.Equal(t, tt.input, ns.Bytes(), "Should round-trip correctly")
			}
		})
	}
}

func TestNamespaceFromString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		verify func(t *testing.T, ns *Namespace)
	}{
		{
			name:  "rollkit-headers",
			input: "rollkit-headers",
			verify: func(t *testing.T, ns *Namespace) {
				assert.Equal(t, NamespaceVersionZero, ns.Version)
				assert.True(t, ns.IsValidForVersion0())
				assert.Equal(t, NamespaceSize, len(ns.Bytes()))
				
				// The hash should be deterministic
				ns2 := NamespaceFromString("rollkit-headers")
				assert.Equal(t, ns.Bytes(), ns2.Bytes(), "Same string should produce same namespace")
			},
		},
		{
			name:  "rollkit-data",
			input: "rollkit-data",
			verify: func(t *testing.T, ns *Namespace) {
				assert.Equal(t, NamespaceVersionZero, ns.Version)
				assert.True(t, ns.IsValidForVersion0())
				assert.Equal(t, NamespaceSize, len(ns.Bytes()))
				
				// Different strings should produce different namespaces
				ns2 := NamespaceFromString("rollkit-headers")
				assert.NotEqual(t, ns.Bytes(), ns2.Bytes(), "Different strings should produce different namespaces")
			},
		},
		{
			name:  "empty string",
			input: "",
			verify: func(t *testing.T, ns *Namespace) {
				assert.Equal(t, NamespaceVersionZero, ns.Version)
				assert.True(t, ns.IsValidForVersion0())
				assert.Equal(t, NamespaceSize, len(ns.Bytes()))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns := NamespaceFromString(tt.input)
			require.NotNil(t, ns)
			tt.verify(t, ns)
		})
	}
}

func TestPrepareNamespace(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		description string
		verify      func(t *testing.T, result []byte)
	}{
		{
			name:        "string identifier",
			input:       []byte("rollkit-headers"),
			description: "Should convert string to valid namespace",
			verify: func(t *testing.T, result []byte) {
				assert.Equal(t, NamespaceSize, len(result))
				assert.Equal(t, byte(0), result[0], "Should be version 0")
				
				// Verify first 18 bytes of ID are zeros
				for i := 1; i <= 18; i++ {
					assert.Equal(t, byte(0), result[i], "First 18 bytes of ID should be zero")
				}
			},
		},
		{
			name:        "already valid namespace",
			input:       append([]byte{0}, append(make([]byte, 18), []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}...)...),
			description: "Should return already valid namespace unchanged",
			verify: func(t *testing.T, result []byte) {
				assert.Equal(t, NamespaceSize, len(result))
				assert.Equal(t, byte(0), result[0])
				// Should pass through unchanged
				expected := append([]byte{0}, append(make([]byte, 18), []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}...)...)
				assert.Equal(t, expected, result)
			},
		},
		{
			name:        "invalid 29-byte input",
			input:       append([]byte{0}, append([]byte{1}, make([]byte, 27)...)...), // Invalid: non-zero in prefix
			description: "Should treat invalid 29-byte input as string and hash it",
			verify: func(t *testing.T, result []byte) {
				assert.Equal(t, NamespaceSize, len(result))
				assert.Equal(t, byte(0), result[0])
				
				// Should be hashed, not passed through
				invalidNs := append([]byte{0}, append([]byte{1}, make([]byte, 27)...)...)
				assert.NotEqual(t, invalidNs, result, "Should not pass through invalid namespace")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PrepareNamespace(tt.input)
			require.NotNil(t, result)
			tt.verify(t, result)
		})
	}
}

func TestHexStringConversion(t *testing.T) {
	ns, err := NewNamespaceV0([]byte{1, 2, 3, 4, 5})
	require.NoError(t, err)
	
	// Test HexString
	hexStr := ns.HexString()
	assert.True(t, len(hexStr) > 2, "Hex string should not be empty")
	assert.Equal(t, "0x", hexStr[:2], "Should have 0x prefix")
	
	// Test ParseHexNamespace
	parsed, err := ParseHexNamespace(hexStr)
	require.NoError(t, err)
	assert.Equal(t, ns.Bytes(), parsed.Bytes(), "Should round-trip through hex")
	
	// Test without 0x prefix
	parsed2, err := ParseHexNamespace(hexStr[2:])
	require.NoError(t, err)
	assert.Equal(t, ns.Bytes(), parsed2.Bytes(), "Should work without 0x prefix")
	
	// Test invalid hex
	_, err = ParseHexNamespace("invalid-hex")
	assert.Error(t, err, "Should fail with invalid hex")
	
	// Test wrong size hex
	_, err = ParseHexNamespace("0x0011")
	assert.Error(t, err, "Should fail with wrong size")
}

func TestCelestiaSpecCompliance(t *testing.T) {
	// Test that our implementation follows the Celestia namespace specification
	
	t.Run("namespace structure", func(t *testing.T) {
		ns, err := NewNamespaceV0([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A})
		require.NoError(t, err)
		
		nsBytes := ns.Bytes()
		
		// Check total size is 29 bytes (1 version + 28 ID)
		assert.Equal(t, 29, len(nsBytes), "Total namespace size should be 29 bytes")
		
		// Check version byte is at position 0
		assert.Equal(t, byte(0), nsBytes[0], "Version byte should be 0")
		
		// Check ID is 28 bytes starting at position 1
		assert.Equal(t, 28, len(nsBytes[1:]), "ID should be 28 bytes")
	})
	
	t.Run("version 0 requirements", func(t *testing.T) {
		ns, err := NewNamespaceV0([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
		require.NoError(t, err)
		
		nsBytes := ns.Bytes()
		
		// For version 0, first 18 bytes of ID must be zero
		for i := 1; i <= 18; i++ {
			assert.Equal(t, byte(0), nsBytes[i], "Bytes 1-18 should be zero for version 0")
		}
		
		// Last 10 bytes should contain our data
		expectedData := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
		actualData := nsBytes[19:29]
		assert.Equal(t, expectedData, actualData, "Last 10 bytes should contain user data")
	})
	
	t.Run("example from spec", func(t *testing.T) {
		// Create a namespace similar to the example in the spec
		ns, err := NewNamespaceV0([]byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01})
		require.NoError(t, err)
		
		hexStr := ns.HexString()
		t.Logf("Example namespace: %s", hexStr)
		
		// Verify it matches the expected format
		assert.Equal(t, 60, len(hexStr), "Hex string should be 60 chars (0x + 58 hex chars)")
		// The prefix should be: 0x (2 chars) + version byte 00 (2 chars) + 18 zero bytes (36 chars) = 40 chars total
		expectedPrefix := "0x00000000000000000000000000000000000000"
		assert.Equal(t, expectedPrefix, hexStr[:40], "Should have correct zero prefix")
	})
}

func TestRealWorldNamespaces(t *testing.T) {
	// Test with actual namespace strings used in rollkit
	namespaces := []string{
		"rollkit-headers",
		"rollkit-data",
		"legacy-namespace",
		"test-headers",
		"test-data",
	}
	
	seen := make(map[string]bool)
	
	for _, nsStr := range namespaces {
		t.Run(nsStr, func(t *testing.T) {
			// Convert string to namespace
			nsBytes := PrepareNamespace([]byte(nsStr))
			
			// Verify it's valid
			ns, err := NamespaceFromBytes(nsBytes)
			require.NoError(t, err)
			assert.True(t, ns.IsValidForVersion0())
			
			// Verify uniqueness
			hexStr := hex.EncodeToString(nsBytes)
			assert.False(t, seen[hexStr], "Namespace should be unique")
			seen[hexStr] = true
			
			// Verify deterministic
			nsBytes2 := PrepareNamespace([]byte(nsStr))
			assert.True(t, bytes.Equal(nsBytes, nsBytes2), "Should be deterministic")
			
			t.Logf("Namespace for '%s': %s", nsStr, hex.EncodeToString(nsBytes))
		})
	}
}