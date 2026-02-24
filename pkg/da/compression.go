package da

import (
	"fmt"

	"github.com/klauspost/compress/zstd"
)

// magic is the 4-byte prefix prepended to all compressed blobs.
// ASCII "ZSTD" = 0x5A 0x53 0x54 0x44.
var magic = []byte{0x5A, 0x53, 0x54, 0x44}

// encoder and decoder are package-level singletons. They are safe for
// concurrent use per the klauspost/compress documentation.
var (
	encoder *zstd.Encoder
	decoder *zstd.Decoder
)

func init() {
	var err error
	encoder, err = zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		panic(fmt.Sprintf("compression: create zstd encoder: %v", err))
	}
	decoder, err = zstd.NewReader(nil)
	if err != nil {
		panic(fmt.Sprintf("compression: create zstd decoder: %v", err))
	}
}

// Compress compresses data using zstd and prepends the magic prefix.
// Returns the original data unchanged if it is empty.
func Compress(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	compressed := encoder.EncodeAll(data, nil)

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
		return nil, fmt.Errorf("compression: zstd decompress failed: %w", err)
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
