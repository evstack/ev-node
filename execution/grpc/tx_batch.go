package grpc

import (
	"fmt"

	pb "github.com/evstack/ev-node/execution/grpc/types/pb/evnode/v1"
)

// maxTxBatchTxSize is the largest transaction length representable in TxBatch.TxSizes:
// 4 GiB - 1 byte, or 4,294,967,295 bytes.
const maxTxBatchTxSize = uint64(1<<32 - 1)

func encodeTxBatch(txs [][]byte) (*pb.TxBatch, error) {
	if len(txs) == 0 {
		return &pb.TxBatch{}, nil
	}

	maxInt := uint64(int(^uint(0) >> 1))
	var total uint64
	txSizes := make([]uint32, len(txs))
	for i, tx := range txs {
		txLen := uint64(len(tx))
		if txLen > maxTxBatchTxSize {
			return nil, fmt.Errorf("tx %d size %d exceeds uint32", i, txLen)
		}
		total += txLen
		if total > maxInt {
			return nil, fmt.Errorf("tx batch size %d exceeds int", total)
		}
		txSizes[i] = uint32(txLen)
	}

	data := make([]byte, 0, int(total))
	for _, tx := range txs {
		data = append(data, tx...)
	}

	return &pb.TxBatch{
		Data:    data,
		TxSizes: txSizes,
	}, nil
}

func decodeTxBatch(batch *pb.TxBatch) ([][]byte, error) {
	if batch == nil {
		return nil, nil
	}
	if len(batch.TxSizes) == 0 {
		if len(batch.Data) != 0 {
			return nil, fmt.Errorf("tx batch has %d data bytes but no tx sizes", len(batch.Data))
		}
		return nil, nil
	}

	var total uint64
	for i, txSize := range batch.TxSizes {
		total += uint64(txSize)
		if total > uint64(len(batch.Data)) {
			return nil, fmt.Errorf("tx sizes exceed data length at index %d", i)
		}
	}
	if total != uint64(len(batch.Data)) {
		return nil, fmt.Errorf("tx sizes total %d does not match data length %d", total, len(batch.Data))
	}

	txs := make([][]byte, len(batch.TxSizes))
	offset := 0
	for i, txSize := range batch.TxSizes {
		end := offset + int(txSize)
		txs[i] = batch.Data[offset:end:end]
		offset = end
	}
	return txs, nil
}
