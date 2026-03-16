//go:build evm

package benchmark

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sort"
	"testing"
	"time"

	"github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
	"github.com/ethereum/go-ethereum/ethclient"
	e2e "github.com/evstack/ev-node/test/e2e"
)

// blockMetrics holds aggregated gas and transaction data across a range of blocks.
// Only blocks containing at least one transaction are counted in BlockCount,
// GasPerBlock, and TxPerBlock. All blocks (including empty) are counted in
// TotalBlockCount and contribute to BlockIntervals.
type blockMetrics struct {
	StartBlock      uint64
	EndBlock        uint64
	BlockCount      int             // non-empty blocks only
	TotalBlockCount int             // all blocks in range including empty
	TotalGasUsed    uint64          // cumulative gas across non-empty blocks
	TotalTxCount    int             // cumulative tx count across non-empty blocks
	GasPerBlock     []uint64        // per-block gas for non-empty blocks only
	TxPerBlock      []int           // per-block tx count for non-empty blocks only
	BlockIntervals  []time.Duration // time between all consecutive blocks (including empty)
	FirstBlockTime  time.Time       // timestamp of the first non-empty block
	LastBlockTime   time.Time       // timestamp of the last non-empty block
}

// steadyStateDuration returns the wall-clock time between the first and last
// non-empty blocks. This excludes warm-up (empty blocks before the first tx)
// and cool-down (empty blocks after the last tx) from throughput calculations.
func (m *blockMetrics) steadyStateDuration() time.Duration {
	if m.FirstBlockTime.IsZero() || m.LastBlockTime.IsZero() {
		return 0
	}
	return m.LastBlockTime.Sub(m.FirstBlockTime)
}

// avgGasPerBlock returns TotalGasUsed / BlockCount, i.e. the mean gas used
// per non-empty block.
func (m *blockMetrics) avgGasPerBlock() float64 {
	if m.BlockCount == 0 {
		return 0
	}
	return float64(m.TotalGasUsed) / float64(m.BlockCount)
}

// avgTxPerBlock returns TotalTxCount / BlockCount, i.e. the mean transaction
// count per non-empty block.
func (m *blockMetrics) avgTxPerBlock() float64 {
	if m.BlockCount == 0 {
		return 0
	}
	return float64(m.TotalTxCount) / float64(m.BlockCount)
}

// nonEmptyRatio returns (BlockCount / TotalBlockCount) * 100, i.e. the
// percentage of blocks in the range that contained at least one transaction.
func (m *blockMetrics) nonEmptyRatio() float64 {
	if m.TotalBlockCount == 0 {
		return 0
	}
	return float64(m.BlockCount) / float64(m.TotalBlockCount) * 100
}

// blockIntervalStats computes percentile statistics over the time gaps between
// all consecutive blocks (including empty ones). This measures block production
// cadence and jitter rather than transaction throughput.
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

// gasPerBlockStats returns the 50th and 99th percentile of gas used across
// non-empty blocks. Shows the distribution of per-block gas consumption.
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

// txPerBlockStats returns the 50th and 99th percentile of transaction counts
// across non-empty blocks. Shows the distribution of per-block tx throughput.
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

// percentile returns the p-th percentile from a pre-sorted float64 slice using
// linear interpolation between adjacent ranks. For example, p=0.50 returns the
// median and p=0.99 returns the value below which 99% of observations fall.
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

// mgasPerSec computes totalGasUsed / elapsed / 1e6, giving throughput in
// millions of gas units per second (MGas/s).
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
				log("failed to get spamoor metrics: %v", mErr)
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
func waitForDrain(ctx context.Context, log func(string, ...any), client *ethclient.Client, consecutiveEmpty int) error {
	var emptyRun int
	var lastBlock uint64
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("drain timeout after %d consecutive empty blocks (needed %d): %w", emptyRun, consecutiveEmpty, ctx.Err())
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
				return nil
			}
		}
	}
}

// blockMetricsSummary holds all derived statistics from a blockMetrics measurement
// window. Every field is computed from the raw block data by summarize().
type blockMetricsSummary struct {
	// SteadyState is the wall-clock duration between the first and last non-empty
	// blocks, used as the denominator for throughput calculations.
	SteadyState time.Duration
	// AchievedMGas is total gas / steady-state seconds / 1e6 (megagas per second).
	AchievedMGas float64
	// AchievedTPS is total tx count / steady-state seconds.
	AchievedTPS float64
	// IntervalP50, IntervalP99, IntervalMax are percentile and max statistics
	// over the time between all consecutive blocks (including empty).
	IntervalP50 time.Duration
	IntervalP99 time.Duration
	IntervalMax time.Duration
	// GasP50, GasP99 are the 50th/99th percentile of gas used per non-empty block.
	GasP50 float64
	GasP99 float64
	// TxP50, TxP99 are the 50th/99th percentile of tx count per non-empty block.
	TxP50 float64
	TxP99 float64
	// AvgGas is the mean gas per non-empty block (TotalGasUsed / BlockCount).
	AvgGas float64
	// AvgTx is the mean tx count per non-empty block (TotalTxCount / BlockCount).
	AvgTx float64
	// BlocksPerSec is non-empty blocks / steady-state seconds.
	BlocksPerSec float64
	// AvgBlockInterval is the mean time between all consecutive blocks.
	AvgBlockInterval time.Duration
	// NonEmptyRatio is (non-empty blocks / total blocks) * 100.
	NonEmptyRatio float64
}

// summarize computes all derived statistics from the raw block-level data in one
// pass. It delegates to the individual stat methods (steadyStateDuration,
// blockIntervalStats, gasPerBlockStats, etc.) and packages the results into a
// blockMetricsSummary for logging and result-writing.
func (m *blockMetrics) summarize() *blockMetricsSummary {
	ss := m.steadyStateDuration()
	intervalP50, intervalP99, intervalMax := m.blockIntervalStats()
	gasP50, gasP99 := m.gasPerBlockStats()
	txP50, txP99 := m.txPerBlockStats()

	var blocksPerSec, achievedTPS float64
	if ss > 0 {
		blocksPerSec = float64(m.BlockCount) / ss.Seconds()
		achievedTPS = float64(m.TotalTxCount) / ss.Seconds()
	}

	var avgBlockInterval time.Duration
	if len(m.BlockIntervals) > 0 {
		var total time.Duration
		for _, d := range m.BlockIntervals {
			total += d
		}
		avgBlockInterval = total / time.Duration(len(m.BlockIntervals))
	}

	return &blockMetricsSummary{
		SteadyState:   ss,
		AchievedMGas:  mgasPerSec(m.TotalGasUsed, ss),
		AchievedTPS:   achievedTPS,
		IntervalP50:   intervalP50,
		IntervalP99:   intervalP99,
		IntervalMax:   intervalMax,
		GasP50:        gasP50,
		GasP99:        gasP99,
		TxP50:         txP50,
		TxP99:         txP99,
		AvgGas:        m.avgGasPerBlock(),
		AvgTx:         m.avgTxPerBlock(),
		BlocksPerSec:     blocksPerSec,
		AvgBlockInterval: avgBlockInterval,
		NonEmptyRatio:    m.nonEmptyRatio(),
	}
}

// log prints the block range, interval stats, per-block gas/tx stats, and
// throughput (MGas/s + TPS) to the test log. startBlock/endBlock and
// totalBlocks/nonEmptyBlocks are passed separately because they live on the
// raw blockMetrics, not the summary.
func (s *blockMetricsSummary) log(t testing.TB, startBlock, endBlock uint64, totalBlocks, nonEmptyBlocks int, wallClock time.Duration) {
	t.Logf("block range: %d-%d (%d total, %d non-empty, %.1f%% non-empty)",
		startBlock, endBlock, totalBlocks, nonEmptyBlocks, s.NonEmptyRatio)
	t.Logf("block intervals: avg=%s, p50=%s, p99=%s, max=%s",
		s.AvgBlockInterval.Round(time.Millisecond), s.IntervalP50.Round(time.Millisecond), s.IntervalP99.Round(time.Millisecond), s.IntervalMax.Round(time.Millisecond))
	t.Logf("gas/block (non-empty): avg=%.0f, p50=%.0f, p99=%.0f", s.AvgGas, s.GasP50, s.GasP99)
	t.Logf("tx/block (non-empty): avg=%.1f, p50=%.0f, p99=%.0f", s.AvgTx, s.TxP50, s.TxP99)
	t.Logf("throughput: %.2f MGas/s, %.1f TPS over %s steady-state (%s wall clock)",
		s.AchievedMGas, s.AchievedTPS, s.SteadyState.Round(time.Millisecond), wallClock.Round(time.Millisecond))
}

// entries returns all summary metrics as result writer entries in the
// customSmallerIsBetter format expected by github-action-benchmark. Each entry
// is prefixed with the given label (e.g. "ERC20Throughput") so results from
// different tests are distinguishable in the same output file.
func (s *blockMetricsSummary) entries(prefix string) []entry {
	return []entry{
		{Name: prefix + " - MGas/s", Unit: "MGas/s", Value: s.AchievedMGas},
		{Name: prefix + " - TPS", Unit: "tx/s", Value: s.AchievedTPS},
		{Name: prefix + " - avg gas/block", Unit: "gas", Value: s.AvgGas},
		{Name: prefix + " - avg tx/block", Unit: "count", Value: s.AvgTx},
		{Name: prefix + " - blocks/s", Unit: "blocks/s", Value: s.BlocksPerSec},
		{Name: prefix + " - non-empty block ratio", Unit: "%", Value: s.NonEmptyRatio},
		{Name: prefix + " - avg block interval", Unit: "ms", Value: float64(s.AvgBlockInterval.Milliseconds())},
		{Name: prefix + " - block interval p50", Unit: "ms", Value: float64(s.IntervalP50.Milliseconds())},
		{Name: prefix + " - block interval p99", Unit: "ms", Value: float64(s.IntervalP99.Milliseconds())},
		{Name: prefix + " - gas/block p50", Unit: "gas", Value: s.GasP50},
		{Name: prefix + " - gas/block p99", Unit: "gas", Value: s.GasP99},
		{Name: prefix + " - tx/block p50", Unit: "count", Value: s.TxP50},
		{Name: prefix + " - tx/block p99", Unit: "count", Value: s.TxP99},
	}
}

// evNodeOverhead computes the fraction of block production time spent outside
// EVM execution. It looks up the average durations of BlockExecutor.ProduceBlock
// (the outer span covering the full block lifecycle) and Executor.ExecuteTxs
// (the inner span covering only EVM tx execution), then returns:
//
//	overhead% = (avgProduce - avgExecute) / avgProduce * 100
//
// This captures time spent on sequencing, DA submission, header construction,
// and other ev-node orchestration work. Returns false if either span is missing.
func evNodeOverhead(spans []e2e.TraceSpan) (float64, bool) {
	stats := e2e.AggregateSpanStats(spans)
	produce, ok := stats["BlockExecutor.ProduceBlock"]
	if !ok {
		return 0, false
	}
	execute, ok := stats["Executor.ExecuteTxs"]
	if !ok {
		return 0, false
	}
	produceAvg := float64(produce.Total.Microseconds()) / float64(produce.Count)
	executeAvg := float64(execute.Total.Microseconds()) / float64(execute.Count)
	if produceAvg <= 0 {
		return 0, false
	}
	return (produceAvg - executeAvg) / produceAvg * 100, true
}

// waitForMetricTarget polls a metric getter function every 2s until the
// returned value >= target, or fails the test on timeout.
func waitForMetricTarget(t testing.TB, name string, poll func() (float64, error), target float64, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		v, err := poll()
		if err == nil && v >= target {
			t.Logf("metric %s reached %.0f (target %.0f)", name, v, target)
			return
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("metric %s did not reach target %.0f within %v", name, target, timeout)
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
