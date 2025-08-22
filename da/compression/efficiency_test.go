package compression

import (
	"bytes"
	"testing"
	"time"

	"github.com/evstack/ev-node/core/da"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBatchCompression tests the batch compression functions
func TestBatchCompression(t *testing.T) {
	// Create test data
	testBlobs := []da.Blob{
		bytes.Repeat([]byte("compressible "), 100),  // Should compress
		bytes.Repeat([]byte("a"), 1000),             // Highly compressible
		[]byte("small"),                              // Small blob
		make([]byte, 100),                           // Random data (may not compress)
	}
	
	// Fill random data
	for i := range testBlobs[3] {
		testBlobs[3][i] = byte(i * 7 % 256)
	}

	t.Run("CompressBatch", func(t *testing.T) {
		compressed, err := CompressBatch(testBlobs)
		require.NoError(t, err)
		require.Len(t, compressed, len(testBlobs))
		
		// Verify each blob has a header
		for i, blob := range compressed {
			require.GreaterOrEqual(t, len(blob), CompressionHeaderSize, 
				"Blob %d should have compression header", i)
		}
	})

	t.Run("RoundTrip", func(t *testing.T) {
		// Compress
		compressed, err := CompressBatch(testBlobs)
		require.NoError(t, err)
		
		// Decompress
		decompressed, err := DecompressBatch(compressed)
		require.NoError(t, err)
		require.Len(t, decompressed, len(testBlobs))
		
		// Verify data integrity
		for i, original := range testBlobs {
			assert.Equal(t, original, decompressed[i], 
				"Blob %d should match after round trip", i)
		}
	})

	t.Run("EmptyBatch", func(t *testing.T) {
		compressed, err := CompressBatch([]da.Blob{})
		require.NoError(t, err)
		require.Empty(t, compressed)
		
		decompressed, err := DecompressBatch([]da.Blob{})
		require.NoError(t, err)
		require.Empty(t, decompressed)
	})

	t.Run("MixedCompressionResults", func(t *testing.T) {
		compressed, err := CompressBatch(testBlobs)
		require.NoError(t, err)
		
		// Check compression info for each blob
		for i, blob := range compressed {
			info := GetCompressionInfo(blob)
			t.Logf("Blob %d: Compressed=%v, Ratio=%.3f", 
				i, info.IsCompressed, info.CompressionRatio)
		}
	})
}

// BenchmarkHelperEfficiency compares the performance of single vs batch operations
func BenchmarkHelperEfficiency(b *testing.B) {
	// Create test data
	numBlobs := 10
	testBlobs := make([]da.Blob, numBlobs)
	for i := range testBlobs {
		testBlobs[i] = bytes.Repeat([]byte("test data "), 100)
	}

	b.Run("Single_Compress", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, blob := range testBlobs {
				_, err := CompressBlob(blob)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})

	b.Run("Batch_Compress", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := CompressBatch(testBlobs)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Pre-compress for decompression benchmarks
	compressedBlobs, _ := CompressBatch(testBlobs)

	b.Run("Single_Decompress", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, blob := range compressedBlobs {
				_, err := DecompressBlob(blob)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})

	b.Run("Batch_Decompress", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := DecompressBatch(compressedBlobs)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// TestHelperCompressorSingleton verifies the helper compressor is properly initialized
func TestHelperCompressorSingleton(t *testing.T) {
	// Get helper instance
	helper1 := getHelperCompressor()
	require.NotNil(t, helper1)
	require.NotNil(t, helper1.encoder)
	require.NotNil(t, helper1.decoder)
	
	// Get again - should be same instance
	helper2 := getHelperCompressor()
	assert.Same(t, helper1, helper2, "Should return same singleton instance")
}

// TestConcurrentHelperUsage tests thread safety of the helper compressor
func TestConcurrentHelperUsage(t *testing.T) {
	testData := bytes.Repeat([]byte("concurrent test "), 50)
	
	// Run concurrent compressions
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			compressed, err := CompressBlob(testData)
			require.NoError(t, err)
			
			decompressed, err := DecompressBlob(compressed)
			require.NoError(t, err)
			require.Equal(t, testData, decompressed)
			
			done <- true
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// BenchmarkPoolOverhead measures the overhead of pool operations
func BenchmarkPoolOverhead(b *testing.B) {
	b.Run("GetPut_Encoder", func(b *testing.B) {
		initPools()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			encoder := getEncoder(DefaultZstdLevel)
			putEncoder(encoder, DefaultZstdLevel)
		}
	})

	b.Run("GetPut_Decoder", func(b *testing.B) {
		initPools()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			decoder := getDecoder()
			putDecoder(decoder)
		}
	})

	b.Run("Helper_Lock_Unlock", func(b *testing.B) {
		helper := getHelperCompressor()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			helper.mu.Lock()
			helper.mu.Unlock()
		}
	})
}

// TestMemoryAllocationOptimization verifies the header allocation optimization
func TestMemoryAllocationOptimization(t *testing.T) {
	testData := []byte("test data for header")
	originalSize := uint64(len(testData))
	flag := uint8(FlagZstd)

	// Test instance method
	config := DefaultConfig()
	compressor, err := NewCompressibleDA(nil, config)
	require.NoError(t, err)
	defer compressor.Close()

	start := time.Now()
	for i := 0; i < 10000; i++ {
		_ = compressor.addCompressionHeader(testData, flag, originalSize)
	}
	instanceTime := time.Since(start)

	// Test standalone function
	start = time.Now()
	for i := 0; i < 10000; i++ {
		_ = addCompressionHeaderStandalone(testData, flag, originalSize)
	}
	standaloneTime := time.Since(start)

	t.Logf("Instance method: %v", instanceTime)
	t.Logf("Standalone function: %v", standaloneTime)
	
	// Both should be similarly fast with optimized allocation
	ratio := float64(instanceTime) / float64(standaloneTime)
	assert.InDelta(t, 1.0, ratio, 0.5, 
		"Both methods should have similar performance after optimization")
}