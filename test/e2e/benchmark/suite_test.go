//go:build evm

package benchmark

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tastoradocker "github.com/celestiaorg/tastora/framework/docker"
	"github.com/celestiaorg/tastora/framework/docker/evstack/reth"
	"github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
	"github.com/celestiaorg/tastora/framework/docker/victoriatraces"
	"github.com/celestiaorg/tastora/framework/testutil/maps"
	tastoratypes "github.com/celestiaorg/tastora/framework/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"

	e2e "github.com/evstack/ev-node/test/e2e"
)

// SpamoorSuite groups benchmarks that use Spamoor for load generation
// and VictoriaTraces for distributed tracing. Docker client and network are shared
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
	traces             traceProvider
	spamoorAPI         *spamoor.API
	ethClient          *ethclient.Client
	evNodeServiceName  string
	evRethServiceName  string
}

// TODO: temporary hardcoded tag, will be replaced with a proper release tag
const defaultRethTag = "latest"

func rethTag() string {
	if tag := os.Getenv("EV_RETH_TAG"); tag != "" {
		return tag
	}
	return defaultRethTag
}

// setupEnv dispatches to either setupExternalEnv or setupLocalEnv based on
// the BENCH_ETH_RPC_URL environment variable.
func (s *SpamoorSuite) setupEnv(cfg benchConfig) *env {
	if rpcURL := os.Getenv("BENCH_ETH_RPC_URL"); rpcURL != "" {
		return s.setupExternalEnv(cfg, rpcURL)
	}
	return s.setupLocalEnv(cfg)
}

// setupLocalEnv creates a VictoriaTraces + reth + sequencer + Spamoor environment
// for a single test. Each call spins up isolated infrastructure so tests
// can't interfere with each other.
func (s *SpamoorSuite) setupLocalEnv(cfg benchConfig) *env {
	t := s.T()
	ctx := t.Context()
	sut := e2e.NewSystemUnderTest(t)

	// victoriatraces
	vtCfg := victoriatraces.Config{Logger: zaptest.NewLogger(t), DockerClient: s.dockerCli, DockerNetworkID: s.networkID}
	vt, err := victoriatraces.New(ctx, vtCfg, t.Name(), 0)
	s.Require().NoError(err, "failed to create victoriatraces node")
	t.Cleanup(func() { _ = vt.Remove(t.Context()) })
	s.Require().NoError(vt.Start(ctx), "failed to start victoriatraces node")

	// reth + local DA with OTLP tracing to VictoriaTraces
	evmEnv := e2e.SetupCommonEVMEnv(t, sut, s.dockerCli, s.networkID,
		e2e.WithRethOpts(func(b *reth.NodeBuilder) {
			b.WithTag(rethTag()).
				WithAdditionalStartArgs(
					"--rpc.max-connections", "5000",
					"--rpc.max-tracing-requests", "1000",
				).
				WithEnv(
					// ev-reth's Rust OTLP exporter auto-appends /v1/traces, so give it
					// only the base path (/insert/opentelemetry).
					"OTEL_EXPORTER_OTLP_ENDPOINT="+vt.Internal.OTLPBaseEndpoint(),
					"OTEL_EXPORTER_OTLP_PROTOCOL=http",
					"RUST_LOG=debug",
					"OTEL_SDK_DISABLED=false",
				)
			if cfg.GasLimit != "" {
				genesis := reth.DefaultEvolveGenesisJSON(func(bz []byte) ([]byte, error) {
					return maps.SetField(bz, "gasLimit", cfg.GasLimit)
				})
				b.WithGenesis([]byte(genesis))
			}
		}),
	)

	// sequencer with tracing
	sequencerHome := filepath.Join(t.TempDir(), "sequencer")
	otlpHTTP := vt.External.IngestHTTPEndpoint()
	e2e.SetupSequencerNode(t, sut, sequencerHome, evmEnv.SequencerJWT, evmEnv.GenesisHash, evmEnv.Endpoints,
		"--evnode.instrumentation.tracing=true",
		"--evnode.instrumentation.tracing_endpoint", otlpHTTP,
		"--evnode.instrumentation.tracing_sample_rate", "1.0",
		"--evnode.instrumentation.tracing_service_name", cfg.ServiceName,
		"--evnode.node.block_time", cfg.BlockTime,
		"--evnode.node.scrape_interval", cfg.ScrapeInterval,
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
		WithAdditionalStartArgs("--slot-duration", cfg.SlotDuration, "--startup-delay", "0")

	spNode, err := spBuilder.Build(ctx)
	s.Require().NoError(err, "failed to build spamoor node")
	t.Cleanup(func() { _ = spNode.Remove(t.Context()) })
	s.Require().NoError(spNode.Start(ctx), "failed to start spamoor node")

	spInfo, err := spNode.GetNetworkInfo(ctx)
	s.Require().NoError(err, "failed to get spamoor network info")
	apiAddr := "http://127.0.0.1:" + spInfo.External.Ports.HTTP
	requireHostUp(t, apiAddr+"/api/spammers", 30*time.Second)

	return &env{
		traces: &victoriaTraceProvider{
			queryURL:  vt.External.QueryURL(),
			t:         t,
			startTime: time.Now(),
		},
		spamoorAPI:        spNode.API(),
		ethClient:         ethClient,
		evNodeServiceName: cfg.ServiceName,
		evRethServiceName: "ev-reth",
	}
}

// setupExternalEnv connects to pre-deployed infrastructure identified by
// rpcURL. Only spamoor is provisioned by the test; reth, DA, and sequencer
// are assumed to already be running.
func (s *SpamoorSuite) setupExternalEnv(cfg benchConfig, rpcURL string) *env {
	t := s.T()
	ctx := t.Context()

	t.Logf("external mode: using RPC %s", rpcURL)

	privateKey := os.Getenv("BENCH_PRIVATE_KEY")
	s.Require().NotEmpty(privateKey, "BENCH_PRIVATE_KEY must be set in external mode")

	// eth client
	ethClient, err := ethclient.Dial(rpcURL)
	s.Require().NoError(err, "failed to dial external RPC %s", rpcURL)
	t.Cleanup(func() { ethClient.Close() })

	// spamoor — connects to the external RPC via host networking so it can
	// resolve the same hostnames as the host machine.
	spBuilder := spamoor.NewNodeBuilder(t.Name()).
		WithDockerClient(s.dockerCli).
		WithDockerNetworkID(s.networkID).
		WithLogger(zaptest.NewLogger(t)).
		WithRPCHosts(rpcURL).
		WithPrivateKey(privateKey).
		WithHostNetwork().
		WithAdditionalStartArgs("--slot-duration", cfg.SlotDuration, "--startup-delay", "0")

	spNode, err := spBuilder.Build(ctx)
	s.Require().NoError(err, "failed to build spamoor node")
	t.Cleanup(func() { _ = spNode.Remove(t.Context()) })
	s.Require().NoError(spNode.Start(ctx), "failed to start spamoor node")

	spInfo, err := spNode.GetNetworkInfo(ctx)
	s.Require().NoError(err, "failed to get spamoor network info")
	apiAddr := "http://127.0.0.1:" + spInfo.External.Ports.HTTP
	requireHostUp(t, apiAddr+"/api/spammers", 30*time.Second)

	// trace provider
	traceURL := os.Getenv("BENCH_TRACE_QUERY_URL")
	s.Require().NotEmpty(traceURL, "BENCH_TRACE_QUERY_URL must be set in external mode")
	t.Logf("external mode: using trace query URL %s", traceURL)

	return &env{
		traces: &victoriaTraceProvider{
			queryURL:  traceURL,
			t:         t,
			startTime: time.Now(),
		},
		spamoorAPI:        spNode.API(),
		ethClient:         ethClient,
		evNodeServiceName: envOrDefault("BENCH_EVNODE_SERVICE_NAME", "ev-node"),
		evRethServiceName: envOrDefault("BENCH_EVRETH_SERVICE_NAME", "ev-reth"),
	}
}

// collectTraces fetches ev-node traces (required) and ev-reth traces (optional)
// from the configured trace provider, then displays flowcharts.
func (s *SpamoorSuite) collectTraces(e *env) *traceResult {
	t := s.T()
	ctx := t.Context()

	evNodeSpans, err := e.traces.collectSpans(ctx, e.evNodeServiceName)
	s.Require().NoError(err, "failed to collect %s traces", e.evNodeServiceName)

	tr := &traceResult{
		evNode: evNodeSpans,
		evReth: e.traces.tryCollectSpans(ctx, e.evRethServiceName),
	}

	if link := e.traces.uiURL(e.evNodeServiceName); link != "" {
		t.Logf("traces UI: %s", link)
	}

	if rc, ok := e.traces.(richSpanCollector); ok {
		if spans, err := rc.collectRichSpans(ctx, e.evNodeServiceName); err == nil {
			tr.evNodeRich = spans
		}
		if spans, err := rc.collectRichSpans(ctx, e.evRethServiceName); err == nil {
			tr.evRethRich = spans
		}
	}

	if rac, ok := e.traces.(resourceAttrCollector); ok {
		tr.evNodeAttrs = rac.fetchResourceAttrs(ctx, e.evNodeServiceName)
		tr.evRethAttrs = rac.fetchResourceAttrs(ctx, e.evRethServiceName)
	}

	tr.displayFlowcharts(t, e.evNodeServiceName)
	return tr
}

