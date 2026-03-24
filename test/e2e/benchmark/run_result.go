//go:build evm

package benchmark

import (
	"encoding/json"
	"os"
	"runtime"
	"runtime/debug"
	"testing"
	"time"

	e2e "github.com/evstack/ev-node/test/e2e"
)

// runResult is the structured output of a single benchmark run.
// written to the path specified by BENCH_RESULT_OUTPUT.
type runResult struct {
	SchemaVersion     int               `json:"schema_version"`
	Test              string            `json:"test"`
	Objective         string            `json:"objective,omitempty"`
	Timestamp         string            `json:"timestamp"`
	Platform          string            `json:"platform"`
	Config            runConfig         `json:"config"`
	Tags              runTags           `json:"tags"`
	Metrics           runMetrics        `json:"metrics"`
	BlockRange        runBlockRange     `json:"block_range"`
	Spamoor           *runSpamoorStats  `json:"spamoor,omitempty"`
	FieldDescriptions map[string]string `json:"field_descriptions"`
}

type runConfig struct {
	BlockTime       string `json:"block_time"`
	SlotDuration    string `json:"slot_duration"`
	GasLimit        string `json:"gas_limit"`
	ScrapeInterval  string `json:"scrape_interval"`
	NumSpammers     int    `json:"num_spammers"`
	CountPerSpammer int    `json:"count_per_spammer"`
	Throughput      int    `json:"throughput"`
	WarmupTxs       int    `json:"warmup_txs"`
	GasUnitsToBurn  int    `json:"gas_units_to_burn"`
	MaxWallets      int    `json:"max_wallets"`
	WaitTimeout     string `json:"wait_timeout"`
}

type runTags struct {
	EvReth string `json:"ev_reth"`
	EvNode string `json:"ev_node"`
}

type runMetrics struct {
	// throughput
	MGasPerSec   float64 `json:"mgas_per_sec"`
	TPS          float64 `json:"tps"`
	BlocksPerSec float64 `json:"blocks_per_sec"`

	// block fill
	NonEmptyRatioPct float64 `json:"non_empty_ratio_pct"`
	AvgGasPerBlock   float64 `json:"avg_gas_per_block"`
	AvgTxPerBlock    float64 `json:"avg_tx_per_block"`
	GasBlockP50      float64 `json:"gas_block_p50"`
	GasBlockP99      float64 `json:"gas_block_p99"`
	TxBlockP50       float64 `json:"tx_block_p50"`
	TxBlockP99       float64 `json:"tx_block_p99"`

	// block intervals
	AvgBlockIntervalMs float64 `json:"avg_block_interval_ms"`
	BlockIntervalP50Ms float64 `json:"block_interval_p50_ms"`
	BlockIntervalP99Ms float64 `json:"block_interval_p99_ms"`

	// duration
	SteadyStateSec float64 `json:"steady_state_sec"`
	WallClockSec   float64 `json:"wall_clock_sec"`

	// trace-derived (optional, omitted when tracing is unavailable)
	OverheadPct      *float64 `json:"overhead_pct,omitempty"`
	EvRethGGasPerSec *float64 `json:"ev_reth_ggas_per_sec,omitempty"`
	SecsPerGigagas   *float64 `json:"secs_per_gigagas,omitempty"`

	// engine span timings (optional)
	ProduceBlockAvgMs *float64 `json:"produce_block_avg_ms,omitempty"`
	ProduceBlockMinMs *float64 `json:"produce_block_min_ms,omitempty"`
	ProduceBlockMaxMs *float64 `json:"produce_block_max_ms,omitempty"`
	GetPayloadAvgMs   *float64 `json:"get_payload_avg_ms,omitempty"`
	GetPayloadMinMs   *float64 `json:"get_payload_min_ms,omitempty"`
	GetPayloadMaxMs   *float64 `json:"get_payload_max_ms,omitempty"`
	NewPayloadAvgMs   *float64 `json:"new_payload_avg_ms,omitempty"`
	NewPayloadMinMs   *float64 `json:"new_payload_min_ms,omitempty"`
	NewPayloadMaxMs   *float64 `json:"new_payload_max_ms,omitempty"`
}

type runBlockRange struct {
	Start    uint64 `json:"start"`
	End      uint64 `json:"end"`
	Total    int    `json:"total"`
	NonEmpty int    `json:"non_empty"`
}

type runSpamoorStats struct {
	Sent   float64 `json:"sent"`
	Failed float64 `json:"failed"`
}

// emitRunResult builds a structured result from the benchmark data and writes it
// to the path in BENCH_RESULT_OUTPUT. no-op when the env var is unset.
func emitRunResult(t testing.TB, cfg benchConfig, br *benchmarkResult, wallClock time.Duration, spamoor *runSpamoorStats) {
	outputPath := os.Getenv("BENCH_RESULT_OUTPUT")
	if outputPath == "" {
		return
	}

	rr := buildRunResult(cfg, br, wallClock, spamoor)

	data, err := json.MarshalIndent(rr, "", "  ")
	if err != nil {
		t.Logf("WARNING: failed to marshal run result: %v", err)
		return
	}
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		t.Logf("WARNING: failed to write run result to %s: %v", outputPath, err)
		return
	}
	t.Logf("wrote structured result to %s", outputPath)
}

func buildRunResult(cfg benchConfig, br *benchmarkResult, wallClock time.Duration, spamoor *runSpamoorStats) *runResult {
	s := br.summary

	// compute span stats once for all trace-derived metrics
	stats := e2e.AggregateSpanStats(br.traces.evNode)

	m := runMetrics{
		MGasPerSec:         s.AchievedMGas,
		TPS:                s.AchievedTPS,
		BlocksPerSec:       s.BlocksPerSec,
		NonEmptyRatioPct:   s.NonEmptyRatio,
		AvgGasPerBlock:     s.AvgGas,
		AvgTxPerBlock:      s.AvgTx,
		GasBlockP50:        s.GasP50,
		GasBlockP99:        s.GasP99,
		TxBlockP50:         s.TxP50,
		TxBlockP99:         s.TxP99,
		AvgBlockIntervalMs: float64(s.AvgBlockInterval.Milliseconds()),
		BlockIntervalP50Ms: float64(s.IntervalP50.Milliseconds()),
		BlockIntervalP99Ms: float64(s.IntervalP99.Milliseconds()),
		SteadyStateSec:     s.SteadyState.Seconds(),
		WallClockSec:       wallClock.Seconds(),
	}

	if overhead, ok := overheadFromStats(stats); ok {
		m.OverheadPct = &overhead
	}
	if ggas, ok := rethRateFromStats(stats, br.bm.TotalGasUsed); ok {
		m.EvRethGGasPerSec = &ggas
	}
	if s.AchievedMGas > 0 {
		v := 1000.0 / s.AchievedMGas
		m.SecsPerGigagas = &v
	}

	setEngineSpanTimings(&m, stats)

	return &runResult{
		SchemaVersion: 1,
		Test:          br.prefix,
		Objective:     os.Getenv("BENCH_OBJECTIVE"),
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Platform:      envOrDefault("BENCH_PLATFORM", runtime.GOOS+"/"+runtime.GOARCH),
		Config: runConfig{
			BlockTime:       cfg.BlockTime,
			SlotDuration:    cfg.SlotDuration,
			GasLimit:        cfg.GasLimit,
			ScrapeInterval:  cfg.ScrapeInterval,
			NumSpammers:     cfg.NumSpammers,
			CountPerSpammer: cfg.CountPerSpammer,
			Throughput:      cfg.Throughput,
			WarmupTxs:       cfg.WarmupTxs,
			GasUnitsToBurn:  cfg.GasUnitsToBurn,
			MaxWallets:      cfg.MaxWallets,
			WaitTimeout:     cfg.WaitTimeout.String(),
		},
		Tags: runTags{
			EvReth: envOrDefault("EV_RETH_TAG", "latest"),
			EvNode: evNodeTag(),
		},
		Metrics: m,
		BlockRange: runBlockRange{
			Start:    br.bm.StartBlock,
			End:      br.bm.EndBlock,
			Total:    br.bm.TotalBlockCount,
			NonEmpty: br.bm.BlockCount,
		},
		Spamoor:           spamoor,
		FieldDescriptions: fieldDescriptions(),
	}
}

func setEngineSpanTimings(m *runMetrics, stats map[string]*e2e.SpanStats) {
	type spanTarget struct {
		name          string
		avg, min, max **float64
	}
	targets := []spanTarget{
		{spanProduceBlock, &m.ProduceBlockAvgMs, &m.ProduceBlockMinMs, &m.ProduceBlockMaxMs},
		{spanGetPayload, &m.GetPayloadAvgMs, &m.GetPayloadMinMs, &m.GetPayloadMaxMs},
		{spanNewPayload, &m.NewPayloadAvgMs, &m.NewPayloadMinMs, &m.NewPayloadMaxMs},
	}
	for _, target := range targets {
		s, ok := stats[target.name]
		if !ok || s.Count == 0 {
			continue
		}
		avg := float64(s.Total.Microseconds()) / float64(s.Count) / 1000.0
		min := float64(s.Min.Microseconds()) / 1000.0
		max := float64(s.Max.Microseconds()) / 1000.0
		*target.avg = &avg
		*target.min = &min
		*target.max = &max
	}
}

func evNodeTag() string {
	if tag := os.Getenv("EV_NODE_TAG"); tag != "" {
		return tag
	}
	info, ok := debug.ReadBuildInfo()
	if ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				if len(s.Value) > 8 {
					return s.Value[:8]
				}
				return s.Value
			}
		}
	}
	return "unknown"
}

func fieldDescriptions() map[string]string {
	return map[string]string{
		"config.block_time":        "target block production interval",
		"config.slot_duration":     "spamoor slot cadence for tx injection",
		"config.gas_limit":         "max gas per block (genesis config, hex string)",
		"config.scrape_interval":   "sequencer metrics scrape interval",
		"config.num_spammers":      "parallel spamoor instances injecting load",
		"config.count_per_spammer": "total txs each spammer sends",
		"config.throughput":        "target tx/s per spammer",
		"config.warmup_txs":        "txs sent before measurement window starts",
		"config.gas_units_to_burn": "gas consumed per tx (gasburner/state-pressure only)",
		"config.max_wallets":       "sender wallets per spammer",
		"config.wait_timeout":      "max time to wait for all txs to be sent",

		"tags.ev_reth": "ev-reth docker image tag",
		"tags.ev_node": "ev-node git commit or tag",

		"metrics.mgas_per_sec":          "total gas / steady-state seconds / 1e6",
		"metrics.tps":                   "total tx count / steady-state seconds",
		"metrics.blocks_per_sec":        "non-empty blocks / steady-state seconds",
		"metrics.non_empty_ratio_pct":   "percentage of blocks containing >= 1 tx",
		"metrics.avg_gas_per_block":     "mean gas per non-empty block",
		"metrics.avg_tx_per_block":      "mean tx count per non-empty block",
		"metrics.gas_block_p50":         "50th percentile gas per non-empty block",
		"metrics.gas_block_p99":         "99th percentile gas per non-empty block",
		"metrics.tx_block_p50":          "50th percentile tx count per non-empty block",
		"metrics.tx_block_p99":          "99th percentile tx count per non-empty block",
		"metrics.avg_block_interval_ms": "mean time between consecutive blocks (all blocks)",
		"metrics.block_interval_p50_ms": "50th percentile block interval",
		"metrics.block_interval_p99_ms": "99th percentile block interval",
		"metrics.steady_state_sec":      "wall-clock seconds between first and last non-empty block",
		"metrics.wall_clock_sec":        "total elapsed seconds including warmup and drain",
		"metrics.overhead_pct":          "(ProduceBlock - ExecuteTxs) / ProduceBlock — ev-node orchestration cost",
		"metrics.ev_reth_ggas_per_sec":  "total gas / cumulative NewPayload time / 1e9",
		"metrics.secs_per_gigagas":      "inverse of mgas_per_sec scaled to gigagas",
		"metrics.produce_block_avg_ms":  "avg BlockExecutor.ProduceBlock span (full block lifecycle)",
		"metrics.produce_block_min_ms":  "min ProduceBlock span",
		"metrics.produce_block_max_ms":  "max ProduceBlock span",
		"metrics.get_payload_avg_ms":    "avg Engine.GetPayload span (reth block building)",
		"metrics.get_payload_min_ms":    "min GetPayload span",
		"metrics.get_payload_max_ms":    "max GetPayload span",
		"metrics.new_payload_avg_ms":    "avg Engine.NewPayload span (reth validation + state commit)",
		"metrics.new_payload_min_ms":    "min NewPayload span",
		"metrics.new_payload_max_ms":    "max NewPayload span",

		"block_range.start":     "first block in measurement window",
		"block_range.end":       "last block in measurement window",
		"block_range.total":     "all blocks in range including empty",
		"block_range.non_empty": "blocks containing at least one tx",

		"spamoor.sent":   "total txs successfully sent by spamoor",
		"spamoor.failed": "total txs that failed",
	}
}
