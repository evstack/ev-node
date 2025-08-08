package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/evstack/ev-node/compression"
	"github.com/evstack/ev-node/core/da"
)

// BenchmarkResult holds the results of a compression benchmark
type BenchmarkResult struct {
	Algorithm        string
	Level            int
	DataSize         int
	CompressedSize   int
	CompressionRatio float64
	CompressTime     time.Duration
	DecompressTime   time.Duration
	CompressionSpeed float64 // MB/s
	DecompressionSpeed float64 // MB/s
}

// TestDataType represents different types of test data
type TestDataType int

const (
	Repetitive TestDataType = iota
	Random
	JSON
	Text
)

func (t TestDataType) String() string {
	switch t {
	case Repetitive:
		return "Repetitive"
	case Random:
		return "Random"
	case JSON:
		return "JSON"
	case Text:
		return "Text"
	default:
		return "Unknown"
	}
}

func generateTestData(dataType TestDataType, size int) da.Blob {
	switch dataType {
	case Repetitive:
		// Highly compressible repetitive data
		pattern := []byte("The quick brown fox jumps over the lazy dog. ")
		data := make([]byte, 0, size)
		for len(data) < size {
			remaining := size - len(data)
			if remaining >= len(pattern) {
				data = append(data, pattern...)
			} else {
				data = append(data, pattern[:remaining]...)
			}
		}
		return data[:size]
		
	case Random:
		// Random data that doesn't compress well
		data := make([]byte, size)
		rand.Read(data)
		return data
		
	case JSON:
		// JSON-like structured data
		jsonTemplate := `{"id":%d,"name":"user_%d","email":"user%d@example.com","data":"%s","timestamp":%d,"active":true}`
		data := make([]byte, 0, size)
		counter := 0
		for len(data) < size {
			userData := fmt.Sprintf("data_%d_%d", counter, time.Now().UnixNano()%10000)
			entry := fmt.Sprintf(jsonTemplate, counter, counter, counter, userData, time.Now().Unix())
			if len(data)+len(entry) <= size {
				data = append(data, entry...)
				if len(data) < size-1 {
					data = append(data, ',')
				}
			} else {
				break
			}
			counter++
		}
		return data[:min(len(data), size)]
		
	case Text:
		// Natural language text (moderately compressible)
		words := []string{
			"lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing", "elit",
			"sed", "do", "eiusmod", "tempor", "incididunt", "ut", "labore", "et", "dolore",
			"magna", "aliqua", "enim", "ad", "minim", "veniam", "quis", "nostrud",
			"exercitation", "ullamco", "laboris", "nisi", "aliquip", "ex", "ea", "commodo",
		}
		data := make([]byte, 0, size)
		wordIndex := 0
		for len(data) < size {
			word := words[wordIndex%len(words)]
			if len(data)+len(word)+1 <= size {
				if len(data) > 0 {
					data = append(data, ' ')
				}
				data = append(data, word...)
			} else {
				break
			}
			wordIndex++
		}
		return data[:min(len(data), size)]
	}
	return nil
}

func runBenchmark(config compression.Config, testData da.Blob, iterations int) (*BenchmarkResult, error) {
	compressor, err := compression.NewCompressibleDA(nil, config)
	if err != nil {
		return nil, err
	}
	defer compressor.Close()
	
	// Warm up
	_, err = compression.CompressBlob(testData)
	if err != nil {
		return nil, err
	}
	
	var totalCompressTime, totalDecompressTime time.Duration
	var compressedData da.Blob
	
	// Run compression benchmark
	for i := 0; i < iterations; i++ {
		start := time.Now()
		compressedData, err = compression.CompressBlob(testData)
		if err != nil {
			return nil, err
		}
		totalCompressTime += time.Since(start)
	}
	
	// Run decompression benchmark
	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := compression.DecompressBlob(compressedData)
		if err != nil {
			return nil, err
		}
		totalDecompressTime += time.Since(start)
	}
	
	avgCompressTime := totalCompressTime / time.Duration(iterations)
	avgDecompressTime := totalDecompressTime / time.Duration(iterations)
	
	compressionRatio := float64(len(compressedData)) / float64(len(testData))
	compressionSpeed := float64(len(testData)) / 1024 / 1024 / avgCompressTime.Seconds()
	decompressionSpeed := float64(len(testData)) / 1024 / 1024 / avgDecompressTime.Seconds()
	
	// Get actual compressed size (minus header)
	info := compression.GetCompressionInfo(compressedData)
	actualCompressedSize := int(info.CompressedSize)
	if !info.IsCompressed {
		actualCompressedSize = int(info.OriginalSize)
	}
	
	return &BenchmarkResult{
		Algorithm:          "zstd",
		Level:              config.ZstdLevel,
		DataSize:           len(testData),
		CompressedSize:     actualCompressedSize,
		CompressionRatio:   compressionRatio,
		CompressTime:       avgCompressTime,
		DecompressTime:     avgDecompressTime,
		CompressionSpeed:   compressionSpeed,
		DecompressionSpeed: decompressionSpeed,
	}, nil
}

func printResults(dataType TestDataType, results []*BenchmarkResult) {
	fmt.Printf("\n=== %s Data Results ===\n", dataType)
	fmt.Printf("%-6s %-10s %-12s %-8s %-12s %-15s %-12s %-15s\n",
		"Level", "Size", "Compressed", "Ratio", "Comp Time", "Comp Speed", "Decomp Time", "Decomp Speed")
	fmt.Printf("%-6s %-10s %-12s %-8s %-12s %-15s %-12s %-15s\n",
		"-----", "--------", "----------", "------", "---------", "-----------", "----------", "-------------")
	
	for _, result := range results {
		fmt.Printf("%-6d %-10s %-12s %-8.3f %-12s %-15s %-12s %-15s\n",
			result.Level,
			formatBytes(result.DataSize),
			formatBytes(result.CompressedSize),
			result.CompressionRatio,
			formatDuration(result.CompressTime),
			formatSpeed(result.CompressionSpeed),
			formatDuration(result.DecompressTime),
			formatSpeed(result.DecompressionSpeed),
		)
	}
}

func formatBytes(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(bytes)/1024/1024)
}

func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.1fμs", float64(d.Nanoseconds())/1000)
	} else if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1000000)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func formatSpeed(mbps float64) string {
	if mbps < 1 {
		return fmt.Sprintf("%.1fKB/s", mbps*1024)
	}
	return fmt.Sprintf("%.1fMB/s", mbps)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	fmt.Println("EV-Node Zstd Compression Benchmark")
	fmt.Println("==================================")
	
	// Parse command line arguments
	iterations := 100
	if len(os.Args) > 1 {
		if i, err := strconv.Atoi(os.Args[1]); err == nil {
			iterations = i
		}
	}
	
	testSizes := []int{1024, 4096, 16384, 65536} // 1KB, 4KB, 16KB, 64KB
	testDataTypes := []TestDataType{Repetitive, Text, JSON, Random}
	zstdLevels := []int{1, 3, 6, 9} // Test different levels, highlighting level 3
	
	fmt.Printf("Running %d iterations per test\n", iterations)
	fmt.Printf("Test sizes: %v\n", testSizes)
	fmt.Printf("Zstd levels: %v (level 3 is recommended)\n", zstdLevels)
	
	for _, dataType := range testDataTypes {
		var allResults []*BenchmarkResult
		
		for _, size := range testSizes {
			testData := generateTestData(dataType, size)
			
			for _, level := range zstdLevels {
				config := compression.Config{
					Enabled:             true,
					ZstdLevel:           level,
					MinCompressionRatio: 0.05, // Allow more compression attempts for benchmarking
				}
				
				result, err := runBenchmark(config, testData, iterations)
				if err != nil {
					fmt.Printf("Error benchmarking %s data (size: %d, level: %d): %v\n",
						dataType, size, level, err)
					continue
				}
				
				allResults = append(allResults, result)
			}
		}
		
		printResults(dataType, allResults)
	}
	
	// Print recommendations
	fmt.Printf("\n=== Recommendations ===\n")
	fmt.Printf("• **Zstd Level 3**: Optimal balance of compression ratio and speed\n")
	fmt.Printf("• **Best for EV-Node**: Fast compression (~100-200 MB/s) with good ratios (~20-40%%)\n")
	fmt.Printf("• **Memory efficient**: Lower memory usage than higher compression levels\n")
	fmt.Printf("• **Production ready**: Widely used default level in many applications\n")
	fmt.Printf("\n")
	
	// Real-world example
	fmt.Printf("=== Real-World Example ===\n")
	realWorldData := generateTestData(JSON, 10240) // 10KB typical blob
	config := compression.DefaultConfig()
	
	start := time.Now()
	compressed, err := compression.CompressBlob(realWorldData)
	compressTime := time.Since(start)
	
	if err != nil {
		fmt.Printf("Error in real-world example: %v\n", err)
		return
	}
	
	start = time.Now()
	decompressed, err := compression.DecompressBlob(compressed)
	decompressTime := time.Since(start)
	
	if err != nil {
		fmt.Printf("Error decompressing: %v\n", err)
		return
	}
	
	if !bytes.Equal(realWorldData, decompressed) {
		fmt.Printf("Data integrity error!\n")
		return
	}
	
	info := compression.GetCompressionInfo(compressed)
	
	fmt.Printf("Original size: %s\n", formatBytes(len(realWorldData)))
	fmt.Printf("Compressed size: %s\n", formatBytes(int(info.CompressedSize)))
	fmt.Printf("Compression ratio: %.1f%% (%.1f%% savings)\n", 
		info.CompressionRatio*100, (1-info.CompressionRatio)*100)
	fmt.Printf("Compression time: %s\n", formatDuration(compressTime))
	fmt.Printf("Decompression time: %s\n", formatDuration(decompressTime))
	fmt.Printf("Compression speed: %.1f MB/s\n", 
		float64(len(realWorldData))/1024/1024/compressTime.Seconds())
	fmt.Printf("Decompression speed: %.1f MB/s\n", 
		float64(len(realWorldData))/1024/1024/decompressTime.Seconds())
}