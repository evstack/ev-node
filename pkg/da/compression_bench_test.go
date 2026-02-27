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

// makeMegaBlobPayload builds a mega-blob by concatenating numBlocks
// independently serialized block payloads. Each block is a separate
// protobuf-encoded Data message at the given per-block size, with its own
// metadata (distinct height, chain ID, last-data-hash). This mirrors the
// production pattern where multiple blocks are batched into a single DA blob.
func makeMegaBlobPayload(numBlocks, perBlockBytes int) []byte {
	var combined []byte
	for i := 0; i < numBlocks; i++ {
		block := makeRealisticBlockPayload(perBlockBytes)
		combined = append(combined, block...)
	}
	return combined
}

// BenchmarkCompressMegaBlob benchmarks Compress on mega-blobs that aggregate
// multiple block payloads into a single compressed blob.
//
// Dimensions:
//
//	blocks:    2, 5, 10, 20  — number of blocks packed into one blob
//	per-block: 10 KB, 100 KB, 500 KB — size of each constituent block
//	level:     fastest, default, best
func BenchmarkCompressMegaBlob(b *testing.B) {
	//blockCounts := []int{2, 5, 10, 20, 30, 40, 50, 100}
	blockCounts := []int{ 150, 200, 250, 300, 350, 400, 500}
	perBlockSizes := []struct {
		name  string
		bytes int
	}{
		{"300", 300},
		{"1KB", 1 * 1024},
		//{"10KB", 10 * 1024},
		//{"100KB", 100 * 1024},
		//{"500KB", 500 * 1024},
	}

	levels := []struct {
		name  string
		level CompressionLevel
	}{
		{"fastest", LevelFastest},
		{"default", LevelDefault},
		{"best", LevelBest},
	}

	for _, nBlocks := range blockCounts {
		for _, pbs := range perBlockSizes {
			payload := makeMegaBlobPayload(nBlocks, pbs.bytes)

			for _, lvl := range levels {
				name := fmt.Sprintf("blocks=%d/per_block=%s/level=%s", nBlocks, pbs.name, lvl.name)
				b.Run(name, func(b *testing.B) {
					b.SetBytes(int64(len(payload)))
					b.ReportAllocs()

					compressed, _ := Compress(payload, lvl.level)
					ratio := float64(len(compressed)) / float64(len(payload))
					b.ResetTimer()
					b.ReportMetric(ratio, "ratio")
					b.ReportMetric(float64(len(payload)), "input_bytes")
					b.ReportMetric(float64(len(compressed)), "output_bytes")
					b.ReportMetric(float64(nBlocks), "num_blocks")
					for i := 0; i < b.N; i++ {
						_, _ = Compress(payload, lvl.level)
					}
				})
			}
		}
	}
}

// BenchmarkDecompressMegaBlob benchmarks Decompress on mega-blobs containing
// multiple blocks, paired with BenchmarkCompressMegaBlob.
func BenchmarkDecompressMegaBlob(b *testing.B) {
	blockCounts := []int{2, 5, 10, 20, 50, 100, 150}
	perBlockSizes := []struct {
		name  string
		bytes int
	}{
		{"200", 300},
		{"300", 300},
		{"1kB", 1 * 1024},
		{"10KB", 10 * 1024},
		//{"100KB", 100 * 1024},
		//{"500KB", 500 * 1024},
	}

	for _, nBlocks := range blockCounts {
		for _, pbs := range perBlockSizes {
			payload := makeMegaBlobPayload(nBlocks, pbs.bytes)
			compressed, err := Compress(payload, LevelDefault)
			if err != nil {
				b.Fatal(err)
			}
			ratio := float64(len(compressed)) / float64(len(payload))
			name := fmt.Sprintf("blocks=%d/per_block=%s", nBlocks, pbs.name)
			b.Run(name, func(b *testing.B) {
				b.SetBytes(int64(len(compressed)))
				b.ReportAllocs()
				b.ResetTimer()
				b.ReportMetric(ratio, "ratio")
				b.ReportMetric(float64(len(payload)), "input_bytes")
				b.ReportMetric(float64(len(compressed)), "output_bytes")
				b.ReportMetric(float64(nBlocks), "num_blocks")
				for i := 0; i < b.N; i++ {
					_, _ = Decompress(b.Context(), compressed)
				}
			})
		}
	}
}

// makeRealisticHeaderPayload builds a protobuf-serialized DAHeaderEnvelope
// that mirrors what the DA client submits for block headers in production.
// Each envelope contains a Header (version, height, time, hashes, chain ID,
// proposer/validator addresses), a 65-byte block signature, a Signer
// (20-byte address + 33-byte compressed secp256k1 pubkey), and a 65-byte
// envelope signature.
func makeRealisticHeaderPayload(height uint64) []byte {
	now := uint64(time.Now().UnixNano())

	// Deterministic 32-byte hashes (simulate real chain state).
	lastHeaderHash := make([]byte, 32)
	lastHeaderHash[0] = byte(height >> 8)
	lastHeaderHash[1] = byte(height)
	_, _ = rand.Read(lastHeaderHash[2:])

	dataHash := make([]byte, 32)
	_, _ = rand.Read(dataHash)

	appHash := make([]byte, 32)
	_, _ = rand.Read(appHash)

	validatorHash := make([]byte, 32)
	_, _ = rand.Read(validatorHash)

	proposerAddr := make([]byte, 20)
	proposerAddr[0] = 0xab
	proposerAddr[1] = 0xcd
	_, _ = rand.Read(proposerAddr[2:])

	// 33-byte compressed secp256k1 public key (0x02 or 0x03 prefix).
	pubKey := make([]byte, 33)
	pubKey[0] = 0x02
	_, _ = rand.Read(pubKey[1:])

	// 65-byte ECDSA signatures (look random).
	blockSig := make([]byte, 65)
	_, _ = rand.Read(blockSig)

	envelopeSig := make([]byte, 65)
	_, _ = rand.Read(envelopeSig)

	envelope := &pb.DAHeaderEnvelope{
		Header: &pb.Header{
			Version:         &pb.Version{Block: 1, App: 1},
			Height:          height,
			Time:            now,
			LastHeaderHash:  lastHeaderHash,
			DataHash:        dataHash,
			AppHash:         appHash,
			ProposerAddress: proposerAddr,
			ValidatorHash:   validatorHash,
			ChainId:         "evmos_9001-1",
		},
		Signature: blockSig,
		Signer: &pb.Signer{
			Address: proposerAddr,
			PubKey:  pubKey,
		},
		EnvelopeSignature: envelopeSig,
	}

	raw, err := proto.Marshal(envelope)
	if err != nil {
		panic(fmt.Sprintf("proto.Marshal DAHeaderEnvelope: %v", err))
	}
	return raw
}

// makeMegaBlobHeaderPayload concatenates numHeaders independently serialized
// DAHeaderEnvelope messages into a single mega-blob, simulating a batch of
// consecutive block headers submitted in one DA blob.
func makeMegaBlobHeaderPayload(numHeaders int) []byte {
	var combined []byte
	for i := 0; i < numHeaders; i++ {
		hdr := makeRealisticHeaderPayload(uint64(i + 1))
		combined = append(combined, hdr...)
	}
	return combined
}

// BenchmarkCompressMegaBlobHeaders benchmarks Compress on mega-blobs composed
// of realistic ev-node block headers (DAHeaderEnvelope).
//
// Each header is ~330 bytes in production. We test batches of 10–500 headers
// across all compression levels to characterize header blob compression.
func BenchmarkCompressMegaBlobHeaders(b *testing.B) {
	headerCounts := []int{10, 25, 50, 100, 200, 500}

	levels := []struct {
		name  string
		level CompressionLevel
	}{
		{"fastest", LevelFastest},
		{"default", LevelDefault},
		{"best", LevelBest},
	}

	for _, nHeaders := range headerCounts {
		payload := makeMegaBlobHeaderPayload(nHeaders)

		for _, lvl := range levels {
			name := fmt.Sprintf("headers=%d/level=%s", nHeaders, lvl.name)
			b.Run(name, func(b *testing.B) {
				b.SetBytes(int64(len(payload)))
				b.ReportAllocs()

				compressed, _ := Compress(payload, lvl.level)
				ratio := float64(len(compressed)) / float64(len(payload))
				b.ResetTimer()
				b.ReportMetric(ratio, "ratio")
				b.ReportMetric(float64(len(payload)), "input_bytes")
				b.ReportMetric(float64(len(compressed)), "output_bytes")
				b.ReportMetric(float64(nHeaders), "num_headers")
				for i := 0; i < b.N; i++ {
					_, _ = Compress(payload, lvl.level)
				}
			})
		}
	}
}

// BenchmarkDecompressMegaBlobHeaders benchmarks Decompress on mega-blobs
// of block headers, paired with BenchmarkCompressMegaBlobHeaders.
func BenchmarkDecompressMegaBlobHeaders(b *testing.B) {
	headerCounts := []int{10, 25, 50, 100, 200, 500}

	for _, nHeaders := range headerCounts {
		payload := makeMegaBlobHeaderPayload(nHeaders)
		compressed, err := Compress(payload, LevelDefault)
		if err != nil {
			b.Fatal(err)
		}

		ratio := float64(len(compressed)) / float64(len(payload))
		name := fmt.Sprintf("headers=%d", nHeaders)
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(compressed)))
			b.ReportAllocs()
			b.ResetTimer()
			b.ReportMetric(ratio, "ratio")
			b.ReportMetric(float64(len(payload)), "input_bytes")
			b.ReportMetric(float64(len(compressed)), "output_bytes")
			b.ReportMetric(float64(nHeaders), "num_headers")
			for i := 0; i < b.N; i++ {
				_, _ = Decompress(b.Context(), compressed)
			}
		})
	}
}
