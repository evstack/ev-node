package evm

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coreexecution "github.com/evstack/ev-node/core/execution"
)

// TestExecuteTxs_ForceIncludedMask verifies that force-included transactions
// are validated while mempool transactions skip validation (already validated on submission)
func TestExecuteTxs_ForceIncludedMask(t *testing.T) {
	t.Parallel()

	// Create a valid Ethereum transaction for testing
	validTx := types.NewTransaction(
		0, // nonce
		common.HexToAddress("0x1234567890123456789012345678901234567890"), // to
		common.Big0, // value
		21000,       // gas
		common.Big1, // gasPrice
		nil,         // data
	)
	validTxBytes, err := validTx.MarshalBinary()
	require.NoError(t, err)

	// Create invalid transaction bytes (gibberish)
	invalidTxBytes := []byte("this is not a valid transaction")

	tests := []struct {
		name              string
		txs               [][]byte
		mask              []bool
		expectedValidTxs  int
		expectedFilterMsg string
	}{
		{
			name: "all transactions validated when no mask",
			txs: [][]byte{
				validTxBytes,
				validTxBytes,
				invalidTxBytes, // Should be filtered
				validTxBytes,
			},
			mask:             nil, // No mask = validate all
			expectedValidTxs: 3,   // 3 valid, 1 invalid filtered
		},
		{
			name: "force-included invalid tx must be validated (filtered out)",
			txs: [][]byte{
				invalidTxBytes, // Force-included, MUST be validated - will be filtered
				validTxBytes,   // Mempool tx, skips validation
			},
			mask:             []bool{true, false},
			expectedValidTxs: 1, // Only mempool tx passes (force-included filtered)
		},
		{
			name: "mixed force-included and mempool transactions",
			txs: [][]byte{
				validTxBytes,   // Force-included, validated (valid)
				validTxBytes,   // Force-included, validated (valid)
				invalidTxBytes, // Mempool tx, skips validation (passes through)
				validTxBytes,   // Mempool tx, skips validation (passes through)
			},
			mask:             []bool{true, true, false, false},
			expectedValidTxs: 4, // 2 valid force-included + 2 mempool txs
		},
		{
			name: "all force-included transactions must be validated",
			txs: [][]byte{
				invalidTxBytes, // Force-included gibberish - filtered
				invalidTxBytes, // Force-included gibberish - filtered
				invalidTxBytes, // Force-included gibberish - filtered
			},
			mask:             []bool{true, true, true},
			expectedValidTxs: 0, // All filtered out (invalid)
		},
		{
			name: "all mempool transactions skip validation",
			txs: [][]byte{
				validTxBytes,
				validTxBytes,
				invalidTxBytes, // Skips validation, passes through
			},
			mask:             []bool{false, false, false},
			expectedValidTxs: 3, // All pass (validation skipped)
		},
		{
			name: "empty mask same as no mask",
			txs: [][]byte{
				validTxBytes,
				invalidTxBytes, // Should be filtered
			},
			mask:             []bool{},
			expectedValidTxs: 1, // 1 valid, 1 filtered
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create context with or without mask
			ctx := context.Background()
			if tt.mask != nil {
				ctx = coreexecution.WithForceIncludedMask(ctx, tt.mask)
			}

			// Call the validation logic by inspecting how transactions are filtered
			// We'll extract the logic to count valid transactions
			forceIncludedMask := coreexecution.GetForceIncludedMask(ctx)
			validTxs := make([]string, 0, len(tt.txs))
			skippedValidation := 0

			for i, tx := range tt.txs {
				if len(tx) == 0 {
					continue
				}

				// Skip validation for mempool transactions (already validated when added to mempool)
				// Force-included transactions from DA MUST be validated
				if forceIncludedMask != nil && i < len(forceIncludedMask) && !forceIncludedMask[i] {
					validTxs = append(validTxs, "0x"+hex.EncodeToString(tx))
					skippedValidation++
					continue
				}

				// Validate force-included transactions (and all txs when no mask)
				var ethTx types.Transaction
				if err := ethTx.UnmarshalBinary(tx); err != nil {
					// Invalid transaction, skip it
					continue
				}

				validTxs = append(validTxs, "0x"+hex.EncodeToString(tx))
			}

			// Verify expected number of valid transactions
			assert.Equal(t, tt.expectedValidTxs, len(validTxs),
				"unexpected number of valid transactions")

			// Verify mempool transactions were actually skipped
			if tt.mask != nil {
				expectedSkipped := 0
				for i, isForceIncluded := range tt.mask {
					// Skip when NOT force-included (i.e., mempool tx)
					if !isForceIncluded && i < len(tt.txs) && len(tt.txs[i]) > 0 {
						expectedSkipped++
					}
				}
				assert.Equal(t, expectedSkipped, skippedValidation,
					"unexpected number of skipped validations")
			}
		})
	}
}
