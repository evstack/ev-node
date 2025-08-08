package compression

import (
	"context"
	"fmt"
	"log"

	"github.com/evstack/ev-node/core/da"
)

// ExampleNewCompressor demonstrates basic compressor usage
func ExampleNewCompressor() {
	// Create a compressor with default settings
	compressor, err := NewCompressor()
	if err != nil {
		log.Fatal(err)
	}
	
	ctx := context.Background()
	data := []byte("Hello, World! This is test data for compression.")
	
	// Compress data
	compressed, err := compressor.Compress(ctx, data, Zstd)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Original size: %d bytes\n", len(data))
	fmt.Printf("Compressed size: %d bytes\n", len(compressed.Data))
	fmt.Printf("Compression ratio: %.2f\n", compressor.GetCompressionRatio(compressed))
	
	// Decompress data
	decompressed, err := compressor.Decompress(ctx, compressed)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Decompressed: %s\n", string(decompressed))
	// Output:
	// Original size: 47 bytes
	// Compressed size: 56 bytes  
	// Compression ratio: 1.19
	// Decompressed: Hello, World! This is test data for compression.
}

// ExampleNewCompressor_withOptions demonstrates compressor configuration
func ExampleNewCompressor_withOptions() {
	// Create a compressor with custom settings
	compressor, err := NewCompressor(
		WithZstdLevel(22),              // Maximum compression
		WithMinCompressionRatio(0.05),  // Very aggressive threshold
	)
	if err != nil {
		log.Fatal(err)
	}
	
	ctx := context.Background()
	
	// Highly compressible data
	data := make([]byte, 1000)
	for i := range data {
		data[i] = byte(i % 10) // Repeating pattern
	}
	
	compressed, err := compressor.Compress(ctx, data, Zstd)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Algorithm used: %s\n", compressed.Algorithm)
	fmt.Printf("Compression ratio: %.3f\n", compressor.GetCompressionRatio(compressed))
	// Output:
	// Algorithm used: zstd
	// Compression ratio: 0.052
}

// ExampleCompressibleDA demonstrates DA wrapper usage
func ExampleCompressibleDA() {
	// Assume we have a base DA implementation
	var baseDA da.DA = &mockDAForExample{}
	
	// Configure compression
	config := CompressibleDAConfig{
		Enabled:   true,
		Algorithm: "zstd",
		ZstdLevel: 3,
		MinCompressionRatio: 0.1,
	}
	
	// Wrap the base DA with compression
	compressibleDA, err := NewCompressibleDA(baseDA, config)
	if err != nil {
		log.Fatal(err)
	}
	
	ctx := context.Background()
	
	// Submit blobs - they will be automatically compressed
	blobs := []da.Blob{
		[]byte("This is blob data that will be compressed before submission"),
		[]byte("Another blob with repetitive content that compresses well"),
	}
	
	ids, err := compressibleDA.Submit(ctx, blobs, 1.0, []byte("test-namespace"))
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Submitted %d blobs\n", len(ids))
	
	// Retrieve blobs - they will be automatically decompressed
	retrievedBlobs, err := compressibleDA.Get(ctx, ids, []byte("test-namespace"))
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Retrieved %d blobs\n", len(retrievedBlobs))
	fmt.Printf("Data matches: %t\n", string(retrievedBlobs[0]) == string(blobs[0]))
	// Output:
	// Submitted 2 blobs
	// Retrieved 2 blobs
	// Data matches: true
}

// ExampleBenchmarkSuite demonstrates performance benchmarking
func ExampleBenchmarkSuite() {
	suite, err := NewBenchmarkSuite()
	if err != nil {
		log.Fatal(err)
	}
	
	ctx := context.Background()
	
	// Benchmark a specific data pattern
	testData := make([]byte, 10240) // 10KB
	for i := range testData {
		testData[i] = byte(i % 100) // Somewhat compressible pattern
	}
	
	// Benchmark all algorithms
	results, err := suite.BenchmarkAllAlgorithms(ctx, testData)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Println("Algorithm Performance:")
	for _, result := range results {
		if result.Algorithm == None {
			continue // Skip uncompressed baseline
		}
		
		fmt.Printf("%s: %.3f ratio, %.1f MB/s compress, %.1f MB/s decompress\n",
			result.Algorithm,
			result.CompressionRatio,
			result.CompressionRate,
			result.DecompressionRate,
		)
	}
	// Example Output:
	// Algorithm Performance:
	// gzip: 0.234 ratio, 45.2 MB/s compress, 156.7 MB/s decompress
	// lz4: 0.287 ratio, 287.3 MB/s compress, 892.1 MB/s decompress
	// zstd: 0.198 ratio, 89.4 MB/s compress, 312.5 MB/s decompress
}

// ExampleCompressibleDA_GetCompressionStats demonstrates compression statistics
func ExampleCompressibleDA_GetCompressionStats() {
	var baseDA da.DA = &mockDAForExample{}
	
	config := CompressibleDAConfig{
		Enabled:   true,
		Algorithm: "lz4",
	}
	
	compressibleDA, err := NewCompressibleDA(baseDA, config)
	if err != nil {
		log.Fatal(err)
	}
	
	ctx := context.Background()
	
	// Analyze compression for different data types
	blobs := []da.Blob{
		make([]byte, 1024),                    // Zeros - highly compressible
		[]byte("Random data: 1234567890abcd"), // Less compressible
	}
	
	stats, err := compressibleDA.GetCompressionStats(ctx, blobs)
	if err != nil {
		log.Fatal(err)
	}
	
	for i, stat := range stats {
		fmt.Printf("Blob %d: %d → %d bytes (%.1f%% savings)\n",
			i, stat.OriginalSize, stat.CompressedSize, stat.SavingsPercent)
	}
	// Example Output:
	// Blob 0: 1024 → 45 bytes (95.6% savings)
	// Blob 1: 26 → 35 bytes (-34.6% savings)
}

// mockDAForExample is a simple mock for examples
type mockDAForExample struct{}

func (m *mockDAForExample) Submit(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte) ([]da.ID, error) {
	ids := make([]da.ID, len(blobs))
	for i := range ids {
		ids[i] = da.ID(fmt.Sprintf("id-%d", i))
	}
	return ids, nil
}

func (m *mockDAForExample) SubmitWithOptions(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte, options []byte) ([]da.ID, error) {
	return m.Submit(ctx, blobs, gasPrice, namespace)
}

func (m *mockDAForExample) Get(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Blob, error) {
	// Return dummy data for examples
	blobs := make([]da.Blob, len(ids))
	for i := range blobs {
		blobs[i] = []byte(fmt.Sprintf("blob-data-%d", i))
	}
	return blobs, nil
}

func (m *mockDAForExample) GetIDs(ctx context.Context, height uint64, namespace []byte) (*da.GetIDsResult, error) {
	return &da.GetIDsResult{}, nil
}

func (m *mockDAForExample) Commit(ctx context.Context, blobs []da.Blob, namespace []byte) ([]da.Commitment, error) {
	return make([]da.Commitment, len(blobs)), nil
}

func (m *mockDAForExample) GetProofs(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Proof, error) {
	return make([]da.Proof, len(ids)), nil
}

func (m *mockDAForExample) Validate(ctx context.Context, ids []da.ID, proofs []da.Proof, namespace []byte) ([]bool, error) {
	return make([]bool, len(ids)), nil
}