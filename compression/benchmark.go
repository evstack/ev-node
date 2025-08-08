package compression

import (
	"context"
	"crypto/rand"
	"fmt"
	"runtime"
	"time"
)

// BenchmarkSuite provides comprehensive benchmarking for compression algorithms
type BenchmarkSuite struct {
	compressor *DefaultCompressor
}

// NewBenchmarkSuite creates a new benchmark suite
func NewBenchmarkSuite() (*BenchmarkSuite, error) {
	compressor, err := NewCompressor()
	if err != nil {
		return nil, err
	}
	
	return &BenchmarkSuite{
		compressor: compressor,
	}, nil
}

// BenchmarkAllAlgorithms benchmarks all compression algorithms against the given data
func (bs *BenchmarkSuite) BenchmarkAllAlgorithms(ctx context.Context, data []byte) ([]BenchmarkResult, error) {
	algorithms := []Algorithm{None, Gzip, LZ4, Zstd}
	results := make([]BenchmarkResult, len(algorithms))
	
	for i, algorithm := range algorithms {
		result, err := bs.BenchmarkAlgorithm(ctx, data, algorithm)
		if err != nil {
			return nil, fmt.Errorf("benchmark failed for %s: %w", algorithm, err)
		}
		results[i] = result
	}
	
	return results, nil
}

// BenchmarkAlgorithm benchmarks a specific compression algorithm
func (bs *BenchmarkSuite) BenchmarkAlgorithm(ctx context.Context, data []byte, algorithm Algorithm) (BenchmarkResult, error) {
	if algorithm == None {
		return BenchmarkResult{
			Algorithm:         None,
			OriginalSize:     len(data),
			CompressedSize:   len(data),
			CompressionRatio: 1.0,
			CompressTime:     0,
			DecompressTime:   0,
			CompressionRate:  0,
			DecompressionRate: 0,
		}, nil
	}
	
	// Warm up
	for i := 0; i < 3; i++ {
		_, err := bs.compressor.Compress(ctx, data, algorithm)
		if err != nil {
			return BenchmarkResult{}, err
		}
	}
	
	// Force garbage collection before benchmarking
	runtime.GC()
	
	// Benchmark compression
	startTime := time.Now()
	compressedBlob, err := bs.compressor.Compress(ctx, data, algorithm)
	compressTime := time.Since(startTime).Nanoseconds()
	
	if err != nil {
		return BenchmarkResult{}, err
	}
	
	// Force garbage collection before decompression benchmark
	runtime.GC()
	
	// Benchmark decompression
	startTime = time.Now()
	decompressedData, err := bs.compressor.Decompress(ctx, compressedBlob)
	decompressTime := time.Since(startTime).Nanoseconds()
	
	if err != nil {
		return BenchmarkResult{}, err
	}
	
	// Verify correctness
	if len(decompressedData) != len(data) {
		return BenchmarkResult{}, fmt.Errorf("decompressed data length mismatch: expected %d, got %d", len(data), len(decompressedData))
	}
	
	// Calculate rates (MB/s)
	originalSizeMB := float64(len(data)) / (1024 * 1024)
	compressionRate := originalSizeMB / (float64(compressTime) / 1e9)
	decompressionRate := originalSizeMB / (float64(decompressTime) / 1e9)
	
	return BenchmarkResult{
		Algorithm:         algorithm,
		OriginalSize:     len(data),
		CompressedSize:   len(compressedBlob.Data),
		CompressionRatio: float64(len(compressedBlob.Data)) / float64(len(data)),
		CompressTime:     compressTime,
		DecompressTime:   decompressTime,
		CompressionRate:  compressionRate,
		DecompressionRate: decompressionRate,
	}, nil
}

// GenerateTestData creates test data of various patterns for benchmarking
func (bs *BenchmarkSuite) GenerateTestData() []TestData {
	return []TestData{
		{
			Name: "Random Data (1KB)",
			Data: generateRandomData(1024),
		},
		{
			Name: "Random Data (10KB)",
			Data: generateRandomData(10240),
		},
		{
			Name: "Random Data (100KB)",
			Data: generateRandomData(102400),
		},
		{
			Name: "Random Data (1MB)",
			Data: generateRandomData(1024 * 1024),
		},
		{
			Name: "Highly Compressible (10KB zeros)",
			Data: make([]byte, 10240), // All zeros
		},
		{
			Name: "Repeated Pattern (10KB)",
			Data: generateRepeatedPattern(10240, []byte("Hello, World! This is a test pattern. ")),
		},
		{
			Name: "JSON-like Structure (10KB)",
			Data: generateJSONLikeData(10240),
		},
		{
			Name: "Binary Protobuf-like (10KB)",
			Data: generateProtobufLikeData(10240),
		},
	}
}

// TestData represents test data with a description
type TestData struct {
	Name string
	Data []byte
}

// RunComprehensiveBenchmark runs benchmarks on various data types
func (bs *BenchmarkSuite) RunComprehensiveBenchmark(ctx context.Context) (*ComprehensiveBenchmarkResult, error) {
	testDatasets := bs.GenerateTestData()
	results := make(map[string][]BenchmarkResult)
	
	for _, testData := range testDatasets {
		benchmarkResults, err := bs.BenchmarkAllAlgorithms(ctx, testData.Data)
		if err != nil {
			return nil, fmt.Errorf("benchmark failed for %s: %w", testData.Name, err)
		}
		results[testData.Name] = benchmarkResults
	}
	
	return &ComprehensiveBenchmarkResult{
		Results: results,
		Summary: bs.generateSummary(results),
	}, nil
}

// ComprehensiveBenchmarkResult contains results from comprehensive benchmarking
type ComprehensiveBenchmarkResult struct {
	Results map[string][]BenchmarkResult
	Summary BenchmarkSummary
}

// BenchmarkSummary provides aggregate statistics
type BenchmarkSummary struct {
	BestCompressionRatio Algorithm
	FastestCompression   Algorithm
	FastestDecompression Algorithm
	BestOverall         Algorithm
	AverageRatios       map[Algorithm]float64
	AverageCompressTime  map[Algorithm]int64
	AverageDecompressTime map[Algorithm]int64
}

// generateSummary creates a summary of benchmark results
func (bs *BenchmarkSuite) generateSummary(results map[string][]BenchmarkResult) BenchmarkSummary {
	algorithms := []Algorithm{Gzip, LZ4, Zstd}
	
	ratioSums := make(map[Algorithm]float64)
	compressTimeSums := make(map[Algorithm]int64)
	decompressTimeSums := make(map[Algorithm]int64)
	counts := make(map[Algorithm]int)
	
	bestRatio := float64(1.0)
	bestCompression := None
	fastestCompress := None
	fastestDecompress := None
	
	var bestCompressTime int64 = 1e9 // 1 second
	var bestDecompressTime int64 = 1e9
	
	// Collect statistics
	for _, benchmarkResults := range results {
		for _, result := range benchmarkResults {
			if result.Algorithm == None {
				continue
			}
			
			ratioSums[result.Algorithm] += result.CompressionRatio
			compressTimeSums[result.Algorithm] += result.CompressTime
			decompressTimeSums[result.Algorithm] += result.DecompressTime
			counts[result.Algorithm]++
			
			if result.CompressionRatio < bestRatio {
				bestRatio = result.CompressionRatio
				bestCompression = result.Algorithm
			}
			
			if result.CompressTime < bestCompressTime {
				bestCompressTime = result.CompressTime
				fastestCompress = result.Algorithm
			}
			
			if result.DecompressTime < bestDecompressTime {
				bestDecompressTime = result.DecompressTime
				fastestDecompress = result.Algorithm
			}
		}
	}
	
	// Calculate averages
	averageRatios := make(map[Algorithm]float64)
	averageCompressTime := make(map[Algorithm]int64)
	averageDecompressTime := make(map[Algorithm]int64)
	
	for _, alg := range algorithms {
		if counts[alg] > 0 {
			averageRatios[alg] = ratioSums[alg] / float64(counts[alg])
			averageCompressTime[alg] = compressTimeSums[alg] / int64(counts[alg])
			averageDecompressTime[alg] = decompressTimeSums[alg] / int64(counts[alg])
		}
	}
	
	// Determine best overall (balanced score)
	bestOverall := None
	bestScore := float64(0)
	
	for _, alg := range algorithms {
		if counts[alg] == 0 {
			continue
		}
		
		// Score based on compression ratio (lower is better) and speed (higher is better)
		// Normalize to [0,1] range and weight equally
		compressionScore := 1.0 - averageRatios[alg] // Higher is better
		speedScore := 1.0 / (float64(averageCompressTime[alg]+averageDecompressTime[alg]) / 1e9) // Higher is better
		
		totalScore := (compressionScore + speedScore) / 2
		
		if totalScore > bestScore {
			bestScore = totalScore
			bestOverall = alg
		}
	}
	
	return BenchmarkSummary{
		BestCompressionRatio:  bestCompression,
		FastestCompression:    fastestCompress,
		FastestDecompression:  fastestDecompress,
		BestOverall:          bestOverall,
		AverageRatios:        averageRatios,
		AverageCompressTime:   averageCompressTime,
		AverageDecompressTime: averageDecompressTime,
	}
}

// generateRandomData creates random data of the specified size
func generateRandomData(size int) []byte {
	data := make([]byte, size)
	rand.Read(data)
	return data
}

// generateRepeatedPattern creates data with a repeated pattern
func generateRepeatedPattern(size int, pattern []byte) []byte {
	data := make([]byte, size)
	for i := 0; i < size; i++ {
		data[i] = pattern[i%len(pattern)]
	}
	return data
}

// generateJSONLikeData creates JSON-like structured data
func generateJSONLikeData(size int) []byte {
	template := `{"id":%d,"name":"user_%d","email":"user%d@example.com","active":true,"metadata":{"created":"2023-01-01","tags":["tag1","tag2"],"score":%.2f}}`
	
	var data []byte
	id := 1
	
	for len(data) < size {
		jsonStr := fmt.Sprintf(template, id, id, id, float64(id)*1.5)
		data = append(data, []byte(jsonStr)...)
		if len(data) < size {
			data = append(data, ',')
		}
		id++
	}
	
	// Truncate to exact size
	if len(data) > size {
		data = data[:size]
	}
	
	return data
}

// generateProtobufLikeData creates binary data similar to protobuf encoding
func generateProtobufLikeData(size int) []byte {
	data := make([]byte, size)
	
	// Simulate protobuf-like structure with field tags and variable length encoding
	pos := 0
	fieldId := 1
	
	for pos < size-10 {
		// Field tag (varint)
		data[pos] = byte(fieldId<<3 | 2) // Wire type 2 (length-delimited)
		pos++
		
		// Length (varint)
		length := (fieldId % 20) + 5
		data[pos] = byte(length)
		pos++
		
		// Data
		for i := 0; i < length && pos < size; i++ {
			data[pos] = byte((fieldId * i) % 256)
			pos++
		}
		
		fieldId++
	}
	
	return data
}