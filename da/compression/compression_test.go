package compression

import (
	"bytes"
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/evstack/ev-node/core/da"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDA implements a simple in-memory DA for testing
type mockDA struct {
	blobs map[string]da.Blob
	ids   []da.ID
}

func newMockDA() *mockDA {
	return &mockDA{
		blobs: make(map[string]da.Blob),
		ids:   make([]da.ID, 0),
	}
}

func (m *mockDA) Get(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Blob, error) {
	blobs := make([]da.Blob, len(ids))
	for i, id := range ids {
		blob, exists := m.blobs[string(id)]
		if !exists {
			return nil, da.ErrBlobNotFound
		}
		blobs[i] = blob
	}
	return blobs, nil
}

func (m *mockDA) Submit(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte) ([]da.ID, error) {
	ids := make([]da.ID, len(blobs))
	for i, blob := range blobs {
		id := da.ID([]byte{byte(len(m.ids))})
		m.blobs[string(id)] = blob
		m.ids = append(m.ids, id)
		ids[i] = id
	}
	return ids, nil
}

func (m *mockDA) SubmitWithOptions(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte, options []byte) ([]da.ID, error) {
	return m.Submit(ctx, blobs, gasPrice, namespace)
}

func (m *mockDA) GetIDs(ctx context.Context, height uint64, namespace []byte) (*da.GetIDsResult, error) {
	return &da.GetIDsResult{IDs: m.ids, Timestamp: time.Now()}, nil
}

func (m *mockDA) GetProofs(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Proof, error) {
	proofs := make([]da.Proof, len(ids))
	for i := range ids {
		proofs[i] = da.Proof("mock_proof")
	}
	return proofs, nil
}

func (m *mockDA) Commit(ctx context.Context, blobs []da.Blob, namespace []byte) ([]da.Commitment, error) {
	commitments := make([]da.Commitment, len(blobs))
	for i, blob := range blobs {
		commitments[i] = da.Commitment(blob[:min(len(blob), 32)])
	}
	return commitments, nil
}

func (m *mockDA) Validate(ctx context.Context, ids []da.ID, proofs []da.Proof, namespace []byte) ([]bool, error) {
	results := make([]bool, len(ids))
	for i := range ids {
		results[i] = true
	}
	return results, nil
}

func (m *mockDA) GasPrice(ctx context.Context) (float64, error) {
	return 1.0, nil
}

func (m *mockDA) GasMultiplier(ctx context.Context) (float64, error) {
	return 1.0, nil
}

func TestCompression_ZstdLevel3(t *testing.T) {
	config := Config{
		Enabled:             true,
		ZstdLevel:           3,
		MinCompressionRatio: 0.1,
	}

	compressor, err := NewCompressibleDA(nil, config)
	require.NoError(t, err)
	defer compressor.Close()

	// Test with compressible data
	originalData := bytes.Repeat([]byte("hello world "), 100)

	compressed, err := compressor.compressBlob(originalData)
	require.NoError(t, err)

	// Check that compression header is present
	require.GreaterOrEqual(t, len(compressed), CompressionHeaderSize)

	// Verify compression flag
	flag := compressed[0]
	assert.Equal(t, uint8(FlagZstd), flag)

	// Decompress and verify
	decompressed, err := compressor.decompressBlob(compressed)
	require.NoError(t, err)
	assert.Equal(t, originalData, decompressed)
}

func TestCompression_UncompressedFallback(t *testing.T) {
	config := Config{
		Enabled:             true,
		ZstdLevel:           3,
		MinCompressionRatio: 0.1,
	}

	compressor, err := NewCompressibleDA(nil, config)
	require.NoError(t, err)
	defer compressor.Close()

	// Generate random data that won't compress well
	randomData := make([]byte, 100)
	_, err = rand.Read(randomData)
	require.NoError(t, err)

	compressed, err := compressor.compressBlob(randomData)
	require.NoError(t, err)

	// Should use uncompressed flag
	flag := compressed[0]
	assert.Equal(t, uint8(FlagUncompressed), flag)

	// Decompress and verify
	decompressed, err := compressor.decompressBlob(compressed)
	require.NoError(t, err)
	assert.Equal(t, randomData, decompressed)
}

func TestCompression_DisabledMode(t *testing.T) {
	config := Config{
		Enabled:             false,
		ZstdLevel:           3,
		MinCompressionRatio: 0.1,
	}

	compressor, err := NewCompressibleDA(nil, config)
	require.NoError(t, err)
	defer compressor.Close()

	originalData := bytes.Repeat([]byte("test data "), 50)

	compressed, err := compressor.compressBlob(originalData)
	require.NoError(t, err)

	// Should use uncompressed flag when disabled
	flag := compressed[0]
	assert.Equal(t, uint8(FlagUncompressed), flag)

	decompressed, err := compressor.decompressBlob(compressed)
	require.NoError(t, err)
	assert.Equal(t, originalData, decompressed)
}

func TestCompression_LegacyBlobs(t *testing.T) {
	config := DefaultConfig()
	compressor, err := NewCompressibleDA(nil, config)
	require.NoError(t, err)
	defer compressor.Close()

	// Test with legacy blob (no compression header)
	legacyBlob := []byte("legacy data without header")

	// Should return as-is
	decompressed, err := compressor.decompressBlob(legacyBlob)
	require.NoError(t, err)
	assert.Equal(t, legacyBlob, decompressed)
}

func TestCompression_ErrorCases(t *testing.T) {
	t.Run("nil base DA", func(t *testing.T) {
		_, err := NewCompressibleDA(nil, DefaultConfig())
		assert.Error(t, err)
	})

	t.Run("invalid compression flag", func(t *testing.T) {
		config := DefaultConfig()
		compressor, err := NewCompressibleDA(nil, config)
		require.NoError(t, err)
		defer compressor.Close()

		// Create blob with invalid flag
		invalidBlob := make([]byte, CompressionHeaderSize+10)
		invalidBlob[0] = 0xFF // Invalid flag

		_, err = compressor.decompressBlob(invalidBlob)
		assert.ErrorIs(t, err, ErrInvalidCompressionFlag)
	})
}

func TestCompressionInfo(t *testing.T) {
	config := DefaultConfig()
	compressor, err := NewCompressibleDA(nil, config)
	require.NoError(t, err)
	defer compressor.Close()

	// Test with compressible data
	originalData := bytes.Repeat([]byte("compress me "), 100)

	compressed, err := compressor.compressBlob(originalData)
	require.NoError(t, err)

	info := GetCompressionInfo(compressed)
	assert.True(t, info.IsCompressed)
	assert.Equal(t, "zstd", info.Algorithm)
	assert.Equal(t, uint64(len(originalData)), info.OriginalSize)
	assert.Less(t, info.CompressionRatio, 1.0)
	assert.Greater(t, info.CompressionRatio, 0.0)
}

func TestHelperFunctions(t *testing.T) {
	originalData := bytes.Repeat([]byte("test "), 100)

	// Test standalone compress function
	compressed, err := CompressBlob(originalData)
	require.NoError(t, err)

	// Test standalone decompress function
	decompressed, err := DecompressBlob(compressed)
	require.NoError(t, err)

	assert.Equal(t, originalData, decompressed)
}
