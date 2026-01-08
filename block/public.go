package block

import (
	"context"

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

// AsyncBlockRetriever provides background prefetching of individual DA blocks
type AsyncBlockRetriever interface {
	Start()
	Stop()
	GetCachedBlock(ctx context.Context, daHeight uint64) (*da.BlockData, error)
	UpdateCurrentHeight(height uint64)
}

// ForcedInclusionRetriever defines the interface for retrieving forced inclusion transactions from DA
type ForcedInclusionRetriever interface {
	RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*da.ForcedInclusionEvent, error)
}

// NewAsyncBlockRetriever creates a new async block retriever for background prefetching.
// Parameters:
//   - client: DA client for fetching data
//   - config: Ev-node config
//   - logger: structured logger
//   - daStartHeight: genesis DA start height
//   - prefetchWindow: how many blocks ahead to prefetch (10-20 recommended)
func NewAsyncBlockRetriever(
	client DAClient,
	cfg config.Config,
	logger zerolog.Logger,
	daStartHeight uint64,
	prefetchWindow uint64,
) AsyncBlockRetriever {
	return da.NewAsyncBlockRetriever(client, logger, cfg, daStartHeight, prefetchWindow)
}

// NewForcedInclusionRetriever creates a new forced inclusion retriever.
// The asyncFetcher parameter is required for background prefetching of DA block data.
// It accepts either AsyncBlockRetriever (recommended) or AsyncEpochFetcher (deprecated) for backward compatibility.
func NewForcedInclusionRetriever(
	client DAClient,
	logger zerolog.Logger,
	daStartHeight, daEpochSize uint64,
	asyncFetcher AsyncBlockRetriever,
) ForcedInclusionRetriever {
	return da.NewForcedInclusionRetriever(client, logger, daStartHeight, daEpochSize, asyncFetcher)
}
