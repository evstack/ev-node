//go:build evm

package benchmark

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
	"github.com/ethereum/go-ethereum/ethclient"
)

// blockMetrics holds aggregated gas and transaction data across a range of blocks.
type blockMetrics struct {
	StartBlock   uint64
	EndBlock     uint64
	BlockCount   int
	TotalGasUsed uint64
	TotalTxCount int
	GasPerBlock  []uint64
	TxPerBlock   []int
}

// avgGasPerBlock returns the mean gas used per block.
func (m *blockMetrics) avgGasPerBlock() float64 {
	if m.BlockCount == 0 {
		return 0
	}
	return float64(m.TotalGasUsed) / float64(m.BlockCount)
}

// avgTxPerBlock returns the mean transaction count per block.
func (m *blockMetrics) avgTxPerBlock() float64 {
	if m.BlockCount == 0 {
		return 0
	}
	return float64(m.TotalTxCount) / float64(m.BlockCount)
}

// mgasPerSec calculates the achieved throughput in megagas per second.
func mgasPerSec(totalGasUsed uint64, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	return float64(totalGasUsed) / elapsed.Seconds() / 1e6
}

// waitForSpamoorDone polls spamoor metrics until the total sent count reaches
// the target or the context is cancelled. It returns the final sent and failed
// counts.
func waitForSpamoorDone(ctx context.Context, api *spamoor.API, targetCount int, pollInterval time.Duration) (sent, failed float64, err error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return sent, failed, fmt.Errorf("timed out waiting for spamoor to send %d txs (sent %.0f): %w", targetCount, sent, ctx.Err())
		case <-ticker.C:
			metrics, mErr := api.GetMetrics()
			if mErr != nil {
				continue
			}
			sent = sumCounter(metrics["spamoor_transactions_sent_total"])
			failed = sumCounter(metrics["spamoor_transactions_failed_total"])
			if sent >= float64(targetCount) {
				return sent, failed, nil
			}
		}
	}
}

// collectBlockMetrics fetches block headers from startBlock to endBlock
// (inclusive) and aggregates gas and transaction counts. Blocks with zero
// transactions are skipped to avoid skewing averages with empty blocks.
func collectBlockMetrics(ctx context.Context, client *ethclient.Client, startBlock, endBlock uint64) (*blockMetrics, error) {
	if endBlock < startBlock {
		return nil, fmt.Errorf("endBlock %d < startBlock %d", endBlock, startBlock)
	}

	m := &blockMetrics{
		StartBlock: startBlock,
		EndBlock:   endBlock,
	}

	for n := startBlock; n <= endBlock; n++ {
		block, err := client.BlockByNumber(ctx, new(big.Int).SetUint64(n))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch block %d: %w", n, err)
		}

		txCount := len(block.Transactions())
		if txCount == 0 {
			continue
		}

		gasUsed := block.GasUsed()
		m.BlockCount++
		m.TotalGasUsed += gasUsed
		m.TotalTxCount += txCount
		m.GasPerBlock = append(m.GasPerBlock, gasUsed)
		m.TxPerBlock = append(m.TxPerBlock, txCount)
	}

	return m, nil
}
