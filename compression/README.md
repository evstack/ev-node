# EV-Node Blob Compression

This package provides transparent blob compression for EV-Node using **Zstd level 3** compression algorithm. It's designed to reduce bandwidth usage, storage costs, and improve overall performance of the EV node network while maintaining full backward compatibility.

## Features

- **Single Algorithm**: Uses Zstd level 3 for optimal balance of speed and compression ratio
- **Transparent Integration**: Wraps any existing DA layer without code changes
- **Smart Compression**: Only compresses when beneficial (configurable threshold)
- **Backward Compatibility**: Seamlessly handles existing uncompressed blobs
- **Zero Dependencies**: Minimal external dependencies (only zstd)
- **Production Ready**: Comprehensive test coverage and error handling

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "github.com/evstack/ev-node/compression"
    "github.com/evstack/ev-node/core/da"
)

func main() {
    // Wrap your existing DA layer
    baseDA := da.NewDummyDA(1024*1024, 1.0, 1.0, time.Second)
    
    config := compression.DefaultConfig() // Uses zstd level 3
    compressibleDA, err := compression.NewCompressibleDA(baseDA, config)
    if err != nil {
        panic(err)
    }
    defer compressibleDA.Close()
    
    // Use normally - compression is transparent
    ctx := context.Background()
    namespace := []byte("my-namespace")
    
    blobs := []da.Blob{
        []byte("Hello, compressed world!"),
        []byte("This data will be compressed automatically"),
    }
    
    // Submit (compresses automatically)
    ids, err := compressibleDA.Submit(ctx, blobs, 1.0, namespace)
    if err != nil {
        panic(err)
    }
    
    // Get (decompresses automatically)  
    retrieved, err := compressibleDA.Get(ctx, ids, namespace)
    if err != nil {
        panic(err)
    }
    
    // Data is identical to original
    fmt.Println("Original:", string(blobs[0]))
    fmt.Println("Retrieved:", string(retrieved[0]))
}
```

### Custom Configuration

```go
config := compression.Config{
    Enabled:             true,
    ZstdLevel:           3,  // Recommended level
    MinCompressionRatio: 0.1, // Only compress if >10% savings
}

compressibleDA, err := compression.NewCompressibleDA(baseDA, config)
```

### Standalone Compression

```go
// Compress a single blob
compressed, err := compression.CompressBlob(originalData)
if err != nil {
    return err
}

// Decompress  
decompressed, err := compression.DecompressBlob(compressed)
if err != nil {
    return err
}

// Analyze compression
info := compression.GetCompressionInfo(compressed)
fmt.Printf("Compressed: %v, Algorithm: %s, Ratio: %.2f\n", 
    info.IsCompressed, info.Algorithm, info.CompressionRatio)
```

## Performance

Based on benchmarks with typical EV-Node blob sizes (1-64KB):

| Data Type | Compression Ratio | Speed | Best Use Case |
|-----------|-------------------|-------|---------------|
| **Repetitive** | ~20-30% | 150-300 MB/s | Logs, repeated data |
| **JSON/Structured** | ~25-40% | 100-200 MB/s | Metadata, transactions |
| **Text** | ~35-50% | 120-250 MB/s | Natural language |
| **Random** | ~95-100% (uncompressed) | N/A | Encrypted data |

### Why Zstd Level 3?

- **Balanced Performance**: Good compression ratio with fast speed
- **Memory Efficient**: Lower memory usage than higher levels
- **Industry Standard**: Widely used default in production systems
- **EV-Node Optimized**: Ideal for typical blockchain blob sizes

## Compression Format

Each compressed blob includes a 9-byte header:

```
[Flag:1][OriginalSize:8][CompressedPayload:N]
```

- **Flag**: `0x00` = uncompressed, `0x01` = zstd
- **OriginalSize**: Little-endian uint64 of original data size
- **CompressedPayload**: The compressed (or original) data

This format ensures:
- **Backward Compatibility**: Legacy blobs without headers work seamlessly
- **Future Extensibility**: Flag byte allows for algorithm upgrades
- **Integrity Checking**: Original size validation after decompression

## Integration Examples

### With Celestia DA

```go
import (
    "github.com/evstack/ev-node/compression"
    "github.com/celestiaorg/celestia-node/nodebuilder"
)

// Create Celestia client
celestiaDA := celestia.NewCelestiaDA(client, namespace)

// Add compression layer
config := compression.DefaultConfig()
compressibleDA, err := compression.NewCompressibleDA(celestiaDA, config)
if err != nil {
    return err
}

// Use in EV-Node
node.SetDA(compressibleDA)
```

### With Custom DA

```go
// Any DA implementation
type CustomDA struct {
    // ... your implementation
}

func (c *CustomDA) Submit(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte) ([]da.ID, error) {
    // Add compression transparently
    config := compression.DefaultConfig()
    compressibleDA, err := compression.NewCompressibleDA(c, config)
    if err != nil {
        return nil, err
    }
    defer compressibleDA.Close()
    
    return compressibleDA.Submit(ctx, blobs, gasPrice, namespace)
}
```

## Benchmarking

Run performance benchmarks:

```bash
# Run default benchmark
go run ./compression/cmd/benchmark/main.go

# Run with custom iterations
go run ./compression/cmd/benchmark/main.go 1000

# Example output:
# === JSON Data Results ===
# Level  Size       Compressed   Ratio    Comp Time    Comp Speed      Decomp Time  Decomp Speed   
# -----  --------   ----------   ------   ---------    -----------     ----------   -------------  
# 3      10.0KB     3.2KB        0.320    45.2μs       221.0MB/s       28.1μs       355.2MB/s
```

## Testing

Run comprehensive tests:

```bash
# Unit tests
go test ./compression/...

# With coverage
go test -cover ./compression/...

# Verbose output
go test -v ./compression/...

# Benchmark tests
go test -bench=. ./compression/...
```

## Error Handling

The package provides specific error types:

```go
var (
    ErrInvalidHeader           = errors.New("invalid compression header")
    ErrInvalidCompressionFlag  = errors.New("invalid compression flag") 
    ErrDecompressionFailed     = errors.New("decompression failed")
)
```

Robust error handling:

```go
compressed, err := compression.CompressBlob(data)
if err != nil {
    log.Printf("Compression failed: %v", err)
    // Handle gracefully - could store uncompressed
}

decompressed, err := compression.DecompressBlob(compressed)
if err != nil {
    if errors.Is(err, compression.ErrDecompressionFailed) {
        log.Printf("Decompression failed, data may be corrupted: %v", err)
        // Handle corruption
    }
    return err
}
```

## Configuration Options

### Config Struct

```go
type Config struct {
    // Enabled controls whether compression is active
    Enabled bool
    
    // ZstdLevel is the compression level (1-22, recommended: 3)
    ZstdLevel int
    
    // MinCompressionRatio is the minimum savings required to store compressed
    // If compression doesn't achieve this ratio, data is stored uncompressed
    MinCompressionRatio float64
}
```

### Recommended Settings

```go
// Production (default)
config := compression.Config{
    Enabled:             true,
    ZstdLevel:           3,      // Balanced performance
    MinCompressionRatio: 0.1,    // 10% minimum savings
}

// High throughput
config := compression.Config{
    Enabled:             true,
    ZstdLevel:           1,      // Fastest compression
    MinCompressionRatio: 0.05,   // 5% minimum savings
}

// Maximum compression
config := compression.Config{
    Enabled:             true,
    ZstdLevel:           9,      // Better compression
    MinCompressionRatio: 0.15,   // 15% minimum savings
}

// Disabled (pass-through)
config := compression.Config{
    Enabled: false,
}
```

## Troubleshooting

### Common Issues

**Q: Compression not working?**
A: Check that `Config.Enabled = true` and your data meets the `MinCompressionRatio` threshold.

**Q: Performance slower than expected?**
A: Try lowering `ZstdLevel` to 1 for faster compression, or increase `MinCompressionRatio` to avoid compressing data that doesn't benefit.

**Q: Getting decompression errors?**
A: Ensure all nodes use compatible versions. Legacy blobs (without compression headers) are handled automatically.

**Q: Memory usage high?**
A: Call `compressibleDA.Close()` when done to free compression resources.

### Debug Information

```go
// Analyze blob compression status
info := compression.GetCompressionInfo(blob)
fmt.Printf("Compressed: %v\n", info.IsCompressed)
fmt.Printf("Algorithm: %s\n", info.Algorithm)  
fmt.Printf("Original: %d bytes\n", info.OriginalSize)
fmt.Printf("Compressed: %d bytes\n", info.CompressedSize)
fmt.Printf("Ratio: %.2f (%.1f%% savings)\n", 
    info.CompressionRatio, (1-info.CompressionRatio)*100)
```

## License

This package is part of EV-Node and follows the same license terms.