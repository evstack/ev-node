package runner

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/evstack/ev-node/apps/loadgen/internal/matrix"
	"github.com/evstack/ev-node/apps/loadgen/internal/spamoor"

	dto "github.com/prometheus/client_model/go"
)

const (
	metricSentTotal   = "spamoor_transactions_sent_total"
	metricFailedTotal = "spamoor_transactions_failed_total"
)

type matrixOpts struct {
	totalTxTarget    int
	allowProbability bool
	waitForSync      bool
}

// ExecuteMatrixFromFile loads a matrix from disk and executes it with
// probability filtering and an initial WaitForSync call.
func ExecuteMatrixFromFile(ctx context.Context, matrixPath string, api spamoor.Client) error {
	m, err := matrix.Load(matrixPath)
	if err != nil {
		return fmt.Errorf("load matrix: %w", err)
	}
	return executeMatrix(ctx, *m, api, matrixOpts{
		allowProbability: true,
		waitForSync:      true,
	})
}

// ExecuteMatrixWithOverridesFromFile loads a matrix from disk and executes it
// with BENCH_COUNT_PER_SPAMMER overridden to totalTxTarget / NumSpammers.
// Skips probability filtering and sync waiting (caller is responsible for sync).
func ExecuteMatrixWithOverridesFromFile(ctx context.Context, matrixPath string, api spamoor.Client, totalTxTarget int) error {
	m, err := matrix.Load(matrixPath)
	if err != nil {
		return fmt.Errorf("load matrix: %w", err)
	}
	return executeMatrix(ctx, *m, api, matrixOpts{
		totalTxTarget: totalTxTarget,
	})
}

func executeMatrix(ctx context.Context, m matrix.Matrix, api spamoor.Client, opts matrixOpts) error {
	log.Printf("spamoor API: %s", api.URL())
	if opts.totalTxTarget > 0 {
		log.Printf("loaded %d matrix entries (totalTxTarget=%d)", len(m.Entries), opts.totalTxTarget)
	} else {
		log.Printf("loaded %d matrix entries", len(m.Entries))
	}

	if opts.waitForSync {
		if err := WaitForSync(ctx, api); err != nil {
			return fmt.Errorf("waiting for spamoor sync: %w", err)
		}
	}

	var failures []string
	for i, entry := range m.Entries {
		if err := ctx.Err(); err != nil {
			log.Printf("cancelled before entry %d/%d", i+1, len(m.Entries))
			return err
		}

		log.Printf("--- [%d/%d] %s ---", i+1, len(m.Entries), entry.TestName)

		if opts.allowProbability && entry.Probability != nil {
			roll := rand.Float64()
			if roll >= *entry.Probability {
				log.Printf("[%s] skipped (probability=%.2f, roll=%.4f)", entry.TestName, *entry.Probability, roll)
				continue
			}
			log.Printf("[%s] triggered (probability=%.2f, roll=%.4f)", entry.TestName, *entry.Probability, roll)
		}

		if opts.totalTxTarget > 0 {
			countPerSpammer := opts.totalTxTarget / entry.NumSpammers
			if countPerSpammer < 1 {
				countPerSpammer = 1
			}
			entry.Env[matrix.EnvCountPerSpammer] = fmt.Sprintf("%d", countPerSpammer)
			entry.CountPerSpammer = countPerSpammer
		}

		timeout := 15 * time.Minute
		if entry.ParsedTimeout > 0 {
			timeout = entry.ParsedTimeout
		}

		entryCtx, cancel := context.WithTimeout(ctx, timeout)
		if err := runEntry(entryCtx, api, entry); err != nil {
			log.Printf("FAIL [%s]: %v", entry.TestName, err)
			failures = append(failures, entry.TestName)
		}
		cancel()
	}

	log.Printf("=== completed: %d/%d succeeded ===", len(m.Entries)-len(failures), len(m.Entries))
	if len(failures) > 0 {
		return fmt.Errorf("failed entries: %v", failures)
	}
	return nil
}

type waitForDoneFunc func(ctx context.Context, api spamoor.Client, targetCount int, baselineSent, baselineFailed float64, namePrefix string) (float64, float64, error)

func runEntry(ctx context.Context, api spamoor.Client, entry matrix.Entry) error {
	return runEntryWithWait(ctx, api, entry, waitForSpamoorDone)
}

func runEntryWithWait(ctx context.Context, api spamoor.Client, entry matrix.Entry, wait waitForDoneFunc) error {
	totalCount := entry.NumSpammers * entry.CountPerSpammer

	log.Printf("[%s] scenario=%s spammers=%d count_per=%d total=%d",
		entry.TestName, entry.Scenario, entry.NumSpammers, entry.CountPerSpammer, totalCount)

	namePrefix := spammerPrefix(entry.TestName)

	baselineSent, baselineFailed, err := getSpamoorCountersForPrefix(api, namePrefix)
	if err != nil {
		return fmt.Errorf("get baseline metrics: %w", err)
	}

	scenarioCfg := spamoor.BuildScenarioConfig(entry.Env)

	var spammerIDs []int
	for i := range entry.NumSpammers {
		name := fmt.Sprintf("%s%d", namePrefix, i)
		id, err := api.CreateSpammer(name, entry.Scenario, scenarioCfg, true)
		if err != nil {
			return fmt.Errorf("create spammer %s: %w", name, err)
		}
		spammerIDs = append(spammerIDs, id)
		log.Printf("[%s] created spammer %s (id=%d)", entry.TestName, name, id)
	}
	defer func() {
		for _, id := range spammerIDs {
			if err := api.DeleteSpammer(id); err != nil {
				log.Printf("[%s] warning: cleanup spammer %d failed: %v", entry.TestName, id, err)
			}
		}
	}()

	for _, id := range spammerIDs {
		sp, err := api.GetSpammer(id)
		if err != nil {
			return fmt.Errorf("get spammer %d: %w", id, err)
		}
		if sp.Status == 0 {
			return fmt.Errorf("spammer %d (%s) failed to start (status=0)", id, sp.Name)
		}
	}

	start := time.Now()
	sent, failed, err := wait(ctx, api, totalCount, baselineSent, baselineFailed, namePrefix)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("[%s] ERROR: %v (sent=%.0f failed=%.0f elapsed=%s)",
			entry.TestName, err, sent, failed, elapsed.Round(time.Second))
		return err
	}

	log.Printf("[%s] DONE: sent=%.0f failed=%.0f elapsed=%s avg_rate=%.1f tx/s",
		entry.TestName, sent, failed, elapsed.Round(time.Second), sent/elapsed.Seconds())

	return nil
}

// DeleteAllSpammers removes all active spammers from the spamoor daemon.
func DeleteAllSpammers(api spamoor.Client) error {
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

func waitForSpamoorDone(ctx context.Context, api spamoor.Client, targetCount int, baselineSent, baselineFailed float64, namePrefix string) (sent, failed float64, err error) {
	return waitForSpamoorDoneWithInterval(ctx, api, targetCount, baselineSent, baselineFailed, namePrefix, 2*time.Second)
}

func waitForSpamoorDoneWithInterval(ctx context.Context, api spamoor.Client, targetCount int, baselineSent, baselineFailed float64, namePrefix string, pollInterval time.Duration) (sent, failed float64, err error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	start := time.Now()
	var prevSent float64

	for {
		select {
		case <-ctx.Done():
			return sent, failed, fmt.Errorf("timed out waiting for %d txs (sent %.0f): %w", targetCount, sent, ctx.Err())
		case <-ticker.C:
			currentSent, currentFailed, mErr := getSpamoorCountersForPrefix(api, namePrefix)
			if mErr != nil {
				log.Printf("warning: failed to get metrics: %v", mErr)
				continue
			}

			sent = currentSent - baselineSent
			failed = currentFailed - baselineFailed
			if sent < 0 {
				sent = 0
			}
			if failed < 0 {
				failed = 0
			}

			delta := sent - prevSent
			rate := delta / pollInterval.Seconds()
			elapsed := time.Since(start)
			log.Printf("  progress: %.0f/%d sent (%.0f tx/s instant, %.1f tx/s avg, %.0f failed) [%s]",
				sent, targetCount, rate, sent/elapsed.Seconds(), failed, elapsed.Round(time.Second))
			prevSent = sent

			if sent >= float64(targetCount) {
				return sent, failed, nil
			}
		}
	}
}

func getSpamoorCountersForPrefix(api spamoor.Client, namePrefix string) (sent, failed float64, err error) {
	metrics, err := api.GetMetrics()
	if err != nil {
		return 0, 0, err
	}
	return sumCounterWithPrefix(metrics[metricSentTotal], namePrefix),
		sumCounterWithPrefix(metrics[metricFailedTotal], namePrefix),
		nil
}

// WaitForSync polls spamoor until its block processing rate stabilizes,
// indicating it has caught up to the chain tip.
func WaitForSync(ctx context.Context, api spamoor.Client) error {
	return waitForSync(ctx, api, 3*time.Second)
}

func waitForSync(ctx context.Context, api spamoor.Client, pollInterval time.Duration) error {
	log.Printf("waiting for spamoor to sync to chain tip...")

	const syncThreshold uint64 = 10

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var lastHeight uint64

	for {
		height, err := getSpamoorHeight(api)
		if err != nil {
			log.Printf("warning: %v", err)
		} else {
			if lastHeight > 0 && height > 0 {
				delta := height - lastHeight
				if delta < syncThreshold {
					log.Printf("spamoor synced at block %d (delta=%d in %s)", height, delta, pollInterval)
					return nil
				}
				log.Printf("syncing... block %d (+%d in %s)", height, delta, pollInterval)
			} else {
				log.Printf("syncing... block %d", height)
			}
			lastHeight = height
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("cancelled waiting for sync: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func getSpamoorHeight(api spamoor.Client) (uint64, error) {
	clients, err := api.GetClients()
	if err != nil {
		return 0, fmt.Errorf("get clients: %w", err)
	}
	if len(clients) == 0 {
		return 0, fmt.Errorf("no RPC clients configured")
	}
	return clients[0].Height, nil
}

func sumCounterWithPrefix(f *dto.MetricFamily, namePrefix string) float64 {
	if f == nil || f.GetType() != dto.MetricType_COUNTER {
		return 0
	}
	var sum float64
	for _, m := range f.GetMetric() {
		if m.GetCounter() == nil || m.GetCounter().Value == nil {
			continue
		}
		if namePrefix != "" && !hasLabelPrefix(m, "spammer_name", namePrefix) {
			continue
		}
		sum += m.GetCounter().GetValue()
	}
	return sum
}

func spammerPrefix(testName string) string {
	return fmt.Sprintf("bench-%s-", testName)
}

func hasLabelPrefix(m *dto.Metric, name, prefix string) bool {
	for _, lp := range m.GetLabel() {
		if lp.GetName() == name {
			return strings.HasPrefix(lp.GetValue(), prefix)
		}
	}
	return false
}
