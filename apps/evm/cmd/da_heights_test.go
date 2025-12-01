package cmd

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/evstack/ev-node/pkg/store"
)

func TestDAHeightsCmd_NoData(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create store with no data
	kvStore, err := store.NewDefaultKVStore(tempDir, "db", "test")
	assert.NilError(t, err)
	defer kvStore.Close()

	st := store.New(kvStore)

	// Should return empty result when no DA included height exists
	result, err := fetchDAHeights(ctx, st, 10)
	assert.NilError(t, err)
	assert.Equal(t, 0, len(result))
}

func TestDAHeightsCmd_SingleHeight(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	kvStore, err := store.NewDefaultKVStore(tempDir, "db", "test")
	assert.NilError(t, err)
	defer kvStore.Close()

	st := store.New(kvStore)

	// Set DA included height to 1
	daIncludedHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(daIncludedHeightBytes, 1)
	assert.NilError(t, st.SetMetadata(ctx, store.DAIncludedHeightKey, daIncludedHeightBytes))

	// Set header and data DA heights for height 1
	headerBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(headerBytes, 100)
	assert.NilError(t, st.SetMetadata(ctx, store.GetHeightToDAHeightHeaderKey(1), headerBytes))

	dataBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(dataBytes, 101)
	assert.NilError(t, st.SetMetadata(ctx, store.GetHeightToDAHeightDataKey(1), dataBytes))

	// Fetch last 10 heights
	result, err := fetchDAHeights(ctx, st, 10)
	assert.NilError(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, uint64(1), result[0].Height)
	assert.Equal(t, uint64(100), result[0].HeaderDAHeight)
	assert.Equal(t, uint64(101), result[0].DataDAHeight)
	assert.Equal(t, false, result[0].Missing)
}

func TestDAHeightsCmd_MultipleHeights(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	kvStore, err := store.NewDefaultKVStore(tempDir, "db", "test")
	assert.NilError(t, err)
	defer kvStore.Close()

	st := store.New(kvStore)

	// Set DA included height to 5
	daIncludedHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(daIncludedHeightBytes, 5)
	assert.NilError(t, st.SetMetadata(ctx, store.DAIncludedHeightKey, daIncludedHeightBytes))

	// Set header and data DA heights for heights 1-5
	for i := uint64(1); i <= 5; i++ {
		headerBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(headerBytes, 100+i)
		assert.NilError(t, st.SetMetadata(ctx, store.GetHeightToDAHeightHeaderKey(i), headerBytes))

		dataBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(dataBytes, 200+i)
		assert.NilError(t, st.SetMetadata(ctx, store.GetHeightToDAHeightDataKey(i), dataBytes))
	}

	// Fetch last 3 heights
	result, err := fetchDAHeights(ctx, st, 3)
	assert.NilError(t, err)
	assert.Equal(t, 3, len(result))

	// Results should be in descending order (most recent first)
	assert.Equal(t, uint64(5), result[0].Height)
	assert.Equal(t, uint64(105), result[0].HeaderDAHeight)
	assert.Equal(t, uint64(205), result[0].DataDAHeight)

	assert.Equal(t, uint64(4), result[1].Height)
	assert.Equal(t, uint64(104), result[1].HeaderDAHeight)
	assert.Equal(t, uint64(204), result[1].DataDAHeight)

	assert.Equal(t, uint64(3), result[2].Height)
	assert.Equal(t, uint64(103), result[2].HeaderDAHeight)
	assert.Equal(t, uint64(203), result[2].DataDAHeight)
}

func TestDAHeightsCmd_MissingHeaderHeight(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	kvStore, err := store.NewDefaultKVStore(tempDir, "db", "test")
	assert.NilError(t, err)
	defer kvStore.Close()

	st := store.New(kvStore)

	// Set DA included height to 3
	daIncludedHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(daIncludedHeightBytes, 3)
	assert.NilError(t, st.SetMetadata(ctx, store.DAIncludedHeightKey, daIncludedHeightBytes))

	// Set heights for 1 and 3, but not 2 (missing header)
	for _, i := range []uint64{1, 3} {
		headerBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(headerBytes, 100+i)
		assert.NilError(t, st.SetMetadata(ctx, store.GetHeightToDAHeightHeaderKey(i), headerBytes))

		dataBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(dataBytes, 200+i)
		assert.NilError(t, st.SetMetadata(ctx, store.GetHeightToDAHeightDataKey(i), dataBytes))
	}

	// Fetch all heights
	result, err := fetchDAHeights(ctx, st, 10)
	assert.NilError(t, err)
	assert.Equal(t, 3, len(result))

	// Height 2 should be marked as missing
	assert.Equal(t, uint64(2), result[1].Height)
	assert.Equal(t, true, result[1].Missing)
}

func TestDAHeightsCmd_MissingDataHeight(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	kvStore, err := store.NewDefaultKVStore(tempDir, "db", "test")
	assert.NilError(t, err)
	defer kvStore.Close()

	st := store.New(kvStore)

	// Set DA included height to 2
	daIncludedHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(daIncludedHeightBytes, 2)
	assert.NilError(t, st.SetMetadata(ctx, store.DAIncludedHeightKey, daIncludedHeightBytes))

	// Set header for height 1 but not data
	headerBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(headerBytes, 100)
	assert.NilError(t, st.SetMetadata(ctx, store.GetHeightToDAHeightHeaderKey(1), headerBytes))

	// Set both for height 2
	headerBytes2 := make([]byte, 8)
	binary.LittleEndian.PutUint64(headerBytes2, 102)
	assert.NilError(t, st.SetMetadata(ctx, store.GetHeightToDAHeightHeaderKey(2), headerBytes2))

	dataBytes2 := make([]byte, 8)
	binary.LittleEndian.PutUint64(dataBytes2, 202)
	assert.NilError(t, st.SetMetadata(ctx, store.GetHeightToDAHeightDataKey(2), dataBytes2))

	// Fetch all heights
	result, err := fetchDAHeights(ctx, st, 10)
	assert.NilError(t, err)
	assert.Equal(t, 2, len(result))

	// Height 1 should be marked as missing (missing data)
	assert.Equal(t, uint64(1), result[1].Height)
	assert.Equal(t, true, result[1].Missing)
}

func TestFormatDAHeightsOutput(t *testing.T) {
	tests := []struct {
		name     string
		heights  []daHeightInfo
		expected string
	}{
		{
			name:     "empty results",
			heights:  []daHeightInfo{},
			expected: "No DA included heights found\n",
		},
		{
			name: "single height no missing",
			heights: []daHeightInfo{
				{Height: 1, HeaderDAHeight: 100, DataDAHeight: 101, Missing: false},
			},
			expected: "Block Height | Header DA Height | Data DA Height | Status\n" +
				"------------ | ---------------- | -------------- | ------\n" +
				"           1 |              100 |            101 | OK\n",
		},
		{
			name: "multiple heights with missing",
			heights: []daHeightInfo{
				{Height: 3, HeaderDAHeight: 103, DataDAHeight: 203, Missing: false},
				{Height: 2, HeaderDAHeight: 0, DataDAHeight: 0, Missing: true},
				{Height: 1, HeaderDAHeight: 101, DataDAHeight: 201, Missing: false},
			},
			expected: "Block Height | Header DA Height | Data DA Height | Status\n" +
				"------------ | ---------------- | -------------- | ------\n" +
				"           3 |              103 |            203 | OK\n" +
				"           2 |                - |              - | MISSING\n" +
				"           1 |              101 |            201 | OK\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatDAHeightsOutput(&buf, tt.heights)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}
