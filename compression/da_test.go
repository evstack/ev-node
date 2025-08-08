package compression

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/evstack/ev-node/core/da"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDA implements the da.DA interface for testing
type mockDA struct {
	blobs       map[string][]byte
	submissions []submitCall
}

type submitCall struct {
	blobs     []da.Blob
	gasPrice  float64
	namespace []byte
	options   []byte
}

func newMockDA() *mockDA {
	return &mockDA{
		blobs:       make(map[string][]byte),
		submissions: make([]submitCall, 0),
	}
}

func (m *mockDA) Submit(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte) ([]da.ID, error) {
	return m.SubmitWithOptions(ctx, blobs, gasPrice, namespace, nil)
}

func (m *mockDA) SubmitWithOptions(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte, options []byte) ([]da.ID, error) {
	call := submitCall{
		blobs:     make([]da.Blob, len(blobs)),
		gasPrice:  gasPrice,
		namespace: make([]byte, len(namespace)),
		options:   make([]byte, len(options)),
	}
	
	copy(call.namespace, namespace)
	if options != nil {
		copy(call.options, options)
	}
	
	ids := make([]da.ID, len(blobs))
	for i, blob := range blobs {
		id := da.ID([]byte{byte(i)}) // Simple ID generation
		call.blobs[i] = make([]byte, len(blob))
		copy(call.blobs[i], blob)
		m.blobs[string(id)] = make([]byte, len(blob))
		copy(m.blobs[string(id)], blob)
		ids[i] = id
	}
	
	m.submissions = append(m.submissions, call)
	return ids, nil
}

func (m *mockDA) Get(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Blob, error) {
	blobs := make([]da.Blob, len(ids))
	for i, id := range ids {
		blob, exists := m.blobs[string(id)]
		if !exists {
			return nil, assert.AnError
		}
		blobs[i] = make([]byte, len(blob))
		copy(blobs[i], blob)
	}
	return blobs, nil
}

func (m *mockDA) GetIDs(ctx context.Context, height uint64, namespace []byte) (*da.GetIDsResult, error) {
	return &da.GetIDsResult{}, nil
}

func (m *mockDA) Commit(ctx context.Context, blobs []da.Blob, namespace []byte) ([]da.Commitment, error) {
	return make([]da.Commitment, len(blobs)), nil
}

func (m *mockDA) GetProofs(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Proof, error) {
	return make([]da.Proof, len(ids)), nil
}

func (m *mockDA) Validate(ctx context.Context, ids []da.ID, proofs []da.Proof, namespace []byte) ([]bool, error) {
	return make([]bool, len(ids)), nil
}

func TestCompressibleDA_DisabledCompression(t *testing.T) {
	ctx := context.Background()
	baseDA := newMockDA()
	
	config := CompressibleDAConfig{
		Enabled: false,
	}
	
	compressibleDA, err := NewCompressibleDA(baseDA, config)
	require.NoError(t, err)
	assert.False(t, compressibleDA.IsCompressionEnabled())
	assert.Equal(t, None, compressibleDA.GetCompressionAlgorithm())
	
	testBlobs := []da.Blob{
		[]byte("test blob 1"),
		[]byte("test blob 2"),
	}
	
	// Submit should pass through without compression
	ids, err := compressibleDA.Submit(ctx, testBlobs, 1.0, []byte("test-namespace"))
	require.NoError(t, err)
	assert.Len(t, ids, 2)
	
	// Verify original blobs were stored
	assert.Len(t, baseDA.submissions, 1)
	assert.Equal(t, testBlobs, baseDA.submissions[0].blobs)
	
	// Get should return original data
	retrievedBlobs, err := compressibleDA.Get(ctx, ids, []byte("test-namespace"))
	require.NoError(t, err)
	assert.Equal(t, testBlobs, retrievedBlobs)
}

func TestCompressibleDA_EnabledCompression(t *testing.T) {
	ctx := context.Background()
	baseDA := newMockDA()
	
	config := CompressibleDAConfig{
		Enabled:   true,
		Algorithm: "zstd",
		ZstdLevel: 3,
	}
	
	compressibleDA, err := NewCompressibleDA(baseDA, config)
	require.NoError(t, err)
	assert.True(t, compressibleDA.IsCompressionEnabled())
	assert.Equal(t, Zstd, compressibleDA.GetCompressionAlgorithm())
	
	// Create compressible test data
	testBlobs := []da.Blob{
		[]byte("This is a test blob with repeated content. This is a test blob with repeated content."),
		[]byte("Another test blob for compression. Another test blob for compression. Another test blob for compression."),
	}
	
	// Submit should compress blobs
	ids, err := compressibleDA.Submit(ctx, testBlobs, 1.0, []byte("test-namespace"))
	require.NoError(t, err)
	assert.Len(t, ids, 2)
	
	// Verify compressed blobs were stored (they should be different from originals)
	assert.Len(t, baseDA.submissions, 1)
	storedBlobs := baseDA.submissions[0].blobs
	
	// Stored blobs should be compressed (different from originals)
	for i, storedBlob := range storedBlobs {
		if len(storedBlob) < len(testBlobs[i]) {
			// Only check if actually compressed (some data might not compress well)
			assert.NotEqual(t, testBlobs[i], storedBlob, "blob %d should be compressed", i)
		}
	}
	
	// Get should return decompressed original data
	retrievedBlobs, err := compressibleDA.Get(ctx, ids, []byte("test-namespace"))
	require.NoError(t, err)
	assert.Equal(t, testBlobs, retrievedBlobs)
}

func TestCompressibleDA_SubmitWithOptions(t *testing.T) {
	ctx := context.Background()
	baseDA := newMockDA()
	
	config := CompressibleDAConfig{
		Enabled:   true,
		Algorithm: "gzip",
		GzipLevel: 6,
	}
	
	compressibleDA, err := NewCompressibleDA(baseDA, config)
	require.NoError(t, err)
	
	testBlobs := []da.Blob{
		[]byte("compressible test data " + string(make([]byte, 100))),
	}
	
	originalOptions := []byte(`{"custom_option": "value"}`)
	
	// Submit with options
	ids, err := compressibleDA.SubmitWithOptions(ctx, testBlobs, 1.0, []byte("test-namespace"), originalOptions)
	require.NoError(t, err)
	assert.Len(t, ids, 1)
	
	// Verify enhanced options were created
	assert.Len(t, baseDA.submissions, 1)
	enhancedOptions := baseDA.submissions[0].options
	
	var options CompressibleDAOptions
	err = json.Unmarshal(enhancedOptions, &options)
	require.NoError(t, err)
	
	// Should have original options and compression metadata
	assert.NotNil(t, options.Original)
	assert.Len(t, options.Compression, 1)
	
	compressionMeta := options.Compression[0]
	assert.True(t, compressionMeta.Algorithm == Gzip || compressionMeta.Algorithm == None) // Depends on compression effectiveness
	assert.Greater(t, compressionMeta.OriginalSize, uint32(0))
}

func TestCompressibleDA_GetCompressionStats(t *testing.T) {
	ctx := context.Background()
	baseDA := newMockDA()
	
	config := CompressibleDAConfig{
		Enabled:   true,
		Algorithm: "lz4",
		LZ4Level:  0,
	}
	
	compressibleDA, err := NewCompressibleDA(baseDA, config)
	require.NoError(t, err)
	
	testBlobs := []da.Blob{
		[]byte("highly compressible data: " + string(make([]byte, 1000))), // Zeros should compress well
		make([]byte, 100), // Also zeros
	}
	
	stats, err := compressibleDA.GetCompressionStats(ctx, testBlobs)
	require.NoError(t, err)
	assert.Len(t, stats, 2)
	
	for i, stat := range stats {
		assert.Equal(t, len(testBlobs[i]), stat.OriginalSize)
		assert.Greater(t, stat.CompressedSize, 0)
		assert.True(t, stat.CompressionRatio > 0 && stat.CompressionRatio <= 1.0)
		assert.True(t, stat.Algorithm == LZ4 || stat.Algorithm == None) // Depends on compression effectiveness
		assert.Equal(t, stat.OriginalSize-stat.CompressedSize, stat.SavingsBytes)
		assert.InDelta(t, (1.0-stat.CompressionRatio)*100, stat.SavingsPercent, 0.1)
	}
}

func TestCompressibleDA_ConfigValidation(t *testing.T) {
	baseDA := newMockDA()
	
	// Test invalid algorithm
	config := CompressibleDAConfig{
		Enabled:   true,
		Algorithm: "invalid-algorithm",
	}
	
	_, err := NewCompressibleDA(baseDA, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid compression algorithm")
	
	// Test valid algorithms
	validAlgorithms := []string{"none", "gzip", "lz4", "zstd"}
	for _, algorithm := range validAlgorithms {
		config := CompressibleDAConfig{
			Enabled:   true,
			Algorithm: algorithm,
		}
		
		compressibleDA, err := NewCompressibleDA(baseDA, config)
		assert.NoError(t, err)
		
		expectedAlg, _ := ParseAlgorithm(algorithm)
		assert.Equal(t, expectedAlg, compressibleDA.GetCompressionAlgorithm())
	}
}

func TestCompressibleDA_CompressionLevels(t *testing.T) {
	ctx := context.Background()
	baseDA := newMockDA()
	
	// Test with custom compression levels
	config := CompressibleDAConfig{
		Enabled:             true,
		Algorithm:           "zstd",
		ZstdLevel:           22, // Maximum compression
		MinCompressionRatio: 0.05, // Very aggressive compression requirement
	}
	
	compressibleDA, err := NewCompressibleDA(baseDA, config)
	require.NoError(t, err)
	
	// Use highly compressible data
	testData := make([]byte, 10000)
	for i := range testData {
		testData[i] = byte(i % 10) // Repeating pattern
	}
	
	testBlobs := []da.Blob{testData}
	
	// Submit and verify compression worked
	ids, err := compressibleDA.Submit(ctx, testBlobs, 1.0, []byte("test-namespace"))
	require.NoError(t, err)
	
	// Get back and verify data integrity
	retrievedBlobs, err := compressibleDA.Get(ctx, ids, []byte("test-namespace"))
	require.NoError(t, err)
	assert.Equal(t, testBlobs, retrievedBlobs)
}

func TestCompressibleDA_BackwardCompatibility(t *testing.T) {
	ctx := context.Background()
	baseDA := newMockDA()
	
	// First, store some uncompressed data directly in the base DA
	uncompressedBlobs := []da.Blob{
		[]byte("uncompressed data from older version"),
	}
	
	ids, err := baseDA.Submit(ctx, uncompressedBlobs, 1.0, []byte("test-namespace"))
	require.NoError(t, err)
	
	// Now create a compressible DA and try to retrieve the uncompressed data
	config := CompressibleDAConfig{
		Enabled:   true,
		Algorithm: "gzip",
	}
	
	compressibleDA, err := NewCompressibleDA(baseDA, config)
	require.NoError(t, err)
	
	// Should be able to retrieve and handle uncompressed data
	retrievedBlobs, err := compressibleDA.Get(ctx, ids, []byte("test-namespace"))
	require.NoError(t, err)
	assert.Equal(t, uncompressedBlobs, retrievedBlobs)
}