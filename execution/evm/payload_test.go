package evm

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestPayloadTransactionsMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    [][]byte
		expected string
	}{
		{
			name:     "empty",
			input:    [][]byte{},
			expected: "[]",
		},
		{
			name:     "nil",
			input:    nil,
			expected: "[]",
		},
		{
			name: "single tx",
			input: [][]byte{
				{0x01, 0x02, 0x03},
			},
			expected: `["0x010203"]`,
		},
		{
			name: "multiple txs",
			input: [][]byte{
				{0xaa, 0xbb},
				{0xcc, 0xdd},
			},
			expected: `["0xaabb","0xccdd"]`,
		},
		{
			name: "skip empty tx",
			input: [][]byte{
				{0x01},
				{},
				{0x02},
			},
			expected: `["0x01","0x02"]`,
		},
		{
			name: "all empty",
			input: [][]byte{
				{},
				{},
			},
			expected: "[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pt := PayloadTransactions(tt.input)
			b, err := pt.MarshalJSON()
			require.NoError(t, err)
			require.Equal(t, tt.expected, string(b))
		})
	}
}

func BenchmarkPayloadTransactionsMarshalJSON(b *testing.B) {
	// Setup large payload
	txs := make([][]byte, 1000)
	for i := range txs {
		txs[i] = make([]byte, 1024) // 1KB tx
	}
	pt := PayloadTransactions(txs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pt.MarshalJSON()
	}
}

func BenchmarkStandardJSONMarshal(b *testing.B) {
	// Setup large payload
	txs := make([][]byte, 1000)
	for i := range txs {
		txs[i] = make([]byte, 1024) // 1KB tx
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate old behavior: allocate strings + json marshal
		validTxs := make([]string, 0, len(txs))
		for _, tx := range txs {
			validTxs = append(validTxs, hexutil.Encode(tx))
		}
		_, _ = json.Marshal(validTxs)
	}
}
