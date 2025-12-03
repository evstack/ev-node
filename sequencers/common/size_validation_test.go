package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateBlobSize(t *testing.T) {
	tests := []struct {
		name     string
		blobSize int
		want     bool
	}{
		{
			name:     "empty blob",
			blobSize: 0,
			want:     true,
		},
		{
			name:     "small blob",
			blobSize: 100,
			want:     true,
		},
		{
			name:     "exactly at limit",
			blobSize: int(AbsoluteMaxBlobSize),
			want:     true,
		},
		{
			name:     "one byte over limit",
			blobSize: int(AbsoluteMaxBlobSize) + 1,
			want:     false,
		},
		{
			name:     "far exceeds limit",
			blobSize: int(AbsoluteMaxBlobSize) * 2,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blob := make([]byte, tt.blobSize)
			got := ValidateBlobSize(blob)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWouldExceedCumulativeSize(t *testing.T) {
	tests := []struct {
		name        string
		currentSize int
		blobSize    int
		maxBytes    uint64
		want        bool
	}{
		{
			name:        "empty batch, small blob",
			currentSize: 0,
			blobSize:    50,
			maxBytes:    100,
			want:        false,
		},
		{
			name:        "would fit exactly",
			currentSize: 50,
			blobSize:    50,
			maxBytes:    100,
			want:        false,
		},
		{
			name:        "would exceed by one byte",
			currentSize: 50,
			blobSize:    51,
			maxBytes:    100,
			want:        true,
		},
		{
			name:        "far exceeds",
			currentSize: 80,
			blobSize:    100,
			maxBytes:    100,
			want:        true,
		},
		{
			name:        "zero max bytes",
			currentSize: 0,
			blobSize:    1,
			maxBytes:    0,
			want:        true,
		},
		{
			name:        "current already at limit",
			currentSize: 100,
			blobSize:    1,
			maxBytes:    100,
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WouldExceedCumulativeSize(tt.currentSize, tt.blobSize, tt.maxBytes)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetBlobSize(t *testing.T) {
	tests := []struct {
		name     string
		blobSize int
		want     int
	}{
		{
			name:     "empty blob",
			blobSize: 0,
			want:     0,
		},
		{
			name:     "small blob",
			blobSize: 42,
			want:     42,
		},
		{
			name:     "large blob",
			blobSize: 1024 * 1024,
			want:     1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blob := make([]byte, tt.blobSize)
			got := GetBlobSize(blob)
			assert.Equal(t, tt.want, got)
		})
	}
}
