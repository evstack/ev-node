package compression

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlgorithm_String(t *testing.T) {
	tests := []struct {
		algorithm Algorithm
		expected  string
	}{
		{None, "none"},
		{Gzip, "gzip"},
		{LZ4, "lz4"},
		{Zstd, "zstd"},
		{Algorithm(999), "unknown"},
	}
	
	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result := test.algorithm.String()
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestParseAlgorithm(t *testing.T) {
	tests := []struct {
		input       string
		expected    Algorithm
		expectError bool
	}{
		{"none", None, false},
		{"gzip", Gzip, false},
		{"lz4", LZ4, false},
		{"zstd", Zstd, false},
		{"invalid", None, true},
		{"", None, true},
	}
	
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result, err := ParseAlgorithm(test.input)
			
			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, result)
			}
		})
	}
}

func TestCompressorOptions(t *testing.T) {
	compressor, err := NewCompressor(
		WithGzipLevel(9),
		WithLZ4Level(12),
		WithZstdLevel(22),
		WithMinCompressionRatio(0.2),
	)
	require.NoError(t, err)
	
	assert.Equal(t, 9, compressor.config.GzipLevel)
	assert.Equal(t, 12, compressor.config.LZ4Level)
	assert.Equal(t, 22, compressor.config.ZstdLevel)
	assert.Equal(t, 0.2, compressor.config.MinCompressionRatio)
}

func TestCompression_AllAlgorithms(t *testing.T) {
	ctx := context.Background()
	compressor, err := NewCompressor()
	require.NoError(t, err)
	
	testData := []byte("Hello, World! This is a test string that should compress well when repeated. " +
		"Hello, World! This is a test string that should compress well when repeated. " +
		"Hello, World! This is a test string that should compress well when repeated.")
	
	algorithms := []Algorithm{None, Gzip, LZ4, Zstd}
	
	for _, algorithm := range algorithms {
		t.Run(algorithm.String(), func(t *testing.T) {
			// Compress
			compressedBlob, err := compressor.Compress(ctx, testData, algorithm)
			require.NoError(t, err)
			assert.NotNil(t, compressedBlob)
			assert.Equal(t, algorithm, compressedBlob.Algorithm)
			assert.Equal(t, uint32(len(testData)), compressedBlob.OriginalSize)
			
			// Decompress
			decompressedData, err := compressor.Decompress(ctx, compressedBlob)
			require.NoError(t, err)
			assert.Equal(t, testData, decompressedData)
			
			// Test compression ratio
			ratio := compressor.GetCompressionRatio(compressedBlob)
			if algorithm == None {
				assert.Equal(t, 1.0, ratio)
			} else {
				assert.True(t, ratio > 0 && ratio <= 1.0, "compression ratio should be between 0 and 1")
			}
		})
	}
}

func TestCompression_IsCompressed(t *testing.T) {
	ctx := context.Background()
	compressor, err := NewCompressor()
	require.NoError(t, err)
	
	testData := []byte("test data for compression detection")
	
	// Test uncompressed data
	assert.False(t, compressor.IsCompressed(testData))
	
	// Test compressed data
	compressedBlob, err := compressor.Compress(ctx, testData, Gzip)
	require.NoError(t, err)
	
	if compressedBlob.Algorithm != None {
		assert.True(t, compressor.IsCompressed(compressedBlob.Data))
	}
}

func TestCompression_ParseCompressedBlob(t *testing.T) {
	ctx := context.Background()
	compressor, err := NewCompressor()
	require.NoError(t, err)
	
	testData := []byte("test data for blob parsing")
	
	// Test with compressed data
	originalBlob, err := compressor.Compress(ctx, testData, Zstd)
	require.NoError(t, err)
	
	parsedBlob, err := compressor.ParseCompressedBlob(originalBlob.Data)
	require.NoError(t, err)
	
	assert.Equal(t, originalBlob.Algorithm, parsedBlob.Algorithm)
	assert.Equal(t, originalBlob.OriginalSize, parsedBlob.OriginalSize)
	assert.Equal(t, originalBlob.Data, parsedBlob.Data)
	
	// Test with uncompressed data
	parsedUncompressed, err := compressor.ParseCompressedBlob(testData)
	require.NoError(t, err)
	
	assert.Equal(t, None, parsedUncompressed.Algorithm)
	assert.Equal(t, uint32(len(testData)), parsedUncompressed.OriginalSize)
	assert.Equal(t, testData, parsedUncompressed.Data)
}

func TestCompression_MinCompressionRatio(t *testing.T) {
	ctx := context.Background()
	
	// Create compressor with high minimum compression ratio
	compressor, err := NewCompressor(WithMinCompressionRatio(0.9))
	require.NoError(t, err)
	
	// Random data that won't compress well
	randomData := make([]byte, 1000)
	for i := range randomData {
		randomData[i] = byte(i % 256)
	}
	
	// Should return uncompressed due to poor compression ratio
	compressedBlob, err := compressor.Compress(ctx, randomData, Gzip)
	require.NoError(t, err)
	
	// Should fallback to None if compression wasn't worthwhile
	assert.True(t, compressedBlob.Algorithm == None || compressor.GetCompressionRatio(compressedBlob) < 0.9)
}

func TestCompression_EdgeCases(t *testing.T) {
	ctx := context.Background()
	compressor, err := NewCompressor()
	require.NoError(t, err)
	
	// Empty data
	emptyData := []byte{}
	compressedEmpty, err := compressor.Compress(ctx, emptyData, Gzip)
	require.NoError(t, err)
	
	decompressedEmpty, err := compressor.Decompress(ctx, compressedEmpty)
	require.NoError(t, err)
	assert.Equal(t, emptyData, decompressedEmpty)
	
	// Single byte
	singleByte := []byte{42}
	compressedSingle, err := compressor.Compress(ctx, singleByte, LZ4)
	require.NoError(t, err)
	
	decompressedSingle, err := compressor.Decompress(ctx, compressedSingle)
	require.NoError(t, err)
	assert.Equal(t, singleByte, decompressedSingle)
}

func TestCompression_InvalidData(t *testing.T) {
	ctx := context.Background()
	compressor, err := NewCompressor()
	require.NoError(t, err)
	
	// Test invalid compressed blob (too short)
	invalidBlob := &CompressedBlob{
		Algorithm:    Gzip,
		OriginalSize: 100,
		Data:        []byte{1, 2, 3}, // Too short
	}
	
	_, err = compressor.Decompress(ctx, invalidBlob)
	assert.Error(t, err)
	
	// Test unsupported algorithm
	_, err = compressor.Compress(ctx, []byte("test"), Algorithm(999))
	assert.Error(t, err)
}

func BenchmarkCompression_Gzip(b *testing.B) {
	benchmarkCompressionAlgorithm(b, Gzip)
}

func BenchmarkCompression_LZ4(b *testing.B) {
	benchmarkCompressionAlgorithm(b, LZ4)
}

func BenchmarkCompression_Zstd(b *testing.B) {
	benchmarkCompressionAlgorithm(b, Zstd)
}

func benchmarkCompressionAlgorithm(b *testing.B, algorithm Algorithm) {
	ctx := context.Background()
	compressor, err := NewCompressor()
	require.NoError(b, err)
	
	testData := make([]byte, 10240) // 10KB test data
	for i := range testData {
		testData[i] = byte(i % 100) // Somewhat compressible
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		compressedBlob, err := compressor.Compress(ctx, testData, algorithm)
		if err != nil {
			b.Fatal(err)
		}
		
		_, err = compressor.Decompress(ctx, compressedBlob)
		if err != nil {
			b.Fatal(err)
		}
	}
}