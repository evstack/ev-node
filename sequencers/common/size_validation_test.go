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
