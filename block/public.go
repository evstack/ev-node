package block

import (
	"context"
	"time"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/block/internal/da"
	"github.com/evstack/ev-node/da/jsonrpc"
	"github.com/evstack/ev-node/pkg/config"
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

// NewDAClient creates a new DA client with configuration.
// It always dials the blob RPC endpoint configured in config.DA.
func NewDAClient(
	config config.Config,
	logger zerolog.Logger,
) DAClient {
	var (
		blobClient da.BlobAPI
		err        error
	)

	blobClient, err = jsonrpc.NewClient(context.Background(), logger, config.DA.Address, config.DA.AuthToken, common.DefaultMaxBlobSize)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to create blob jsonrpc client, falling back to local in-memory client")
		blobClient = da.NewLocalBlobAPI(common.DefaultMaxBlobSize)
	}

	return da.NewClient(da.Config{
		BlobAPI:        blobClient,
		Logger:         logger,
		DefaultTimeout: 10 * time.Second,
		Namespace:      config.DA.GetNamespace(),
		DataNamespace:  config.DA.GetDataNamespace(),
	})
}

// NewLocalDAClient creates a new DA client using an in-memory LocalBlobAPI.
// This is useful for tests where multiple nodes need to share the same DA state.
func NewLocalDAClient(
	config config.Config,
	logger zerolog.Logger,
) DAClient {
	blobClient := da.NewLocalBlobAPI(common.DefaultMaxBlobSize)
	return da.NewClient(da.Config{
		BlobAPI:        blobClient,
		Logger:         logger,
		DefaultTimeout: 10 * time.Second,
		Namespace:      config.DA.GetNamespace(),
		DataNamespace:  config.DA.GetDataNamespace(),
	})
}
