//go:build evm

package benchmark

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
)

// TestStatePressure measures throughput under maximum storage write pressure.
// Each tx maximizes SSTORE operations, creating rapid state growth that stresses
// the state trie and disk I/O.
//
// Shares system configuration with TestERC20Throughput (100ms blocks, 100M gas,
// 25ms scrape) so results are directly comparable. The gap between
// TestEVMComputeCeiling and this test isolates state root + storage I/O cost.
//
// Primary metrics: MGas/s.
// Diagnostic metrics: Engine.NewPayload latency, ev-node overhead %.
func (s *SpamoorSuite) TestStatePressure() {
	cfg := newBenchConfig("ev-node-state-pressure")

	t := s.T()
	ctx := t.Context()
	cfg.log(t)
	w := newResultWriter(t, "StatePressure")
	defer w.flush()

	var result *benchmarkResult
	var wallClock time.Duration
	var spamoorStats *runSpamoorStats
	defer func() {
		if result != nil {
			emitRunResult(t, cfg, result, wallClock, spamoorStats)
		}
	}()

	e := s.setupEnv(cfg)

	storageSpamConfig := map[string]any{
		"throughput":        cfg.Throughput,
		"total_count":       cfg.CountPerSpammer,
		"gas_units_to_burn": cfg.GasUnitsToBurn,
		"max_pending":       cfg.MaxPending,
		"max_wallets":       cfg.MaxWallets,
		"rebroadcast":       cfg.Rebroadcast,
		"base_fee":          cfg.BaseFee,
		"tip_fee":           cfg.TipFee,
		"refill_amount":     "5000000000000000000", // 5 ETH
		"refill_balance":    "2000000000000000000", // 2 ETH
		"refill_interval":   600,
	}

	s.Require().NoError(deleteAllSpammers(e.spamoorAPI), "failed to delete stale spammers")

	var spammerIDs []int
	for i := range cfg.NumSpammers {
		name := fmt.Sprintf("bench-storage-%d", i)
		id, err := e.spamoorAPI.CreateSpammer(name, spamoor.ScenarioStorageSpam, storageSpamConfig, true)
		s.Require().NoError(err, "failed to create spammer %s", name)
		spammerIDs = append(spammerIDs, id)
		t.Cleanup(func() { _ = e.spamoorAPI.DeleteSpammer(id) })
	}

	requireSpammersRunning(t, e.spamoorAPI, spammerIDs)

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
	waitForMetricTarget(t, "spamoor_transactions_sent_total", pollSentTotal, float64(cfg.totalCount()), cfg.WaitTimeout)

	// wait for pending txs to drain
	drainCtx, drainCancel := context.WithTimeout(ctx, 30*time.Second)
	defer drainCancel()
	if err := waitForDrain(drainCtx, t.Logf, e.ethClient, 10); err != nil {
		t.Logf("warning: %v", err)
	}
	wallClock = time.Since(loadStart)

	endHeader, err := e.ethClient.HeaderByNumber(ctx, nil)
	s.Require().NoError(err, "failed to get end block header")
	endBlock := endHeader.Number.Uint64()
	t.Logf("end block: %d (range %d blocks)", endBlock, endBlock-startBlock)

	// collect block-level gas/tx metrics
	bm, err := collectBlockMetrics(ctx, e.ethClient, startBlock, endBlock)
	s.Require().NoError(err, "failed to collect block metrics")

	traces := s.collectTraces(e)

	result = newBenchmarkResult("StatePressure", bm, traces)
	s.Require().Greater(result.summary.SteadyState, time.Duration(0), "expected non-zero steady-state duration")
	result.log(t, wallClock)
	w.addEntries(result.entries())

	metrics, mErr := e.spamoorAPI.GetMetrics()
	s.Require().NoError(mErr, "failed to get final metrics")
	sent := sumCounter(metrics["spamoor_transactions_sent_total"])
	failed := sumCounter(metrics["spamoor_transactions_failed_total"])
	spamoorStats = &runSpamoorStats{Sent: sent, Failed: failed}

	s.Require().Greater(sent, float64(0), "at least one transaction should have been sent")
	s.Require().Zero(failed, "no transactions should have failed")
}
