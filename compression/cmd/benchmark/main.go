package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"time"

	"github.com/evstack/ev-node/compression"
)

func main() {
	ctx := context.Background()
	
	fmt.Println("EV Node Blob Compression Benchmark")
	fmt.Println("==================================")
	fmt.Println()
	
	// Create benchmark suite
	suite, err := compression.NewBenchmarkSuite()
	if err != nil {
		log.Fatal("Failed to create benchmark suite:", err)
	}
	
	// Run comprehensive benchmarks
	fmt.Println("Running comprehensive benchmarks...")
	start := time.Now()
	
	result, err := suite.RunComprehensiveBenchmark(ctx)
	if err != nil {
		log.Fatal("Benchmark failed:", err)
	}
	
	duration := time.Since(start)
	fmt.Printf("Benchmarks completed in %v\n\n", duration)
	
	// Print detailed results
	printDetailedResults(result)
	
	// Print summary
	printSummary(result.Summary)
	
	// Generate recommendations
	printRecommendations(result.Summary)
	
	// Save results to JSON file if requested
	if len(os.Args) > 1 && os.Args[1] == "--save-json" {
		saveResultsToJSON(result)
	}
}

func printDetailedResults(result *compression.ComprehensiveBenchmarkResult) {
	fmt.Println("Detailed Benchmark Results")
	fmt.Println("==========================")
	
	for datasetName, results := range result.Results {
		fmt.Printf("\n%s:\n", datasetName)
		
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Algorithm\tOriginal Size\tCompressed Size\tRatio\tCompress Time\tDecompress Time\tCompress Rate\tDecompress Rate")
		fmt.Fprintln(w, "--------\t-------------\t---------------\t-----\t-------------\t---------------\t-------------\t---------------")
		
		for _, result := range results {
			compressTime := formatDuration(result.CompressTime)
			decompressTime := formatDuration(result.DecompressTime)
			compressRate := formatRate(result.CompressionRate)
			decompressRate := formatRate(result.DecompressionRate)
			
			fmt.Fprintf(w, "%s\t%s\t%s\t%.3f\t%s\t%s\t%s\t%s\n",
				result.Algorithm.String(),
				formatBytes(result.OriginalSize),
				formatBytes(result.CompressedSize),
				result.CompressionRatio,
				compressTime,
				decompressTime,
				compressRate,
				decompressRate,
			)
		}
		
		w.Flush()
	}
}

func printSummary(summary compression.BenchmarkSummary) {
	fmt.Println("\n\nBenchmark Summary")
	fmt.Println("================")
	
	fmt.Printf("Best Compression Ratio: %s\n", summary.BestCompressionRatio.String())
	fmt.Printf("Fastest Compression: %s\n", summary.FastestCompression.String())
	fmt.Printf("Fastest Decompression: %s\n", summary.FastestDecompression.String())
	fmt.Printf("Best Overall: %s\n", summary.BestOverall.String())
	
	fmt.Println("\nAverage Performance:")
	
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Algorithm\tAvg Ratio\tAvg Compress Time\tAvg Decompress Time")
	fmt.Fprintln(w, "--------\t---------\t-----------------\t-------------------")
	
	algorithms := []compression.Algorithm{compression.Gzip, compression.LZ4, compression.Zstd}
	for _, alg := range algorithms {
		if ratio, exists := summary.AverageRatios[alg]; exists {
			fmt.Fprintf(w, "%s\t%.3f\t%s\t%s\n",
				alg.String(),
				ratio,
				formatDuration(summary.AverageCompressTime[alg]),
				formatDuration(summary.AverageDecompressTime[alg]),
			)
		}
	}
	
	w.Flush()
}

func printRecommendations(summary compression.BenchmarkSummary) {
	fmt.Println("\n\nRecommendations")
	fmt.Println("===============")
	
	if summary.BestOverall != compression.None {
		fmt.Printf("🎯 Overall Best Choice: %s\n", summary.BestOverall.String())
		fmt.Println("   This algorithm provides the best balance of compression ratio and speed.")
	}
	
	if summary.BestCompressionRatio != compression.None {
		fmt.Printf("🗜️  Maximum Compression: %s\n", summary.BestCompressionRatio.String())
		fmt.Println("   Use this when storage/bandwidth costs are the primary concern.")
	}
	
	if summary.FastestCompression != compression.None {
		fmt.Printf("⚡ Fastest Compression: %s\n", summary.FastestCompression.String())
		fmt.Println("   Use this when compression speed is critical.")
	}
	
	if summary.FastestDecompression != compression.None {
		fmt.Printf("🚀 Fastest Decompression: %s\n", summary.FastestDecompression.String())
		fmt.Println("   Use this when decompression speed is critical (read-heavy workloads).")
	}
	
	fmt.Println("\nUse Case Guidelines:")
	fmt.Println("- High-throughput ingestion: LZ4 (fast compression)")
	fmt.Println("- Cost optimization: Zstd or Gzip (better compression)")
	fmt.Println("- Balanced workloads: Zstd (good compression + reasonable speed)")
	fmt.Println("- Legacy compatibility: Gzip (universally supported)")
}

func saveResultsToJSON(result *compression.ComprehensiveBenchmarkResult) {
	filename := fmt.Sprintf("compression_benchmark_%d.json", time.Now().Unix())
	
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal results: %v", err)
		return
	}
	
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		log.Printf("Failed to save results: %v", err)
		return
	}
	
	fmt.Printf("\n📊 Results saved to: %s\n", filename)
}

func formatBytes(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatDuration(nanoseconds int64) string {
	if nanoseconds == 0 {
		return "0"
	}
	
	duration := time.Duration(nanoseconds)
	
	if duration < time.Microsecond {
		return fmt.Sprintf("%dns", duration.Nanoseconds())
	} else if duration < time.Millisecond {
		return fmt.Sprintf("%.1fμs", float64(duration.Nanoseconds())/1000.0)
	} else if duration < time.Second {
		return fmt.Sprintf("%.1fms", float64(duration.Nanoseconds())/1000000.0)
	} else {
		return fmt.Sprintf("%.2fs", duration.Seconds())
	}
}

func formatRate(mbPerSecond float64) string {
	if mbPerSecond == 0 {
		return "0"
	}
	
	if mbPerSecond < 1 {
		return fmt.Sprintf("%.2f MB/s", mbPerSecond)
	} else if mbPerSecond < 100 {
		return fmt.Sprintf("%.1f MB/s", mbPerSecond)
	} else {
		return fmt.Sprintf("%.0f MB/s", mbPerSecond)
	}
}