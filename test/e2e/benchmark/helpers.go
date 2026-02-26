//go:build evm

package benchmark

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sort"
	"time"

	"github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
	"github.com/ethereum/go-ethereum/ethclient"
)

// blockMetrics holds aggregated gas and transaction data across a range of blocks.
type blockMetrics struct {
	StartBlock     uint64
	EndBlock       uint64
	BlockCount     int // non-empty blocks only
	TotalBlockCount int // all blocks in range including empty
	TotalGasUsed   uint64
	TotalTxCount   int
	GasPerBlock    []uint64        // non-empty blocks only
	TxPerBlock     []int           // non-empty blocks only
	BlockIntervals []time.Duration // intervals between all consecutive blocks (for drift)
	FirstBlockTime time.Time
	LastBlockTime  time.Time
}

// steadyStateDuration returns the time between the first and last active blocks.
func (m *blockMetrics) steadyStateDuration() time.Duration {
	if m.FirstBlockTime.IsZero() || m.LastBlockTime.IsZero() {
		return 0
	}
	return m.LastBlockTime.Sub(m.FirstBlockTime)
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

// nonEmptyRatio returns the percentage of blocks that contained transactions.
func (m *blockMetrics) nonEmptyRatio() float64 {
	if m.TotalBlockCount == 0 {
		return 0
	}
	return float64(m.BlockCount) / float64(m.TotalBlockCount) * 100
}

// blockIntervalStats returns p50, p99, and max of block intervals.
func (m *blockMetrics) blockIntervalStats() (p50, p99, max time.Duration) {
	if len(m.BlockIntervals) == 0 {
		return 0, 0, 0
	}
	sorted := make([]float64, len(m.BlockIntervals))
	for i, d := range m.BlockIntervals {
		sorted[i] = float64(d)
	}
	sort.Float64s(sorted)
	return time.Duration(percentile(sorted, 0.50)),
		time.Duration(percentile(sorted, 0.99)),
		time.Duration(sorted[len(sorted)-1])
}

// gasPerBlockStats returns p50 and p99 of gas used per non-empty block.
func (m *blockMetrics) gasPerBlockStats() (p50, p99 float64) {
	if len(m.GasPerBlock) == 0 {
		return 0, 0
	}
	sorted := make([]float64, len(m.GasPerBlock))
	for i, g := range m.GasPerBlock {
		sorted[i] = float64(g)
	}
	sort.Float64s(sorted)
	return percentile(sorted, 0.50), percentile(sorted, 0.99)
}

// txPerBlockStats returns p50 and p99 of tx count per non-empty block.
func (m *blockMetrics) txPerBlockStats() (p50, p99 float64) {
	if len(m.TxPerBlock) == 0 {
		return 0, 0
	}
	sorted := make([]float64, len(m.TxPerBlock))
	for i, c := range m.TxPerBlock {
		sorted[i] = float64(c)
	}
	sort.Float64s(sorted)
	return percentile(sorted, 0.50), percentile(sorted, 0.99)
}

// percentile returns the p-th percentile from a pre-sorted float64 slice
// using linear interpolation.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	rank := p * float64(len(sorted)-1)
	lower := int(math.Floor(rank))
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	frac := rank - float64(lower)
	return sorted[lower] + frac*(sorted[upper]-sorted[lower])
}

// mgasPerSec calculates the achieved throughput in megagas per second.
func mgasPerSec(totalGasUsed uint64, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	return float64(totalGasUsed) / elapsed.Seconds() / 1e6
}

// waitForSpamoorDone polls spamoor metrics until the total sent count reaches
// the target or the context is cancelled. It logs the send rate at each poll
// interval and returns the final sent and failed counts.
func waitForSpamoorDone(ctx context.Context, log func(string, ...any), api *spamoor.API, targetCount int, pollInterval time.Duration) (sent, failed float64, err error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	start := time.Now()
	var prevSent float64

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

			delta := sent - prevSent
			rate := delta / pollInterval.Seconds()
			elapsed := time.Since(start).Round(time.Second)
			log("spamoor progress: %.0f/%.0f sent (%.0f tx/s instant, %.0f tx/s avg, %.0f failed) [%s]",
				sent, float64(targetCount), rate, sent/time.Since(start).Seconds(), failed, elapsed)
			prevSent = sent

			if sent >= float64(targetCount) {
				return sent, failed, nil
			}
		}
	}
}

// deleteAllSpammers removes any pre-existing spammers from the daemon.
// This prevents stale spammers (from previous failed runs) being restored
// from the spamoor SQLite database.
func deleteAllSpammers(api *spamoor.API) error {
	existing, err := api.ListSpammers()
	if err != nil {
		return fmt.Errorf("list spammers: %w", err)
	}
	for _, sp := range existing {
		if err := api.DeleteSpammer(sp.ID); err != nil {
			return fmt.Errorf("delete spammer %d: %w", sp.ID, err)
		}
	}
	return nil
}

// waitForDrain polls the latest block until consecutiveEmpty consecutive empty
// blocks are observed, indicating the mempool has drained.
func waitForDrain(ctx context.Context, log func(string, ...any), client *ethclient.Client, consecutiveEmpty int) {
	var emptyRun int
	var lastBlock uint64
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log("drain timeout after %d consecutive empty blocks (needed %d)", emptyRun, consecutiveEmpty)
			return
		case <-ticker.C:
			header, err := client.HeaderByNumber(ctx, nil)
			if err != nil {
				continue
			}
			num := header.Number.Uint64()
			if num == lastBlock {
				continue
			}

			txCount, err := client.TransactionCount(ctx, header.Hash())
			if err != nil {
				continue
			}

			lastBlock = num
			if txCount == 0 {
				emptyRun++
			} else {
				emptyRun = 0
			}

			if emptyRun >= consecutiveEmpty {
				log("mempool drained: %d consecutive empty blocks at block %d", emptyRun, num)
				return
			}
		}
	}
}

// collectBlockMetrics iterates all headers in [startBlock, endBlock] to collect
// gas and transaction metrics. Empty blocks are skipped for gas/tx aggregation
// but included in block interval tracking.
func collectBlockMetrics(ctx context.Context, client *ethclient.Client, startBlock, endBlock uint64) (*blockMetrics, error) {
	if endBlock < startBlock {
		return nil, fmt.Errorf("endBlock %d < startBlock %d", endBlock, startBlock)
	}

	m := &blockMetrics{StartBlock: startBlock, EndBlock: endBlock}

	var prevBlockTime time.Time
	for n := startBlock; n <= endBlock; n++ {
		header, err := client.HeaderByNumber(ctx, new(big.Int).SetUint64(n))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch header %d: %w", n, err)
		}

		blockTime := time.Unix(int64(header.Time), 0)
		m.TotalBlockCount++

		// track intervals between all consecutive blocks
		if !prevBlockTime.IsZero() {
			m.BlockIntervals = append(m.BlockIntervals, blockTime.Sub(prevBlockTime))
		}
		prevBlockTime = blockTime

		txCount, err := client.TransactionCount(ctx, header.Hash())
		if err != nil {
			return nil, fmt.Errorf("failed to fetch tx count for block %d: %w", n, err)
		}

		if txCount == 0 {
			continue
		}

		// non-empty block: aggregate gas and tx metrics
		if m.BlockCount == 0 {
			m.FirstBlockTime = blockTime
		}
		m.LastBlockTime = blockTime

		m.BlockCount++
		m.TotalGasUsed += header.GasUsed
		m.TotalTxCount += int(txCount)
		m.GasPerBlock = append(m.GasPerBlock, header.GasUsed)
		m.TxPerBlock = append(m.TxPerBlock, int(txCount))
	}

	return m, nil
}
