//go:build evm

package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	tastoradocker "github.com/celestiaorg/tastora/framework/docker"
	"github.com/celestiaorg/tastora/framework/docker/container"
	"github.com/celestiaorg/tastora/framework/docker/evstack/reth"
	"github.com/ethereum/go-ethereum/common"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/execution/evm"
)

const (
	forkCyclePragueTimestamp    = uint64(12)
	forkCycleOsakaTimestamp     = uint64(24)
	forkCycleAmsterdamTimestamp = uint64(36)
)

// TestEngineAPIForkUpgradeCycleE2E drives ev-node's EngineClient against a real
// ev-reth Docker node whose chainspec activates Osaka and Amsterdam at fixed
// payload timestamps. An Engine RPC proxy records the method versions used by
// ev-node, so this verifies the V4 -> V5 -> V6 upgrade path instead of only
// asserting that blocks are produced.
//
// The existing Tastora ev-reth image is used by default. Set EV_RETH_TAG, or set
// EV_RETH_IMAGE_REPO with optional EV_RETH_TAG, to point at a local or registry image.
func TestEngineAPIForkUpgradeCycleE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping due to short mode")
	}

	rethImageOpt := rethImageOptFromEnv()
	ctx := context.Background()
	dockerClient, networkID := tastoradocker.Setup(t)

	genesis := reth.DefaultEvolveGenesisJSON(withEngineForkTimes(t, forkCycleOsakaTimestamp, forkCycleAmsterdamTimestamp))
	rethNode := SetupTestRethNode(t, dockerClient, networkID,
		rethImageOpt,
		func(b *reth.NodeBuilder) {
			b.WithGenesis([]byte(genesis))
		},
	)

	networkInfo, err := rethNode.GetNetworkInfo(ctx)
	require.NoError(t, err)

	ethURL := "http://127.0.0.1:" + networkInfo.External.Ports.RPC
	engineURL := "http://127.0.0.1:" + networkInfo.External.Ports.Engine
	engineRecorder := newEngineRPCRecorder(t, engineURL)
	defer engineRecorder.Close()

	genesisHash, err := rethNode.GenesisHash(ctx)
	require.NoError(t, err)

	store := dssync.MutexWrap(ds.NewMapDatastore())
	executionClient, err := evm.NewEngineExecutionClient(
		ethURL,
		engineRecorder.URL(),
		rethNode.JWTSecretHex(),
		common.HexToHash(genesisHash),
		common.Address{},
		store,
		false,
		zerolog.Nop(),
	)
	require.NoError(t, err)

	stateRoot, err := executionClient.InitChain(ctx, time.Unix(0, 0), 1, "engine-fork-cycle")
	require.NoError(t, err)

	steps := []struct {
		name      string
		height    uint64
		timestamp uint64
	}{
		{name: "prague", height: 1, timestamp: forkCyclePragueTimestamp},
		{name: "osaka", height: 2, timestamp: forkCycleOsakaTimestamp},
		{name: "amsterdam", height: 3, timestamp: forkCycleAmsterdamTimestamp},
	}

	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			nextStateRoot, err := executionClient.ExecuteTxs(ctx, nil, step.height, time.Unix(int64(step.timestamp), 0), stateRoot)
			require.NoError(t, err)
			stateRoot = nextStateRoot
		})
	}

	engineRecorder.RequireCalled(t, "engine_forkchoiceUpdatedV3")
	engineRecorder.RequireCalled(t, "engine_getPayloadV4")
	engineRecorder.RequireCalled(t, "engine_newPayloadV4")
	engineRecorder.RequireCalled(t, "engine_getPayloadV5")
	engineRecorder.RequireCalled(t, "engine_forkchoiceUpdatedV4")
	engineRecorder.RequireCalled(t, "engine_getPayloadV6")
	engineRecorder.RequireCalled(t, "engine_newPayloadV5")
	require.True(t, engineRecorder.SawNewPayloadV5BlockAccessList(),
		"engine_newPayloadV5 should preserve executionPayload.blockAccessList from engine_getPayloadV6")
}

func withEngineForkTimes(t testing.TB, osakaTime, amsterdamTime uint64) reth.GenesisOpt {
	t.Helper()
	return func(genesis []byte) ([]byte, error) {
		var doc map[string]any
		if err := json.Unmarshal(genesis, &doc); err != nil {
			return nil, err
		}
		config, ok := doc["config"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("genesis config missing or invalid")
		}
		config["osakaTime"] = osakaTime
		config["amsterdamTime"] = amsterdamTime
		return json.Marshal(doc)
	}
}

func rethImageOptFromEnv() RethNodeOpt {
	repo := strings.TrimSpace(os.Getenv("EV_RETH_IMAGE_REPO"))
	tag := strings.TrimSpace(os.Getenv("EV_RETH_TAG"))
	if repo == "" && tag == "" {
		return nil
	}
	if tag == "" {
		tag = "latest"
	}
	return func(b *reth.NodeBuilder) {
		if repo != "" {
			b.WithImage(container.NewImage(repo, tag, ""))
			return
		}
		b.WithTag(tag)
	}
}

type engineRPCRecorder struct {
	server   *http.Server
	listener net.Listener
	target   *url.URL

	mu                             sync.Mutex
	calls                          []string
	counts                         map[string]int
	sawNewPayloadV5BlockAccessList bool
}

func newEngineRPCRecorder(t testing.TB, target string) *engineRPCRecorder {
	t.Helper()

	targetURL, err := url.Parse(target)
	require.NoError(t, err)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	recorder := &engineRPCRecorder{
		listener: listener,
		target:   targetURL,
		counts:   make(map[string]int),
	}
	recorder.server = &http.Server{Handler: http.HandlerFunc(recorder.proxy)}

	go func() {
		if err := recorder.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Logf("engine RPC recorder stopped unexpectedly: %v", err)
		}
	}()

	return recorder
}

func (r *engineRPCRecorder) URL() string {
	return "http://" + r.listener.Addr().String()
}

func (r *engineRPCRecorder) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = r.server.Shutdown(ctx)
}

func (r *engineRPCRecorder) RequireCalled(t testing.TB, method string) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	require.Greater(t, r.counts[method], 0, "expected Engine RPC method %s to be called; calls=%v", method, r.calls)
}

func (r *engineRPCRecorder) SawNewPayloadV5BlockAccessList() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sawNewPayloadV5BlockAccessList
}

func (r *engineRPCRecorder) proxy(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = req.Body.Close()

	method := rpcMethod(body)
	if method != "" {
		r.record(method, body)
	}

	targetReq, err := http.NewRequestWithContext(req.Context(), req.Method, r.target.String(), bytes.NewReader(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	targetReq.Header = req.Header.Clone()

	resp, err := http.DefaultTransport.RoundTrip(targetReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (r *engineRPCRecorder) record(method string, body []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls = append(r.calls, method)
	r.counts[method]++

	if method == "engine_newPayloadV5" && requestHasBlockAccessList(body) {
		r.sawNewPayloadV5BlockAccessList = true
	}
}

func rpcMethod(body []byte) string {
	var req struct {
		Method string `json:"method"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}
	return req.Method
}

func requestHasBlockAccessList(body []byte) bool {
	var req struct {
		Params []json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(body, &req); err != nil || len(req.Params) == 0 {
		return false
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(req.Params[0], &payload); err != nil {
		return false
	}
	_, ok := payload["blockAccessList"]
	return ok
}
