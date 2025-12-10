package block

import (
	"context"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/block/internal/da"
	"github.com/evstack/ev-node/pkg/config"
	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
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

// DAClient is the interface representing the DA client for public use.
type DAClient = da.Interface

// NewDAClient creates a new DA client backed by the Celestia blob JSON-RPC API.
func NewDAClient(
	blobRPC *blobrpc.Client,
	config config.Config,
	logger zerolog.Logger,
) DAClient {
	return da.NewClient(da.Config{
		Client:                   blobRPC,
		Logger:                   logger,
		Namespace:                config.DA.GetNamespace(),
		DefaultTimeout:           config.DA.RequestTimeout.Duration,
		DataNamespace:            config.DA.GetDataNamespace(),
		ForcedInclusionNamespace: config.DA.GetForcedInclusionNamespace(),
	})
}

// ErrForceInclusionNotConfigured is returned when force inclusion is not configured.
// It is exported because sequencers needs to check for this error.
var ErrForceInclusionNotConfigured = da.ErrForceInclusionNotConfigured

// ForcedInclusionEvent represents forced inclusion transactions retrieved from DA
type ForcedInclusionEvent = da.ForcedInclusionEvent

// ForcedInclusionRetriever defines the interface for retrieving forced inclusion transactions from DA
type ForcedInclusionRetriever interface {
	RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*da.ForcedInclusionEvent, error)
}

// NewForcedInclusionRetriever creates a new forced inclusion retriever
func NewForcedInclusionRetriever(
	client DAClient,
	genesis genesis.Genesis,
	logger zerolog.Logger,
) ForcedInclusionRetriever {
	return da.NewForcedInclusionRetriever(client, genesis, logger)
}
