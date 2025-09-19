package da

import (
	"bytes"
	"encoding/hex"
	"testing"
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
				if err == nil {
					t.Errorf("%s: expected error but got nil", tt.description)
				}
				if ns != nil {
					t.Errorf("expected nil namespace but got %v", ns)
				}
			} else {
				if err != nil {
					t.Fatalf("%s: unexpected error: %v", tt.description, err)
				}
				if ns == nil {
					t.Fatal("expected non-nil namespace but got nil")
				}

				// Verify version is 0
				if ns.Version != NamespaceVersionZero {
					t.Errorf("Version should be 0, got %d", ns.Version)
				}

				// Verify first 18 bytes of ID are zeros
				for i := range NamespaceVersionZeroPrefixSize {
					if ns.ID[i] != byte(0) {
						t.Errorf("First 18 bytes should be zero, but byte %d is %d", i, ns.ID[i])
					}
				}

				// Verify data is in the last 10 bytes
				expectedData := make([]byte, NamespaceVersionZeroDataSize)
				copy(expectedData, tt.data)
				actualData := ns.ID[NamespaceVersionZeroPrefixSize:]
				if !bytes.Equal(expectedData, actualData) {
					t.Errorf("Data should match in last 10 bytes, expected %v, got %v", expectedData, actualData)
				}

				// Verify total size
				if len(ns.Bytes()) != NamespaceSize {
					t.Errorf("Total namespace size should be 29 bytes, got %d", len(ns.Bytes()))
				}

				// Verify it's valid for version 0
				if !ns.IsValidForVersion0() {
					t.Error("Should be valid for version 0")
				}
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
				if err == nil {
					t.Errorf("%s: expected error but got nil", tt.description)
				}
				if ns != nil {
					t.Errorf("expected nil namespace but got %v", ns)
				}
			} else {
				if err != nil {
					t.Fatalf("%s: unexpected error: %v", tt.description, err)
				}
				if ns == nil {
					t.Fatal("expected non-nil namespace but got nil")
				}
				if !bytes.Equal(tt.input, ns.Bytes()) {
					t.Errorf("Should round-trip correctly, expected %v, got %v", tt.input, ns.Bytes())
				}
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
				if ns.Version != NamespaceVersionZero {
					t.Errorf("expected version %d, got %d", NamespaceVersionZero, ns.Version)
				}
				if !ns.IsValidForVersion0() {
					t.Error("namespace should be valid for version 0")
				}
				if len(ns.Bytes()) != NamespaceSize {
					t.Errorf("expected namespace size %d, got %d", NamespaceSize, len(ns.Bytes()))
				}

				// The hash should be deterministic
				ns2 := NamespaceFromString("rollkit-headers")
				if !bytes.Equal(ns.Bytes(), ns2.Bytes()) {
					t.Error("Same string should produce same namespace")
				}
			},
		},
		{
			name:  "rollkit-data",
			input: "rollkit-data",
			verify: func(t *testing.T, ns *Namespace) {
				if ns.Version != NamespaceVersionZero {
					t.Errorf("expected version %d, got %d", NamespaceVersionZero, ns.Version)
				}
				if !ns.IsValidForVersion0() {
					t.Error("namespace should be valid for version 0")
				}
				if len(ns.Bytes()) != NamespaceSize {
					t.Errorf("expected namespace size %d, got %d", NamespaceSize, len(ns.Bytes()))
				}

				// Different strings should produce different namespaces
				ns2 := NamespaceFromString("rollkit-headers")
				if bytes.Equal(ns.Bytes(), ns2.Bytes()) {
					t.Error("Different strings should produce different namespaces")
				}
			},
		},
		{
			name:  "empty string",
			input: "",
			verify: func(t *testing.T, ns *Namespace) {
				if ns.Version != NamespaceVersionZero {
					t.Errorf("expected version %d, got %d", NamespaceVersionZero, ns.Version)
				}
				if !ns.IsValidForVersion0() {
					t.Error("namespace should be valid for version 0")
				}
				if len(ns.Bytes()) != NamespaceSize {
					t.Errorf("expected namespace size %d, got %d", NamespaceSize, len(ns.Bytes()))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns := NamespaceFromString(tt.input)
			if ns == nil {
				t.Fatal("expected non-nil namespace but got nil")
			}
			tt.verify(t, ns)
		})
	}
}

func TestHexStringConversion(t *testing.T) {
	ns, err := NewNamespaceV0([]byte{1, 2, 3, 4, 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test HexString
	hexStr := ns.HexString()
	if len(hexStr) <= 2 {
		t.Error("Hex string should not be empty")
	}
	if hexStr[:2] != "0x" {
		t.Errorf("Should have 0x prefix, got %s", hexStr[:2])
	}

	// Test ParseHexNamespace
	parsed, err := ParseHexNamespace(hexStr)
	if err != nil {
		t.Fatalf("unexpected error parsing hex: %v", err)
	}
	if !bytes.Equal(ns.Bytes(), parsed.Bytes()) {
		t.Error("Should round-trip through hex")
	}

	// Test without 0x prefix
	parsed2, err := ParseHexNamespace(hexStr[2:])
	if err != nil {
		t.Fatalf("unexpected error parsing hex without prefix: %v", err)
	}
	if !bytes.Equal(ns.Bytes(), parsed2.Bytes()) {
		t.Error("Should work without 0x prefix")
	}

	// Test invalid hex
	_, err = ParseHexNamespace("invalid-hex")
	if err == nil {
		t.Error("Should fail with invalid hex")
	}

	// Test wrong size hex
	_, err = ParseHexNamespace("0x0011")
	if err == nil {
		t.Error("Should fail with wrong size")
	}
}

func TestCelestiaSpecCompliance(t *testing.T) {
	// Test that our implementation follows the Celestia namespace specification

	t.Run("namespace structure", func(t *testing.T) {
		ns, err := NewNamespaceV0([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		nsBytes := ns.Bytes()

		// Check total size is 29 bytes (1 version + 28 ID)
		if len(nsBytes) != 29 {
			t.Errorf("Total namespace size should be 29 bytes, got %d", len(nsBytes))
		}

		// Check version byte is at position 0
		if nsBytes[0] != byte(0) {
			t.Errorf("Version byte should be 0, got %d", nsBytes[0])
		}

		// Check ID is 28 bytes starting at position 1
		if len(nsBytes[1:]) != 28 {
			t.Errorf("ID should be 28 bytes, got %d", len(nsBytes[1:]))
		}
	})

	t.Run("version 0 requirements", func(t *testing.T) {
		ns, err := NewNamespaceV0([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		nsBytes := ns.Bytes()

		// For version 0, first 18 bytes of ID must be zero
		for i := 1; i <= 18; i++ {
			if nsBytes[i] != byte(0) {
				t.Errorf("Bytes 1-18 should be zero for version 0, but byte %d is %d", i, nsBytes[i])
			}
		}

		// Last 10 bytes should contain our data
		expectedData := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
		actualData := nsBytes[19:29]
		if !bytes.Equal(expectedData, actualData) {
			t.Errorf("Last 10 bytes should contain user data, expected %v, got %v", expectedData, actualData)
		}
	})

	t.Run("example from spec", func(t *testing.T) {
		// Create a namespace similar to the example in the spec
		ns, err := NewNamespaceV0([]byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		hexStr := ns.HexString()
		t.Logf("Example namespace: %s", hexStr)

		// Verify it matches the expected format
		if len(hexStr) != 60 {
			t.Errorf("Hex string should be 60 chars (0x + 58 hex chars), got %d", len(hexStr))
		}
		// The prefix should be: 0x (2 chars) + version byte 00 (2 chars) + 18 zero bytes (36 chars) = 40 chars total
		expectedPrefix := "0x00000000000000000000000000000000000000"
		if hexStr[:40] != expectedPrefix {
			t.Errorf("Should have correct zero prefix, expected %s, got %s", expectedPrefix, hexStr[:40])
		}
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
			nsBytes := NamespaceFromString(nsStr)

			// Verify it's valid
			ns, err := NamespaceFromBytes(nsBytes.Bytes())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !ns.IsValidForVersion0() {
				t.Error("namespace should be valid for version 0")
			}

			// Verify uniqueness
			hexStr := hex.EncodeToString(nsBytes.Bytes())
			if seen[hexStr] {
				t.Errorf("Namespace should be unique, but %s was already seen", hexStr)
			}
			seen[hexStr] = true

			// Verify deterministic
			nsBytes2 := NamespaceFromString(nsStr)
			if !bytes.Equal(nsBytes.Bytes(), nsBytes2.Bytes()) {
				t.Error("Should be deterministic")
			}

			t.Logf("Namespace for '%s': %s", nsStr, hex.EncodeToString(nsBytes.Bytes()))
		})
	}
}
