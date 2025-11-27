package celestia

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/nmt"
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

func TestClient_Submit(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	validNamespace := make([]byte, 29)
	validBlob := &Blob{
		Namespace: validNamespace,
		Data:      []byte("test data"),
	}

	tests := []struct {
		name    string
		blobs   []*Blob
		wantRPC bool
	}{
		{
			name:    "single blob",
			blobs:   []*Blob{validBlob},
			wantRPC: true,
		},
		{
			name: "multiple blobs",
			blobs: []*Blob{
				validBlob,
				{
					Namespace: validNamespace,
					Data:      []byte("more data"),
				},
			},
			wantRPC: true,
		},
		{
			name:    "empty list delegates to celestia-node",
			blobs:   []*Blob{},
			wantRPC: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(ctx, logger, "http://localhost:26658", "token", 1024*1024)
			require.NoError(t, err)
			defer client.Close()

			_, err = client.submit(ctx, tt.blobs, nil)

			if tt.wantRPC {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to submit blobs")
			}
		})
	}
}

func TestClient_Get(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	client, err := NewClient(ctx, logger, "http://localhost:26658", "token", 1024*1024)
	require.NoError(t, err)
	defer client.Close()

	validNamespace := make([]byte, 29)
	validCommitment := []byte("commitment")

	_, err = client.get(ctx, 100, validNamespace, validCommitment)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get blob")
}

func TestClient_GetAll(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	client, err := NewClient(ctx, logger, "http://localhost:26658", "token", 1024*1024)
	require.NoError(t, err)
	defer client.Close()

	validNamespace := make([]byte, 29)
	namespaces := []Namespace{validNamespace}

	_, err = client.getAll(ctx, 100, namespaces)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get blobs")
}

func TestClient_GetProof(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	client, err := NewClient(ctx, logger, "http://localhost:26658", "token", 1024*1024)
	require.NoError(t, err)
	defer client.Close()

	validNamespace := make([]byte, 29)
	validCommitment := []byte("commitment")

	_, err = client.getProof(ctx, 100, validNamespace, validCommitment)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get proof")
}

func TestClient_Included(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	client, err := NewClient(ctx, logger, "http://localhost:26658", "token", 1024*1024)
	require.NoError(t, err)
	defer client.Close()

	validNamespace := make([]byte, 29)
	validCommitment := []byte("commitment")
	proof := Proof{&nmt.Proof{}}

	_, err = client.included(ctx, 100, validNamespace, &proof, validCommitment)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check inclusion")
}
