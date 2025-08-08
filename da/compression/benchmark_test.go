package compression

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/evstack/ev-node/core/da"
)

// Benchmark compression performance with different data types
func BenchmarkZstdCompression(b *testing.B) {
	config := DefaultConfig()
	compressor, err := NewCompressibleDA(nil, config)
	if err != nil {
		b.Fatal(err)
	}
	defer compressor.Close()

	testCases := []struct {
		name string
		data da.Blob
	}{
		{
			name: "Repetitive_1KB",
			data: bytes.Repeat([]byte("hello world "), 85), // ~1KB
		},
		{
			name: "Repetitive_10KB", 
			data: bytes.Repeat([]byte("The quick brown fox jumps over the lazy dog. "), 227), // ~10KB
		},
		{
			name: "JSON_1KB",
			data: []byte(`{"id":1,"name":"user_1","data":"` + string(bytes.Repeat([]byte("x"), 900)) + `","timestamp":1234567890}`),
		},
		{
			name: "Random_1KB",
			data: func() da.Blob {
				data := make([]byte, 1024)
				rand.Read(data)
				return data
			}(),
		},
	}

	for _, tc := range testCases {
		b.Run("Compress_"+tc.name, func(b *testing.B) {
			b.ResetTimer()
			b.SetBytes(int64(len(tc.data)))
			for i := 0; i < b.N; i++ {
				_, err := compressor.compressBlob(tc.data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		// Benchmark decompression
		compressed, err := compressor.compressBlob(tc.data)
		if err != nil {
			b.Fatal(err)
		}

		b.Run("Decompress_"+tc.name, func(b *testing.B) {
			b.ResetTimer()
			b.SetBytes(int64(len(tc.data)))
			for i := 0; i < b.N; i++ {
				_, err := compressor.decompressBlob(compressed)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Benchmark helper functions
func BenchmarkCompressBlob(b *testing.B) {
	data := bytes.Repeat([]byte("benchmark data "), 64) // ~1KB
	b.SetBytes(int64(len(data)))
	
	for i := 0; i < b.N; i++ {
		_, err := CompressBlob(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecompressBlob(b *testing.B) {
	data := bytes.Repeat([]byte("benchmark data "), 64) // ~1KB
	compressed, err := CompressBlob(data)
	if err != nil {
		b.Fatal(err)
	}
	
	b.SetBytes(int64(len(data)))
	
	for i := 0; i < b.N; i++ {
		_, err := DecompressBlob(compressed)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark end-to-end DA operations
func BenchmarkCompressibleDA_Submit(b *testing.B) {
	mockDA := newMockDA()
	config := DefaultConfig()
	
	compressibleDA, err := NewCompressibleDA(mockDA, config)
	if err != nil {
		b.Fatal(err)
	}
	defer compressibleDA.Close()
	
	testBlobs := []da.Blob{
		bytes.Repeat([]byte("test data "), 100),
		bytes.Repeat([]byte("more data "), 100), 
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := compressibleDA.Submit(nil, testBlobs, 1.0, []byte("test"))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompressibleDA_Get(b *testing.B) {
	mockDA := newMockDA()
	config := DefaultConfig()
	
	compressibleDA, err := NewCompressibleDA(mockDA, config)
	if err != nil {
		b.Fatal(err)
	}
	defer compressibleDA.Close()
	
	testBlobs := []da.Blob{
		bytes.Repeat([]byte("test data "), 100),
		bytes.Repeat([]byte("more data "), 100),
	}
	
	// Submit first
	ids, err := compressibleDA.Submit(nil, testBlobs, 1.0, []byte("test"))
	if err != nil {
		b.Fatal(err)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := compressibleDA.Get(nil, ids, []byte("test"))
		if err != nil {
			b.Fatal(err)
		}
	}
}