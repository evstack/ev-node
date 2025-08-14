package jsonrpc

import (
	"bytes"
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/da/compression"
)

// mockRPCServer simulates the RPC server for testing
type mockRPCServer struct {
	blobs       map[string]da.Blob
	compressionDetected bool
}

func newMockRPCServer() *mockRPCServer {
	return &mockRPCServer{
		blobs: make(map[string]da.Blob),
	}
}

func (m *mockRPCServer) submit(blobs []da.Blob) []da.ID {
	ids := make([]da.ID, len(blobs))
	for i, blob := range blobs {
		// Check if blob is compressed
		if len(blob) >= compression.CompressionHeaderSize {
			info := compression.GetCompressionInfo(blob)
			if info.IsCompressed || blob[0] == compression.FlagUncompressed {
				m.compressionDetected = true
			}
		}
		
		id := da.ID([]byte{byte(len(m.blobs))})
		m.blobs[string(id)] = blob
		ids[i] = id
	}
	return ids
}

func (m *mockRPCServer) get(ids []da.ID) []da.Blob {
	blobs := make([]da.Blob, len(ids))
	for i, id := range ids {
		blobs[i] = m.blobs[string(id)]
	}
	return blobs
}

// TestClientCompressionSubmit tests that the client compresses data before submission
func TestClientCompressionSubmit(t *testing.T) {
	logger := zerolog.Nop()
	mockServer := newMockRPCServer()
	
	// Create API with compression enabled
	api := &API{
		Logger:             logger,
		MaxBlobSize:        1024 * 1024,
		compressionEnabled: true,
		compressionConfig: compression.Config{
			Enabled:             true,
			ZstdLevel:           3,
			MinCompressionRatio: 0.1,
		},
	}
	
	// Mock the internal submit function
	api.Internal.Submit = func(ctx context.Context, blobs []da.Blob, gasPrice float64, ns []byte) ([]da.ID, error) {
		return mockServer.submit(blobs), nil
	}
	
	// Test data that should compress well
	testBlobs := []da.Blob{
		bytes.Repeat([]byte("compress me "), 100),
		bytes.Repeat([]byte("a"), 1000),
	}
	
	ctx := context.Background()
	ids, err := api.Submit(ctx, testBlobs, 1.0, []byte("test"))
	require.NoError(t, err)
	require.Len(t, ids, len(testBlobs))
	
	// Verify compression was applied
	assert.True(t, mockServer.compressionDetected, "Compression should be detected in submitted blobs")
	
	// Check that compressed blobs are smaller
	for _, id := range ids {
		compressedBlob := mockServer.blobs[string(id)]
		info := compression.GetCompressionInfo(compressedBlob)
		
		// At least one blob should be compressed
		if info.IsCompressed {
			assert.Less(t, info.CompressedSize, info.OriginalSize, 
				"Compressed size should be less than original")
		}
	}
}

// TestClientCompressionGet tests that the client decompresses data after retrieval
func TestClientCompressionGet(t *testing.T) {
	logger := zerolog.Nop()
	mockServer := newMockRPCServer()
	
	// Original test data
	originalBlobs := []da.Blob{
		bytes.Repeat([]byte("test data "), 50),
		[]byte("small data"),
	}
	
	// Compress blobs manually for the mock server
	compressedBlobs := make([]da.Blob, len(originalBlobs))
	for i, blob := range originalBlobs {
		compressed, err := compression.CompressBlob(blob)
		require.NoError(t, err)
		compressedBlobs[i] = compressed
	}
	
	// Store compressed blobs in mock server
	ids := mockServer.submit(compressedBlobs)
	
	// Create API with compression enabled
	api := &API{
		Logger:             logger,
		compressionEnabled: true,
		compressionConfig: compression.Config{
			Enabled: true,
		},
	}
	
	// Mock the internal get function
	api.Internal.Get = func(ctx context.Context, ids []da.ID, ns []byte) ([]da.Blob, error) {
		return mockServer.get(ids), nil
	}
	
	// Retrieve and decompress
	ctx := context.Background()
	retrievedBlobs, err := api.Get(ctx, ids, []byte("test"))
	require.NoError(t, err)
	require.Len(t, retrievedBlobs, len(originalBlobs))
	
	// Verify data integrity
	for i, retrieved := range retrievedBlobs {
		assert.Equal(t, originalBlobs[i], retrieved, 
			"Retrieved blob should match original after decompression")
	}
}

// TestClientCompressionDisabled tests that compression can be disabled
func TestClientCompressionDisabled(t *testing.T) {
	logger := zerolog.Nop()
	mockServer := newMockRPCServer()
	
	// Create API with compression disabled
	api := &API{
		Logger:             logger,
		MaxBlobSize:        1024 * 1024,
		compressionEnabled: false,
	}
	
	// Mock the internal submit function
	api.Internal.Submit = func(ctx context.Context, blobs []da.Blob, gasPrice float64, ns []byte) ([]da.ID, error) {
		return mockServer.submit(blobs), nil
	}
	
	// Test data
	testBlobs := []da.Blob{
		bytes.Repeat([]byte("don't compress me "), 50),
	}
	
	ctx := context.Background()
	ids, err := api.Submit(ctx, testBlobs, 1.0, []byte("test"))
	require.NoError(t, err)
	require.Len(t, ids, len(testBlobs))
	
	// Verify no compression was applied
	assert.False(t, mockServer.compressionDetected, 
		"Compression should not be detected when disabled")
	
	// Check that blobs are unmodified
	for i, id := range ids {
		storedBlob := mockServer.blobs[string(id)]
		assert.Equal(t, testBlobs[i], storedBlob, 
			"Blob should be unmodified when compression is disabled")
	}
}

// TestClientOptionsCreation tests creating clients with different compression options
func TestClientOptionsCreation(t *testing.T) {
	t.Run("DefaultOptions", func(t *testing.T) {
		opts := DefaultClientOptions()
		assert.True(t, opts.CompressionEnabled)
		assert.Equal(t, compression.DefaultZstdLevel, opts.CompressionLevel)
		assert.Equal(t, compression.DefaultMinCompressionRatio, opts.MinCompressionRatio)
	})
	
	t.Run("CustomOptions", func(t *testing.T) {
		opts := ClientOptions{
			CompressionEnabled:  true,
			CompressionLevel:    9,
			MinCompressionRatio: 0.2,
		}
		
		assert.True(t, opts.CompressionEnabled)
		assert.Equal(t, 9, opts.CompressionLevel)
		assert.Equal(t, 0.2, opts.MinCompressionRatio)
	})
}

// TestSubmitWithOptionsCompression tests compression with SubmitWithOptions
func TestSubmitWithOptionsCompression(t *testing.T) {
	logger := zerolog.Nop()
	mockServer := newMockRPCServer()
	
	// Create API with compression enabled
	api := &API{
		Logger:             logger,
		MaxBlobSize:        1024 * 1024,
		compressionEnabled: true,
		compressionConfig: compression.Config{
			Enabled:             true,
			ZstdLevel:           3,
			MinCompressionRatio: 0.1,
		},
	}
	
	// Mock the internal submit with options function
	api.Internal.SubmitWithOptions = func(ctx context.Context, blobs []da.Blob, gasPrice float64, ns []byte, options []byte) ([]da.ID, error) {
		// Verify blobs are compressed
		for _, blob := range blobs {
			if len(blob) >= compression.CompressionHeaderSize {
				info := compression.GetCompressionInfo(blob)
				if info.IsCompressed || blob[0] == compression.FlagUncompressed {
					mockServer.compressionDetected = true
					break
				}
			}
		}
		return mockServer.submit(blobs), nil
	}
	
	// Test data that should compress well
	testBlobs := []da.Blob{
		bytes.Repeat([]byte("compress with options "), 100),
	}
	
	ctx := context.Background()
	ids, err := api.SubmitWithOptions(ctx, testBlobs, 1.0, []byte("test"), []byte("options"))
	require.NoError(t, err)
	require.Len(t, ids, len(testBlobs))
	
	// Verify compression was applied
	assert.True(t, mockServer.compressionDetected, 
		"Compression should be detected in submitted blobs with options")
}

// TestCommitWithCompression tests that Commit handles compression
func TestCommitWithCompression(t *testing.T) {
	logger := zerolog.Nop()
	
	// Create API with compression enabled
	api := &API{
		Logger:             logger,
		compressionEnabled: true,
		compressionConfig: compression.Config{
			Enabled:             true,
			ZstdLevel:           3,
			MinCompressionRatio: 0.1,
		},
	}
	
	compressionDetected := false
	
	// Mock the internal commit function
	api.Internal.Commit = func(ctx context.Context, blobs []da.Blob, ns []byte) ([]da.Commitment, error) {
		// Check if blobs are compressed
		for _, blob := range blobs {
			if len(blob) >= compression.CompressionHeaderSize {
				info := compression.GetCompressionInfo(blob)
				if info.IsCompressed || blob[0] == compression.FlagUncompressed {
					compressionDetected = true
					break
				}
			}
		}
		
		// Return mock commitments
		commitments := make([]da.Commitment, len(blobs))
		for i := range blobs {
			commitments[i] = da.Commitment([]byte{byte(i)})
		}
		return commitments, nil
	}
	
	// Test data
	testBlobs := []da.Blob{
		bytes.Repeat([]byte("commit this "), 100),
	}
	
	ctx := context.Background()
	commitments, err := api.Commit(ctx, testBlobs, []byte("test"))
	require.NoError(t, err)
	require.Len(t, commitments, len(testBlobs))
	
	// Verify compression was applied
	assert.True(t, compressionDetected, 
		"Compression should be detected in committed blobs")
}