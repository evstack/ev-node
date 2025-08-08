package compression

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
)

// magicBytes is used to identify compressed blobs
var magicBytes = [4]byte{0x45, 0x56, 0x43, 0x5A} // "EVCZ" in hex

// DefaultCompressor implements the Compressor interface
type DefaultCompressor struct {
	config CompressorConfig
	zstdEncoder *zstd.Encoder
	zstdDecoder *zstd.Decoder
}

// NewCompressor creates a new compressor with the given options
func NewCompressor(opts ...CompressorOption) (*DefaultCompressor, error) {
	config := CompressorConfig{
		GzipLevel:           6, // default gzip level
		LZ4Level:            0, // fast mode
		ZstdLevel:           3, // default zstd level
		MinCompressionRatio: 0.1, // must save at least 10%
	}
	
	for _, opt := range opts {
		opt(&config)
	}
	
	zstdEncoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(config.ZstdLevel)))
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd encoder: %w", err)
	}
	
	zstdDecoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
	}
	
	return &DefaultCompressor{
		config:      config,
		zstdEncoder: zstdEncoder,
		zstdDecoder: zstdDecoder,
	}, nil
}

// Compress compresses the input data using the specified algorithm
func (c *DefaultCompressor) Compress(ctx context.Context, data []byte, algorithm Algorithm) (*CompressedBlob, error) {
	if algorithm == None {
		return &CompressedBlob{
			Algorithm:    None,
			OriginalSize: uint32(len(data)),
			Data:        data,
		}, nil
	}
	
	var compressedData []byte
	var err error
	
	switch algorithm {
	case Gzip:
		compressedData, err = c.compressGzip(data)
	case LZ4:
		compressedData, err = c.compressLZ4(data)
	case Zstd:
		compressedData, err = c.compressZstd(data)
	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %s", algorithm)
	}
	
	if err != nil {
		return nil, fmt.Errorf("compression failed: %w", err)
	}
	
	// Check if compression is worthwhile
	ratio := float64(len(compressedData)) / float64(len(data))
	if ratio > (1.0 - c.config.MinCompressionRatio) {
		// Compression didn't save enough space, return uncompressed
		return &CompressedBlob{
			Algorithm:    None,
			OriginalSize: uint32(len(data)),
			Data:        data,
		}, nil
	}
	
	// Create the final compressed blob with header
	finalData := c.createCompressedBlob(algorithm, uint32(len(data)), compressedData)
	
	return &CompressedBlob{
		Algorithm:    algorithm,
		OriginalSize: uint32(len(data)),
		Data:        finalData,
	}, nil
}

// Decompress decompresses the compressed blob back to original data
func (c *DefaultCompressor) Decompress(ctx context.Context, blob *CompressedBlob) ([]byte, error) {
	if blob.Algorithm == None {
		return blob.Data, nil
	}
	
	// Extract the compressed data (skip header)
	if len(blob.Data) < 9 { // magic(4) + algorithm(1) + originalSize(4)
		return nil, errors.New("invalid compressed blob: too short")
	}
	
	compressedData := blob.Data[9:]
	
	switch blob.Algorithm {
	case Gzip:
		return c.decompressGzip(compressedData)
	case LZ4:
		return c.decompressLZ4(compressedData, int(blob.OriginalSize))
	case Zstd:
		return c.decompressZstd(compressedData)
	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %s", blob.Algorithm)
	}
}

// IsCompressed checks if the given blob data appears to be compressed
func (c *DefaultCompressor) IsCompressed(data []byte) bool {
	if len(data) < 9 { // magic(4) + algorithm(1) + originalSize(4)
		return false
	}
	
	return bytes.Equal(data[:4], magicBytes[:])
}

// GetCompressionRatio returns the compression ratio for the given compressed blob
func (c *DefaultCompressor) GetCompressionRatio(blob *CompressedBlob) float64 {
	if blob.Algorithm == None {
		return 1.0
	}
	return float64(len(blob.Data)) / float64(blob.OriginalSize)
}

// createCompressedBlob creates a blob with compression header
func (c *DefaultCompressor) createCompressedBlob(algorithm Algorithm, originalSize uint32, compressedData []byte) []byte {
	buf := make([]byte, 9+len(compressedData))
	
	// Magic bytes
	copy(buf[0:4], magicBytes[:])
	
	// Algorithm
	buf[4] = byte(algorithm)
	
	// Original size (big endian)
	binary.BigEndian.PutUint32(buf[5:9], originalSize)
	
	// Compressed data
	copy(buf[9:], compressedData)
	
	return buf
}

// parseCompressedBlob parses a compressed blob from raw data
func (c *DefaultCompressor) parseCompressedBlob(data []byte) (*CompressedBlob, error) {
	if !c.IsCompressed(data) {
		// Not compressed, return as-is
		return &CompressedBlob{
			Algorithm:    None,
			OriginalSize: uint32(len(data)),
			Data:        data,
		}, nil
	}
	
	algorithm := Algorithm(data[4])
	originalSize := binary.BigEndian.Uint32(data[5:9])
	
	return &CompressedBlob{
		Algorithm:    algorithm,
		OriginalSize: originalSize,
		Data:        data,
	}, nil
}

// compressGzip compresses data using gzip
func (c *DefaultCompressor) compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, c.config.GzipLevel)
	if err != nil {
		return nil, err
	}
	
	_, err = writer.Write(data)
	if err != nil {
		return nil, err
	}
	
	err = writer.Close()
	if err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

// decompressGzip decompresses gzip data
func (c *DefaultCompressor) decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	
	return io.ReadAll(reader)
}

// compressLZ4 compresses data using LZ4
func (c *DefaultCompressor) compressLZ4(data []byte) ([]byte, error) {
	buf := make([]byte, lz4.CompressBlockBound(len(data)))
	
	var compressedSize int
	var err error
	
	if c.config.LZ4Level == 0 {
		// Fast mode
		compressedSize, err = lz4.CompressBlock(data, buf, nil)
	} else {
		// High compression mode
		compressedSize, err = lz4.CompressBlockHC(data, buf, c.config.LZ4Level)
	}
	
	if err != nil {
		return nil, err
	}
	
	return buf[:compressedSize], nil
}

// decompressLZ4 decompresses LZ4 data
func (c *DefaultCompressor) decompressLZ4(data []byte, originalSize int) ([]byte, error) {
	buf := make([]byte, originalSize)
	decompressedSize, err := lz4.UncompressBlock(data, buf)
	if err != nil {
		return nil, err
	}
	
	if decompressedSize != originalSize {
		return nil, fmt.Errorf("decompressed size mismatch: expected %d, got %d", originalSize, decompressedSize)
	}
	
	return buf, nil
}

// compressZstd compresses data using Zstandard
func (c *DefaultCompressor) compressZstd(data []byte) ([]byte, error) {
	return c.zstdEncoder.EncodeAll(data, make([]byte, 0, len(data))), nil
}

// decompressZstd decompresses Zstandard data
func (c *DefaultCompressor) decompressZstd(data []byte) ([]byte, error) {
	return c.zstdDecoder.DecodeAll(data, nil)
}

// ParseCompressedBlob parses a compressed blob from raw data
func (c *DefaultCompressor) ParseCompressedBlob(data []byte) (*CompressedBlob, error) {
	return c.parseCompressedBlob(data)
}