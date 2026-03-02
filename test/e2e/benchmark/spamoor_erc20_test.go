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
	const (
		numSpammers     = 2
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

	summary := bm.summarize()
	s.Require().Greater(summary.SteadyState, time.Duration(0), "expected non-zero steady-state duration")
	summary.log(t, startBlock, endBlock, bm.TotalBlockCount, bm.BlockCount, wallClock)

	// collect and report traces
	traces := s.collectTraces(e, serviceName)

	if overhead, ok := evNodeOverhead(traces.evNode); ok {
		t.Logf("ev-node overhead: %.1f%%", overhead)
		w.addEntry(entry{Name: "ERC20Throughput - ev-node overhead", Unit: "%", Value: overhead})
	}

	w.addEntries(summary.entries("ERC20Throughput"))
	w.addSpans(traces.allSpans())

	s.Require().Greater(sent, float64(0), "at least one transaction should have been sent")
	s.Require().Zero(failed, "no transactions should have failed")
}
