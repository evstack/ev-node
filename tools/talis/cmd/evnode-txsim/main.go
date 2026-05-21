// Command evnode-txsim drives the ev-node aggregator's HTTP /tx endpoint
// at a fixed rate for a fixed duration. Stdlib-only; deployed by talis
// onto a dedicated load-gen instance that lives separately from
// ev-node and bridge boxes so its own CPU / network do not bias the
// measurement.
//
// Output format (final line, machine-grep'able):
//
//	TXSIM: target=http://X:7777/tx duration=30s tx_size=10240
//	       concurrency=8 sent=300000 ok=300000 err=0
//	       wall_s=30.00 sent_per_s=10000 mb_per_s=97.66
//	       rtt_p50_us=145 rtt_p99_us=820
//
// Concurrency model: N goroutines, each posting txs at most as fast as
// the server accepts them. There's no client-side rate cap by design —
// the goal is to back-pressure the server and measure its absorption
// rate, not to simulate a paced client.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type cliFlags struct {
	target      string
	txSize      int
	concurrency int
	duration    time.Duration
	timeout     time.Duration
	verbose     bool
}

func parseFlags() cliFlags {
	var c cliFlags
	flag.StringVar(&c.target, "target", envOr("TARGET", "http://127.0.0.1:7777/tx"),
		"ev-node tx-ingest endpoint (POST raw bytes)")
	flag.IntVar(&c.txSize, "tx-size", intFromEnv("TX_SIZE", 10*1024),
		"per-tx payload size in bytes")
	flag.IntVar(&c.concurrency, "concurrency", intFromEnv("CONCURRENCY", 8),
		"number of concurrent posters")
	flag.DurationVar(&c.duration, "duration", durFromEnv("DURATION", 30*time.Second),
		"how long to pump (0 = until SIGTERM)")
	flag.DurationVar(&c.timeout, "timeout", durFromEnv("TIMEOUT", 5*time.Second),
		"per-request HTTP timeout")
	flag.BoolVar(&c.verbose, "verbose", false, "log every error to stderr")
	flag.Parse()
	return c
}

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func intFromEnv(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func durFromEnv(name string, def time.Duration) time.Duration {
	if v := os.Getenv(name); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func main() {
	cli := parseFlags()
	if err := run(cli); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run(cli cliFlags) error {
	if cli.txSize <= 16 {
		return fmt.Errorf("--tx-size must be > 16 (header is 16 bytes: seq + emit_time)")
	}

	// Pre-fill a randomness pool sized for cheap per-tx sampling. At
	// 100 MiB/s of tx bytes, calling rand.Read per tx is itself the
	// hot path; sampling from a fixed pool is dramatically cheaper
	// and the experiment doesn't care about cryptographic uniqueness.
	poolSize := 8 * cli.txSize
	if poolSize < (1 << 20) {
		poolSize = 1 << 20
	}
	pool := make([]byte, poolSize)
	if _, err := rand.Read(pool); err != nil {
		return fmt.Errorf("seed random pool: %w", err)
	}

	// Bump per-host idle connections so concurrent goroutines reuse
	// keep-alive sockets instead of churning TCP+TLS handshakes —
	// stdlib default MaxIdleConnsPerHost=2 caps in-flight requests
	// to 2 keep-alive sockets per target, which serializes any
	// concurrency>2 onto fresh connections each request.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 2 * cli.concurrency
	transport.MaxIdleConnsPerHost = 2 * cli.concurrency
	transport.MaxConnsPerHost = 2 * cli.concurrency
	httpClient := &http.Client{Timeout: cli.timeout, Transport: transport}

	// pprof on a dedicated listener — `_ "net/http/pprof"` registers
	// handlers on http.DefaultServeMux. Always-on at 127.0.0.1:6060
	// since this is a load-tester binary, not a production daemon;
	// SSH port-forward to grab profiles under load:
	//
	//	ssh -L 6060:127.0.0.1:6060 root@loadgen \
	//	  go tool pprof http://localhost:6060/debug/pprof/profile?seconds=10
	go func() {
		_ = http.ListenAndServe("127.0.0.1:6060", nil)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	if cli.duration > 0 {
		ctx, cancel = context.WithTimeout(ctx, cli.duration)
	}
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	var (
		sent       atomic.Uint64
		ok         atomic.Uint64
		fail       atomic.Uint64
		totalBytes atomic.Uint64
	)

	// RTT samples are collected in per-worker buffers and merged at
	// the end. With a 30 s pump at 10 KiB/s per worker × 8 workers,
	// that's ~30 000 samples per worker, which fits in memory comfortably.
	rttBufs := make([][]int64, cli.concurrency) // microseconds

	wg := sync.WaitGroup{}
	start := time.Now()
	for i := range cli.concurrency {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			rtts := make([]int64, 0, 16384)
			defer func() { rttBufs[idx] = rtts }()

			buf := make([]byte, cli.txSize)
			poolLen := len(pool)
			var localSeq uint64
			for {
				if ctx.Err() != nil {
					return
				}
				localSeq++
				now := time.Now()
				binary.BigEndian.PutUint64(buf, uint64(idx)<<32|localSeq)
				binary.BigEndian.PutUint64(buf[8:], uint64(now.UnixNano()))
				offset := int((localSeq * 7919) % uint64(poolLen-cli.txSize+16))
				copy(buf[16:], pool[offset:offset+cli.txSize-16])

				req, err := http.NewRequestWithContext(ctx, http.MethodPost, cli.target, bytes.NewReader(buf))
				if err != nil {
					sent.Add(1)
					fail.Add(1)
					if cli.verbose {
						fmt.Fprintln(os.Stderr, "request build:", err)
					}
					continue
				}
				req.Header.Set("Content-Type", "application/octet-stream")

				rttStart := time.Now()
				resp, err := httpClient.Do(req)
				rtt := time.Since(rttStart)
				sent.Add(1)
				if err != nil {
					fail.Add(1)
					if cli.verbose {
						fmt.Fprintln(os.Stderr, "post:", err)
					}
					continue
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					ok.Add(1)
					totalBytes.Add(uint64(cli.txSize))
				} else {
					fail.Add(1)
					if cli.verbose {
						fmt.Fprintln(os.Stderr, "http:", resp.StatusCode)
					}
				}
				rtts = append(rtts, rtt.Microseconds())
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Merge RTT buffers and compute percentiles. Sorting in place is
	// fine — the buffer goroutines have all returned by now.
	var allRTT []int64
	totalRTT := 0
	for _, b := range rttBufs {
		totalRTT += len(b)
	}
	allRTT = make([]int64, 0, totalRTT)
	for _, b := range rttBufs {
		allRTT = append(allRTT, b...)
	}
	sort.Slice(allRTT, func(i, j int) bool { return allRTT[i] < allRTT[j] })

	p50 := percentileMicros(allRTT, 0.50)
	p99 := percentileMicros(allRTT, 0.99)

	mb := float64(totalBytes.Load()) / (1024 * 1024)
	mbPerS := mb / elapsed.Seconds()
	sentPerS := float64(sent.Load()) / elapsed.Seconds()

	fmt.Printf("TXSIM: target=%s duration=%s tx_size=%d concurrency=%d sent=%d ok=%d err=%d wall_s=%.2f sent_per_s=%.0f mb_per_s=%.2f rtt_p50_us=%d rtt_p99_us=%d\n",
		cli.target, cli.duration, cli.txSize, cli.concurrency,
		sent.Load(), ok.Load(), fail.Load(),
		elapsed.Seconds(), sentPerS, mbPerS,
		p50, p99,
	)
	return nil
}

func percentileMicros(sorted []int64, p float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}
