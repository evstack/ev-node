//go:build fibre

package cnfibertest_test

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
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/p2p/key"
	pkgsigner "github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/signer/file"
	"github.com/evstack/ev-node/pkg/sequencers/solo"
	"github.com/evstack/ev-node/pkg/store"
	datypes "github.com/evstack/ev-node/pkg/da/types"

	"github.com/celestiaorg/celestia-node/api/client"

	cnfiber "github.com/evstack/ev-node/tools/celestia-node-fiber"
	cnfibertest "github.com/evstack/ev-node/tools/celestia-node-fiber/testing"
)

const (
	evnodeBlockTime    = 200 * time.Millisecond
	evnodeDABlockTime  = 1 * time.Second
	evnodeHeaderNS     = "ev-fib-ht"
	evnodeDataNS       = "ev-fib-da"
	evnodeChainID      = "ev-fiber-test"
	evnodeBlockTimeout = 30 * time.Second
	evnodePassphrase   = "test-passphrase-evnode"
)

// TestEvNode_FiberDA_Posting wires a full ev-node in-memory to the
// celestia-node-fiber adapter and verifies that block data is posted
// to the Fibre DA layer. The test:
//   - Starts a single-validator Celestia chain + Fibre server + bridge
//   - Creates a celestia-node-fiber adapter (block.FiberClient)
//   - Constructs an ev-node aggregator node that uses the adapter as DA
//   - Subscribes to the data namespace via adapter.Listen before uploading
//   - Injects a transaction and waits for block production
//   - Confirms the DA submitter pushed blobs to Fiber by receiving events
//     on the subscription and round-tripping each through Download
func TestEvNode_FiberDA_Posting(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)

	network := cnfibertest.StartNetwork(t, ctx)
	bridge := cnfibertest.StartBridge(t, ctx, network)

	adapter, err := cnfiber.New(ctx, cnfiber.Config{
		Client: client.Config{
			ReadConfig: client.ReadConfig{
				BridgeDAAddr: bridge.RPCAddr(),
				DAAuthToken:  bridge.AdminToken,
				EnableDATLS:  false,
			},
			SubmitConfig: client.SubmitConfig{
				DefaultKeyName: network.ClientAccount,
				Network:        "private",
				CoreGRPCConfig: client.CoreGRPCConfig{
					Addr: network.ConsensusGRPCAddr(),
				},
			},
		},
	}, network.Consensus.Keyring)
	require.NoError(t, err, "constructing adapter")
	t.Cleanup(func() { _ = adapter.Close() })

	// Subscribe to the header namespace BEFORE starting the node so we
	// don't race against the first DA submission. fromHeight=0 follows
	// the live tip. The adapter expects the 10-byte v0 namespace ID
	// (the last 10 bytes of the full 29-byte namespace), matching what
	// fiberDAClient.Submit extracts before calling fiber.Upload.
	fullHeaderNS := datypes.NamespaceFromString(evnodeHeaderNS).Bytes()
	headerNSID := fullHeaderNS[len(fullHeaderNS)-10:]
	events, err := adapter.Listen(ctx, headerNSID, 0)
	require.NoError(t, err, "starting fiber Listen on header namespace")

	rollnode, exec, nodeCleanup := newFiberEvNode(t, ctx, adapter)
	t.Cleanup(nodeCleanup)

	nodeErrCh := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				nodeErrCh <- fmt.Errorf("node panicked: %v", r)
			}
		}()
		nodeErrCh <- rollnode.Run(ctx)
	}()

	txPayload := fmt.Sprintf("fiber-key=fiber-value-%d", time.Now().UnixNano())
	exec.InjectTx([]byte(txPayload))

	require.Eventually(t, func() bool {
		stats := exec.Stats()
		t.Logf("blocks=%d txs=%d", stats.BlocksProduced, stats.TotalExecutedTxs)
		return stats.BlocksProduced >= 1 && stats.TotalExecutedTxs >= 1
	}, evnodeBlockTimeout, 200*time.Millisecond, "ev-node should produce at least one block with the transaction")

	// Drain at least one Fiber BlobEvent from the subscription to prove
	// the DA submitter pushed data through the fiber adapter's Upload
	// path and the settlement landed on-chain.
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
	}, evnodeBlockTimeout, 500*time.Millisecond, "expected at least one Fiber BlobEvent from DA submission")

	for _, ev := range seen {
		got, err := adapter.Download(ctx, ev.BlobID)
		require.NoError(t, err, "adapter.Download blob_id=%x", ev.BlobID)
		require.NotEmpty(t, got, "downloaded blob must not be empty")
		t.Logf("download ok: blob_id=%x bytes=%d", ev.BlobID, len(got))
	}

	select {
	case err := <-nodeErrCh:
		t.Fatalf("node exited unexpectedly: %v", err)
	default:
	}
}

type inMemExecutor struct {
	mu   sync.Mutex
	data map[string]string

	txChan           chan []byte
	blocksProduced   uint64
	totalExecutedTxs uint64
}

func newInMemExecutor() *inMemExecutor {
	return &inMemExecutor{
		data:   make(map[string]string),
		txChan: make(chan []byte, 10000),
	}
}

func (e *inMemExecutor) InjectTx(tx []byte) {
	select {
	case e.txChan <- tx:
	default:
	}
}

type execStats struct {
	BlocksProduced   uint64
	TotalExecutedTxs uint64
}

func (e *inMemExecutor) Stats() execStats {
	e.mu.Lock()
	defer e.mu.Unlock()
	return execStats{BlocksProduced: e.blocksProduced, TotalExecutedTxs: e.totalExecutedTxs}
}

func (e *inMemExecutor) InitChain(_ context.Context, _ time.Time, _ uint64, _ string) ([]byte, error) {
	return []byte("inmem-genesis-root"), nil
}

func (e *inMemExecutor) GetTxs(_ context.Context) ([][]byte, error) {
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

func (e *inMemExecutor) ExecuteTxs(_ context.Context, txs [][]byte, _ uint64, _ time.Time, _ []byte) ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, tx := range txs {
		k, v, ok := parseKV(tx)
		if ok {
			e.data[k] = v
		}
	}
	e.blocksProduced++
	e.totalExecutedTxs += uint64(len(txs))
	return []byte(fmt.Sprintf("root-%d", e.blocksProduced)), nil
}

func (e *inMemExecutor) SetFinal(_ context.Context, _ uint64) error { return nil }
func (e *inMemExecutor) Rollback(_ context.Context, _ uint64) error  { return nil }
func (e *inMemExecutor) GetExecutionInfo(_ context.Context) (coreexecution.ExecutionInfo, error) {
	return coreexecution.ExecutionInfo{MaxGas: 0}, nil
}
func (e *inMemExecutor) FilterTxs(_ context.Context, txs [][]byte, _, _ uint64, _ bool) ([]coreexecution.FilterStatus, error) {
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

func newFiberEvNode(t *testing.T, ctx context.Context, fiberClient block.FiberClient) (node.Node, *inMemExecutor, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	// Create a file-backed signer so the executor can sign blocks.
	signerDir := tmpDir
	fs, err := file.CreateFileSystemSigner(signerDir, []byte(evnodePassphrase))
	require.NoError(t, err, "creating file signer")
	signerAddr, err := fs.GetAddress()
	require.NoError(t, err, "getting signer address")

	// Generate a separate libp2p node key for P2P networking.
	nodePrivKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err, "generating node key")
	nodeKey := &key.NodeKey{PrivKey: nodePrivKey}

	genesis := genesispkg.NewGenesis(evnodeChainID, 1, time.Now(), signerAddr)
	require.NoError(t, genesis.Validate(), "validating genesis")

	cfg := config.DefaultConfig()
	cfg.RootDir = tmpDir
	cfg.DBPath = "data"
	cfg.Node.Aggregator = true
	cfg.Node.BlockTime = config.DurationWrapper{Duration: evnodeBlockTime}
	cfg.Node.LazyMode = false
	cfg.DA.BlockTime = config.DurationWrapper{Duration: evnodeDABlockTime}
	cfg.DA.Namespace = evnodeHeaderNS
	cfg.DA.DataNamespace = evnodeDataNS
	cfg.DA.BatchingStrategy = "immediate"
	cfg.DA.Fiber.Enabled = true
	cfg.DA.RequestTimeout = config.DurationWrapper{Duration: 60 * time.Second}
	cfg.P2P.ListenAddress = "/ip4/0.0.0.0/tcp/0"
	cfg.P2P.DisableConnectionGater = true
	cfg.Instrumentation.Prometheus = false
	cfg.Instrumentation.Pprof = false
	cfg.RPC.Address = "127.0.0.1:0"
	cfg.Log.Level = "debug"
	cfg.Signer.SignerType = "file"
	cfg.Signer.SignerPath = signerDir

	// Build the full signer via the factory (needed for consistency with
	// how the real node boots).
	signer, err := pkgsigner.NewSigner(ctx, &cfg, evnodePassphrase)
	require.NoError(t, err, "creating signer via factory")

	ds, err := store.NewDefaultKVStore(tmpDir, cfg.DBPath, "testdb")
	require.NoError(t, err, "creating datastore")

	executor := newInMemExecutor()
	sequencer := solo.NewSoloSequencer(logger, []byte(genesis.ChainID), executor)
	daClient := block.NewFiberDAClient(fiberClient, cfg, logger)
	p2pClient, err := p2p.NewClient(cfg.P2P, nodeKey.PrivKey, datastore.NewMapDatastore(), genesis.ChainID, logger, nil)
	require.NoError(t, err, "creating p2p client")

	rollnode, err := node.NewNode(
		cfg,
		executor,
		sequencer,
		daClient,
		signer,
		p2pClient,
		genesis,
		ds,
		node.DefaultMetricsProvider(cfg.Instrumentation),
		logger,
		node.NodeOptions{},
	)
	require.NoError(t, err, "creating node")

	return rollnode, executor, func() {}
}

var _ coreexecution.Executor = (*inMemExecutor)(nil)
