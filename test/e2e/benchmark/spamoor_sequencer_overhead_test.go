//go:build evm

package benchmark

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
)

// TestSequencerOverhead measures ev-node coordination cost using simple EOA
// transfers (21K gas each). EVM execution is minimal so the bottleneck shifts
// to reaper polling, sequencer batching, and Engine API round-trips.
//
// Primary metrics: TPS, ev-node overhead %.
// Diagnostic metrics: Sequencer.GetNextBatch, Executor.FilterTxs, Executor.GetTxs latency.
func (s *SpamoorSuite) TestSequencerOverhead() {
	const (
		numSpammers     = 5
		countPerSpammer = 100000
		totalCount      = numSpammers * countPerSpammer
		waitTimeout     = 10 * time.Minute
	)

	cfg := newBenchConfig("ev-node-sequencer-overhead")

	t := s.T()
	ctx := t.Context()
	w := newResultWriter(t, "SequencerOverhead")
	defer w.flush()

	e := s.setupEnv(cfg)

	eoaConfig := map[string]any{
		"throughput":      100, // 100 tx per 100ms slot = 1000 tx/s per spammer, 5000 tx/s total
		"total_count":     countPerSpammer,
		"max_pending":     50000,
		"max_wallets":     500, // many wallets to avoid nonce contention at 5000 tx/s
		"amount":          100,
		"random_amount":   true,
		"random_target":   true,
		"base_fee":        20,
		"tip_fee":         2,
		"refill_amount":   "100000000000000000",  // 0.1 ETH (transfers are cheap)
		"refill_balance":  "50000000000000000",   // 0.05 ETH
		"refill_interval": 600,
	}

	s.Require().NoError(deleteAllSpammers(e.spamoorAPI), "failed to delete stale spammers")

	var spammerIDs []int
	for i := range numSpammers {
		name := fmt.Sprintf("bench-eoa-%d", i)
		id, err := e.spamoorAPI.CreateSpammer(name, spamoor.ScenarioEOATX, eoaConfig, true)
		s.Require().NoError(err, "failed to create spammer %s", name)
		spammerIDs = append(spammerIDs, id)
		t.Cleanup(func() { _ = e.spamoorAPI.DeleteSpammer(id) })
	}

	time.Sleep(3 * time.Second)
	assertSpammersRunning(t, e.spamoorAPI, spammerIDs)

	// wait for wallet funding to finish before recording start block
	pollSentTotal := func() (float64, error) {
		metrics, mErr := e.spamoorAPI.GetMetrics()
		if mErr != nil {
			return 0, mErr
		}
		return sumCounter(metrics["spamoor_transactions_sent_total"]), nil
	}
	waitForMetricTarget(t, "spamoor_transactions_sent_total (warmup)", pollSentTotal, float64(cfg.WarmupTxs), cfg.WaitTimeout)

	// reset trace window to exclude warmup spans
	e.traces.resetStartTime()

	startHeader, err := e.ethClient.HeaderByNumber(ctx, nil)
	s.Require().NoError(err, "failed to get start block header")
	startBlock := startHeader.Number.Uint64()
	loadStart := time.Now()
	t.Logf("start block: %d (after warmup)", startBlock)

	// wait for all transactions to be sent
	waitForMetricTarget(t, "spamoor_transactions_sent_total", pollSentTotal, float64(totalCount), cfg.WaitTimeout)

	// wait for pending txs to drain
	drainCtx, drainCancel := context.WithTimeout(ctx, 30*time.Second)
	defer drainCancel()
	if err := waitForDrain(drainCtx, t.Logf, e.ethClient, 10); err != nil {
		t.Logf("warning: %v", err)
	}
	wallClock := time.Since(loadStart)

	endHeader, err := e.ethClient.HeaderByNumber(ctx, nil)
	s.Require().NoError(err, "failed to get end block header")
	endBlock := endHeader.Number.Uint64()
	t.Logf("end block: %d (range %d blocks)", endBlock, endBlock-startBlock)

	// collect block-level gas/tx metrics
	bm, err := collectBlockMetrics(ctx, e.ethClient, startBlock, endBlock)
	s.Require().NoError(err, "failed to collect block metrics")

	traces := s.collectTraces(e, cfg.ServiceName)

	result := newBenchmarkResult("SequencerOverhead", bm, traces)
	s.Require().Greater(result.summary.SteadyState, time.Duration(0), "expected non-zero steady-state duration")
	result.log(t, wallClock)
	w.addEntries(result.entries())
}
