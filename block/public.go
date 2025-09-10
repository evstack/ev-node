package block

import "github.com/evstack/ev-node/block/internal/common"

// BlockOptions defines the options for creating block components
type BlockOptions = common.BlockOptions

// DefaultBlockOptions returns the default block options
func DefaultBlockOptions() BlockOptions {
	return common.DefaultBlockOptions()
}

// Expose Metrics for constructor
type Metrics = common.Metrics

// PrometheusMetrics creates a new PrometheusMetrics instance with the given namespace and labelsAndValues.
func PrometheusMetrics(namespace string, labelsAndValues ...string) *Metrics {
	return common.PrometheusMetrics(namespace, labelsAndValues...)
}

// NopMetrics creates a new NopMetrics instance.
func NopMetrics() *Metrics {
	return common.NopMetrics()
}
