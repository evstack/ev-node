package compression

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBenchmarkSuite_BenchmarkAlgorithm(t *testing.T) {
	ctx := context.Background()
	suite, err := NewBenchmarkSuite()
	require.NoError(t, err)
	
	testData := []byte("This is a test string for benchmarking compression algorithms. " +
		"It contains repeated patterns and should compress reasonably well. " +
		"This is a test string for benchmarking compression algorithms.")
	
	algorithms := []Algorithm{None, Gzip, LZ4, Zstd}
	
	for _, algorithm := range algorithms {
		t.Run(algorithm.String(), func(t *testing.T) {
			result, err := suite.BenchmarkAlgorithm(ctx, testData, algorithm)
			require.NoError(t, err)
			
			assert.Equal(t, algorithm, result.Algorithm)
			assert.Equal(t, len(testData), result.OriginalSize)
			assert.True(t, result.CompressedSize > 0)
			assert.True(t, result.CompressionRatio > 0)
			
			if algorithm == None {
				assert.Equal(t, len(testData), result.CompressedSize)
				assert.Equal(t, 1.0, result.CompressionRatio)
				assert.Equal(t, int64(0), result.CompressTime)
				assert.Equal(t, int64(0), result.DecompressTime)
			} else {
				assert.True(t, result.CompressTime > 0)
				assert.True(t, result.DecompressTime > 0)
				assert.True(t, result.CompressionRate >= 0)
				assert.True(t, result.DecompressionRate >= 0)
			}
		})
	}
}

func TestBenchmarkSuite_BenchmarkAllAlgorithms(t *testing.T) {
	ctx := context.Background()
	suite, err := NewBenchmarkSuite()
	require.NoError(t, err)
	
	testData := make([]byte, 1024)
	for i := range testData {
		testData[i] = byte(i % 50) // Create somewhat compressible pattern
	}
	
	results, err := suite.BenchmarkAllAlgorithms(ctx, testData)
	require.NoError(t, err)
	
	assert.Len(t, results, 4) // None, Gzip, LZ4, Zstd
	
	// Verify each algorithm is represented
	algorithmsSeen := make(map[Algorithm]bool)
	for _, result := range results {
		algorithmsSeen[result.Algorithm] = true
		assert.Equal(t, len(testData), result.OriginalSize)
		assert.True(t, result.CompressedSize > 0)
	}
	
	assert.True(t, algorithmsSeen[None])
	assert.True(t, algorithmsSeen[Gzip])
	assert.True(t, algorithmsSeen[LZ4])
	assert.True(t, algorithmsSeen[Zstd])
}

func TestBenchmarkSuite_GenerateTestData(t *testing.T) {
	suite, err := NewBenchmarkSuite()
	require.NoError(t, err)
	
	testDatasets := suite.GenerateTestData()
	require.Greater(t, len(testDatasets), 0)
	
	expectedDatasets := []string{
		"Random Data (1KB)",
		"Random Data (10KB)", 
		"Random Data (100KB)",
		"Random Data (1MB)",
		"Highly Compressible (10KB zeros)",
		"Repeated Pattern (10KB)",
		"JSON-like Structure (10KB)",
		"Binary Protobuf-like (10KB)",
	}
	
	assert.Len(t, testDatasets, len(expectedDatasets))
	
	for i, expected := range expectedDatasets {
		assert.Equal(t, expected, testDatasets[i].Name)
		assert.NotNil(t, testDatasets[i].Data)
		assert.Greater(t, len(testDatasets[i].Data), 0)
	}
	
	// Verify sizes
	assert.Equal(t, 1024, len(testDatasets[0].Data))        // 1KB
	assert.Equal(t, 10240, len(testDatasets[1].Data))       // 10KB
	assert.Equal(t, 102400, len(testDatasets[2].Data))      // 100KB
	assert.Equal(t, 1024*1024, len(testDatasets[3].Data))   // 1MB
	assert.Equal(t, 10240, len(testDatasets[4].Data))       // 10KB zeros
	
	// Verify zeros dataset is all zeros
	zerosData := testDatasets[4].Data
	for _, b := range zerosData {
		assert.Equal(t, byte(0), b)
	}
}

func TestBenchmarkSuite_RunComprehensiveBenchmark(t *testing.T) {
	ctx := context.Background()
	suite, err := NewBenchmarkSuite()
	require.NoError(t, err)
	
	// Use a smaller subset for testing to avoid long test times
	originalGenerateTestData := suite.GenerateTestData
	suite.GenerateTestData = func() []TestData {
		return []TestData{
			{Name: "Small Test (1KB)", Data: make([]byte, 1024)},
			{Name: "Pattern Test", Data: generateRepeatedPattern(512, []byte("ABC"))},
		}
	}
	defer func() { suite.GenerateTestData = originalGenerateTestData }()
	
	result, err := suite.RunComprehensiveBenchmark(ctx)
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Verify results structure
	assert.Len(t, result.Results, 2)
	assert.Contains(t, result.Results, "Small Test (1KB)")
	assert.Contains(t, result.Results, "Pattern Test")
	
	// Each dataset should have results for all algorithms
	for _, benchmarkResults := range result.Results {
		assert.Len(t, benchmarkResults, 4) // None, Gzip, LZ4, Zstd
	}
	
	// Verify summary
	summary := result.Summary
	assert.NotEqual(t, None, summary.BestCompressionRatio) // Should find a best compression algorithm
	assert.Contains(t, []Algorithm{Gzip, LZ4, Zstd}, summary.BestCompressionRatio)
	
	// Verify average statistics are calculated
	assert.Greater(t, len(summary.AverageRatios), 0)
	assert.Greater(t, len(summary.AverageCompressTime), 0)
	assert.Greater(t, len(summary.AverageDecompressTime), 0)
}

func TestGenerateTestDataHelpers(t *testing.T) {
	// Test generateRandomData
	randomData := generateRandomData(100)
	assert.Len(t, randomData, 100)
	
	// Test generateRepeatedPattern
	pattern := []byte("ABC")
	repeated := generateRepeatedPattern(10, pattern)
	assert.Len(t, repeated, 10)
	expected := []byte("ABCABCABCA")
	assert.Equal(t, expected, repeated)
	
	// Test generateJSONLikeData
	jsonData := generateJSONLikeData(500)
	assert.Len(t, jsonData, 500)
	assert.Contains(t, string(jsonData), "id")
	assert.Contains(t, string(jsonData), "name")
	
	// Test generateProtobufLikeData
	protoData := generateProtobufLikeData(200)
	assert.Len(t, protoData, 200)
	assert.Greater(t, len(protoData), 0)
}

func TestBenchmarkSummary(t *testing.T) {
	ctx := context.Background()
	suite, err := NewBenchmarkSuite()
	require.NoError(t, err)
	
	// Create test results
	results := map[string][]BenchmarkResult{
		"Test1": {
			{Algorithm: Gzip, OriginalSize: 1000, CompressedSize: 500, CompressionRatio: 0.5, CompressTime: 1000000, DecompressTime: 500000},
			{Algorithm: LZ4, OriginalSize: 1000, CompressedSize: 600, CompressionRatio: 0.6, CompressTime: 200000, DecompressTime: 100000},
			{Algorithm: Zstd, OriginalSize: 1000, CompressedSize: 450, CompressionRatio: 0.45, CompressTime: 800000, DecompressTime: 400000},
		},
		"Test2": {
			{Algorithm: Gzip, OriginalSize: 2000, CompressedSize: 1000, CompressionRatio: 0.5, CompressTime: 2000000, DecompressTime: 1000000},
			{Algorithm: LZ4, OriginalSize: 2000, CompressedSize: 1200, CompressionRatio: 0.6, CompressTime: 400000, DecompressTime: 200000},
			{Algorithm: Zstd, OriginalSize: 2000, CompressedSize: 900, CompressionRatio: 0.45, CompressTime: 1600000, DecompressTime: 800000},
		},
	}
	
	summary := suite.generateSummary(results)
	
	// Best compression ratio should be Zstd (0.45 average)
	assert.Equal(t, Zstd, summary.BestCompressionRatio)
	
	// Fastest compression should be LZ4 (lowest average compress time)
	assert.Equal(t, LZ4, summary.FastestCompression)
	
	// Fastest decompression should be LZ4 (lowest average decompress time)
	assert.Equal(t, LZ4, summary.FastestDecompression)
	
	// Verify average calculations
	assert.InDelta(t, 0.5, summary.AverageRatios[Gzip], 0.001)
	assert.InDelta(t, 0.6, summary.AverageRatios[LZ4], 0.001)
	assert.InDelta(t, 0.45, summary.AverageRatios[Zstd], 0.001)
	
	assert.Equal(t, int64(1500000), summary.AverageCompressTime[Gzip])   // (1000000+2000000)/2
	assert.Equal(t, int64(300000), summary.AverageCompressTime[LZ4])     // (200000+400000)/2
	assert.Equal(t, int64(1200000), summary.AverageCompressTime[Zstd])   // (800000+1600000)/2
}