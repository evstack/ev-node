//go:build evm

package benchmark

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	tastoradocker "github.com/celestiaorg/tastora/framework/docker"
	"github.com/celestiaorg/tastora/framework/docker/evstack/reth"
	"github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
	"github.com/celestiaorg/tastora/framework/docker/jaeger"
	tastoratypes "github.com/celestiaorg/tastora/framework/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"

	e2e "github.com/evstack/ev-node/test/e2e"
)

// SpamoorSuite groups benchmarks that use Spamoor for load generation
// and Jaeger for distributed tracing. Docker client and network are shared
// across all tests in the suite.
type SpamoorSuite struct {
	suite.Suite
	dockerCli tastoratypes.TastoraDockerClient
	networkID string
}

func TestSpamoorSuite(t *testing.T) {
	suite.Run(t, new(SpamoorSuite))
}

func (s *SpamoorSuite) SetupTest() {
	s.dockerCli, s.networkID = tastoradocker.Setup(s.T())
}

// env holds a fully-wired environment created by setupEnv.
type env struct {
	jaeger     *jaeger.Node
	evmEnv     *e2e.EVMEnv
	sut        *e2e.SystemUnderTest
	spamoorAPI *spamoor.API
	ethClient  *ethclient.Client
}

// config parameterizes the per-test environment setup.
type config struct {
	serviceName string
}

// TODO: temporary hardcoded tag, will be replaced with a proper release tag
const defaultRethTag = "pr-140"

func rethTag() string {
	if tag := os.Getenv("EV_RETH_TAG"); tag != "" {
		return tag
	}
	return defaultRethTag
}

// setupEnv creates a Jaeger + reth + sequencer + Spamoor environment for
// a single test. Each call spins up isolated infrastructure so tests
// can't interfere with each other.
func (s *SpamoorSuite) setupEnv(cfg config) *env {
	t := s.T()
	ctx := t.Context()
	sut := e2e.NewSystemUnderTest(t)

	// jaeger
	jcfg := jaeger.Config{Logger: zaptest.NewLogger(t), DockerClient: s.dockerCli, DockerNetworkID: s.networkID}
	jg, err := jaeger.New(ctx, jcfg, t.Name(), 0)
	s.Require().NoError(err, "failed to create jaeger node")
	t.Cleanup(func() { _ = jg.Remove(t.Context()) })
	s.Require().NoError(jg.Start(ctx), "failed to start jaeger node")

	// reth + local DA with OTLP tracing to Jaeger
	evmEnv := e2e.SetupCommonEVMEnv(t, sut, s.dockerCli, s.networkID,
		e2e.WithRethOpts(func(b *reth.NodeBuilder) {
			b.WithTag(rethTag()).
				// increase values to facilitate spamoor.
				WithAdditionalStartArgs(
					"--rpc.max-connections", "5000",
					"--rpc.max-tracing-requests", "1000",
				).
				WithEnv(
					// ev-reth reads OTEL_EXPORTER_OTLP_ENDPOINT and passes it directly
					// to with_endpoint(). opentelemetry-otlp v0.31 HTTP exporter does
					// not auto-append /v1/traces, so the full path is required.
					"OTEL_EXPORTER_OTLP_ENDPOINT="+jg.Internal.IngestHTTPEndpoint()+"/v1/traces",
					"OTEL_EXPORTER_OTLP_PROTOCOL=http",
					"RUST_LOG=debug",
					"OTEL_SDK_DISABLED=false",
				)
		}),
	)

	// sequencer with tracing
	sequencerHome := filepath.Join(t.TempDir(), "sequencer")
	otlpHTTP := jg.External.IngestHTTPEndpoint()
	e2e.SetupSequencerNode(t, sut, sequencerHome, evmEnv.SequencerJWT, evmEnv.GenesisHash, evmEnv.Endpoints,
		"--evnode.instrumentation.tracing=true",
		"--evnode.instrumentation.tracing_endpoint", otlpHTTP,
		// TODO: setting this to 1 produced too many spans for the local Jaeger deployment alongside everything else.
		"--evnode.instrumentation.tracing_sample_rate", "0.1",
		"--evnode.instrumentation.tracing_service_name", cfg.serviceName,
	)
	t.Log("sequencer node is up")

	// eth client
	ethClient, err := ethclient.Dial(evmEnv.Endpoints.GetSequencerEthURL())
	s.Require().NoError(err, "failed to dial sequencer eth endpoint")
	t.Cleanup(func() { ethClient.Close() })

	// spamoor
	ni, err := evmEnv.RethNode.GetNetworkInfo(ctx)
	s.Require().NoError(err, "failed to get reth network info")
	internalRPC := "http://" + ni.Internal.RPCAddress()

	spBuilder := spamoor.NewNodeBuilder(t.Name()).
		WithDockerClient(evmEnv.RethNode.DockerClient).
		WithDockerNetworkID(evmEnv.RethNode.NetworkID).
		WithLogger(evmEnv.RethNode.Logger).
		WithRPCHosts(internalRPC).
		WithPrivateKey(e2e.TestPrivateKey).
		WithAdditionalStartArgs("--slot-duration", "100ms", "--startup-delay", "0")

	spNode, err := spBuilder.Build(ctx)
	s.Require().NoError(err, "failed to build spamoor node")
	t.Cleanup(func() { _ = spNode.Remove(t.Context()) })
	s.Require().NoError(spNode.Start(ctx), "failed to start spamoor node")

	spInfo, err := spNode.GetNetworkInfo(ctx)
	s.Require().NoError(err, "failed to get spamoor network info")
	apiAddr := "http://127.0.0.1:" + spInfo.External.Ports.HTTP
	requireHostUp(t, apiAddr+"/api/spammers", 30*time.Second)

	return &env{
		jaeger:     jg,
		evmEnv:     evmEnv,
		sut:        sut,
		spamoorAPI: spNode.API(),
		ethClient:  ethClient,
	}
}

// traceResult holds the collected spans from ev-node and (optionally) ev-reth.
type traceResult struct {
	evNode []e2e.TraceSpan
	evReth []e2e.TraceSpan
}

// allSpans returns ev-node and ev-reth spans concatenated.
func (tr *traceResult) allSpans() []e2e.TraceSpan {
	return append(tr.evNode, tr.evReth...)
}

// collectTraces fetches ev-node traces (required) and ev-reth traces (optional)
// from Jaeger, then prints reports for both.
func (s *SpamoorSuite) collectTraces(e *env, serviceName string) *traceResult {
	t := s.T()
	tr := &traceResult{
		evNode: s.collectServiceTraces(e, serviceName),
		evReth: s.tryCollectServiceTraces(e, "ev-reth"),
	}
	e2e.PrintTraceReport(t, serviceName, tr.evNode)
	if len(tr.evReth) > 0 {
		e2e.PrintTraceReport(t, "ev-reth", tr.evReth)
	}
	return tr
}

// collectServiceTraces fetches traces from Jaeger for the given service and returns the spans.
func (s *SpamoorSuite) collectServiceTraces(e *env, serviceName string) []e2e.TraceSpan {
	ctx, cancel := context.WithTimeout(s.T().Context(), 3*time.Minute)
	defer cancel()

	ok, err := e.jaeger.External.WaitForTraces(ctx, serviceName, 1, 2*time.Second)
	s.Require().NoError(err, "error waiting for %s traces; UI: %s", serviceName, e.jaeger.External.QueryURL())
	s.Require().True(ok, "expected at least one trace from %s; UI: %s", serviceName, e.jaeger.External.QueryURL())

	traces, err := e.jaeger.External.Traces(ctx, serviceName, 10000)
	s.Require().NoError(err, "failed to fetch %s traces", serviceName)

	return toTraceSpans(extractSpansFromTraces(traces))
}

// tryCollectServiceTraces fetches traces from Jaeger for the given service,
// returning nil instead of failing the test if traces are unavailable.
func (s *SpamoorSuite) tryCollectServiceTraces(e *env, serviceName string) []e2e.TraceSpan {
	t := s.T()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	ok, err := e.jaeger.External.WaitForTraces(ctx, serviceName, 1, 2*time.Second)
	if err != nil || !ok {
		t.Logf("warning: could not collect %s traces (err=%v, ok=%v)", serviceName, err, ok)
		return nil
	}

	traces, err := e.jaeger.External.Traces(ctx, serviceName, 10000)
	if err != nil {
		t.Logf("warning: failed to fetch %s traces: %v", serviceName, err)
		return nil
	}

	spans := toTraceSpans(extractSpansFromTraces(traces))
	t.Logf("collected %d %s spans", len(spans), serviceName)
	return spans
}
