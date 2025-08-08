# Blob Compression for EV Node

This package provides comprehensive blob compression functionality for EV Node, reducing bandwidth usage, storage costs, and improving overall network performance.

## Features

- **Multiple Compression Algorithms**: Gzip (compatibility), LZ4 (speed), Zstd (balance)
- **Transparent Integration**: Drop-in wrapper for existing DA implementations
- **Smart Compression**: Only compresses when beneficial based on configurable thresholds
- **Backward Compatibility**: Seamlessly handles both compressed and uncompressed blobs
- **Comprehensive Benchmarking**: Built-in performance analysis and algorithm comparison
- **Configurable Parameters**: Customizable compression levels and behavior

## Quick Start

### Basic Compression

```go
import "github.com/evstack/ev-node/compression"

// Create a compressor
compressor, err := compression.NewCompressor()
if err != nil {
    log.Fatal(err)
}

// Compress data
data := []byte("Your blob data here")
compressed, err := compressor.Compress(ctx, data, compression.Zstd)
if err != nil {
    log.Fatal(err)
}

// Decompress data
decompressed, err := compressor.Decompress(ctx, compressed)
if err != nil {
    log.Fatal(err)
}
```

### DA Integration

```go
// Wrap your existing DA with compression
config := compression.CompressibleDAConfig{
    Enabled:   true,
    Algorithm: "zstd",
    ZstdLevel: 3,
    MinCompressionRatio: 0.1, // Only keep compression if it saves >= 10%
}

compressibleDA, err := compression.NewCompressibleDA(baseDA, config)
if err != nil {
    log.Fatal(err)
}

// Use normally - compression happens automatically
ids, err := compressibleDA.Submit(ctx, blobs, gasPrice, namespace)
retrievedBlobs, err := compressibleDA.Get(ctx, ids, namespace)
```

## Compression Algorithms

| Algorithm | Use Case | Compression Ratio | Speed | Compatibility |
|-----------|----------|-------------------|-------|---------------|
| **Gzip** | Maximum compatibility | Good | Moderate | Universal |
| **LZ4** | High-throughput ingestion | Moderate | Very Fast | Good |
| **Zstd** | Balanced workloads | Very Good | Fast | Good |

### Algorithm Selection Guidelines

- **High-throughput scenarios**: Use LZ4 for fast compression with reasonable ratios
- **Cost optimization**: Use Zstd or Gzip for better compression ratios
- **Legacy systems**: Use Gzip for maximum compatibility
- **Balanced workloads**: Use Zstd for the best overall performance

## Configuration Options

### Compressor Options

```go
compressor, err := compression.NewCompressor(
    compression.WithZstdLevel(22),              // Maximum compression
    compression.WithGzipLevel(9),               // Maximum gzip compression
    compression.WithLZ4Level(12),               // High compression LZ4
    compression.WithMinCompressionRatio(0.05),  // Very aggressive threshold
)
```

### DA Configuration

```go
config := compression.CompressibleDAConfig{
    Enabled:             true,
    Algorithm:           "zstd",
    ZstdLevel:           3,     // 1-22 (higher = better compression, slower)
    GzipLevel:           6,     // 1-9 (higher = better compression, slower)  
    LZ4Level:            0,     // 0 = fast mode, 1-12 = high compression
    MinCompressionRatio: 0.1,   // Don't compress if savings < 10%
}
```

## Benchmarking

### Run Comprehensive Benchmarks

```go
suite, err := compression.NewBenchmarkSuite()
result, err := suite.RunComprehensiveBenchmark(ctx)

// Print results
fmt.Printf("Best overall algorithm: %s\n", result.Summary.BestOverall)
fmt.Printf("Best compression ratio: %s\n", result.Summary.BestCompressionRatio)
fmt.Printf("Fastest compression: %s\n", result.Summary.FastestCompression)
```

### CLI Benchmark Tool

```bash
go run ./compression/cmd/benchmark/main.go
go run ./compression/cmd/benchmark/main.go --save-json  # Save results to JSON
```

### Benchmark Your Data

```go
suite, _ := compression.NewBenchmarkSuite()
results, _ := suite.BenchmarkAllAlgorithms(ctx, yourData)

for _, result := range results {
    fmt.Printf("%s: %.3f ratio, %.1f MB/s\n", 
        result.Algorithm, 
        result.CompressionRatio,
        result.CompressionRate)
}
```

## Performance Characteristics

### Typical Performance (10KB data)

| Algorithm | Compression Ratio | Compress Speed | Decompress Speed |
|-----------|-------------------|----------------|------------------|
| Gzip      | 0.25-0.35        | 40-80 MB/s     | 150-300 MB/s     |
| LZ4       | 0.35-0.45        | 200-400 MB/s   | 800-1200 MB/s    |
| Zstd      | 0.20-0.30        | 80-150 MB/s    | 300-500 MB/s     |

*Performance varies significantly based on data characteristics*

## Data Format

### Compressed Blob Structure

```
[Magic Bytes: 4] [Algorithm: 1] [Original Size: 4] [Compressed Data: N]
     "EVCZ"         0-3 enum      big-endian uint32      variable length
```

- **Magic Bytes**: `0x45564356` ("EVCZ") identifies compressed blobs
- **Algorithm**: 0=None, 1=Gzip, 2=LZ4, 3=Zstd
- **Original Size**: Uncompressed data size for validation
- **Compressed Data**: The actual compressed payload

### Backward Compatibility

The system automatically detects and handles:
- Existing uncompressed blobs (pass-through)
- Blobs compressed with different algorithms
- Mixed compressed/uncompressed data in the same namespace

## Advanced Usage

### Compression Statistics

```go
stats, err := compressibleDA.GetCompressionStats(ctx, blobs)
for i, stat := range stats {
    fmt.Printf("Blob %d: %d → %d bytes (%.1f%% savings)\n",
        i, stat.OriginalSize, stat.CompressedSize, stat.SavingsPercent)
}
```

### Custom Algorithm Selection

```go
// Benchmark your specific data patterns
testDatasets := [][]byte{
    yourProtobufData,
    yourJSONData,
    yourBinaryData,
}

bestAlgorithm := compression.None
bestScore := 0.0

for _, data := range testDatasets {
    results, _ := suite.BenchmarkAllAlgorithms(ctx, data)
    for _, result := range results {
        // Custom scoring function based on your priorities
        score := (1.0 - result.CompressionRatio) * 0.7 + // 70% weight on compression
                 (result.CompressionRate / 100.0) * 0.3   // 30% weight on speed
        
        if score > bestScore {
            bestScore = score
            bestAlgorithm = result.Algorithm
        }
    }
}
```

### Error Handling

The compression system is designed to be robust:

- **Compression failures**: Falls back to uncompressed data
- **Decompression failures**: Returns appropriate errors
- **Invalid data**: Detects and handles corrupted compressed blobs
- **Unsupported algorithms**: Graceful error handling

## Integration Examples

### With Block Manager

```go
// In your block submission logic
config := compression.CompressibleDAConfig{
    Enabled:   true,
    Algorithm: "zstd",
    ZstdLevel: 3,
}

compressibleDA, _ := compression.NewCompressibleDA(baseDA, config)

// Headers and data are now automatically compressed
headerIDs, _ := compressibleDA.Submit(ctx, headerBlobs, gasPrice, headerNamespace)
dataIDs, _ := compressibleDA.Submit(ctx, dataBlobs, gasPrice, dataNamespace)
```

### Configuration Management

```go
// Add to your node configuration
type NodeConfig struct {
    // ... existing fields
    Compression compression.CompressibleDAConfig `yaml:"compression"`
}

// In your initialization code
if config.Compression.Enabled {
    node.DA, err = compression.NewCompressibleDA(node.DA, config.Compression)
}
```

## Monitoring and Observability

### Metrics to Track

- Compression ratios achieved
- Compression/decompression latency
- Bandwidth savings
- Storage savings
- Algorithm performance over time

### Logging

The compression system provides structured logging for:
- Compression decisions (compress vs. skip)
- Performance metrics
- Error conditions
- Algorithm selection rationale

## Best Practices

1. **Profile Your Data**: Use the benchmarking tools to test with your actual data patterns
2. **Monitor Performance**: Track compression ratios and latency in production
3. **Start Conservative**: Begin with moderate compression levels and adjust based on results
4. **Consider Network vs. CPU**: Balance compression overhead against bandwidth savings
5. **Test Thoroughly**: Validate round-trip compression/decompression with your data
6. **Plan for Growth**: Consider how compression performance scales with data size

## Troubleshooting

### Common Issues

**Poor Compression Ratios**:
- Data may already be compressed or encrypted
- Try different algorithms
- Adjust `MinCompressionRatio` threshold

**High CPU Usage**:
- Reduce compression levels
- Switch to LZ4 for faster compression
- Monitor compression vs. bandwidth tradeoffs

**Compatibility Issues**:
- Ensure all nodes support the same compression algorithms
- Use Gzip for maximum compatibility
- Test backward compatibility thoroughly