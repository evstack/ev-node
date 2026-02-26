//go:build evm

package benchmark

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
	e2e "github.com/evstack/ev-node/test/e2e"
)

// TestERC20Throughput measures ERC-20 token transfer throughput in isolation.
// Each tx is ~65K gas (contract call with storage reads/writes)
//
// Primary metric: achieved MGas/s.
// Diagnostic metrics: per-span latency breakdown, gas/block, tx/block.
func (s *SpamoorSuite) TestERC20Throughput() {
	const (
		numSpammers     = 5
		countPerSpammer = 50000
		totalCount      = numSpammers * countPerSpammer
		serviceName     = "ev-node-erc20"
		waitTimeout     = 5 * time.Minute
	)

	t := s.T()
	ctx := t.Context()
	w := newResultWriter(t, "ERC20Throughput")
	defer w.flush()

	e := s.setupEnv(config{
		rethTag:     "pr-140",
		serviceName: serviceName,
	})

	erc20Config := map[string]any{
		"throughput":      50, // 50 tx per 100ms slot = 500 tx/s per spammer
		"total_count":     countPerSpammer,
		"max_pending":     50000,
		"max_wallets":     200,
		"base_fee":        20,
		"tip_fee":         3,
		"refill_amount":   "5000000000000000000",
		"refill_balance":  "2000000000000000000",
		"refill_interval": 600,
	}

	// clear any stale spammers restored from the persistent spamoor database
	s.Require().NoError(deleteAllSpammers(e.spamoorAPI), "failed to delete stale spammers")

	// launch all spammers before recording startBlock so warm-up
	// (contract deploy + wallet funding) is excluded from the measurement window.
	for i := range numSpammers {
		name := fmt.Sprintf("bench-erc20-%d", i)
		id, err := e.spamoorAPI.CreateSpammer(name, spamoor.ScenarioERC20TX, erc20Config, true)
		s.Require().NoError(err, "failed to create spammer %s", name)
		t.Cleanup(func() { _ = e.spamoorAPI.DeleteSpammer(id) })
	}

	// record the starting block after all spammers are launched so the
	// measurement window excludes the warm-up period.
	startHeader, err := e.ethClient.HeaderByNumber(ctx, nil)
	s.Require().NoError(err, "failed to get start block header")
	startBlock := startHeader.Number.Uint64()
	loadStart := time.Now()

	// wait for spamoor to finish sending all transactions
	waitCtx, cancel := context.WithTimeout(ctx, waitTimeout)
	defer cancel()
	sent, failed, err := waitForSpamoorDone(waitCtx, t.Logf, e.spamoorAPI, totalCount, 2*time.Second)
	s.Require().NoError(err, "spamoor did not finish in time")

	// wait for pending txs to drain: once we see several consecutive empty
	// blocks, the mempool is drained and we can stop.
	drainCtx, drainCancel := context.WithTimeout(ctx, 30*time.Second)
	defer drainCancel()
	waitForDrain(drainCtx, t.Logf, e.ethClient, 10)
	wallClock := time.Since(loadStart)

	endHeader, err := e.ethClient.HeaderByNumber(ctx, nil)
	s.Require().NoError(err, "failed to get end block header")
	endBlock := endHeader.Number.Uint64()

	// collect block-level gas/tx metrics
	bm, err := collectBlockMetrics(ctx, e.ethClient, startBlock, endBlock)
	s.Require().NoError(err, "failed to collect block metrics")

	// use steady-state window (first active block to last active block) for
	// throughput calculation, excluding warm-up and cool-down periods
	steadyState := bm.steadyStateDuration()
	s.Require().Greater(steadyState, time.Duration(0), "expected non-zero steady-state duration")

	achievedMGas := mgasPerSec(bm.TotalGasUsed, steadyState)
	achievedTPS := float64(bm.TotalTxCount) / steadyState.Seconds()

	intervalP50, intervalP99, intervalMax := bm.blockIntervalStats()
	gasP50, gasP99 := bm.gasPerBlockStats()
	txP50, txP99 := bm.txPerBlockStats()

	t.Logf("block range: %d-%d (%d total, %d non-empty, %.1f%% non-empty)",
		startBlock, endBlock, bm.TotalBlockCount, bm.BlockCount, bm.nonEmptyRatio())
	t.Logf("block intervals: p50=%s, p99=%s, max=%s",
		intervalP50.Round(time.Millisecond), intervalP99.Round(time.Millisecond), intervalMax.Round(time.Millisecond))
	t.Logf("gas/block (non-empty): avg=%.0f, p50=%.0f, p99=%.0f", bm.avgGasPerBlock(), gasP50, gasP99)
	t.Logf("tx/block (non-empty): avg=%.1f, p50=%.0f, p99=%.0f", bm.avgTxPerBlock(), txP50, txP99)
	t.Logf("ERC20 throughput: %.2f MGas/s, %.1f TPS over %s steady-state (%s wall clock)",
		achievedMGas, achievedTPS, steadyState.Round(time.Millisecond), wallClock.Round(time.Millisecond))

	// collect and report traces
	evNodeSpans := s.collectServiceTraces(e, serviceName)
	evRethSpans := s.tryCollectServiceTraces(e, "ev-reth")
	e2e.PrintTraceReport(t, serviceName, evNodeSpans)
	if len(evRethSpans) > 0 {
		e2e.PrintTraceReport(t, "ev-reth", evRethSpans)
	}

	// compute ev-node overhead ratio
	spanStats := e2e.AggregateSpanStats(evNodeSpans)
	if produceBlock, ok := spanStats["BlockExecutor.ProduceBlock"]; ok {
		if executeTxs, ok2 := spanStats["Executor.ExecuteTxs"]; ok2 {
			produceAvg := float64(produceBlock.Total.Microseconds()) / float64(produceBlock.Count)
			executeAvg := float64(executeTxs.Total.Microseconds()) / float64(executeTxs.Count)
			if produceAvg > 0 {
				overhead := (produceAvg - executeAvg) / produceAvg * 100
				t.Logf("ev-node overhead: %.1f%%", overhead)
				w.addEntry(entry{
					Name:  "ERC20Throughput - ev-node overhead",
					Unit:  "%",
					Value: overhead,
				})
			}
		}
	}

	// write all metrics to benchmark output
	w.addEntry(entry{
		Name:  "ERC20Throughput - MGas/s",
		Unit:  "MGas/s",
		Value: achievedMGas,
	})
	w.addEntry(entry{
		Name:  "ERC20Throughput - TPS",
		Unit:  "tx/s",
		Value: achievedTPS,
	})
	w.addEntry(entry{
		Name:  "ERC20Throughput - avg gas/block",
		Unit:  "gas",
		Value: bm.avgGasPerBlock(),
	})
	w.addEntry(entry{
		Name:  "ERC20Throughput - avg tx/block",
		Unit:  "count",
		Value: bm.avgTxPerBlock(),
	})
	w.addEntry(entry{
		Name:  "ERC20Throughput - blocks/s",
		Unit:  "blocks/s",
		Value: float64(bm.BlockCount) / steadyState.Seconds(),
	})
	w.addEntry(entry{
		Name:  "ERC20Throughput - non-empty block ratio",
		Unit:  "%",
		Value: bm.nonEmptyRatio(),
	})
	w.addEntry(entry{
		Name:  "ERC20Throughput - block interval p50",
		Unit:  "ms",
		Value: float64(intervalP50.Milliseconds()),
	})
	w.addEntry(entry{
		Name:  "ERC20Throughput - block interval p99",
		Unit:  "ms",
		Value: float64(intervalP99.Milliseconds()),
	})
	w.addEntry(entry{
		Name:  "ERC20Throughput - gas/block p50",
		Unit:  "gas",
		Value: gasP50,
	})
	w.addEntry(entry{
		Name:  "ERC20Throughput - gas/block p99",
		Unit:  "gas",
		Value: gasP99,
	})
	w.addEntry(entry{
		Name:  "ERC20Throughput - tx/block p50",
		Unit:  "count",
		Value: txP50,
	})
	w.addEntry(entry{
		Name:  "ERC20Throughput - tx/block p99",
		Unit:  "count",
		Value: txP99,
	})

	// add per-span avg latencies
	w.addSpans(append(evNodeSpans, evRethSpans...))

	// assertions
	s.Require().Greater(sent, float64(0), "at least one transaction should have been sent")
	s.Require().Zero(failed, "no transactions should have failed")

	// assert expected ev-node spans are present
	assertSpanNames(t, evNodeSpans, []string{
		"BlockExecutor.ProduceBlock",
		"BlockExecutor.ApplyBlock",
		"Executor.ExecuteTxs",
		"Executor.SetFinal",
		"Engine.ForkchoiceUpdated",
		"Engine.NewPayload",
		"Engine.GetPayload",
		"Sequencer.GetNextBatch",
		"DA.Submit",
	}, serviceName)
}
