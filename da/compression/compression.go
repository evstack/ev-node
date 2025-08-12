package compression

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/evstack/ev-node/core/da"
	"github.com/klauspost/compress/zstd"
)

// Compression constants
const (
	// CompressionHeaderSize is the size of the compression metadata header
	CompressionHeaderSize = 9 // 1 byte flags + 8 bytes original size

	// Compression levels
	DefaultZstdLevel = 3

	// Flags
	FlagUncompressed = 0x00
	FlagZstd         = 0x01

	// Default minimum compression ratio threshold (10% savings)
	DefaultMinCompressionRatio = 0.1
)

var (
	ErrInvalidHeader          = errors.New("invalid compression header")
	ErrInvalidCompressionFlag = errors.New("invalid compression flag")
	ErrDecompressionFailed    = errors.New("decompression failed")
)

// Config holds compression configuration
type Config struct {
	// Enabled controls whether compression is active
	Enabled bool

	// ZstdLevel is the compression level for zstd (1-22, default 3)
	ZstdLevel int

	// MinCompressionRatio is the minimum compression ratio required to store compressed data
	// If compression doesn't achieve this ratio, original data is stored uncompressed
	MinCompressionRatio float64
}

// DefaultConfig returns a configuration optimized for zstd level 3
func DefaultConfig() Config {
	return Config{
		Enabled:             true,
		ZstdLevel:           DefaultZstdLevel,
		MinCompressionRatio: DefaultMinCompressionRatio,
	}
}

// Global sync.Pools for encoder/decoder reuse
var (
	encoderPools map[int]*sync.Pool
	decoderPool  *sync.Pool
	poolsOnce    sync.Once
)

// initPools initializes the encoder and decoder pools
func initPools() {
	poolsOnce.Do(func() {
		// Create encoder pools for different compression levels
		encoderPools = make(map[int]*sync.Pool)
		
		// Pre-create pools for common compression levels (1-9)
		for level := 1; level <= 9; level++ {
			lvl := level // Capture loop variable
			encoderPools[lvl] = &sync.Pool{
				New: func() interface{} {
					encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(lvl)))
					if err != nil {
						// This should not happen with valid levels
						panic(fmt.Sprintf("failed to create zstd encoder with level %d: %v", lvl, err))
					}
					return encoder
				},
			}
		}
		
		// Create decoder pool
		decoderPool = &sync.Pool{
			New: func() interface{} {
				decoder, err := zstd.NewReader(nil)
				if err != nil {
					// This should not happen
					panic(fmt.Sprintf("failed to create zstd decoder: %v", err))
				}
				return decoder
			},
		}
	})
}

// getEncoder retrieves an encoder from the pool for the specified compression level
func getEncoder(level int) *zstd.Encoder {
	initPools()
	
	pool, exists := encoderPools[level]
	if !exists {
		// Create a new pool for this level if it doesn't exist
		pool = &sync.Pool{
			New: func() interface{} {
				encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(level)))
				if err != nil {
					panic(fmt.Sprintf("failed to create zstd encoder with level %d: %v", level, err))
				}
				return encoder
			},
		}
		encoderPools[level] = pool
	}
	
	return pool.Get().(*zstd.Encoder)
}

// putEncoder returns an encoder to the pool
func putEncoder(encoder *zstd.Encoder, level int) {
	if encoder == nil {
		return
	}
	
	// Reset the encoder for reuse
	encoder.Reset(nil)
	
	if pool, exists := encoderPools[level]; exists {
		pool.Put(encoder)
	}
}

// getDecoder retrieves a decoder from the pool
func getDecoder() *zstd.Decoder {
	initPools()
	return decoderPool.Get().(*zstd.Decoder)
}

// putDecoder returns a decoder to the pool
func putDecoder(decoder *zstd.Decoder) {
	if decoder == nil {
		return
	}
	
	// Reset the decoder for reuse
	decoder.Reset(nil)
	decoderPool.Put(decoder)
}

// CompressibleDA wraps a DA implementation to add transparent compression support
type CompressibleDA struct {
	baseDA  da.DA
	config  Config
	encoder *zstd.Encoder
	decoder *zstd.Decoder
}

// NewCompressibleDA creates a new CompressibleDA wrapper
func NewCompressibleDA(baseDA da.DA, config Config) (*CompressibleDA, error) {
	// Allow nil baseDA for testing purposes (when only using compression functions)
	// The baseDA will only be used when calling Submit, Get, GetIDs methods

	var encoder *zstd.Encoder
	var decoder *zstd.Decoder
	var err error

	if config.Enabled {
		// Create zstd encoder with specified level
		encoder, err = zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(config.ZstdLevel)))
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd encoder: %w", err)
		}

		// Create zstd decoder
		decoder, err = zstd.NewReader(nil)
		if err != nil {
			encoder.Close()
			return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
		}
	}

	return &CompressibleDA{
		baseDA:  baseDA,
		config:  config,
		encoder: encoder,
		decoder: decoder,
	}, nil
}

// Close cleans up compression resources
func (c *CompressibleDA) Close() error {
	if c.encoder != nil {
		c.encoder.Close()
	}
	if c.decoder != nil {
		c.decoder.Close()
	}
	return nil
}

// compressBlob compresses a single blob using zstd
func (c *CompressibleDA) compressBlob(blob da.Blob) (da.Blob, error) {
	if !c.config.Enabled || len(blob) == 0 {
		return c.addCompressionHeader(blob, FlagUncompressed, uint64(len(blob))), nil
	}

	// Compress the blob
	compressed := c.encoder.EncodeAll(blob, make([]byte, 0, len(blob)))

	// Check if compression is beneficial
	compressionRatio := float64(len(compressed)) / float64(len(blob))
	if compressionRatio > (1.0 - c.config.MinCompressionRatio) {
		// Compression not beneficial, store uncompressed
		return c.addCompressionHeader(blob, FlagUncompressed, uint64(len(blob))), nil
	}

	return c.addCompressionHeader(compressed, FlagZstd, uint64(len(blob))), nil
}

// decompressBlob decompresses a single blob
func (c *CompressibleDA) decompressBlob(compressedBlob da.Blob) (da.Blob, error) {
	if len(compressedBlob) < CompressionHeaderSize {
		// Assume legacy uncompressed blob
		return compressedBlob, nil
	}

	flag, originalSize, payload, err := c.parseCompressionHeader(compressedBlob)
	if err != nil {
		// Assume legacy uncompressed blob
		return compressedBlob, nil
	}

	switch flag {
	case FlagUncompressed:
		return payload, nil
	case FlagZstd:
		if !c.config.Enabled {
			return nil, errors.New("received compressed blob but compression is disabled")
		}

		decompressed, err := c.decoder.DecodeAll(payload, make([]byte, 0, originalSize))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrDecompressionFailed, err)
		}

		if uint64(len(decompressed)) != originalSize {
			return nil, fmt.Errorf("decompressed size mismatch: expected %d, got %d", originalSize, len(decompressed))
		}

		return decompressed, nil
	default:
		return nil, fmt.Errorf("%w: flag %d", ErrInvalidCompressionFlag, flag)
	}
}

// addCompressionHeader adds compression metadata to the blob
func (c *CompressibleDA) addCompressionHeader(payload da.Blob, flag uint8, originalSize uint64) da.Blob {
	header := make([]byte, CompressionHeaderSize)
	header[0] = flag
	binary.LittleEndian.PutUint64(header[1:9], originalSize)

	result := make([]byte, CompressionHeaderSize+len(payload))
	copy(result, header)
	copy(result[CompressionHeaderSize:], payload)

	return result
}

// parseCompressionHeader extracts compression metadata from a blob
func (c *CompressibleDA) parseCompressionHeader(blob da.Blob) (uint8, uint64, da.Blob, error) {
	if len(blob) < CompressionHeaderSize {
		return 0, 0, nil, ErrInvalidHeader
	}

	flag := blob[0]
	originalSize := binary.LittleEndian.Uint64(blob[1:9])
	payload := blob[CompressionHeaderSize:]

	return flag, originalSize, payload, nil
}

// DA interface implementation - these methods pass through to the base DA with compression

// Get retrieves and decompresses blobs
func (c *CompressibleDA) Get(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Blob, error) {
	compressedBlobs, err := c.baseDA.Get(ctx, ids, namespace)
	if err != nil {
		return nil, err
	}

	blobs := make([]da.Blob, len(compressedBlobs))
	for i, compressedBlob := range compressedBlobs {
		blob, err := c.decompressBlob(compressedBlob)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress blob at index %d: %w", i, err)
		}
		blobs[i] = blob
	}

	return blobs, nil
}

// Submit compresses and submits blobs
func (c *CompressibleDA) Submit(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte) ([]da.ID, error) {
	compressedBlobs := make([]da.Blob, len(blobs))
	for i, blob := range blobs {
		compressedBlob, err := c.compressBlob(blob)
		if err != nil {
			return nil, fmt.Errorf("failed to compress blob at index %d: %w", i, err)
		}
		compressedBlobs[i] = compressedBlob
	}

	return c.baseDA.Submit(ctx, compressedBlobs, gasPrice, namespace)
}

// SubmitWithOptions compresses and submits blobs with options
func (c *CompressibleDA) SubmitWithOptions(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte, options []byte) ([]da.ID, error) {
	compressedBlobs := make([]da.Blob, len(blobs))
	for i, blob := range blobs {
		compressedBlob, err := c.compressBlob(blob)
		if err != nil {
			return nil, fmt.Errorf("failed to compress blob at index %d: %w", i, err)
		}
		compressedBlobs[i] = compressedBlob
	}

	return c.baseDA.SubmitWithOptions(ctx, compressedBlobs, gasPrice, namespace, options)
}

// Commit creates commitments for compressed blobs
func (c *CompressibleDA) Commit(ctx context.Context, blobs []da.Blob, namespace []byte) ([]da.Commitment, error) {
	compressedBlobs := make([]da.Blob, len(blobs))
	for i, blob := range blobs {
		compressedBlob, err := c.compressBlob(blob)
		if err != nil {
			return nil, fmt.Errorf("failed to compress blob at index %d: %w", i, err)
		}
		compressedBlobs[i] = compressedBlob
	}

	return c.baseDA.Commit(ctx, compressedBlobs, namespace)
}

// Pass-through methods (no compression needed)

func (c *CompressibleDA) GetIDs(ctx context.Context, height uint64, namespace []byte) (*da.GetIDsResult, error) {
	return c.baseDA.GetIDs(ctx, height, namespace)
}

func (c *CompressibleDA) GetProofs(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Proof, error) {
	return c.baseDA.GetProofs(ctx, ids, namespace)
}

func (c *CompressibleDA) Validate(ctx context.Context, ids []da.ID, proofs []da.Proof, namespace []byte) ([]bool, error) {
	return c.baseDA.Validate(ctx, ids, proofs, namespace)
}

func (c *CompressibleDA) GasPrice(ctx context.Context) (float64, error) {
	return c.baseDA.GasPrice(ctx)
}

func (c *CompressibleDA) GasMultiplier(ctx context.Context) (float64, error) {
	return c.baseDA.GasMultiplier(ctx)
}

// Helper functions for external use

// CompressBlob compresses a blob using the default zstd level 3 configuration
func CompressBlob(blob da.Blob) (da.Blob, error) {
	config := DefaultConfig()
	
	if !config.Enabled || len(blob) == 0 {
		// Return with uncompressed header
		return addCompressionHeaderStandalone(blob, FlagUncompressed, uint64(len(blob))), nil
	}
	
	// Get encoder from pool
	encoder := getEncoder(config.ZstdLevel)
	defer putEncoder(encoder, config.ZstdLevel)
	
	// Compress the blob
	compressed := encoder.EncodeAll(blob, make([]byte, 0, len(blob)))
	
	// Check if compression is beneficial
	compressionRatio := float64(len(compressed)) / float64(len(blob))
	if compressionRatio > (1.0 - config.MinCompressionRatio) {
		// Compression not beneficial, store uncompressed
		return addCompressionHeaderStandalone(blob, FlagUncompressed, uint64(len(blob))), nil
	}
	
	return addCompressionHeaderStandalone(compressed, FlagZstd, uint64(len(blob))), nil
}

// DecompressBlob decompresses a blob
func DecompressBlob(compressedBlob da.Blob) (da.Blob, error) {
	if len(compressedBlob) < CompressionHeaderSize {
		// Assume legacy uncompressed blob
		return compressedBlob, nil
	}
	
	flag, originalSize, payload, err := parseCompressionHeaderStandalone(compressedBlob)
	if err != nil {
		// Assume legacy uncompressed blob
		return compressedBlob, nil
	}
	
	switch flag {
	case FlagUncompressed:
		return payload, nil
	case FlagZstd:
		// Get decoder from pool
		decoder := getDecoder()
		defer putDecoder(decoder)
		
		decompressed, err := decoder.DecodeAll(payload, make([]byte, 0, originalSize))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrDecompressionFailed, err)
		}
		
		if uint64(len(decompressed)) != originalSize {
			return nil, fmt.Errorf("decompressed size mismatch: expected %d, got %d", originalSize, len(decompressed))
		}
		
		return decompressed, nil
	default:
		return nil, fmt.Errorf("unsupported compression flag: %d", flag)
	}
}

// Standalone helper functions for use without CompressibleDA instance

// addCompressionHeaderStandalone adds compression metadata header to data
func addCompressionHeaderStandalone(data []byte, flag uint8, originalSize uint64) []byte {
	header := make([]byte, CompressionHeaderSize)
	header[0] = flag
	binary.BigEndian.PutUint64(header[1:], originalSize)
	return append(header, data...)
}

// parseCompressionHeaderStandalone parses compression metadata from blob
func parseCompressionHeaderStandalone(blob []byte) (flag uint8, originalSize uint64, payload []byte, err error) {
	if len(blob) < CompressionHeaderSize {
		return 0, 0, nil, errors.New("blob too small for compression header")
	}
	
	flag = blob[0]
	originalSize = binary.BigEndian.Uint64(blob[1:9])
	payload = blob[CompressionHeaderSize:]
	
	return flag, originalSize, payload, nil
}

// CompressionInfo provides information about a blob's compression
type CompressionInfo struct {
	IsCompressed     bool
	Algorithm        string
	OriginalSize     uint64
	CompressedSize   uint64
	CompressionRatio float64
}

// GetCompressionInfo analyzes a blob to determine its compression status
func GetCompressionInfo(blob da.Blob) CompressionInfo {
	info := CompressionInfo{
		IsCompressed:   false,
		Algorithm:      "none",
		OriginalSize:   uint64(len(blob)),
		CompressedSize: uint64(len(blob)),
	}

	if len(blob) < CompressionHeaderSize {
		return info
	}

	flag := blob[0]
	originalSize := binary.LittleEndian.Uint64(blob[1:9])
	payloadSize := uint64(len(blob) - CompressionHeaderSize)

	switch flag {
	case FlagZstd:
		info.IsCompressed = true
		info.Algorithm = "zstd"
		info.OriginalSize = originalSize
		info.CompressedSize = payloadSize
		if originalSize > 0 {
			info.CompressionRatio = float64(payloadSize) / float64(originalSize)
		}
	case FlagUncompressed:
		info.Algorithm = "none"
		info.OriginalSize = originalSize
		info.CompressedSize = payloadSize
	}

	return info
}
