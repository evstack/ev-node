package da

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompressDecompress_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "small payload",
			data: []byte("hello world, this is a test payload for compression"),
		},
		{
			name: "protobuf-like repeated data",
			data: bytes.Repeat([]byte{0x0a, 0x10, 0x08, 0x01, 0x12, 0x0c}, 1000),
		},
		{
			name: "single byte",
			data: []byte{0xFF},
		},
		{
			name: "1MB payload",
			data: bytes.Repeat([]byte("rollkit blob compression test data "), 30000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, err := Compress(tt.data, LevelDefault)
			require.NoError(t, err)

			assert.True(t, IsCompressed(compressed), "compressed data should have magic prefix")

			decompressed, err := Decompress(compressed)
			require.NoError(t, err)

			assert.Equal(t, tt.data, decompressed, "round-trip should preserve data")
		})
	}
}

func TestCompress_AllLevelsRoundTrip(t *testing.T) {
	data := bytes.Repeat([]byte("adaptive compression level test "), 5000)
	levels := []struct {
		name  string
		level CompressionLevel
	}{
		{"fastest", LevelFastest},
		{"default", LevelDefault},
		{"best", LevelBest},
	}

	var sizes []int
	for _, lvl := range levels {
		t.Run(lvl.name, func(t *testing.T) {
			compressed, err := Compress(data, lvl.level)
			require.NoError(t, err)
			assert.True(t, IsCompressed(compressed))

			sizes = append(sizes, len(compressed))

			decompressed, err := Decompress(compressed)
			require.NoError(t, err)
			assert.Equal(t, data, decompressed)

			t.Logf("level=%s compressed=%d ratio=%.4f", lvl.name, len(compressed), float64(len(compressed))/float64(len(data)))
		})
	}

	// Best should produce equal or smaller output than Fastest
	if len(sizes) == 3 {
		assert.LessOrEqual(t, sizes[2], sizes[0], "LevelBest should compress at least as well as LevelFastest")
	}
}

func TestCompress_Empty(t *testing.T) {
	compressed, err := Compress(nil, LevelDefault)
	require.NoError(t, err)
	assert.Nil(t, compressed)

	compressed, err = Compress([]byte{}, LevelDefault)
	require.NoError(t, err)
	assert.Empty(t, compressed)
}

func TestDecompress_UncompressedPassthrough(t *testing.T) {
	// Data without magic prefix should pass through unchanged
	raw := []byte("this is uncompressed protobuf data")
	result, err := Decompress(raw)
	require.NoError(t, err)
	assert.Equal(t, raw, result)
}

func TestDecompress_Empty(t *testing.T) {
	result, err := Decompress(nil)
	require.NoError(t, err)
	assert.Nil(t, result)

	result, err = Decompress([]byte{})
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestDecompress_ShortData(t *testing.T) {
	// Data shorter than magic prefix should pass through
	result, err := Decompress([]byte{0x5A, 0x53})
	require.NoError(t, err)
	assert.Equal(t, []byte{0x5A, 0x53}, result)
}

func TestDecompress_CorruptCompressedData(t *testing.T) {
	// Magic prefix followed by invalid zstd data
	corrupt := append([]byte{0x5A, 0x53, 0x54, 0x44}, []byte("not valid zstd")...)
	_, err := Decompress(corrupt)
	assert.Error(t, err, "should fail on corrupt compressed data")
}

func TestIsCompressed(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{name: "nil", data: nil, expected: false},
		{name: "empty", data: []byte{}, expected: false},
		{name: "short", data: []byte{0x5A}, expected: false},
		{name: "magic prefix only", data: []byte{0x5A, 0x53, 0x54, 0x44}, expected: true},
		{name: "magic with data", data: []byte{0x5A, 0x53, 0x54, 0x44, 0x01, 0x02}, expected: true},
		{name: "wrong prefix", data: []byte{0x00, 0x53, 0x54, 0x44}, expected: false},
		{name: "protobuf data", data: []byte{0x0a, 0x10, 0x08, 0x01}, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsCompressed(tt.data))
		})
	}
}

func TestCompress_AchievesCompression(t *testing.T) {
	data := bytes.Repeat([]byte("rollkit block data with repeated content "), 10000)
	compressed, err := Compress(data, LevelDefault)
	require.NoError(t, err)

	ratio := float64(len(compressed)) / float64(len(data))
	t.Logf("compression ratio: %.4f (original: %d, compressed: %d)", ratio, len(data), len(compressed))
	assert.Less(t, ratio, 0.1, "highly repetitive data should compress to <10%% of original size")
}

func TestCompress_RandomDataStillWorks(t *testing.T) {
	data := make([]byte, 4096)
	_, err := rand.Read(data)
	require.NoError(t, err)

	compressed, err := Compress(data, LevelFastest)
	require.NoError(t, err)

	decompressed, err := Decompress(compressed)
	require.NoError(t, err)
	assert.Equal(t, data, decompressed)
}

func TestDecompress_DataStartingWithMagicButUncompressed(t *testing.T) {
	fakeCompressed := append([]byte{0x5A, 0x53, 0x54, 0x44}, bytes.Repeat([]byte{0x00}, 100)...)
	_, err := Decompress(fakeCompressed)
	assert.Error(t, err, "data starting with magic but containing invalid zstd should error")
}

func TestCompress_InvalidLevel(t *testing.T) {
	// Out-of-range level should fall back to LevelDefault
	data := []byte("test data for invalid level")
	compressed, err := Compress(data, CompressionLevel(99))
	require.NoError(t, err)

	decompressed, err := Decompress(compressed)
	require.NoError(t, err)
	assert.Equal(t, data, decompressed)
}
