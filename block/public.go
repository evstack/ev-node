package block

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/block/internal/da"
	"github.com/evstack/ev-node/pkg/config"
	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
)

type BlockOptions = common.BlockOptions

func DefaultBlockOptions() BlockOptions {
	return common.DefaultBlockOptions()
}

func SetMaxBlobSize(n uint64) {
	common.DefaultMaxBlobSize = n
}

type Metrics = common.Metrics

func PrometheusMetrics(namespace string, labelsAndValues ...string) *Metrics {
	return common.PrometheusMetrics(namespace, labelsAndValues...)
}

func NopMetrics() *Metrics {
	return common.NopMetrics()
}

type DAClient = da.Client

type DAVerifier = da.Verifier

type FullDAClient = da.FullClient

type FiberClient = da.FiberClient

type (
	FiberBlobID       = da.BlobID
	FiberUploadResult = da.UploadResult
	FiberBlobEvent    = da.BlobEvent
)

func NewDAClient(
	blobRPC *blobrpc.Client,
	config config.Config,
	logger zerolog.Logger,
) FullDAClient {
	base := da.NewClient(da.Config{
		DA:                       blobRPC,
		Logger:                   logger,
		Namespace:                config.DA.GetNamespace(),
		DefaultTimeout:           config.DA.RequestTimeout.Duration,
		DataNamespace:            config.DA.GetDataNamespace(),
		ForcedInclusionNamespace: config.DA.GetForcedInclusionNamespace(),
	})
	if config.Instrumentation.IsTracingEnabled() {
		return da.WithTracingClient(base)
	}
	return base
}

func NewFiberDAClient(
	fiberClient da.FiberClient,
	config config.Config,
	logger zerolog.Logger,
	lastKnownDaHeight uint64,
) FullDAClient {
	base, err := da.NewFiberClient(da.FiberConfig{
		Client:            fiberClient,
		Logger:            logger,
		DefaultTimeout:    config.DA.RequestTimeout.Duration,
		Namespace:         config.DA.GetNamespace(),
		DataNamespace:     config.DA.GetDataNamespace(),
		LastKnownDAHeight: lastKnownDaHeight,
	})
	if err != nil {
		panic(err)
	}

	if config.Instrumentation.IsTracingEnabled() {
		return da.WithTracingClient(base)
	}

	return base
}

var (
	ErrForceInclusionNotConfigured = da.ErrForceInclusionNotConfigured
	ErrNoBatch                     = common.ErrNoBatch
)

type ForcedInclusionEvent = da.ForcedInclusionEvent

type ForcedInclusionRetriever interface {
	RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error)
	Stop()
	Start(ctx context.Context)
}

func NewForcedInclusionRetriever(
	client DAClient,
	cfg config.Config,
	logger zerolog.Logger,
	daStartHeight, daEpochSize uint64,
) ForcedInclusionRetriever {
	return da.NewForcedInclusionRetriever(
		client,
		logger,
		cfg.DA.BlockTime.Duration,
		cfg.Instrumentation.IsTracingEnabled(),
		daStartHeight,
		daEpochSize,
	)
}

type RaftNode = common.RaftNode
