package block

import (
	"fmt"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/block/internal/syncing"
	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/rs/zerolog"
)

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

// NewDaRetriver creates a new DA retriever instance.
func NewDARetriever(
	da coreda.DA,
	config config.Config,
	genesis genesis.Genesis,
	logger zerolog.Logger,
) (syncing.DARetriever, error) {
	cacheManager, err := cache.NewManager(config, nil /* pending data not used */, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache manager: %w", err)
	}

	return syncing.NewDARetriever(
		da,
		cacheManager,
		config,
		genesis,
		logger,
	), nil
}
