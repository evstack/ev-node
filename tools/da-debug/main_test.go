package main

import (
	"encoding/hex"
	"testing"

	coreda "github.com/evstack/ev-node/core/da"
)

func TestParseNamespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // expected length in bytes
		wantErr  bool
	}{
		{
			name:     "valid hex namespace with 0x prefix",
			input:    "0x000000000000000000000000000000000000000000000000000000746573743031",
			expected: 29,
			wantErr:  false,
		},
		{
			name:     "valid hex namespace without prefix",
			input:    "000000000000000000000000000000000000000000000000000000746573743031",
			expected: 29,
			wantErr:  false,
		},
		{
			name:     "string identifier",
			input:    "test-namespace",
			expected: 29,
			wantErr:  false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 29,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseNamespace(tt.input)

			if tt.wantErr && err == nil {
				t.Errorf("parseNamespace() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("parseNamespace() unexpected error: %v", err)
			}

			if len(result) != tt.expected {
				t.Errorf("parseNamespace() result length = %d, expected %d", len(result), tt.expected)
			}
		})
	}
}

func TestTryDecodeHeader(t *testing.T) {
	// Test with invalid data
	result := tryDecodeHeader([]byte("invalid"))
	if result != nil {
		t.Errorf("tryDecodeHeader() with invalid data should return nil")
	}

	// Test with empty data
	result = tryDecodeHeader([]byte{})
	if result != nil {
		t.Errorf("tryDecodeHeader() with empty data should return nil")
	}
}

func TestTryDecodeData(t *testing.T) {
	// Test with invalid data
	result := tryDecodeData([]byte("invalid"))
	if result != nil {
		t.Errorf("tryDecodeData() with invalid data should return nil")
	}

	// Test with empty data
	result = tryDecodeData([]byte{})
	if result != nil {
		t.Errorf("tryDecodeData() with empty data should return nil")
	}
}

func TestParseHex(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "with 0x prefix",
			input:    "0xdeadbeef",
			expected: "deadbeef",
			wantErr:  false,
		},
		{
			name:     "without prefix",
			input:    "deadbeef",
			expected: "deadbeef",
			wantErr:  false,
		},
		{
			name:    "invalid hex",
			input:   "xyz123",
			wantErr: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseHex(tt.input)

			if tt.wantErr && err == nil {
				t.Errorf("parseHex() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("parseHex() unexpected error: %v", err)
			}

			if !tt.wantErr {
				resultHex := hex.EncodeToString(result)
				if resultHex != tt.expected {
					t.Errorf("parseHex() result = %s, expected %s", resultHex, tt.expected)
				}
			}
		})
	}
}

func TestIsPrintable(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{
			name:     "printable ASCII",
			input:    []byte("Hello, World!"),
			expected: true,
		},
		{
			name:     "with newlines and tabs",
			input:    []byte("Hello\nWorld\t!"),
			expected: true,
		},
		{
			name:     "binary data",
			input:    []byte{0x00, 0x01, 0x02, 0xFF},
			expected: false,
		},
		{
			name:     "mixed printable and non-printable",
			input:    []byte("Hello\x00World"),
			expected: false,
		},
		{
			name:     "empty",
			input:    []byte{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPrintable(tt.input)
			if result != tt.expected {
				t.Errorf("isPrintable() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestIDSplitting(t *testing.T) {
	// Test with a mock ID that follows the expected format
	height := uint64(12345)
	commitment := []byte("test-commitment-data")

	// Create an ID using the format from the LocalDA implementation
	id := make([]byte, 8+len(commitment))
	// Use little endian as per the da.go implementation
	id[0] = byte(height)
	id[1] = byte(height >> 8)
	id[2] = byte(height >> 16)
	id[3] = byte(height >> 24)
	id[4] = byte(height >> 32)
	id[5] = byte(height >> 40)
	id[6] = byte(height >> 48)
	id[7] = byte(height >> 56)
	copy(id[8:], commitment)

	// Test splitting
	parsedHeight, parsedCommitment, err := coreda.SplitID(id)
	if err != nil {
		t.Errorf("SplitID() unexpected error: %v", err)
	}

	if parsedHeight != height {
		t.Errorf("SplitID() height = %d, expected %d", parsedHeight, height)
	}

	if string(parsedCommitment) != string(commitment) {
		t.Errorf("SplitID() commitment = %s, expected %s", string(parsedCommitment), string(commitment))
	}
}
