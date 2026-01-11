package common

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/discard"
	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	// MetricsSubsystem is a subsystem shared by all metrics exposed by this
	// package.
	MetricsSubsystem = "sequencer"
)

// DASubmitterFailureReason represents a typed failure reason for DA submission failures
type DASubmitterFailureReason string

const (
	DASubmitterFailureReasonAlreadyRejected    DASubmitterFailureReason = "already_rejected"
	DASubmitterFailureReasonInsufficientFee    DASubmitterFailureReason = "insufficient_fee"
	DASubmitterFailureReasonTimeout            DASubmitterFailureReason = "timeout"
	DASubmitterFailureReasonAlreadyInMempool   DASubmitterFailureReason = "already_in_mempool"
	DASubmitterFailureReasonNotIncludedInBlock DASubmitterFailureReason = "not_included_in_block"
	DASubmitterFailureReasonTooBig             DASubmitterFailureReason = "too_big"
	DASubmitterFailureReasonContextCanceled    DASubmitterFailureReason = "context_canceled"
	DASubmitterFailureReasonUnknown            DASubmitterFailureReason = "unknown"
)

// AllDASubmitterFailureReasons returns all possible failure reasons
func AllDASubmitterFailureReasons() []DASubmitterFailureReason {
	return []DASubmitterFailureReason{
		DASubmitterFailureReasonAlreadyRejected,
		DASubmitterFailureReasonInsufficientFee,
		DASubmitterFailureReasonTimeout,
		DASubmitterFailureReasonAlreadyInMempool,
		DASubmitterFailureReasonNotIncludedInBlock,
		DASubmitterFailureReasonTooBig,
		DASubmitterFailureReasonContextCanceled,
		DASubmitterFailureReasonUnknown,
	}
}

// Metrics contains all metrics exposed by this package.
type Metrics struct {
	// Original metrics
	Height          metrics.Gauge // Height of the chain
	NumTxs          metrics.Gauge // Number of transactions in the latest block
	BlockSizeBytes  metrics.Gauge // Size of the latest block
	TotalTxs        metrics.Gauge // Total number of transactions
	CommittedHeight metrics.Gauge `metrics_name:"latest_block_height"` // The latest block height
	TxsPerBlock     metrics.Histogram

	// Performance metrics
	OperationDuration map[string]metrics.Histogram

	// DA metrics
	DASubmitterFailures     map[DASubmitterFailureReason]metrics.Counter // Counter with reason label
	DASubmitterLastFailure  map[DASubmitterFailureReason]metrics.Gauge   // Timestamp gauge with reason label
	DASubmitterPendingBlobs metrics.Gauge                                // Total number of blobs awaiting submission (backlog)
	DASubmitterResends      metrics.Counter                              // Number of resend attempts
	DARetrievalAttempts     metrics.Counter
	DARetrievalSuccesses    metrics.Counter
	DARetrievalFailures     metrics.Counter
	DAInclusionHeight       metrics.Gauge
	PendingHeadersCount     metrics.Gauge
	PendingDataCount        metrics.Gauge

	// Forced inclusion metrics
	ForcedInclusionTxsInGracePeriod metrics.Gauge   // Number of forced inclusion txs currently in grace period
	ForcedInclusionTxsMalicious     metrics.Counter // Total number of forced inclusion txs marked as malicious

	// Sync mode metrics
	SyncMode        metrics.Gauge   // Current sync mode: 0=catchup, 1=follow
	SubscribeErrors metrics.Counter // Number of subscription failures
	ModeSwitches    metrics.Counter // Number of catchup<->follow mode transitions
}

// PrometheusMetrics returns Metrics built using Prometheus client library
func PrometheusMetrics(namespace string, labelsAndValues ...string) *Metrics {
	labels := []string{}
	for i := 0; i < len(labelsAndValues); i += 2 {
		labels = append(labels, labelsAndValues[i])
	}

	m := &Metrics{
		OperationDuration:      make(map[string]metrics.Histogram),
		DASubmitterFailures:    make(map[DASubmitterFailureReason]metrics.Counter),
		DASubmitterLastFailure: make(map[DASubmitterFailureReason]metrics.Gauge),
	}

	// Original metrics
	m.Height = prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: MetricsSubsystem,
		Name:      "height",
		Help:      "Height of the chain.",
	}, labels).With(labelsAndValues...)

	m.NumTxs = prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: MetricsSubsystem,
		Name:      "num_txs",
		Help:      "Number of transactions.",
	}, labels).With(labelsAndValues...)

	m.BlockSizeBytes = prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: MetricsSubsystem,
		Name:      "block_size_bytes",
		Help:      "Size of the block.",
	}, labels).With(labelsAndValues...)

	m.TotalTxs = prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: MetricsSubsystem,
		Name:      "total_txs",
		Help:      "Total number of transactions.",
	}, labels).With(labelsAndValues...)

	m.CommittedHeight = prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: MetricsSubsystem,
		Name:      "latest_block_height",
		Help:      "The latest block height.",
	}, labels).With(labelsAndValues...)

	m.TxsPerBlock = prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: MetricsSubsystem,
		Name:      "txs_per_block",
		Help:      "Number of transactions per block",
		Buckets:   []float64{0, 1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
	}, labels).With(labelsAndValues...)

	// Initialize operation duration histograms
	operations := []string{"block_production", "da_submission", "block_retrieval", "block_validation", "state_update"}
	for _, op := range operations {
		m.OperationDuration[op] = prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "operation_duration_seconds",
			Help:      "Duration of operations in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			ConstLabels: map[string]string{
				"operation": op,
			},
		}, labels).With(labelsAndValues...)
	}

	// DA metrics
	m.DARetrievalAttempts = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: MetricsSubsystem,
		Name:      "da_retrieval_attempts_total",
		Help:      "Total number of DA retrieval attempts",
	}, labels).With(labelsAndValues...)

	m.DARetrievalSuccesses = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: MetricsSubsystem,
		Name:      "da_retrieval_successes_total",
		Help:      "Total number of successful DA retrievals",
	}, labels).With(labelsAndValues...)

	m.DARetrievalFailures = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: MetricsSubsystem,
		Name:      "da_retrieval_failures_total",
		Help:      "Total number of failed DA retrievals",
	}, labels).With(labelsAndValues...)

	m.DAInclusionHeight = prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: MetricsSubsystem,
		Name:      "da_inclusion_height",
		Help:      "Height at which all blocks have been included in DA",
	}, labels).With(labelsAndValues...)

	m.PendingHeadersCount = prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: MetricsSubsystem,
		Name:      "pending_headers_count",
		Help:      "Number of headers pending DA submission",
	}, labels).With(labelsAndValues...)

	m.PendingDataCount = prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: MetricsSubsystem,
		Name:      "pending_data_count",
		Help:      "Number of data blocks pending DA submission",
	}, labels).With(labelsAndValues...)

	// Forced inclusion metrics
	m.ForcedInclusionTxsInGracePeriod = prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: MetricsSubsystem,
		Name:      "forced_inclusion_txs_in_grace_period",
		Help:      "Number of forced inclusion transactions currently in grace period (past epoch end but within grace boundary)",
	}, labels).With(labelsAndValues...)

	m.ForcedInclusionTxsMalicious = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: MetricsSubsystem,
		Name:      "forced_inclusion_txs_malicious_total",
		Help:      "Total number of forced inclusion transactions marked as malicious (past grace boundary)",
	}, labels).With(labelsAndValues...)

	// Sync mode metrics
	m.SyncMode = prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: MetricsSubsystem,
		Name:      "sync_mode",
		Help:      "Current sync mode: 0=catchup (polling), 1=follow (subscription)",
	}, labels).With(labelsAndValues...)

	m.SubscribeErrors = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: MetricsSubsystem,
		Name:      "subscribe_errors_total",
		Help:      "Total number of DA subscription failures",
	}, labels).With(labelsAndValues...)

	m.ModeSwitches = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: MetricsSubsystem,
		Name:      "mode_switches_total",
		Help:      "Total number of sync mode transitions between catchup and follow",
	}, labels).With(labelsAndValues...)

	// DA Submitter metrics
	m.DASubmitterPendingBlobs = prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: MetricsSubsystem,
		Name:      "da_submitter_pending_blobs",
		Help:      "Total number of blobs awaiting DA submission (backlog)",
	}, labels).With(labelsAndValues...)

	m.DASubmitterResends = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: MetricsSubsystem,
		Name:      "da_submitter_resends_total",
		Help:      "Total number of DA submission retry attempts",
	}, labels).With(labelsAndValues...)

	// Initialize DA submitter failure counters and timestamps for various reasons
	for _, reason := range AllDASubmitterFailureReasons() {
		m.DASubmitterFailures[reason] = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "da_submitter_failures_total",
			Help:      "Total number of DA submission failures by reason",
			ConstLabels: map[string]string{
				"reason": string(reason),
			},
		}, labels).With(labelsAndValues...)

		m.DASubmitterLastFailure[reason] = prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "da_submitter_last_failure_timestamp",
			Help:      "Unix timestamp of the last DA submission failure by reason",
			ConstLabels: map[string]string{
				"reason": string(reason),
			},
		}, labels).With(labelsAndValues...)
	}

	return m
}

// NopMetrics returns no-op Metrics
func NopMetrics() *Metrics {
	m := &Metrics{
		// Original metrics
		Height:          discard.NewGauge(),
		NumTxs:          discard.NewGauge(),
		BlockSizeBytes:  discard.NewGauge(),
		TotalTxs:        discard.NewGauge(),
		CommittedHeight: discard.NewGauge(),
		TxsPerBlock:     discard.NewHistogram(),

		// Extended metrics
		OperationDuration:       make(map[string]metrics.Histogram),
		DARetrievalAttempts:     discard.NewCounter(),
		DARetrievalSuccesses:    discard.NewCounter(),
		DARetrievalFailures:     discard.NewCounter(),
		DAInclusionHeight:       discard.NewGauge(),
		PendingHeadersCount:     discard.NewGauge(),
		PendingDataCount:        discard.NewGauge(),
		DASubmitterFailures:     make(map[DASubmitterFailureReason]metrics.Counter),
		DASubmitterLastFailure:  make(map[DASubmitterFailureReason]metrics.Gauge),
		DASubmitterPendingBlobs: discard.NewGauge(),
		DASubmitterResends:      discard.NewCounter(),

		// Forced inclusion metrics
		ForcedInclusionTxsInGracePeriod: discard.NewGauge(),
		ForcedInclusionTxsMalicious:     discard.NewCounter(),

		// Sync mode metrics
		SyncMode:        discard.NewGauge(),
		SubscribeErrors: discard.NewCounter(),
		ModeSwitches:    discard.NewCounter(),
	}

	// Initialize maps with no-op metrics
	operations := []string{"block_production", "da_submission", "block_retrieval", "block_validation", "state_update"}
	for _, op := range operations {
		m.OperationDuration[op] = discard.NewHistogram()
	}

	// Initialize DA submitter failure maps with no-op metrics
	for _, reason := range AllDASubmitterFailureReasons() {
		m.DASubmitterFailures[reason] = discard.NewCounter()
		m.DASubmitterLastFailure[reason] = discard.NewGauge()
	}

	return m
}
