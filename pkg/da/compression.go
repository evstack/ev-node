package da

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/klauspost/compress/zstd"
)

// magic is the 4-byte prefix prepended to all compressed blobs.
// ASCII "ZSTD" = 0x5A 0x53 0x54 0x44.
var magic = []byte{0x5A, 0x53, 0x54, 0x44}

// maxDecompressedSize is the maximum allowed decompressed output size.
// Matches the WithDecoderMaxMemory cap and provides early rejection
// by inspecting the zstd frame header before allocating anything.
const maxDecompressedSize = 7 * 1024 * 1024 // 7 MiB

// decompressTimeout is the hard wall-clock cap on a single decompression.
// This guards against CPU-based decompression bombs (crafted inputs that
// are slow to decode) independently of the caller's context deadline.
const decompressTimeout = 500 * time.Millisecond

// ErrDecompressedSizeExceeded is returned when the zstd frame header
// declares a decompressed size that exceeds the allowed limit.
var ErrDecompressedSizeExceeded = errors.New("compression: declared decompressed size exceeds limit")

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
	const maxDecoderMemory = 7 * 1024 * 1024 // 7 MiB cap
	var err error
	decoder, err = zstd.NewReader(nil, zstd.WithDecoderMaxMemory(maxDecoderMemory))
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

// decodeResult carries the output of a DecodeAll goroutine.
type decodeResult struct {
	data []byte
	err  error
}

// Decompress decompresses data that was compressed with Compress.
// If the data does not have the magic prefix, it is returned as-is
// (backward-compatible passthrough for uncompressed blobs).
func Decompress(ctx context.Context, data []byte) ([]byte, error) {
	if !IsCompressed(data) {
		return data, nil
	}

	payload := data[len(magic):]

	// Layer 1: Parse frame header to check declared decompressed size
	// before allocating anything. This is a zero-cost upfront rejection.
	var hdr zstd.Header
	if err := hdr.Decode(payload); err == nil && hdr.HasFCS {
		if hdr.FrameContentSize > maxDecompressedSize {
			return nil, fmt.Errorf("%w: %d bytes declared, %d allowed",
				ErrDecompressedSizeExceeded, hdr.FrameContentSize, maxDecompressedSize)
		}
	}

	// Layer 3: Apply the shorter of caller deadline and our hard cap.
	ctx, cancel := context.WithTimeout(ctx, decompressTimeout)
	defer cancel()

	ch := make(chan decodeResult, 1)
	go func() {
		out, err := decoder.DecodeAll(payload, nil)
		ch <- decodeResult{data: out, err: err}
	}()

	select {
	case res := <-ch:
		if res.err != nil {
			return nil, fmt.Errorf("zstd decompress: %w", res.err)
		}
		return res.data, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("zstd decompress timeout: %w", ctx.Err())
	}
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
