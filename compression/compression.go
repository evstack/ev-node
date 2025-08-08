package compression

import (
	"context"
	"errors"
)

// Algorithm represents a compression algorithm type
type Algorithm uint8

const (
	// None represents no compression
	None Algorithm = iota
	// Gzip uses gzip compression for maximum compatibility
	Gzip
	// LZ4 uses LZ4 compression for fast compression/decompression
	LZ4
	// Zstd uses Zstandard compression for balanced performance
	Zstd
)

// String returns the string representation of the algorithm
func (a Algorithm) String() string {
	switch a {
	case None:
		return "none"
	case Gzip:
		return "gzip"
	case LZ4:
		return "lz4"
	case Zstd:
		return "zstd"
	default:
		return "unknown"
	}
}

// ParseAlgorithm parses a string into an Algorithm
func ParseAlgorithm(s string) (Algorithm, error) {
	switch s {
	case "none":
		return None, nil
	case "gzip":
		return Gzip, nil
	case "lz4":
		return LZ4, nil
	case "zstd":
		return Zstd, nil
	default:
		return None, errors.New("unknown compression algorithm")
	}
}

// CompressedBlob represents a blob with compression metadata
type CompressedBlob struct {
	// Algorithm used for compression
	Algorithm Algorithm
	// OriginalSize of the uncompressed data
	OriginalSize uint32
	// Data contains the compressed blob data
	Data []byte
}

// Compressor defines the interface for blob compression operations
type Compressor interface {
	// Compress compresses the input data using the specified algorithm
	Compress(ctx context.Context, data []byte, algorithm Algorithm) (*CompressedBlob, error)
	
	// Decompress decompresses the compressed blob back to original data
	Decompress(ctx context.Context, blob *CompressedBlob) ([]byte, error)
	
	// IsCompressed checks if the given blob data appears to be compressed
	IsCompressed(data []byte) bool
	
	// GetCompressionRatio returns the compression ratio for the given compressed blob
	GetCompressionRatio(blob *CompressedBlob) float64
}

// CompressorOption allows configuration of the compressor
type CompressorOption func(*CompressorConfig)

// CompressorConfig holds configuration for the compressor
type CompressorConfig struct {
	// GzipLevel for gzip compression (1-9, 6 is default)
	GzipLevel int
	// LZ4Level for lz4 compression (1-12, 0 is fast mode)
	LZ4Level int
	// ZstdLevel for zstd compression (1-22, 3 is default)
	ZstdLevel int
	// MinCompressionRatio minimum ratio required to keep compressed data
	MinCompressionRatio float64
}

// WithGzipLevel sets the gzip compression level
func WithGzipLevel(level int) CompressorOption {
	return func(c *CompressorConfig) {
		c.GzipLevel = level
	}
}

// WithLZ4Level sets the LZ4 compression level
func WithLZ4Level(level int) CompressorOption {
	return func(c *CompressorConfig) {
		c.LZ4Level = level
	}
}

// WithZstdLevel sets the Zstd compression level
func WithZstdLevel(level int) CompressorOption {
	return func(c *CompressorConfig) {
		c.ZstdLevel = level
	}
}

// WithMinCompressionRatio sets minimum compression ratio
func WithMinCompressionRatio(ratio float64) CompressorOption {
	return func(c *CompressorConfig) {
		c.MinCompressionRatio = ratio
	}
}

// BenchmarkResult contains performance metrics for compression algorithms
type BenchmarkResult struct {
	Algorithm        Algorithm
	OriginalSize     int
	CompressedSize   int
	CompressionRatio float64
	CompressTime     int64 // nanoseconds
	DecompressTime   int64 // nanoseconds
	CompressionRate  float64 // MB/s
	DecompressionRate float64 // MB/s
}