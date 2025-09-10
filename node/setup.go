package node

import (
	"time"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/p2p"
)

const readHeaderTimeout = 10 * time.Second

// MetricsProvider returns p2p Metrics.
type MetricsProvider func(chainID string) *p2p.Metrics

// DefaultMetricsProvider returns Metrics build using Prometheus client library
// if Prometheus is enabled. Otherwise, it returns no-op Metrics.
func DefaultMetricsProvider(config *config.InstrumentationConfig) MetricsProvider {
	return func(chainID string) *p2p.Metrics {
		if config.Prometheus {
			return p2p.PrometheusMetrics(config.Namespace, "chain_id", chainID)
		}
		return p2p.NopMetrics()
	}
}
