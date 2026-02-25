//go:build evm

package benchmark

import (
	"context"
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
		totalCount  = 3000
		serviceName = "ev-node-erc20"
		waitTimeout = 3 * time.Minute
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
		"throughput":      100,
		"total_count":     totalCount,
		"max_pending":     4000,
		"max_wallets":     300,
		"base_fee":        20,
		"tip_fee":         3,
		"refill_amount":   "5000000000000000000",
		"refill_balance":  "2000000000000000000",
		"refill_interval": 600,
	}

	// record the starting block before generating load
	startHeader, err := e.ethClient.HeaderByNumber(ctx, nil)
	s.Require().NoError(err, "failed to get start block header")
	startBlock := startHeader.Number.Uint64()
	loadStart := time.Now()

	id, err := e.spamoorAPI.CreateSpammer("bench-erc20", spamoor.ScenarioERC20TX, erc20Config, true)
	s.Require().NoError(err, "failed to create erc20 spammer")
	t.Cleanup(func() { _ = e.spamoorAPI.DeleteSpammer(id) })

	// wait for spamoor to finish sending all transactions
	waitCtx, cancel := context.WithTimeout(ctx, waitTimeout)
	defer cancel()
	sent, failed, err := waitForSpamoorDone(waitCtx, e.spamoorAPI, totalCount, 2*time.Second)
	s.Require().NoError(err, "spamoor did not finish in time")

	// allow a short settle period for remaining txs to be included in blocks
	time.Sleep(5 * time.Second)
	wallClock := time.Since(loadStart)

	endHeader, err := e.ethClient.HeaderByNumber(ctx, nil)
	s.Require().NoError(err, "failed to get end block header")
	endBlock := endHeader.Number.Uint64()

	// collect block-level gas/tx metrics
	bm, err := collectBlockMetrics(ctx, e.ethClient, startBlock, endBlock)
	s.Require().NoError(err, "failed to collect block metrics")

	achievedMGas := mgasPerSec(bm.TotalGasUsed, wallClock)
	achievedTPS := float64(bm.TotalTxCount) / wallClock.Seconds()

	t.Logf("ERC20 throughput: %.2f MGas/s, %.1f TPS over %s (%d blocks, %d txs, %.2f avg gas/block)",
		achievedMGas, achievedTPS, wallClock.Round(time.Millisecond),
		bm.BlockCount, bm.TotalTxCount, bm.avgGasPerBlock())

	// collect and report traces
	evNodeSpans := s.collectServiceTraces(e, serviceName)
	evRethSpans := s.collectServiceTraces(e, "ev-reth")
	e2e.PrintTraceReport(t, serviceName, evNodeSpans)
	e2e.PrintTraceReport(t, "ev-reth", evRethSpans)

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
		Value: float64(bm.BlockCount) / wallClock.Seconds(),
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
