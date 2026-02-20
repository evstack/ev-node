package block

import (
	"context"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/block/internal/da"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/rs/zerolog"

	datypes "github.com/evstack/ev-node/pkg/da/types"
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

// DAClient is the interface representing the DA client for public use.
type DAClient = da.Client

// NewDAClient creates a new DA client backed by the provided BlobClient implementation.
func NewDAClient(
	client datypes.BlobClient,
	config config.Config,
	logger zerolog.Logger,
) DAClient {
	base := da.NewClient(da.Config{
		Client:                   client,
		Logger:                   logger,
		Namespace:                config.DA.GetNamespace(),
		DataNamespace:            config.DA.GetDataNamespace(),
		ForcedInclusionNamespace: config.DA.GetForcedInclusionNamespace(),
	})
	if config.Instrumentation.IsTracingEnabled() {
		return da.WithTracingClient(base)
	}
	return base
}

// ErrForceInclusionNotConfigured is returned when force inclusion is not configured.
// It is exported because sequencers needs to check for this error.
var ErrForceInclusionNotConfigured = da.ErrForceInclusionNotConfigured

// ForcedInclusionEvent represents forced inclusion transactions retrieved from DA
type ForcedInclusionEvent = da.ForcedInclusionEvent

// ForcedInclusionRetriever defines the interface for retrieving forced inclusion transactions from DA
type ForcedInclusionRetriever interface {
	RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error)
	Stop()
}

// NewForcedInclusionRetriever creates a new forced inclusion retriever.
// It internally creates and manages an AsyncBlockRetriever for background prefetching.
// Tracing is automatically enabled when configured.
func NewForcedInclusionRetriever(
	client DAClient,
	cfg config.Config,
	logger zerolog.Logger,
	daStartHeight, daEpochSize uint64,
) ForcedInclusionRetriever {
	return da.NewForcedInclusionRetriever(client, logger, cfg, daStartHeight, daEpochSize)
}

// Expose Raft types for consensus integration
type RaftNode = common.RaftNode
