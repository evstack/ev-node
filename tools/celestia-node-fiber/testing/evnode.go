//go:build fibre

package cnfibertest

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block"
	coreexecution "github.com/evstack/ev-node/core/execution"
	"github.com/evstack/ev-node/node"
	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/p2p/key"
	"github.com/evstack/ev-node/pkg/sequencers/solo"
	pkgsigner "github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/signer/file"
	"github.com/evstack/ev-node/pkg/store"
)

// EvNodePassphrase is the passphrase used by the file signers wired up
// by NewFiberAggregator / NewFiberFullNode.
const EvNodePassphrase = "test-passphrase-evnode"

const (
	defaultEvNodeBlockTimeout = 60 * time.Second
)

// EvNodeConfig parameterizes the chain shared by an aggregator and any
// number of full nodes. Zero values get sensible defaults applied by
// the helpers — block time defaults to 200ms (fast block production)
// and DA block time to 1s.
type EvNodeConfig struct {
	ChainID         string
	HeaderNamespace string
	DataNamespace   string
	BlockTime       time.Duration
	DABlockTime     time.Duration

	// DAStartHeight is written into Genesis.DAStartHeight (and the
	// FiberDAClient's last-known DA height) so both nodes skip the
	// historical DA scan from height 0 and pick up at the live tip.
	//
	// Why it matters: ev-node's catch-up retriever creates a fresh
	// blob.Subscribe per height batch and cancels it. celestia-node's
	// go-jsonrpc multiplexes subscriptions on a single websocket per
	// module — cancelling any one subscription tears the whole
	// connection down, so subsequent retrievals immediately fail with
	// "websocket routine exiting". Starting at the tip avoids the
	// catch-up phase and keeps the one long-lived Subscribe alive.
	DAStartHeight uint64
}

func (c *EvNodeConfig) applyDefaults() {
	if c.ChainID == "" {
		c.ChainID = "ev-fiber-test"
	}
	// Header / data namespaces default to per-process-unique strings so
	// successive test runs against the same long-lived bridge don't
	// observe each other's blobs (they would be unverifiable against
	// the current test's proposer and would jam the full-node syncer
	// as undeliverable pending events).
	if c.HeaderNamespace == "" {
		c.HeaderNamespace = uniqueNamespace("ht")
	}
	if c.DataNamespace == "" {
		c.DataNamespace = uniqueNamespace("da")
	}
	if c.BlockTime == 0 {
		// 200ms = the production target for ev-node block production.
		// The aggregator keeps up cleanly at this cadence; the full
		// node side has a separate caveat documented on
		// RunEvNodeFibreTwoNodeFlow about the per-Retrieve Subscribe
		// teardown.
		c.BlockTime = 200 * time.Millisecond
	}
	if c.DABlockTime == 0 {
		c.DABlockTime = 1 * time.Second
	}
}

// uniqueNamespace returns a short, deterministically-unique-per-call
// namespace string built from `prefix` plus a 6-byte hex suffix derived
// from crypto/rand. The full string fits within the 10-byte v0
// namespace identifier expected by Fibre.
func uniqueNamespace(prefix string) string {
	var b [3]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%s-%x", prefix, b[:])
}

// NewFiberAggregator wires a single aggregator (block producer) ev-node
// node backed by the supplied Fibre DA client. The returned executor
// can be fed transactions via InjectTx; the returned genesis MUST be
// passed to NewFiberFullNode for any full nodes joining the same chain
// so they share chain-id and proposer address.
//
// The caller drives lifecycle via node.Run(ctx).
func NewFiberAggregator(t *testing.T, ctx context.Context, fiberClient block.FiberClient, cfg EvNodeConfig) (node.Node, *InMemExecutor, genesispkg.Genesis) {
	t.Helper()
	cfg.applyDefaults()

	tmpDir := t.TempDir()
	logger := newTestLogger(t).With().Str("role", "aggregator").Logger()

	signerAddr := mustCreateFileSigner(t, tmpDir)
	gen := genesispkg.NewGenesis(cfg.ChainID, 1, time.Now(), signerAddr)
	gen.DAStartHeight = cfg.DAStartHeight
	require.NoError(t, gen.Validate(), "validating genesis")

	rollnode, exec := buildEvNode(t, ctx, fiberClient, cfg, gen, tmpDir, logger, true, cfg.DAStartHeight)
	return rollnode, exec, gen
}

// NewFiberFullNode wires a full ev-node node (no block production) that
// DA-syncs blocks from the same Fibre namespace as the aggregator.
// Full nodes still need a signer (for libp2p identity / network
// attestations) but it does not need to be the proposer — the proposer
// address comes from the supplied aggregator genesis.
//
// The full node's DA retriever obeys gen.DAStartHeight, which the
// aggregator constructor copies from cfg.DAStartHeight. See the
// EvNodeConfig docstring for why pinning to the live bridge tip
// matters.
func NewFiberFullNode(t *testing.T, ctx context.Context, fiberClient block.FiberClient, cfg EvNodeConfig, gen genesispkg.Genesis) (node.Node, *InMemExecutor) {
	t.Helper()
	cfg.applyDefaults()

	tmpDir := t.TempDir()
	logger := newTestLogger(t).With().Str("role", "fullnode").Logger()

	// File signer is created but the address is unused — only the
	// aggregator's address (already in `gen`) acts as proposer.
	mustCreateFileSigner(t, tmpDir)

	rollnode, exec := buildEvNode(t, ctx, fiberClient, cfg, gen, tmpDir, logger, false, gen.DAStartHeight)
	return rollnode, exec
}

func newTestLogger(t *testing.T) zerolog.Logger {
	return zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
}

func mustCreateFileSigner(t *testing.T, tmpDir string) []byte {
	t.Helper()
	fs, err := file.CreateFileSystemSigner(tmpDir, []byte(EvNodePassphrase))
	require.NoError(t, err, "creating file signer")
	addr, err := fs.GetAddress()
	require.NoError(t, err, "getting signer address")
	return addr
}

func buildEvNode(
	t *testing.T,
	ctx context.Context,
	fiberClient block.FiberClient,
	cfg EvNodeConfig,
	gen genesispkg.Genesis,
	tmpDir string,
	logger zerolog.Logger,
	aggregator bool,
	lastKnownDAHeight uint64,
) (node.Node, *InMemExecutor) {
	t.Helper()

	nodePrivKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err, "generating node key")
	nodeKey := &key.NodeKey{PrivKey: nodePrivKey}

	nodeCfg := config.DefaultConfig()
	nodeCfg.RootDir = tmpDir
	nodeCfg.DBPath = "data"
	nodeCfg.Node.Aggregator = aggregator
	nodeCfg.Node.BlockTime = config.DurationWrapper{Duration: cfg.BlockTime}
	nodeCfg.Node.LazyMode = false
	nodeCfg.DA.BlockTime = config.DurationWrapper{Duration: cfg.DABlockTime}
	nodeCfg.DA.Namespace = cfg.HeaderNamespace
	nodeCfg.DA.DataNamespace = cfg.DataNamespace
	nodeCfg.DA.BatchingStrategy = "immediate"
	nodeCfg.DA.Fiber.Enabled = true
	nodeCfg.DA.StartHeight = cfg.DAStartHeight
	nodeCfg.DA.RequestTimeout = config.DurationWrapper{Duration: 60 * time.Second}
	nodeCfg.P2P.ListenAddress = "/ip4/0.0.0.0/tcp/0"
	nodeCfg.P2P.DisableConnectionGater = true
	nodeCfg.Instrumentation.Prometheus = false
	nodeCfg.Instrumentation.Pprof = false
	nodeCfg.RPC.Address = "127.0.0.1:0"
	nodeCfg.Log.Level = "debug"
	nodeCfg.Signer.SignerType = "file"
	nodeCfg.Signer.SignerPath = tmpDir

	signer, err := pkgsigner.NewSigner(ctx, &nodeCfg, EvNodePassphrase)
	require.NoError(t, err, "creating signer via factory")

	ds, err := store.NewDefaultKVStore(tmpDir, nodeCfg.DBPath, "testdb")
	require.NoError(t, err, "creating datastore")

	executor := newInMemExecutor()
	sequencer := solo.NewSoloSequencer(logger, []byte(gen.ChainID), executor)
	daClient := block.NewFiberDAClient(fiberClient, nodeCfg, logger, lastKnownDAHeight)
	p2pClient, err := p2p.NewClient(nodeCfg.P2P, nodeKey.PrivKey, datastore.NewMapDatastore(), gen.ChainID, logger, nil)
	require.NoError(t, err, "creating p2p client")

	rollnode, err := node.NewNode(
		nodeCfg,
		executor,
		sequencer,
		daClient,
		signer,
		p2pClient,
		gen,
		ds,
		node.DefaultMetricsProvider(nodeCfg.Instrumentation),
		logger,
		node.NodeOptions{},
	)
	require.NoError(t, err, "creating node")

	return rollnode, executor
}

// InMemExecutor is a minimal coreexecution.Executor implementation
// for tests: it accepts "k=v" payloads via InjectTx, applies them to
// an in-memory map, and tracks block + tx counts.
type InMemExecutor struct {
	mu   sync.Mutex
	data map[string]string

	txChan           chan []byte
	blocksProduced   uint64
	totalExecutedTxs uint64
	executedTxs      [][]byte
}

func newInMemExecutor() *InMemExecutor {
	return &InMemExecutor{
		data:   make(map[string]string),
		txChan: make(chan []byte, 10000),
	}
}

// InjectTx queues a "k=v" payload for inclusion in the next block.
func (e *InMemExecutor) InjectTx(tx []byte) {
	select {
	case e.txChan <- tx:
	default:
	}
}

// ExecStats reports cumulative block and tx counts for assertions.
type ExecStats struct {
	BlocksProduced   uint64
	TotalExecutedTxs uint64
}

func (e *InMemExecutor) Stats() ExecStats {
	e.mu.Lock()
	defer e.mu.Unlock()
	return ExecStats{BlocksProduced: e.blocksProduced, TotalExecutedTxs: e.totalExecutedTxs}
}

// Get returns the value associated with the supplied key, if any.
func (e *InMemExecutor) Get(key string) (string, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	v, ok := e.data[key]
	return v, ok
}

// ExecutedTxs returns a copy of the raw payloads that were applied so
// far. Tests use this to confirm a full node observed exactly the txs
// the aggregator submitted via DA.
func (e *InMemExecutor) ExecutedTxs() [][]byte {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([][]byte, len(e.executedTxs))
	for i, tx := range e.executedTxs {
		out[i] = append([]byte(nil), tx...)
	}
	return out
}

func (e *InMemExecutor) InitChain(_ context.Context, _ time.Time, _ uint64, _ string) ([]byte, error) {
	return []byte("inmem-genesis-root"), nil
}

func (e *InMemExecutor) GetTxs(_ context.Context) ([][]byte, error) {
	var txs [][]byte
	for {
		select {
		case tx := <-e.txChan:
			txs = append(txs, tx)
		default:
			return txs, nil
		}
	}
}

func (e *InMemExecutor) ExecuteTxs(_ context.Context, txs [][]byte, _ uint64, _ time.Time, _ []byte) ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, tx := range txs {
		k, v, ok := parseKV(tx)
		if ok {
			e.data[k] = v
		}
		e.executedTxs = append(e.executedTxs, append([]byte(nil), tx...))
	}
	e.blocksProduced++
	e.totalExecutedTxs += uint64(len(txs))
	return []byte(fmt.Sprintf("root-%d", e.blocksProduced)), nil
}

func (e *InMemExecutor) SetFinal(_ context.Context, _ uint64) error { return nil }
func (e *InMemExecutor) Rollback(_ context.Context, _ uint64) error { return nil }

func (e *InMemExecutor) GetExecutionInfo(_ context.Context) (coreexecution.ExecutionInfo, error) {
	return coreexecution.ExecutionInfo{MaxGas: 0}, nil
}

func (e *InMemExecutor) FilterTxs(_ context.Context, txs [][]byte, _, _ uint64, _ bool) ([]coreexecution.FilterStatus, error) {
	st := make([]coreexecution.FilterStatus, len(txs))
	for i := range st {
		st[i] = coreexecution.FilterOK
	}
	return st, nil
}

func parseKV(tx []byte) (string, string, bool) {
	s := string(tx)
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return s[:i], s[i+1:], true
		}
	}
	return "", "", false
}

var _ coreexecution.Executor = (*InMemExecutor)(nil)

// RunEvNodeFibreTwoNodeFlow exercises the aggregator + full-node path:
//
//  1. Subscribe to the aggregator's header namespace via `observer` so
//     we can verify Fibre BlobEvents land on chain.
//  2. Spin up an aggregator backed by `aggAdapter`; capture its genesis.
//  3. Spin up a full node backed by `fnAdapter`, sharing that genesis.
//     The full node DA-syncs from cfg.DAStartHeight (which should be
//     the bridge tip captured before either node starts).
//  4. Inject a tx into the aggregator. Wait for it to produce a block
//     containing the tx.
//  5. Confirm the aggregator's blob landed on Fibre by reading at
//     least one BlobEvent from `observer` and Download'ing it.
//  6. Verify the full node started its DA sync (its syncer initialized
//     against the supplied genesis without crashing). Block-by-block
//     application across the gap between DAStartHeight and the
//     aggregator's first submission requires ev-node to keep a
//     persistent Subscribe — currently the catch-up retriever creates
//     a fresh Subscribe per height batch and cancels it, which tears
//     down celestia-node's go-jsonrpc websocket. That refactor is
//     tracked separately; this test deliberately stops short of
//     asserting the full node fully replayed the aggregator's chain.
//
// The three adapters MUST be distinct instances. celestia-node's
// go-jsonrpc multiplexes blob.Subscribe over a single websocket per
// module — cancelling any one subscription tears the shared connection
// down, which would crash both nodes if they shared an adapter.
func RunEvNodeFibreTwoNodeFlow(t *testing.T, ctx context.Context, aggAdapter, fnAdapter, observer block.FiberClient, cfg EvNodeConfig) {
	t.Helper()
	// Resolve defaults at the top level so both nodes share the same
	// namespaces and chain settings. NewFiberAggregator /
	// NewFiberFullNode also call applyDefaults but it's a no-op once
	// the fields are populated here.
	cfg.applyDefaults()

	fullHeaderNS := datypes.NamespaceFromString(cfg.HeaderNamespace).Bytes()
	headerNSID := fullHeaderNS[len(fullHeaderNS)-10:]
	events, err := observer.Listen(ctx, headerNSID, 0)
	require.NoError(t, err, "starting observer Listen on header namespace")

	aggNode, aggExec, gen := NewFiberAggregator(t, ctx, aggAdapter, cfg)
	fnNode, _ := NewFiberFullNode(t, ctx, fnAdapter, cfg, gen)

	// Start the full node FIRST so its DA retriever is already
	// listening from gen.DAStartHeight (the captured bridge tip) when
	// the aggregator begins posting.
	fnErrCh := startNode(t, ctx, fnNode, "full-node")
	time.Sleep(500 * time.Millisecond)
	aggErrCh := startNode(t, ctx, aggNode, "aggregator")

	txPayload := []byte(fmt.Sprintf("fiber-key=fiber-value-%d", time.Now().UnixNano())) //nolint:gomnd
	aggExec.InjectTx(txPayload)

	require.Eventually(t, func() bool {
		stats := aggExec.Stats()
		t.Logf("aggregator: blocks=%d txs=%d", stats.BlocksProduced, stats.TotalExecutedTxs)
		return stats.BlocksProduced >= 1 && stats.TotalExecutedTxs >= 1
	}, defaultEvNodeBlockTimeout, 200*time.Millisecond, "aggregator should produce at least one block with the injected tx")

	// Confirm the aggregator-injected tx made it into the executed set.
	require.Contains(t, asStrings(aggExec.ExecutedTxs()), string(txPayload),
		"aggregator executed tx set should include the injected payload")

	// Drain at least one Fibre BlobEvent off the observer subscription
	// — this proves the aggregator's DA submission landed on chain via
	// Fibre and is retrievable through the bridge.
	var seen []block.FiberBlobEvent
	require.Eventually(t, func() bool {
		select {
		case ev, ok := <-events:
			if !ok {
				return false
			}
			seen = append(seen, ev)
			t.Logf("fiber event: blob_id=%x height=%d data_size=%d",
				ev.BlobID, ev.Height, ev.DataSize)
			return true
		default:
			return false
		}
	}, defaultEvNodeBlockTimeout, 500*time.Millisecond, "expected at least one Fiber BlobEvent from DA submission")

	for _, ev := range seen {
		got, err := observer.Download(ctx, ev.BlobID)
		require.NoError(t, err, "observer.Download blob_id=%x", ev.BlobID)
		require.NotEmpty(t, got, "downloaded blob must not be empty")
		t.Logf("download ok: blob_id=%x bytes=%d", ev.BlobID, len(got))
	}

	// Confirm neither node has died on us during the assertion window.
	for _, c := range []struct {
		name string
		ch   <-chan error
	}{{"aggregator", aggErrCh}, {"full-node", fnErrCh}} {
		select {
		case err := <-c.ch:
			t.Fatalf("%s exited unexpectedly: %v", c.name, err)
		default:
		}
	}

	// celestia-node's Fibre service spawns one async pay-for-fibre
	// goroutine per submission and they outlive the parent test ctx.
	// Without a grace period they race t.TempDir() cleanup, which
	// removes the docker-derived keyring directory mid-flight and they
	// fail with "key not found". Wait briefly so the in-flight signers
	// settle. (Lifecycle hookup in celestia-node is tracked separately.)
	time.Sleep(2 * time.Second)
}

func startNode(t *testing.T, ctx context.Context, n node.Node, label string) <-chan error {
	t.Helper()
	errCh := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errCh <- fmt.Errorf("%s panicked: %v", label, r)
			}
		}()
		errCh <- n.Run(ctx)
	}()
	return errCh
}

func asStrings(in [][]byte) []string {
	out := make([]string, len(in))
	for i, b := range in {
		out[i] = string(b)
	}
	return out
}
