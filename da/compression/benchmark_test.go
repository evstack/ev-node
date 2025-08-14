package compression

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/evstack/ev-node/core/da"
	"github.com/stretchr/testify/require"
)

// TestLargeBlobCompressionEfficiency tests compression efficiency for blob sizes from 20KB to 2MB
func TestLargeBlobCompressionEfficiency(t *testing.T) {
	config := DefaultConfig()
	compressor, err := NewCompressibleDA(nil, config)
	require.NoError(t, err)
	defer compressor.Close()

	// Test sizes: 20KB, 50KB, 100KB, 200KB, 500KB, 1MB, 2MB
	testSizes := []int{
		20 * 1024,   // 20KB
		50 * 1024,   // 50KB
		100 * 1024,  // 100KB
		200 * 1024,  // 200KB
		500 * 1024,  // 500KB
		1024 * 1024, // 1MB
		2048 * 1024, // 2MB
	}

	dataTypes := []struct {
		name      string
		generator func(size int) da.Blob
	}{
		{
			name:      "Repetitive",
			generator: generateRepetitiveData,
		},
		{
			name:      "JSON",
			generator: generateJSONData,
		},
		{
			name:      "Text",
			generator: generateTextData,
		},
		{
			name:      "Binary",
			generator: generateBinaryData,
		},
		{
			name:      "Random",
			generator: generateRandomData,
		},
	}

	fmt.Printf("\n=== Large Blob Compression Efficiency Test ===\n")
	fmt.Printf("%-15s %-10s %-12s %-12s %-10s %-15s\n",
		"Data Type", "Size", "Compressed", "Saved", "Ratio", "Compression")
	fmt.Printf("%-15s %-10s %-12s %-12s %-10s %-15s\n",
		"---------", "----", "----------", "-----", "-----", "-----------")

	for _, dt := range dataTypes {
		for _, size := range testSizes {
			data := dt.generator(size)
			compressed, err := compressor.compressBlob(data)
			require.NoError(t, err)

			info := GetCompressionInfo(compressed)

			var saved string
			var compressionStatus string

			if info.IsCompressed {
				savedPercent := (1.0 - info.CompressionRatio) * 100
				saved = fmt.Sprintf("%.1f%%", savedPercent)
				compressionStatus = "Yes"
			} else {
				saved = "0%"
				compressionStatus = "No (inefficient)"
			}

			fmt.Printf("%-15s %-10s %-12s %-12s %-10.3f %-15s\n",
				dt.name,
				formatSize(size),
				formatSize(int(info.CompressedSize)),
				saved,
				info.CompressionRatio,
				compressionStatus,
			)

			// Verify decompression works correctly
			decompressed, err := compressor.decompressBlob(compressed)
			require.NoError(t, err)
			require.Equal(t, data, decompressed, "Decompressed data should match original")
		}
		fmt.Println() // Add spacing between data types
	}
}

// BenchmarkLargeBlobCompression benchmarks compression performance for large blobs
func BenchmarkLargeBlobCompression(b *testing.B) {
	config := DefaultConfig()
	compressor, err := NewCompressibleDA(nil, config)
	if err != nil {
		b.Fatal(err)
	}
	defer compressor.Close()

	benchmarkSizes := []int{
		20 * 1024,   // 20KB
		100 * 1024,  // 100KB
		500 * 1024,  // 500KB
		1024 * 1024, // 1MB
		2048 * 1024, // 2MB
	}

	for _, size := range benchmarkSizes {
		// Benchmark with different data types
		b.Run(fmt.Sprintf("JSON_%s", formatSize(size)), func(b *testing.B) {
			data := generateJSONData(size)
			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := compressor.compressBlob(data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("Text_%s", formatSize(size)), func(b *testing.B) {
			data := generateTextData(size)
			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := compressor.compressBlob(data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("Binary_%s", formatSize(size)), func(b *testing.B) {
			data := generateBinaryData(size)
			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := compressor.compressBlob(data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkLargeBlobDecompression benchmarks decompression performance
func BenchmarkLargeBlobDecompression(b *testing.B) {
	config := DefaultConfig()
	compressor, err := NewCompressibleDA(nil, config)
	if err != nil {
		b.Fatal(err)
	}
	defer compressor.Close()

	benchmarkSizes := []int{
		20 * 1024,   // 20KB
		100 * 1024,  // 100KB
		500 * 1024,  // 500KB
		1024 * 1024, // 1MB
		2048 * 1024, // 2MB
	}

	for _, size := range benchmarkSizes {
		// Pre-compress data for decompression benchmark
		jsonData := generateJSONData(size)
		compressedJSON, _ := compressor.compressBlob(jsonData)

		textData := generateTextData(size)
		compressedText, _ := compressor.compressBlob(textData)

		binaryData := generateBinaryData(size)
		compressedBinary, _ := compressor.compressBlob(binaryData)

		b.Run(fmt.Sprintf("JSON_%s", formatSize(size)), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := compressor.decompressBlob(compressedJSON)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("Text_%s", formatSize(size)), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := compressor.decompressBlob(compressedText)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("Binary_%s", formatSize(size)), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := compressor.decompressBlob(compressedBinary)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// TestCompressionThresholds tests the MinCompressionRatio threshold behavior
func TestCompressionThresholds(t *testing.T) {
	testCases := []struct {
		name                string
		minCompressionRatio float64
		dataSize            int
		dataType            func(int) da.Blob
		expectCompressed    bool
	}{
		{
			name:                "High_Threshold_Repetitive_Data",
			minCompressionRatio: 0.5, // Require 50% savings
			dataSize:            100 * 1024,
			dataType:            generateRepetitiveData,
			expectCompressed:    true, // Repetitive data should achieve >50% savings
		},
		{
			name:                "High_Threshold_Random_Data",
			minCompressionRatio: 0.5,
			dataSize:            100 * 1024,
			dataType:            generateRandomData,
			expectCompressed:    false, // Random data won't achieve 50% savings
		},
		{
			name:                "Default_Threshold_JSON",
			minCompressionRatio: 0.1, // Default 10% savings
			dataSize:            500 * 1024,
			dataType:            generateJSONData,
			expectCompressed:    true, // JSON should achieve >10% savings
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := Config{
				Enabled:             true,
				ZstdLevel:           3,
				MinCompressionRatio: tc.minCompressionRatio,
			}

			compressor, err := NewCompressibleDA(nil, config)
			require.NoError(t, err)
			defer compressor.Close()

			data := tc.dataType(tc.dataSize)
			compressed, err := compressor.compressBlob(data)
			require.NoError(t, err)

			info := GetCompressionInfo(compressed)

			if tc.expectCompressed {
				require.True(t, info.IsCompressed,
					"Expected data to be compressed with threshold %.2f, but it wasn't. Ratio: %.3f",
					tc.minCompressionRatio, info.CompressionRatio)
			} else {
				require.False(t, info.IsCompressed,
					"Expected data to NOT be compressed with threshold %.2f, but it was. Ratio: %.3f",
					tc.minCompressionRatio, info.CompressionRatio)
			}
		})
	}
}

// Data generation functions
func generateRepetitiveData(size int) da.Blob {
	pattern := []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	data := make([]byte, 0, size)
	for len(data) < size {
		remaining := size - len(data)
		if remaining >= len(pattern) {
			data = append(data, pattern...)
		} else {
			data = append(data, pattern[:remaining]...)
		}
	}
	return data
}

func generateJSONData(size int) da.Blob {
	// Generate realistic JSON data with nested structures
	type Record struct {
		ID          int                    `json:"id"`
		Name        string                 `json:"name"`
		Email       string                 `json:"email"`
		Active      bool                   `json:"active"`
		Score       float64                `json:"score"`
		Tags        []string               `json:"tags"`
		Metadata    map[string]interface{} `json:"metadata"`
		Description string                 `json:"description"`
	}

	records := make([]Record, 0)
	currentSize := 0
	id := 0

	for currentSize < size {
		record := Record{
			ID:     id,
			Name:   fmt.Sprintf("User_%d", id),
			Email:  fmt.Sprintf("user%d@example.com", id),
			Active: id%2 == 0,
			Score:  float64(id) * 1.5,
			Tags:   []string{"tag1", "tag2", fmt.Sprintf("tag_%d", id)},
			Metadata: map[string]interface{}{
				"created_at": "2024-01-01",
				"updated_at": "2024-01-02",
				"version":    id,
			},
			Description: fmt.Sprintf("This is a description for record %d with some repetitive content to simulate real data", id),
		}

		records = append(records, record)

		// Estimate size
		tempData, _ := json.Marshal(records)
		currentSize = len(tempData)
		id++

		if currentSize >= size {
			break
		}
	}

	data, _ := json.Marshal(records)
	if len(data) > size {
		data = data[:size]
	}
	return data
}

func generateTextData(size int) da.Blob {
	// Generate natural language text
	sentences := []string{
		"The quick brown fox jumps over the lazy dog.",
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
		"In a hole in the ground there lived a hobbit.",
		"It was the best of times, it was the worst of times.",
		"To be or not to be, that is the question.",
		"All happy families are alike; each unhappy family is unhappy in its own way.",
		"It is a truth universally acknowledged that a single man in possession of a good fortune must be in want of a wife.",
		"The sun did not shine, it was too wet to play.",
	}

	data := make([]byte, 0, size)
	sentenceIndex := 0

	for len(data) < size {
		sentence := sentences[sentenceIndex%len(sentences)]
		if len(data)+len(sentence)+1 <= size {
			if len(data) > 0 {
				data = append(data, ' ')
			}
			data = append(data, sentence...)
		} else {
			remaining := size - len(data)
			if remaining > 0 {
				data = append(data, sentence[:remaining]...)
			}
			break
		}
		sentenceIndex++
	}

	return data
}

func generateBinaryData(size int) da.Blob {
	// Generate semi-structured binary data (like compiled code or encrypted data)
	data := make([]byte, size)

	// Add some structure to make it somewhat compressible
	for i := 0; i < size; i += 256 {
		// Header-like structure
		if i+4 <= size {
			data[i] = 0xDE
			data[i+1] = 0xAD
			data[i+2] = 0xBE
			data[i+3] = 0xEF
		}

		// Some repetitive patterns
		for j := 4; j < 128 && i+j < size; j++ {
			data[i+j] = byte(j % 256)
		}

		// Some random data
		if i+128 < size {
			randSection := make([]byte, min(128, size-i-128))
			rand.Read(randSection)
			copy(data[i+128:], randSection)
		}
	}

	return data
}

func generateRandomData(size int) da.Blob {
	data := make([]byte, size)
	rand.Read(data)
	return data
}

func formatSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%dKB", bytes/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(bytes)/1024/1024)
}
