package compression

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/evstack/ev-node/core/da"
)

// CompressibleDA wraps an existing DA implementation to add compression support
type CompressibleDA struct {
	da.DA
	compressor *DefaultCompressor
	algorithm  Algorithm
	enabled    bool
}

// CompressibleDAConfig holds configuration for the compressible DA
type CompressibleDAConfig struct {
	// Enabled controls whether compression is active
	Enabled bool `json:"enabled"`
	// Algorithm specifies which compression algorithm to use
	Algorithm string `json:"algorithm"`
	// GzipLevel for gzip compression (1-9)
	GzipLevel int `json:"gzip_level,omitempty"`
	// LZ4Level for lz4 compression (1-12, 0 for fast mode)
	LZ4Level int `json:"lz4_level,omitempty"`
	// ZstdLevel for zstd compression (1-22)
	ZstdLevel int `json:"zstd_level,omitempty"`
	// MinCompressionRatio minimum ratio to keep compressed data
	MinCompressionRatio float64 `json:"min_compression_ratio,omitempty"`
}

// NewCompressibleDA creates a new compressible DA wrapper
func NewCompressibleDA(baseDA da.DA, config CompressibleDAConfig) (*CompressibleDA, error) {
	if !config.Enabled {
		// Return a pass-through wrapper when compression is disabled
		return &CompressibleDA{
			DA:      baseDA,
			enabled: false,
		}, nil
	}
	
	algorithm, err := ParseAlgorithm(config.Algorithm)
	if err != nil {
		return nil, fmt.Errorf("invalid compression algorithm: %w", err)
	}
	
	// Set up compressor options
	var opts []CompressorOption
	if config.GzipLevel > 0 {
		opts = append(opts, WithGzipLevel(config.GzipLevel))
	}
	if config.LZ4Level > 0 {
		opts = append(opts, WithLZ4Level(config.LZ4Level))
	}
	if config.ZstdLevel > 0 {
		opts = append(opts, WithZstdLevel(config.ZstdLevel))
	}
	if config.MinCompressionRatio > 0 {
		opts = append(opts, WithMinCompressionRatio(config.MinCompressionRatio))
	}
	
	compressor, err := NewCompressor(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create compressor: %w", err)
	}
	
	return &CompressibleDA{
		DA:         baseDA,
		compressor: compressor,
		algorithm:  algorithm,
		enabled:    true,
	}, nil
}

// Submit compresses blobs before submitting them to the underlying DA
func (cda *CompressibleDA) Submit(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte) ([]da.ID, error) {
	if !cda.enabled {
		return cda.DA.Submit(ctx, blobs, gasPrice, namespace)
	}
	
	compressedBlobs := make([]da.Blob, len(blobs))
	
	for i, blob := range blobs {
		compressedBlob, err := cda.compressor.Compress(ctx, blob, cda.algorithm)
		if err != nil {
			return nil, fmt.Errorf("failed to compress blob %d: %w", i, err)
		}
		compressedBlobs[i] = compressedBlob.Data
	}
	
	return cda.DA.Submit(ctx, compressedBlobs, gasPrice, namespace)
}

// SubmitWithOptions compresses blobs and includes compression metadata in options
func (cda *CompressibleDA) SubmitWithOptions(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte, options []byte) ([]da.ID, error) {
	if !cda.enabled {
		return cda.DA.SubmitWithOptions(ctx, blobs, gasPrice, namespace, options)
	}
	
	compressedBlobs := make([]da.Blob, len(blobs))
	compressionMetadata := make([]CompressionMetadata, len(blobs))
	
	for i, blob := range blobs {
		compressedBlob, err := cda.compressor.Compress(ctx, blob, cda.algorithm)
		if err != nil {
			return nil, fmt.Errorf("failed to compress blob %d: %w", i, err)
		}
		
		compressedBlobs[i] = compressedBlob.Data
		compressionMetadata[i] = CompressionMetadata{
			Algorithm:    compressedBlob.Algorithm,
			OriginalSize: compressedBlob.OriginalSize,
			Compressed:   compressedBlob.Algorithm != None,
		}
	}
	
	// Merge compression metadata with existing options
	enhancedOptions, err := cda.mergeOptions(options, compressionMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to merge compression options: %w", err)
	}
	
	return cda.DA.SubmitWithOptions(ctx, compressedBlobs, gasPrice, namespace, enhancedOptions)
}

// Get retrieves and decompresses blobs from the underlying DA
func (cda *CompressibleDA) Get(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Blob, error) {
	compressedBlobs, err := cda.DA.Get(ctx, ids, namespace)
	if err != nil {
		return nil, err
	}
	
	if !cda.enabled {
		return compressedBlobs, nil
	}
	
	decompressedBlobs := make([]da.Blob, len(compressedBlobs))
	
	for i, compressedBlob := range compressedBlobs {
		// Parse the compressed blob
		parsedBlob, err := cda.compressor.ParseCompressedBlob(compressedBlob)
		if err != nil {
			return nil, fmt.Errorf("failed to parse compressed blob %d: %w", i, err)
		}
		
		// Decompress
		decompressedData, err := cda.compressor.Decompress(ctx, parsedBlob)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress blob %d: %w", i, err)
		}
		
		decompressedBlobs[i] = decompressedData
	}
	
	return decompressedBlobs, nil
}

// CompressionMetadata holds metadata about compressed blobs
type CompressionMetadata struct {
	Algorithm    Algorithm `json:"algorithm"`
	OriginalSize uint32    `json:"original_size"`
	Compressed   bool      `json:"compressed"`
}

// CompressibleDAOptions represents the structure of enhanced options with compression metadata
type CompressibleDAOptions struct {
	// Original options (if any)
	Original json.RawMessage `json:"original,omitempty"`
	// Compression metadata for each blob
	Compression []CompressionMetadata `json:"compression,omitempty"`
}

// mergeOptions combines existing options with compression metadata
func (cda *CompressibleDA) mergeOptions(originalOptions []byte, compressionMetadata []CompressionMetadata) ([]byte, error) {
	var options CompressibleDAOptions
	
	if len(originalOptions) > 0 {
		options.Original = json.RawMessage(originalOptions)
	}
	
	options.Compression = compressionMetadata
	
	return json.Marshal(options)
}

// IsCompressionEnabled returns whether compression is enabled
func (cda *CompressibleDA) IsCompressionEnabled() bool {
	return cda.enabled
}

// GetCompressionAlgorithm returns the configured compression algorithm
func (cda *CompressibleDA) GetCompressionAlgorithm() Algorithm {
	if !cda.enabled {
		return None
	}
	return cda.algorithm
}

// GetCompressionStats returns statistics about compression for the given blobs
func (cda *CompressibleDA) GetCompressionStats(ctx context.Context, blobs []da.Blob) ([]CompressionStats, error) {
	if !cda.enabled {
		return nil, fmt.Errorf("compression is not enabled")
	}
	
	stats := make([]CompressionStats, len(blobs))
	
	for i, blob := range blobs {
		compressedBlob, err := cda.compressor.Compress(ctx, blob, cda.algorithm)
		if err != nil {
			return nil, fmt.Errorf("failed to compress blob %d for stats: %w", i, err)
		}
		
		stats[i] = CompressionStats{
			OriginalSize:     len(blob),
			CompressedSize:   len(compressedBlob.Data),
			CompressionRatio: cda.compressor.GetCompressionRatio(compressedBlob),
			Algorithm:        compressedBlob.Algorithm,
			SavingsBytes:     len(blob) - len(compressedBlob.Data),
			SavingsPercent:   (1.0 - cda.compressor.GetCompressionRatio(compressedBlob)) * 100,
		}
	}
	
	return stats, nil
}

// CompressionStats provides statistics about compression for a blob
type CompressionStats struct {
	OriginalSize     int       `json:"original_size"`
	CompressedSize   int       `json:"compressed_size"`
	CompressionRatio float64   `json:"compression_ratio"`
	Algorithm        Algorithm `json:"algorithm"`
	SavingsBytes     int       `json:"savings_bytes"`
	SavingsPercent   float64   `json:"savings_percent"`
}