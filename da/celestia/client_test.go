package celestia

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	tests := []struct {
		name        string
		addr        string
		token       string
		maxBlobSize uint64
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid parameters",
			addr:        "http://localhost:26658",
			token:       "test-token",
			maxBlobSize: 1024 * 1024,
			wantErr:     false,
		},
		{
			name:        "valid parameters without token",
			addr:        "http://localhost:26658",
			token:       "",
			maxBlobSize: 1024 * 1024,
			wantErr:     false,
		},
		{
			name:        "empty address",
			addr:        "",
			token:       "test-token",
			maxBlobSize: 1024,
			wantErr:     true,
			errContains: "address cannot be empty",
		},
		{
			name:        "zero maxBlobSize",
			addr:        "http://localhost:26658",
			token:       "test-token",
			maxBlobSize: 0,
			wantErr:     true,
			errContains: "maxBlobSize must be greater than 0",
		},
		{
			name:        "invalid address will fail on connection",
			addr:        "not-a-valid-url",
			token:       "test-token",
			maxBlobSize: 1024,
			wantErr:     true,
			errContains: "failed to create JSON-RPC client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(ctx, logger, tt.addr, tt.token, tt.maxBlobSize)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
				assert.Equal(t, tt.maxBlobSize, client.maxBlobSize)
				assert.NotNil(t, client.closer)

				// Clean up
				client.Close()
			}
		})
	}
}

func TestClient_Close(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	client, err := NewClient(ctx, logger, "http://localhost:26658", "token", 1024*1024)
	require.NoError(t, err)
	require.NotNil(t, client)

	// Should not panic
	assert.NotPanics(t, func() {
		client.Close()
	})

	// Should be safe to call multiple times
	assert.NotPanics(t, func() {
		client.Close()
	})
}

func TestClient_BlobAPIMethods(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	client, err := NewClient(ctx, logger, "http://localhost:26658", "token", 1024*1024)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	// Test that all native blob API methods exist and return "not implemented" for now
	t.Run("Submit", func(t *testing.T) {
		_, err := client.Submit(ctx, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
	})

	t.Run("Get", func(t *testing.T) {
		_, err := client.Get(ctx, 0, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
	})

	t.Run("GetAll", func(t *testing.T) {
		_, err := client.GetAll(ctx, 0, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
	})

	t.Run("GetProof", func(t *testing.T) {
		_, err := client.GetProof(ctx, 0, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
	})

	t.Run("Included", func(t *testing.T) {
		_, err := client.Included(ctx, 0, nil, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
	})
}
