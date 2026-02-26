package da

import (
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// makeRealisticBlockPayload builds a protobuf-serialized Data message that
// mirrors what the DA client compresses in production. Each transaction
// resembles a signed EVM call: 1-byte type prefix, 8-byte nonce, 8-byte gas,
// 20-byte address, deterministic calldata, and a 65-byte secp256k1 signature.
func makeRealisticBlockPayload(targetBytes int) []byte {
	const (
		txOverhead = 1 + 8 + 8 + 20 + 65 // type + nonce + gas + to + sig ≈ 102 bytes
		minTxSize  = txOverhead + 4      // smallest calldata = 4 bytes (selector)
	)

	// Decide per-tx calldata size so we land close to targetBytes after proto
	// encoding. Proto overhead per tx is ~4 bytes (field tag + varint length).
	// We aim for 200-byte average calldata for realism, but scale up if needed.
	calldataSize := 200
	estTxSize := txOverhead + calldataSize + 4 // +4 proto overhead
	numTxs := targetBytes / estTxSize
	if numTxs < 1 {
		numTxs = 1
		calldataSize = targetBytes - txOverhead - 4
		if calldataSize < 4 {
			calldataSize = 4
		}
	}

	now := uint64(time.Now().UnixNano())

	txs := make([][]byte, numTxs)
	for i := range txs {
		calldata := make([]byte, calldataSize)
		// First 4 bytes are a function selector (deterministic per-tx)
		calldata[0] = 0xa9
		calldata[1] = 0x05
		calldata[2] = 0x9c
		calldata[3] = 0xbb
		// Remaining calldata is semi-random (simulates ABI-encoded args).
		// Use crypto/rand for realistic entropy — EVM calldata is not
		// compressible in the general case.
		_, _ = rand.Read(calldata[4:])

		tx := make([]byte, 0, txOverhead+calldataSize)
		tx = append(tx, 0x02) // EIP-1559 type prefix

		// nonce (big-endian)
		nonce := uint64(i)
		tx = append(tx,
			byte(nonce>>56), byte(nonce>>48), byte(nonce>>40), byte(nonce>>32),
			byte(nonce>>24), byte(nonce>>16), byte(nonce>>8), byte(nonce),
		)

		// gas limit
		gas := uint64(21_000 + len(calldata)*16)
		tx = append(tx,
			byte(gas>>56), byte(gas>>48), byte(gas>>40), byte(gas>>32),
			byte(gas>>24), byte(gas>>16), byte(gas>>8), byte(gas),
		)

		// to address (deterministic)
		addr := make([]byte, 20)
		addr[0] = 0xde
		addr[1] = 0xad
		addr[19] = byte(i % 256)
		tx = append(tx, addr...)

		// calldata
		tx = append(tx, calldata...)

		// signature (65 bytes of random — signatures look random)
		sig := make([]byte, 65)
		_, _ = rand.Read(sig)
		tx = append(tx, sig...)

		txs[i] = tx
	}
	sig := make([]byte, 32)
	_, _ = rand.Read(sig)
	// Build a protobuf Data message identical to what the submitter serializes.
	dataProto := &pb.Data{
		Metadata: &pb.Metadata{
			ChainId:      "testing",
			Height:       42,
			Time:         now,
			LastDataHash: sig,
		},
		Txs: txs,
	}

	raw, err := proto.Marshal(dataProto)
	if err != nil {
		panic(fmt.Sprintf("proto.Marshal: %v", err))
	}
	return raw
}

// BenchmarkCompress benchmarks Compress across realistic rollup block sizes
// and all three compression levels.
//
// Block sizes are chosen to match real-world DA submission patterns:
//
//	1 KB   — single small transaction (token transfer)
//	10 KB  — a few contract calls
//	100 KB — moderately busy block
//	500 KB — busy block with large calldata
//	1 MB   — heavy block (batch posting, large calldata)
//	2 MB   — near the DefaultMaxBlobSize limit
func BenchmarkCompress(b *testing.B) {
	sizes := []struct {
		name  string
		bytes int
	}{
		{"1KB", 1 * 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"500KB", 500 * 1024},
		{"1MB", 1 * 1024 * 1024},
		{"2MB", 2 * 1024 * 1024},
		{"5MB", 2 * 1024 * 1024},
	}

	levels := []struct {
		name  string
		level CompressionLevel
	}{
		{"fastest", LevelFastest},
		{"default", LevelDefault},
		{"best", LevelBest},
	}

	for _, sz := range sizes {
		payload := makeRealisticBlockPayload(sz.bytes)

		for _, lvl := range levels {
			name := fmt.Sprintf("size=%s/level=%s", sz.name, lvl.name)
			b.Run(name, func(b *testing.B) {
				b.SetBytes(int64(len(payload)))
				b.ReportAllocs()

				// Run once to report compression ratio.
				compressed, _ := Compress(payload, lvl.level)
				ratio := float64(len(compressed)) / float64(len(payload))
				b.ResetTimer()
				b.ReportMetric(ratio, "ratio")
				b.ReportMetric(float64(len(payload)), "input_bytes")
				b.ReportMetric(float64(len(compressed)), "output_bytes")
				for i := 0; i < b.N; i++ {
					_, _ = Compress(payload, lvl.level)
				}
			})
		}
	}
}

// BenchmarkDecompress benchmarks Decompress to pair with the compress benchmark.
func BenchmarkDecompress(b *testing.B) {
	sizes := []struct {
		name  string
		bytes int
	}{
		{"1KB", 1 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1 * 1024 * 1024},
		{"2MB", 2 * 1024 * 1024},
		{"5MB", 2 * 1024 * 1024},
	}

	for _, sz := range sizes {
		payload := makeRealisticBlockPayload(sz.bytes)
		compressed, err := Compress(payload, LevelDefault)
		if err != nil {
			b.Fatal(err)
		}

		ratio := float64(len(compressed)) / float64(len(payload))
		name := fmt.Sprintf("size=%s", sz.name)
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(compressed)))
			b.ReportAllocs()
			b.ResetTimer()
			b.ReportMetric(ratio, "ratio")
			b.ReportMetric(float64(len(payload)), "input_bytes")
			b.ReportMetric(float64(len(compressed)), "output_bytes")
			for i := 0; i < b.N; i++ {
				_, _ = Decompress(b.Context(), compressed)
			}
		})
	}
}
