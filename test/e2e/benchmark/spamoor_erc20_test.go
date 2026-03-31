//go:build evm

package benchmark

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
)

// TestERC20Throughput measures ERC-20 token transfer throughput in isolation.
// Each tx is ~65K gas (contract call with storage reads/writes)
//
// Primary metric: achieved MGas/s.
// Diagnostic metrics: per-span latency breakdown, gas/block, tx/block.
func (s *SpamoorSuite) TestERC20Throughput() {
	cfg := newBenchConfig("ev-node-erc20")

	t := s.T()
	ctx := t.Context()
	w := newResultWriter(t, "ERC20Throughput")
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

	erc20Config := map[string]any{
		"throughput":      cfg.Throughput,
		"total_count":     cfg.CountPerSpammer,
		"max_pending":     cfg.MaxPending,
		"max_wallets":     cfg.MaxWallets,
		"base_fee":        cfg.BaseFee,
		"tip_fee":         cfg.TipFee,
		"refill_amount":   "5000000000000000000",
		"refill_balance":  "2000000000000000000",
		"refill_interval": 600,
	}

	// clear any stale spammers restored from the persistent spamoor database
	s.Require().NoError(deleteAllSpammers(e.spamoorAPI), "failed to delete stale spammers")

	// launch all spammers before recording startBlock so warm-up
	// (contract deploy + wallet funding) is excluded from the measurement window.
	for i := range cfg.NumSpammers {
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
	waitCtx, cancel := context.WithTimeout(ctx, cfg.WaitTimeout)
	defer cancel()
	sent, failed, err := waitForSpamoorDone(waitCtx, t.Logf, e.spamoorAPI, cfg.totalCount(), 2*time.Second)
	s.Require().NoError(err, "spamoor did not finish in time")

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

	// collect block-level gas/tx metrics
	bm, err := collectBlockMetrics(ctx, e.ethClient, startBlock, endBlock)
	s.Require().NoError(err, "failed to collect block metrics")

	traces := s.collectTraces(e)

	result = newBenchmarkResult("ERC20Throughput", bm, traces)
	s.Require().Greater(result.summary.SteadyState, time.Duration(0), "expected non-zero steady-state duration")
	result.log(t, wallClock)
	w.addEntries(result.entries())

	spamoorStats = &runSpamoorStats{Sent: sent, Failed: failed}

	s.Require().Greater(sent, float64(0), "at least one transaction should have been sent")
	s.Require().Zero(failed, "no transactions should have failed")
}
