package da

import (
	"fmt"

	"github.com/klauspost/compress/zstd"
)

// magic is the 4-byte prefix prepended to all compressed blobs.
// ASCII "ZSTD" = 0x5A 0x53 0x54 0x44.
var magic = []byte{0x5A, 0x53, 0x54, 0x44}

// CompressionLevel controls the speed/ratio trade-off for blob compression.
type CompressionLevel int

const (
	// LevelFastest prioritizes speed over compression ratio.
	// Use when backlog is high and throughput matters most.
	LevelFastest CompressionLevel = iota
	// LevelDefault balances speed and compression ratio.
	LevelDefault
	// LevelBest prioritizes compression ratio over speed.
	// Use when backlog is low to save bandwidth and storage.
	LevelBest
)

// encoders holds one zstd encoder per compression level. Each is safe for
// concurrent use per the klauspost/compress documentation.
var encoders [3]*zstd.Encoder

// decoder is a package-level singleton, safe for concurrent use.
var decoder *zstd.Decoder

func init() {
	levels := [3]zstd.EncoderLevel{
		zstd.SpeedFastest,
		zstd.SpeedDefault,
		zstd.SpeedBestCompression,
	}
	for i, lvl := range levels {
		enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(lvl))
		if err != nil {
			panic(fmt.Sprintf("compression: create zstd encoder (level %d): %v", i, err))
		}
		encoders[i] = enc
	}
	var err error
	decoder, err = zstd.NewReader(nil)
	if err != nil {
		panic(fmt.Sprintf("compression: create zstd decoder: %v", err))
	}
}

// Compress compresses data using zstd at the given level and prepends the magic prefix.
// Returns the original data unchanged if it is empty.
func Compress(data []byte, level CompressionLevel) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	if level < LevelFastest || level > LevelBest {
		level = LevelDefault
	}

	compressed := encoders[level].EncodeAll(data, nil)

	// Prepend magic prefix
	result := make([]byte, len(magic)+len(compressed))
	copy(result, magic)
	copy(result[len(magic):], compressed)

	return result, nil
}

// Decompress decompresses data that was compressed with Compress.
// If the data does not have the magic prefix, it is returned as-is
// (backward-compatible passthrough for uncompressed blobs).
func Decompress(data []byte) ([]byte, error) {
	if !IsCompressed(data) {
		return data, nil
	}

	// Strip magic prefix and decompress
	decompressed, err := decoder.DecodeAll(data[len(magic):], nil)
	if err != nil {
		return nil, fmt.Errorf("compression: zstd decompress: %w", err)
	}

	return decompressed, nil
}

// IsCompressed reports whether data starts with the compression magic prefix.
func IsCompressed(data []byte) bool {
	if len(data) < len(magic) {
		return false
	}
	return data[0] == magic[0] &&
		data[1] == magic[1] &&
		data[2] == magic[2] &&
		data[3] == magic[3]
}
