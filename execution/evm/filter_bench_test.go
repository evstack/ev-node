package evm

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
)

const (
	benchPrivateKey = "cece4f25ac74deb1468965160c7185e07dff413f23fcadb611b05ca37ab0a52e"
	benchToAddress  = "0x944fDcD1c868E3cC566C78023CcB38A32cDA836E"
	benchChainID    = "1234"
)

// createBenchClient creates a minimal EngineClient for benchmarking FilterTxs.
// It only needs the logger to be set for the FilterTxs method.
func createBenchClient(b *testing.B) *EngineClient {
	b.Helper()
	baseStore := dssync.MutexWrap(ds.NewMapDatastore())
	store := NewEVMStore(baseStore)
	client := &EngineClient{
		store: store,
	}
	return client
}

// generateSignedTransaction creates a valid signed Ethereum transaction for benchmarking.
func generateSignedTransaction(b *testing.B, nonce uint64, gasLimit uint64) []byte {
	b.Helper()

	privateKey, err := crypto.HexToECDSA(benchPrivateKey)
	if err != nil {
		b.Fatalf("failed to parse private key: %v", err)
	}

	chainID, ok := new(big.Int).SetString(benchChainID, 10)
	if !ok {
		b.Fatalf("failed to parse chain ID")
	}

	toAddress := common.HexToAddress(benchToAddress)
	txValue := big.NewInt(1000000000000000000)
	gasPrice := big.NewInt(30000000000)

	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		b.Fatalf("failed to generate random data: %v", err)
	}

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &toAddress,
		Value:    txValue,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		b.Fatalf("failed to sign transaction: %v", err)
	}

	txBytes, err := signedTx.MarshalBinary()
	if err != nil {
		b.Fatalf("failed to marshal transaction: %v", err)
	}

	return txBytes
}

// generateTransactionBatch creates a batch of signed transactions for benchmarking.
func generateTransactionBatch(b *testing.B, count int, gasLimit uint64) [][]byte {
	b.Helper()
	txs := make([][]byte, count)
	for i := 0; i < count; i++ {
		txs[i] = generateSignedTransaction(b, uint64(i), gasLimit)
	}
	return txs
}

// generateMixedTransactionBatch creates a batch with some valid and some invalid transactions.
// forcedRatio is the percentage of transactions that are "forced" (could include invalid ones).
func generateMixedTransactionBatch(b *testing.B, count int, gasLimit uint64, includeInvalid bool) [][]byte {
	b.Helper()
	txs := make([][]byte, count)
	for i := 0; i < count; i++ {
		if includeInvalid && i%10 == 0 {
			// Every 10th transaction is invalid (random garbage)
			txs[i] = make([]byte, 100)
			if _, err := rand.Read(txs[i]); err != nil {
				b.Fatalf("failed to generate random data: %v", err)
			}
		} else {
			txs[i] = generateSignedTransaction(b, uint64(i), gasLimit)
		}
	}
	return txs
}

func benchName(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%dk", n/1000)
	}
	return fmt.Sprintf("%d", n)
}

// BenchmarkFilterTxs_OnlyNormalTxs benchmarks FilterTxs when hasForceIncludedTransaction is false.
// In this case, UnmarshalBinary is NOT called - mempool transactions are already validated.
func BenchmarkFilterTxs_OnlyNormalTxs(b *testing.B) {
	client := createBenchClient(b)
	ctx := context.Background()

	txCounts := []int{100, 1000, 10000}

	for _, count := range txCounts {
		b.Run(benchName(count), func(b *testing.B) {
			txs := generateTransactionBatch(b, count, 21000)

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				// hasForceIncludedTransaction=false means UnmarshalBinary is skipped
				_, err := client.FilterTxs(ctx, txs, 0, 0, false)
				if err != nil {
					b.Fatalf("FilterTxs failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkFilterTxs_WithForcedTxs benchmarks FilterTxs when hasForceIncludedTransaction is true.
// In this case, UnmarshalBinary IS called for every transaction to validate and extract gas.
func BenchmarkFilterTxs_WithForcedTxs(b *testing.B) {
	client := createBenchClient(b)
	ctx := context.Background()

	txCounts := []int{100, 1000, 10000}

	for _, count := range txCounts {
		b.Run(benchName(count), func(b *testing.B) {
			txs := generateTransactionBatch(b, count, 21000)

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				// hasForceIncludedTransaction=true triggers UnmarshalBinary path
				_, err := client.FilterTxs(ctx, txs, 0, 0, true)
				if err != nil {
					b.Fatalf("FilterTxs failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkFilterTxs_MixedWithInvalidTxs benchmarks FilterTxs with a mix of valid and invalid transactions.
// This tests the UnmarshalBinary error handling path.
func BenchmarkFilterTxs_MixedWithInvalidTxs(b *testing.B) {
	client := createBenchClient(b)
	ctx := context.Background()

	txCounts := []int{100, 1000, 10000}

	for _, count := range txCounts {
		b.Run(benchName(count), func(b *testing.B) {
			// Generate batch with 10% invalid transactions
			txs := generateMixedTransactionBatch(b, count, 21000, true)

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				// hasForceIncludedTransaction=true triggers UnmarshalBinary path
				_, err := client.FilterTxs(ctx, txs, 0, 0, true)
				if err != nil {
					b.Fatalf("FilterTxs failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkFilterTxs_WithGasLimit benchmarks FilterTxs with gas limit enforcement.
// This tests the full path including gas accumulation logic.
func BenchmarkFilterTxs_WithGasLimit(b *testing.B) {
	client := createBenchClient(b)
	ctx := context.Background()

	txCounts := []int{100, 1000}
	// Gas limit that allows roughly half the transactions
	maxGas := uint64(21000 * 500)

	for _, count := range txCounts {
		b.Run(benchName(count), func(b *testing.B) {
			txs := generateTransactionBatch(b, count, 21000)

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				// With gas limit and forced transactions
				_, err := client.FilterTxs(ctx, txs, 0, maxGas, true)
				if err != nil {
					b.Fatalf("FilterTxs failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkFilterTxs_WithSizeLimit benchmarks FilterTxs with byte size limit enforcement.
func BenchmarkFilterTxs_WithSizeLimit(b *testing.B) {
	client := createBenchClient(b)
	ctx := context.Background()

	txCounts := []int{100, 1000}
	// Size limit that allows roughly half the transactions (~110 bytes per tx)
	maxBytes := uint64(110 * 500)

	for _, count := range txCounts {
		b.Run(benchName(count)+"_noForced", func(b *testing.B) {
			txs := generateTransactionBatch(b, count, 21000)

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				// Without forced transactions - UnmarshalBinary skipped
				_, err := client.FilterTxs(ctx, txs, maxBytes, 0, false)
				if err != nil {
					b.Fatalf("FilterTxs failed: %v", err)
				}
			}
		})

		b.Run(benchName(count)+"_withForced", func(b *testing.B) {
			txs := generateTransactionBatch(b, count, 21000)

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				// With forced transactions - UnmarshalBinary called
				_, err := client.FilterTxs(ctx, txs, maxBytes, 0, true)
				if err != nil {
					b.Fatalf("FilterTxs failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkFilterTxs_CompareUnmarshalOverhead directly compares the overhead of UnmarshalBinary.
// Runs the same transaction set with and without the UnmarshalBinary path.
func BenchmarkFilterTxs_CompareUnmarshalOverhead(b *testing.B) {
	client := createBenchClient(b)
	ctx := context.Background()

	count := 1000
	txs := generateTransactionBatch(b, count, 21000)

	b.Run("without_unmarshal", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			_, err := client.FilterTxs(ctx, txs, 0, 0, false)
			if err != nil {
				b.Fatalf("FilterTxs failed: %v", err)
			}
		}
	})

	b.Run("with_unmarshal", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			_, err := client.FilterTxs(ctx, txs, 0, 0, true)
			if err != nil {
				b.Fatalf("FilterTxs failed: %v", err)
			}
		}
	})
}
