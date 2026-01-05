package block

import (
	"context"
	"time"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/block/internal/da"
	"github.com/evstack/ev-node/pkg/config"
	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
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
type DAClient = da.Client

// DAVerifier is the interface for DA proof verification operations.
type DAVerifier = da.Verifier

// FullDAClient combines DAClient and DAVerifier interfaces.
// This is the complete interface implemented by the concrete DA client.
type FullDAClient = da.FullClient

// NewDAClient creates a new DA client backed by the blob JSON-RPC API.
// The returned client implements both DAClient and DAVerifier interfaces.
func NewDAClient(
	blobRPC *blobrpc.Client,
	config config.Config,
	logger zerolog.Logger,
) FullDAClient {
	return da.NewClient(da.Config{
		DA:                       blobRPC,
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

// AsyncEpochFetcher provides background prefetching of DA epoch data
type AsyncEpochFetcher = da.AsyncEpochFetcher

// ForcedInclusionRetriever defines the interface for retrieving forced inclusion transactions from DA
type ForcedInclusionRetriever interface {
	RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*da.ForcedInclusionEvent, error)
}

// NewAsyncEpochFetcher creates a new async epoch fetcher for background prefetching.
// Parameters:
//   - client: DA client for fetching data
//   - logger: structured logger
//   - daStartHeight: genesis DA start height
//   - daEpochSize: number of DA blocks per epoch
//   - prefetchWindow: how many epochs ahead to prefetch (1-2 recommended)
//   - pollInterval: how often to check for new epochs to prefetch
func NewAsyncEpochFetcher(
	client DAClient,
	logger zerolog.Logger,
	daStartHeight, daEpochSize uint64,
	prefetchWindow uint64,
	pollInterval time.Duration,
) *AsyncEpochFetcher {
	return da.NewAsyncEpochFetcher(client, logger, daStartHeight, daEpochSize, prefetchWindow, pollInterval)
}

// NewForcedInclusionRetriever creates a new forced inclusion retriever.
// The asyncFetcher parameter is required for background prefetching of DA epoch data.
func NewForcedInclusionRetriever(
	client DAClient,
	logger zerolog.Logger,
	daStartHeight, daEpochSize uint64,
	asyncFetcher *AsyncEpochFetcher,
) ForcedInclusionRetriever {
	return da.NewForcedInclusionRetriever(client, logger, daStartHeight, daEpochSize, asyncFetcher)
}
