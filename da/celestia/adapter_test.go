package celestia

import (
	"context"
	"testing"

	"github.com/evstack/ev-node/da"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAdapter(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	tests := []struct {
		name        string
		addr        string
		token       string
		maxBlobSize uint64
		wantErr     bool
	}{
		{
			name:        "valid parameters",
			addr:        "http://localhost:26658",
			token:       "test-token",
			maxBlobSize: 1024 * 1024,
			wantErr:     false,
		},
		{
			name:        "empty address",
			addr:        "",
			token:       "test-token",
			maxBlobSize: 1024,
			wantErr:     true,
		},
		{
			name:        "zero maxBlobSize",
			addr:        "http://localhost:26658",
			token:       "test-token",
			maxBlobSize: 0,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := NewAdapter(ctx, logger, tt.addr, tt.token, tt.maxBlobSize)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, adapter)
			} else {
				require.NoError(t, err)
				require.NotNil(t, adapter)
				adapter.Close()
			}
		})
	}
}

func TestAdapter_Submit(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	adapter, err := NewAdapter(ctx, logger, "http://localhost:26658", "token", 1024*1024)
	require.NoError(t, err)
	defer adapter.Close()

	validNamespace := make([]byte, 29)
	blobs := []da.Blob{[]byte("test data")}

	_, err = adapter.Submit(ctx, blobs, 0.002, validNamespace)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to submit blobs")
}

func TestAdapter_SubmitWithInvalidNamespace(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	adapter, err := NewAdapter(ctx, logger, "http://localhost:26658", "token", 1024*1024)
	require.NoError(t, err)
	defer adapter.Close()

	invalidNamespace := make([]byte, 10)
	blobs := []da.Blob{[]byte("test data")}

	_, err = adapter.Submit(ctx, blobs, 0.002, invalidNamespace)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid namespace")
}

func TestAdapter_Get(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	adapter, err := NewAdapter(ctx, logger, "http://localhost:26658", "token", 1024*1024)
	require.NoError(t, err)
	defer adapter.Close()

	validNamespace := make([]byte, 29)
	testID := makeID(100, []byte("test-commitment"))

	_, err = adapter.Get(ctx, []da.ID{testID}, validNamespace)
	require.Error(t, err)
}

func TestAdapter_GetIDs(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	adapter, err := NewAdapter(ctx, logger, "http://localhost:26658", "token", 1024*1024)
	require.NoError(t, err)
	defer adapter.Close()

	validNamespace := make([]byte, 29)

	_, err = adapter.GetIDs(ctx, 100, validNamespace)
	require.Error(t, err)
}

func TestMakeIDAndSplitID(t *testing.T) {
	height := uint64(12345)
	commitment := []byte("test-commitment-data")

	id := makeID(height, commitment)

	retrievedHeight, retrievedCommitment, err := splitID(id)
	require.NoError(t, err)
	assert.Equal(t, height, retrievedHeight)
	assert.Equal(t, commitment, retrievedCommitment)
}

func TestSplitID_InvalidID(t *testing.T) {
	shortID := []byte("short")

	_, _, err := splitID(shortID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid ID length")
}
