package grpc

import (
	"bytes"
	"testing"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

func mustEncodeTxBatch(t *testing.T, txs [][]byte) *pb.TxBatch {
	t.Helper()

	batch, err := encodeTxBatch(txs)
	if err != nil {
		t.Fatalf("encode tx batch: %v", err)
	}
	return batch
}

func TestEncodeDecodeTxBatch(t *testing.T) {
	txs := [][]byte{[]byte("tx1"), nil, []byte("tx3"), []byte{}}

	batch := mustEncodeTxBatch(t, txs)
	decoded, err := decodeTxBatch(batch)
	if err != nil {
		t.Fatalf("decode tx batch: %v", err)
	}
	if len(decoded) != len(txs) {
		t.Fatalf("expected %d txs, got %d", len(txs), len(decoded))
	}
	for i := range txs {
		if !bytes.Equal(decoded[i], txs[i]) {
			t.Fatalf("tx %d: expected %q, got %q", i, txs[i], decoded[i])
		}
	}

	decoded[0] = append(decoded[0], 'x')
	if !bytes.Equal(decoded[2], txs[2]) {
		t.Fatalf("decoded tx slices should not have capacity overlap")
	}
}

func TestDecodeTxBatchRejectsMalformedInput(t *testing.T) {
	tests := []struct {
		name  string
		batch *pb.TxBatch
	}{
		{
			name:  "data without sizes",
			batch: &pb.TxBatch{Data: []byte("tx")},
		},
		{
			name:  "sizes exceed data",
			batch: &pb.TxBatch{Data: []byte("tx"), TxSizes: []uint32{3}},
		},
		{
			name:  "sizes do not consume data",
			batch: &pb.TxBatch{Data: []byte("tx"), TxSizes: []uint32{1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := decodeTxBatch(tt.batch); err == nil {
				t.Fatalf("expected decode error")
			}
		})
	}
}

func TestDecodeTxBatchOrTxsFallsBackToLegacyTxs(t *testing.T) {
	legacyTxs := [][]byte{[]byte("legacy1"), []byte("legacy2")}

	txs, err := decodeTxBatchOrTxs(nil, legacyTxs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txs) != len(legacyTxs) {
		t.Fatalf("expected %d txs, got %d", len(legacyTxs), len(txs))
	}
	for i := range txs {
		if !bytes.Equal(txs[i], legacyTxs[i]) {
			t.Fatalf("tx %d: expected %q, got %q", i, legacyTxs[i], txs[i])
		}
	}
}

func TestDecodeTxBatchOrTxsPrefersTxBatch(t *testing.T) {
	batchTxs := [][]byte{[]byte("batch")}
	legacyTxs := [][]byte{[]byte("legacy")}

	txs, err := decodeTxBatchOrTxs(mustEncodeTxBatch(t, batchTxs), legacyTxs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txs) != len(batchTxs) {
		t.Fatalf("expected %d txs, got %d", len(batchTxs), len(txs))
	}
	if !bytes.Equal(txs[0], batchTxs[0]) {
		t.Fatalf("expected tx_batch %q, got %q", batchTxs[0], txs[0])
	}
}
