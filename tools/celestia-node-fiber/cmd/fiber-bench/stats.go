package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// statsPrinter periodically prints a one-line summary combining counters
// from the in-mem executor and selected Prometheus metrics scraped from
// ev-node's instrumentation endpoint.
//
// Why scrape Prometheus instead of reaching into ev-node? Because the
// metrics ev-node already exports give us the answers we want
// (committed height, txs-per-block, pending blobs, block-production
// duration histogram) and scraping is zero source diff. It also makes
// the same numbers available to a real Prometheus once we move past the
// fail-fast baseline.
type statsPrinter struct {
	exec       *inMemExecutor
	promURL    string
	httpClient *http.Client
	txSize     int
	adapter    *instrumentedAdapter

	mu         sync.Mutex
	startedAt  time.Time
	lastTick   time.Time
	lastInject uint64
	lastTxs    float64
	lastBlocks float64
	lastDaInc  float64
	peakInjRPS float64
	peakTxRPS  float64
	peakDaRPS  float64

	// lastSnapshot caches the last successful Prometheus scrape so
	// the final summary still has values after the node has shut
	// down (its /metrics endpoint goes away with it).
	lastSnapshot map[string]float64
}

func newStatsPrinter(exec *inMemExecutor, promListenAddr string, txSize int, adapter *instrumentedAdapter) *statsPrinter {
	url := ""
	if promListenAddr != "" {
		// PrometheusListenAddr can be ":26660" or "127.0.0.1:26660";
		// normalize to a fetchable URL.
		host := promListenAddr
		if strings.HasPrefix(host, ":") {
			host = "127.0.0.1" + host
		}
		url = "http://" + host + "/metrics"
	}
	return &statsPrinter{
		exec:       exec,
		promURL:    url,
		httpClient: &http.Client{Timeout: 500 * time.Millisecond},
		txSize:     txSize,
		adapter:    adapter,
	}
}

// start prints a header then ticks every interval until ctx is done.
func (p *statsPrinter) start(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Second
	}
	now := time.Now()
	p.mu.Lock()
	p.startedAt = now
	p.lastTick = now
	p.mu.Unlock()

	fmt.Println()
	// Each rate column shows "<tx/s> / <MB/s>" so tps and bandwidth
	// land side by side without doubling the column count. The blob
	// size at the latest block stays as an absolute (blob_KB) since
	// it's a level, not a rate.
	fmt.Printf("%-9s %-15s %-15s %-15s %-7s %-9s %-7s %-8s %-7s %-10s %s\n",
		"elapsed", "inj tps/MBs", "exec tps/MBs", "da tps/MBs",
		"prod_h", "da_inc_h", "txs/blk", "blob_KB", "pending", "drops", "upload latency")
	fmt.Println(strings.Repeat("-", 140))

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				p.tick()
			}
		}
	}()
}

func (p *statsPrinter) tick() {
	now := time.Now()

	injected, dropped, blocks, txs, _ := p.exec.Stats()
	mempool := p.exec.MempoolDepth()

	prom := p.scrapePrometheus()
	if len(prom) > 0 {
		p.mu.Lock()
		p.lastSnapshot = prom
		p.mu.Unlock()
	}
	// ev-node prefixes its metrics with the namespace from the metrics
	// provider — for the aggregator path this is "evnode_sequencer".
	producedHeight := prom["evnode_sequencer_height"]
	daInclusionHeight := prom["evnode_sequencer_da_inclusion_height"]
	totalTxs := prom["evnode_sequencer_total_txs"]
	if totalTxs == 0 {
		totalTxs = float64(txs)
	}
	blockBytes := prom["evnode_sequencer_block_size_bytes"]
	pending := prom["evnode_sequencer_da_submitter_pending_blobs"]
	blocksGauge := float64(blocks)
	if producedHeight > blocksGauge {
		blocksGauge = producedHeight
	}
	txsPerBlock := txsPerBlockMetric(blocksGauge, totalTxs)

	p.mu.Lock()
	dt := now.Sub(p.lastTick).Seconds()
	if dt < 0.001 {
		p.mu.Unlock()
		return
	}
	injRPS := float64(injected-p.lastInject) / dt
	txRPS := (totalTxs - p.lastTxs) / dt
	daSettledRPS := (daInclusionHeight - p.lastDaInc) * txsPerBlock / dt
	if injRPS > p.peakInjRPS {
		p.peakInjRPS = injRPS
	}
	if txRPS > p.peakTxRPS {
		p.peakTxRPS = txRPS
	}
	if daSettledRPS > p.peakDaRPS {
		p.peakDaRPS = daSettledRPS
	}
	elapsed := now.Sub(p.startedAt).Truncate(time.Millisecond)
	p.lastTick = now
	p.lastInject = injected
	p.lastTxs = totalTxs
	p.lastBlocks = blocksGauge
	p.lastDaInc = daInclusionHeight
	p.mu.Unlock()

	txSizeBytes := float64(p.txSize)

	upStats := p.adapter.uploadStats()

	fmt.Printf("%-9s %-15s %-15s %-15s %-7.0f %-9.0f %-7.0f %-8.0f %-7.0f %-10d %s\n",
		elapsed.String(),
		formatRate(injRPS, txSizeBytes),
		formatRate(txRPS, txSizeBytes),
		formatRate(daSettledRPS, txSizeBytes),
		producedHeight, daInclusionHeight, txsPerBlock, blockBytes/1024, pending, dropped,
		formatUploadLatency(upStats),
	)

	_ = mempool // currently we report drops, not depth — the mempool is large enough that depth isn't the meaningful signal
}

// formatUploadLatency renders Upload latency stats as a compact suffix
// for the live table. Returns "-" if no samples yet.
func formatUploadLatency(s uploadStats) string {
	if s.Count == 0 {
		return "upload[-]"
	}
	failPart := ""
	if s.Failures > 0 {
		failPart = fmt.Sprintf(",fails=%d", s.Failures)
	}
	return fmt.Sprintf("upload[n=%d p50=%v p99=%v%s]",
		s.Count, s.P50.Truncate(time.Millisecond), s.P99.Truncate(time.Millisecond), failPart)
}

// formatRate renders "<tps> / <MB/s>" compactly, rounding to whole MB/s
// since sub-MB/s precision isn't useful at our throughput levels and a
// short string keeps the table aligned.
func formatRate(rps, txSizeBytes float64) string {
	mbps := rps * txSizeBytes / (1024 * 1024)
	switch {
	case rps >= 1_000_000:
		return fmt.Sprintf("%.1fM/%.0fMB", rps/1_000_000, mbps)
	case rps >= 1_000:
		return fmt.Sprintf("%.0fk/%.0fMB", rps/1_000, mbps)
	default:
		return fmt.Sprintf("%.0f/%.1fMB", rps, mbps)
	}
}

// txsPerBlockMetric computes the running mean tx/blk over all produced
// blocks. Only meaningful once at least one block has been produced;
// returns 0 otherwise.
func txsPerBlockMetric(blocks, totalTxs float64) float64 {
	if blocks <= 0 {
		return 0
	}
	return totalTxs / blocks
}

// scrapePrometheus pulls the ev-node /metrics endpoint and parses just
// the gauges/counters we care about. Best effort: returns empty map on
// any error so the bench keeps running even if metrics aren't ready yet.
func (p *statsPrinter) scrapePrometheus() map[string]float64 {
	out := map[string]float64{}
	if p.promURL == "" {
		return out
	}
	resp, err := p.httpClient.Get(p.promURL)
	if err != nil {
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return out
	}

	wanted := map[string]struct{}{
		"evnode_sequencer_height":                     {},
		"evnode_sequencer_latest_block_height":        {},
		"evnode_sequencer_da_inclusion_height":        {},
		"evnode_sequencer_total_txs":                  {},
		"evnode_sequencer_num_txs":                    {},
		"evnode_sequencer_block_size_bytes":           {},
		"evnode_sequencer_da_submitter_pending_blobs": {},
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// "metric_name{labels...} value [timestamp]" — strip labels and
		// trailing timestamp; we don't use them.
		nameEnd := strings.IndexAny(line, "{ ")
		if nameEnd < 0 {
			continue
		}
		name := line[:nameEnd]
		if _, ok := wanted[name]; !ok {
			continue
		}
		// Skip past labels if present.
		rest := line[nameEnd:]
		if rest[0] == '{' {
			closeIdx := strings.Index(rest, "}")
			if closeIdx < 0 {
				continue
			}
			rest = rest[closeIdx+1:]
		}
		rest = strings.TrimSpace(rest)
		valEnd := strings.IndexByte(rest, ' ')
		valStr := rest
		if valEnd >= 0 {
			valStr = rest[:valEnd]
		}
		v, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			continue
		}
		out[name] = v
	}
	return out
}

func (p *statsPrinter) printFinalSummary() {
	injected, dropped, blocks, txs, mempoolHigh := p.exec.Stats()
	// Prefer a fresh scrape, but fall back to the last live snapshot:
	// the node's /metrics endpoint goes away as it shuts down, so a
	// post-stop scrape returns an empty map and the summary would
	// otherwise print zeros.
	prom := p.scrapePrometheus()
	p.mu.Lock()
	if len(prom) == 0 && p.lastSnapshot != nil {
		prom = p.lastSnapshot
	}
	p.mu.Unlock()
	producedHeight := uint64(prom["evnode_sequencer_height"])
	daInclusionHeight := uint64(prom["evnode_sequencer_da_inclusion_height"])
	totalTxs := uint64(prom["evnode_sequencer_total_txs"])
	if totalTxs == 0 {
		totalTxs = txs
	}

	p.mu.Lock()
	elapsed := time.Since(p.startedAt)
	peakInj := p.peakInjRPS
	peakTx := p.peakTxRPS
	p.mu.Unlock()

	avgInj := 0.0
	if elapsed.Seconds() > 0 {
		avgInj = float64(injected) / elapsed.Seconds()
	}
	avgTx := 0.0
	if elapsed.Seconds() > 0 {
		avgTx = float64(totalTxs) / elapsed.Seconds()
	}
	txsPerBlock := 0.0
	if blocks > 0 {
		txsPerBlock = float64(totalTxs) / float64(blocks)
	}
	txSize := float64(p.txSize)
	mb := func(rps float64) float64 { return rps * txSize / (1024 * 1024) }

	p.mu.Lock()
	peakDa := p.peakDaRPS
	p.mu.Unlock()

	var avgDaSettled float64
	if daInclusionHeight > 0 && elapsed.Seconds() > 0 {
		avgDaSettled = float64(daInclusionHeight) * txsPerBlock / elapsed.Seconds()
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("                       BASELINE SUMMARY")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("Duration:               %s\n", elapsed.Truncate(time.Millisecond))
	fmt.Printf("Tx size:                %d B\n", p.txSize)
	fmt.Println()
	fmt.Printf("Injection:              avg %.0f tx/s (%.1f MB/s), peak %.0f tx/s (%.0f MB/s)\n",
		avgInj, mb(avgInj), peakInj, mb(peakInj))
	fmt.Printf("Block production:       avg %.0f tx/s (%.2f MB/s), peak %.0f tx/s (%.1f MB/s)\n",
		avgTx, mb(avgTx), peakTx, mb(peakTx))
	fmt.Printf("DA-settled:             avg %.0f tx/s (%.2f MB/s), peak %.0f tx/s (%.1f MB/s)\n",
		avgDaSettled, mb(avgDaSettled), peakDa, mb(peakDa))
	fmt.Println()
	fmt.Printf("Blocks produced:        %d (prod_h=%d)\n", blocks, producedHeight)
	fmt.Printf("DA-included height:     %d (lag = %d blocks behind production)\n",
		daInclusionHeight, producedHeight-daInclusionHeight)
	fmt.Printf("Txs into blocks:        %d (%.1f tx/blk)\n", totalTxs, txsPerBlock)
	fmt.Printf("Dropped (mempool full): %d\n", dropped)
	fmt.Printf("Mempool high-water:     %d\n", mempoolHigh)

	upStats := p.adapter.uploadStats()
	if upStats.Count > 0 {
		fmt.Println()
		fmt.Println("Fibre Upload latency (per call observed at the adapter):")
		fmt.Printf("  count:     %d (failures: %d)\n", upStats.Count, upStats.Failures)
		fmt.Printf("  mean:      %s\n", upStats.Mean.Truncate(time.Millisecond))
		fmt.Printf("  p50:       %s\n", upStats.P50.Truncate(time.Millisecond))
		fmt.Printf("  p99:       %s\n", upStats.P99.Truncate(time.Millisecond))
		fmt.Printf("  max:       %s\n", upStats.Max.Truncate(time.Millisecond))
		// ev-node's submitter runs ONE header-Upload goroutine and
		// ONE data-Upload goroutine concurrently (each TryLock'd via
		// its own mutex in submitter.go). A block settles only when
		// BOTH its header and data Uploads have returned, and each
		// stream submits at most 1 Upload per mean_latency seconds —
		// so the per-stream cap is 1/mean blocks/s, and the block
		// settlement cap (min of the two) equals it. We print this
		// so the operator can compare it to the observed da_inc_h
		// rate and tell apart "Fibre Upload is slow" from "ev-node
		// is leaving capacity on the table".
		if upStats.Mean > 0 {
			capBlocksPerSec := 1.0 / upStats.Mean.Seconds()
			fmt.Printf("  implied cap (1/mean per stream): %.2f blocks/s ≈ %.0f tx/s (%.2f MB/s)\n",
				capBlocksPerSec,
				capBlocksPerSec*txsPerBlock,
				capBlocksPerSec*txsPerBlock*txSize/(1024*1024),
			)
		}
	}
	fmt.Println(strings.Repeat("=", 70))
}
