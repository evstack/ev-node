// Command evnode-fibre runs a long-lived ev-node aggregator wired to
// a celestia-node-fiber adapter. It is the binary that ships to the
// ev-node instance during a `talis deploy` for the multi-FSP
// throughput experiment.
//
// Topology (smallest variant of the experiment):
//
//	[ load-gen ]
//	     │  POST /tx
//	     ▼
//	[ evnode-fibre (this binary)  aggregator + InMem executor ]
//	     │  block.NewFiberDAClient → cnfiber.New
//	     ▼
//	[ celestia-node bridge ]
//	     │  blob.Subscribe / blob.Submit
//	     ▼
//	[ Fibre Server (per validator) ]  +  [ celestia-app validators ]
//
// CLI flags map to talis/SSM-friendly env vars; everything that
// changes per-deploy can be set via flag *or* CELES_* env var.
//
// This binary is intentionally not part of the testapp tree — testapp
// is the canonical small-chain example and we don't want to drag the
// celestia-node-fiber adapter (with its celestia-node + celestia-app
// deps) into testapp's go.mod. By living under tools/celestia-node-fiber
// the runner reuses the adapter's existing dep set as-is.
package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"

	"github.com/celestiaorg/celestia-app/v9/app"
	"github.com/celestiaorg/celestia-app/v9/app/encoding"
	appfibre "github.com/celestiaorg/celestia-app/v9/fibre"
	"github.com/celestiaorg/celestia-node/api/client"
	cnp2p "github.com/celestiaorg/celestia-node/nodebuilder/p2p"

	"github.com/evstack/ev-node/block"
	coreexecution "github.com/evstack/ev-node/core/execution"
	"github.com/evstack/ev-node/node"
	"github.com/evstack/ev-node/pkg/config"
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/p2p/key"
	"github.com/evstack/ev-node/pkg/sequencers/solo"
	pkgsigner "github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/signer/file"
	"github.com/evstack/ev-node/pkg/store"

	cnfiber "github.com/evstack/ev-node/tools/celestia-node-fiber"
)

type cliFlags struct {
	homeDir        string
	chainID        string
	headerNS       string
	dataNS         string
	bridgeAddr     string
	bridgeTokenFp  string
	coreGRPCAddr   string
	coreNetwork    string
	keyringPath    string
	keyName        string
	signerPpFp     string
	httpListen     string
	rpcListen      string
	p2pListen      string
	pprofListen    string
	blockTime      time.Duration
	scrapeInterval time.Duration
	logLevel       string
}

func parseFlags() cliFlags {
	var c cliFlags

	flag.StringVar(&c.homeDir, "home", envOr("EVNODE_HOME", filepath.Join(os.Getenv("HOME"), ".evnode-fibre")),
		"ev-node home directory (datastore, signer, node key live here)")
	flag.StringVar(&c.chainID, "chain-id", envOr("CHAIN_ID", ""),
		"app chain id (must match validators' chain id)")
	flag.StringVar(&c.headerNS, "header-namespace", envOr("HEADER_NS", "ev-fib-ht"),
		"DA header namespace string")
	flag.StringVar(&c.dataNS, "data-namespace", envOr("DATA_NS", "ev-fib-da"),
		"DA data namespace string")
	flag.StringVar(&c.bridgeAddr, "bridge-addr", envOr("BRIDGE_ADDR", ""),
		"celestia-node bridge RPC address (host:port, no scheme)")
	flag.StringVar(&c.bridgeTokenFp, "bridge-token-file", envOr("BRIDGE_TOKEN_FILE", "/root/bridge-jwt.txt"),
		"path to a file containing the bridge admin JWT")
	flag.StringVar(&c.coreGRPCAddr, "core-grpc-addr", envOr("CORE_GRPC_ADDR", ""),
		"celestia-app validator gRPC address (host:port) used by Fibre's submit path for state queries")
	flag.StringVar(&c.coreNetwork, "core-network", envOr("CORE_NETWORK", "private"),
		"celestia-node Network identifier (matches bridge's --p2p.network)")
	flag.StringVar(&c.keyringPath, "keyring-path", envOr("KEYRING_PATH", ""),
		"directory holding the cosmos-sdk file keyring with the Fibre payment account")
	flag.StringVar(&c.keyName, "key-name", envOr("KEY_NAME", "default-fibre"),
		"keyring entry name for the Fibre payment account")
	flag.StringVar(&c.signerPpFp, "signer-passphrase-file", envOr("SIGNER_PASSPHRASE_FILE", ""),
		"path to a file holding the file-backed signer passphrase")
	flag.StringVar(&c.httpListen, "http-listen", envOr("HTTP_LISTEN", "0.0.0.0:7777"),
		"listen addr for the tx-injection HTTP endpoint (POST /tx)")
	flag.StringVar(&c.rpcListen, "rpc-listen", envOr("RPC_LISTEN", "0.0.0.0:7331"),
		"ev-node RPC listen addr")
	flag.StringVar(&c.p2pListen, "p2p-listen", envOr("P2P_LISTEN", "/ip4/0.0.0.0/tcp/7676"),
		"libp2p listen address (kept up for shutdown symmetry; never gossips when Fiber is on)")
	flag.StringVar(&c.pprofListen, "pprof-listen", envOr("PPROF_LISTEN", ""),
		"if set (e.g. 127.0.0.1:6060), serve net/http/pprof on this addr — heap/goroutine/profile useful for diagnosing memory growth")
	flag.DurationVar(&c.blockTime, "block-time", durFromEnv("BLOCK_TIME", 200*time.Millisecond),
		"ev-node BlockTime")
	flag.DurationVar(&c.scrapeInterval, "scrape-interval", durFromEnv("SCRAPE_INTERVAL", 100*time.Millisecond),
		"reaper scrape interval (lower = smaller per-block batches)")
	flag.StringVar(&c.logLevel, "log-level", envOr("LOG_LEVEL", "info"), "log level")

	flag.Parse()
	return c
}

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func durFromEnv(name string, def time.Duration) time.Duration {
	if v := os.Getenv(name); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func main() {
	cfg := parseFlags()
	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run(cli cliFlags) error {
	// Validate the inputs that have no sensible default.
	if cli.chainID == "" {
		return errors.New("--chain-id is required")
	}
	if cli.bridgeAddr == "" {
		return errors.New("--bridge-addr is required")
	}
	if cli.coreGRPCAddr == "" {
		return errors.New("--core-grpc-addr is required (validator-0 gRPC, host:port)")
	}
	if cli.keyringPath == "" {
		return errors.New("--keyring-path is required")
	}

	level, err := zerolog.ParseLevel(cli.logLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("component", "evnode-fibre").Logger()

	if err := os.MkdirAll(cli.homeDir, 0o755); err != nil {
		return fmt.Errorf("create home dir: %w", err)
	}

	// Bridge JWT: read once at startup. The bridge_init.sh writes it
	// to /root/bridge-jwt.txt on the bridge box; the ev-node init
	// script scp's it onto this box at the same path by default.
	authBytes, err := os.ReadFile(cli.bridgeTokenFp)
	if err != nil {
		return fmt.Errorf("read bridge token from %s: %w", cli.bridgeTokenFp, err)
	}
	authToken := string(authBytes)
	for len(authToken) > 0 && (authToken[len(authToken)-1] == '\n' || authToken[len(authToken)-1] == '\r' || authToken[len(authToken)-1] == ' ') {
		authToken = authToken[:len(authToken)-1]
	}

	// Cosmos-sdk file keyring with the Fibre payment account.
	// The deploy step copies this from a validator (or a dedicated
	// pre-funded account) so the keyring already contains cli.keyName.
	encCfg := encoding.MakeConfig(app.ModuleEncodingRegisters...)
	kr, err := keyring.New(app.Name, keyring.BackendTest, cli.keyringPath, nil, encCfg.Codec)
	if err != nil {
		return fmt.Errorf("open keyring at %s: %w", cli.keyringPath, err)
	}
	if _, err := kr.Key(cli.keyName); err != nil {
		return fmt.Errorf("keyring entry %q not found in %s: %w", cli.keyName, cli.keyringPath, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Construct the celestia-node-fiber adapter. The Fibre client
	// defaults (UploadMemoryBudget 512 MiB, RPCTimeout 15 s) are sized
	// for FSP-side concurrency. Bumping the budget alone caused 64 GiB
	// OOMs (4 GiB budget × 16 workers), so we leave that conservative
	// AND raise RPCTimeout to 30 s so a slow-but-healthy validator
	// signature collection isn't cut short under load — under busy
	// conditions a 32 MiB row upload + sig aggregation can exceed the
	// 15 s default.
	fibreCfg := appfibre.DefaultClientConfig()
	fibreCfg.RPCTimeout = 30 * time.Second
	adapter, err := cnfiber.New(ctx, cnfiber.Config{
		Client: client.Config{
			ReadConfig: client.ReadConfig{
				BridgeDAAddr: cli.bridgeAddr,
				DAAuthToken:  authToken,
				EnableDATLS:  false,
			},
			SubmitConfig: client.SubmitConfig{
				DefaultKeyName: cli.keyName,
				Network:        cnp2p.Network(cli.coreNetwork),
				CoreGRPCConfig: client.CoreGRPCConfig{
					Addr: cli.coreGRPCAddr,
				},
				Fibre: &fibreCfg,
			},
		},
	}, kr)
	if err != nil {
		return fmt.Errorf("construct fiber adapter: %w", err)
	}
	defer adapter.Close()

	// File-backed signer for ev-node block-signing key (separate from
	// the cosmos-sdk keyring used to sign Fibre payment promises).
	signerDir := filepath.Join(cli.homeDir, "signer")
	if err := os.MkdirAll(signerDir, 0o755); err != nil {
		return fmt.Errorf("create signer dir: %w", err)
	}
	passphrase, err := readSignerPassphrase(cli.signerPpFp)
	if err != nil {
		return fmt.Errorf("read signer passphrase: %w", err)
	}
	if !signerExists(signerDir) {
		if _, err := file.CreateFileSystemSigner(signerDir, []byte(passphrase)); err != nil {
			return fmt.Errorf("create file signer: %w", err)
		}
	}

	// Generate a libp2p node key. With the syncer's P2P worker gated
	// off in Fiber mode, this key is mostly cosmetic — the host comes
	// up but never gossips. Keeping it ephemeral per restart is fine.
	nodePrivKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate libp2p key: %w", err)
	}
	nodeKey := &key.NodeKey{PrivKey: nodePrivKey}

	// File signer factory needs the address before genesis is built,
	// so construct it here and read the address back.
	fs, err := file.LoadFileSystemSigner(signerDir, []byte(passphrase))
	if err != nil {
		return fmt.Errorf("load signer: %w", err)
	}
	signerAddr, err := fs.GetAddress()
	if err != nil {
		return fmt.Errorf("get signer address: %w", err)
	}

	// Fresh genesis per-run is fine: the chain we're talking to via
	// Fibre is the celestia-app testnet; ev-node's own genesis is
	// self-consistent and never gossiped.
	genesis := genesispkg.NewGenesis(cli.chainID, 1, time.Now(), signerAddr)
	if err := genesis.Validate(); err != nil {
		return fmt.Errorf("validate genesis: %w", err)
	}

	cfg := config.DefaultConfig()
	cfg.RootDir = cli.homeDir
	cfg.DBPath = "data"
	cfg.Node.Aggregator = true
	cfg.Node.BlockTime = config.DurationWrapper{Duration: cli.blockTime}
	cfg.Node.LazyMode = false
	cfg.Node.ScrapeInterval = config.DurationWrapper{Duration: cli.scrapeInterval}
	cfg.DA.Namespace = cli.headerNS
	cfg.DA.DataNamespace = cli.dataNS
	cfg.DA.Fiber.Enabled = true
	cfg.DA.Fiber.ConsensusAddress = cli.coreGRPCAddr
	cfg.DA.Fiber.ConsensusChainID = cli.chainID
	cfg.DA.Fiber.BridgeAddress = cli.bridgeAddr
	cfg.DA.Fiber.KeyringPath = cli.keyringPath
	cfg.DA.Fiber.KeyName = cli.keyName
	cfg.DA.RequestTimeout = config.DurationWrapper{Duration: 60 * time.Second}
	// Fiber-tuned profile: BatchingStrategy=adaptive, BatchMaxDelay=1.5s,
	// DA.BlockTime=1s, MaxPendingHeadersAndData=0, plus 120 MiB blob cap.
	cfg.ApplyFiberDefaults()
	// 100 MiB — bounded by Fibre's hard ~128 MiB per-upload cap (we
	// hit `data size exceeds maximum 134217723` at 128 MiB - 5 B).
	// Set the per-block data cap below that so each block_data item
	// fits in a single Fibre upload after the submitter splits a
	// multi-blob batch into ≤120 MiB chunks.
	block.SetMaxBlobSize(100 * 1024 * 1024)
	cfg.P2P.ListenAddress = cli.p2pListen
	cfg.P2P.DisableConnectionGater = true
	cfg.RPC.Address = cli.rpcListen
	cfg.Log.Level = cli.logLevel
	cfg.Signer.SignerType = "file"
	cfg.Signer.SignerPath = signerDir

	signer, err := pkgsigner.NewSigner(ctx, &cfg, passphrase)
	if err != nil {
		return fmt.Errorf("construct signer via factory: %w", err)
	}

	ds, err := store.NewDefaultKVStore(cli.homeDir, cfg.DBPath, "evnode-fibre")
	if err != nil {
		return fmt.Errorf("create datastore: %w", err)
	}

	executor := newInMemExecutor()
	sequencer := solo.NewSoloSequencer(logger, []byte(genesis.ChainID), executor)
	// Cap the sequencer's in-memory queue at 10× the per-block tx
	// budget. Above this, SubmitBatchTxs returns ErrQueueFull and the
	// runner's reaper-bridge / tx-ingress applies backpressure (txs
	// stay in the executor's txChan until the sequencer drains, and
	// the chan's bound 503's /tx). Without this cap a fast loadgen
	// (32 vCPU pushing >100 MB/s) outruns the 1 block/s drain and
	// the queue grows monotonically — observed pre-fix as 24 GB of
	// retained io.ReadAll bytes in heap snapshots before the daemon
	// hit the 64 GiB box ceiling and OOM-killed.
	// Sized at 10× the per-block tx budget (matches SetMaxBlobSize
	// above; both anchor at the per-blob Fibre cap).
	const seqQueueBytes = 10 * 100 * 1024 * 1024 // 1 GiB
	sequencer.SetMaxQueueBytes(seqQueueBytes)
	daClient := block.NewFiberDAClient(adapter, cfg, logger, 0)
	p2pClient, err := p2p.NewClient(cfg.P2P, nodeKey.PrivKey, datastore.NewMapDatastore(), genesis.ChainID, logger, nil)
	if err != nil {
		return fmt.Errorf("create p2p client: %w", err)
	}

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
	if err != nil {
		return fmt.Errorf("create ev-node: %w", err)
	}

	// pprof on a separate listener (off by default). The `_ "net/http/pprof"`
	// import registers handlers on http.DefaultServeMux; we serve that
	// mux on cli.pprofListen so heap / goroutine / profile dumps don't
	// share a port with the tx-ingress mux. Used to diagnose where the
	// daemon's RSS goes — the AWS run held ~49 GiB at steady state and
	// we don't yet have a breakdown.
	if cli.pprofListen != "" {
		pprofSrv := &http.Server{Addr: cli.pprofListen, ReadHeaderTimeout: 5 * time.Second}
		go func() {
			logger.Info().Str("addr", cli.pprofListen).Msg("starting pprof HTTP server")
			if err := pprofSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Warn().Err(err).Msg("pprof server exited")
			}
		}()
	}

	// HTTP tx ingestion endpoint.
	httpServer := &http.Server{
		Addr:    cli.httpListen,
		Handler: txIngressHandler(executor, logger),
	}
	httpDone := make(chan error, 1)
	go func() {
		logger.Info().Str("addr", cli.httpListen).Msg("starting tx-ingress HTTP server")
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			httpDone <- err
		} else {
			httpDone <- nil
		}
	}()

	// Run the node and trap signals.
	nodeDone := make(chan error, 1)
	go func() {
		nodeDone <- rollnode.Run(ctx)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigCh:
		logger.Info().Msg("signal received, shutting down")
	case err := <-nodeDone:
		if err != nil {
			logger.Error().Err(err).Msg("ev-node exited with error")
		}
	case err := <-httpDone:
		logger.Error().Err(err).Msg("HTTP server exited")
	}
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = httpServer.Shutdown(shutdownCtx)

	return nil
}

func readSignerPassphrase(path string) (string, error) {
	if path == "" {
		// Default: deterministic-but-non-empty passphrase. This is a
		// long-lived testnet daemon, not a custodial wallet.
		return "evnode-fibre-passphrase", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	s := string(b)
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	if s == "" {
		return "", errors.New("passphrase file is empty")
	}
	return s, nil
}

func signerExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "signer.json"))
	return err == nil
}

// ─────────────────────────── tx ingress ────────────────────────────────

func txIngressHandler(exec *inMemExecutor, logger zerolog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/tx", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(body) == 0 {
			http.Error(w, "empty body", http.StatusBadRequest)
			return
		}
		exec.InjectTx(body)
		w.WriteHeader(http.StatusAccepted)
	})
	mux.HandleFunc("/stats", func(w http.ResponseWriter, _ *http.Request) {
		s := exec.Stats()
		fmt.Fprintf(w, "blocks=%d txs=%d\n", s.BlocksProduced, s.TotalExecutedTxs)
	})
	return mux
}

// ─────────────────────── in-memory executor ────────────────────────────
//
// Mirrors the test executor in tools/celestia-node-fiber/testing — accepts
// arbitrary tx bytes, drains a buffered channel into blocks, and tracks
// counts for /stats. Not a real chain; just a generic blob factory for
// the experiment.

type inMemExecutor struct {
	mu          sync.Mutex
	txChan      chan []byte
	maxBlockTxs int
	blocks      atomic.Uint64
	totalTxs    atomic.Uint64
}

// txChan capacity bounds the HTTP /tx ingest queue. Sized at 10K
// slots (~100 MiB at 10 KB tx-size) so a 100 ms reaper cycle can
// absorb a full max-size block's worth of txs without /tx blocking
// the loadgen. Earlier we used 500 slots (~5 MiB) which forced
// backpressure at ~5,000 tx/s — that turned txsim into the limiting
// factor at ~22 MB/s rather than DA upload. With the per-block
// FilterTxs cap (executor.go:RetrieveBatch via DefaultMaxBlobSize=
// 100 MiB) and the submitter chunker now enforcing the actual blob
// budget, the executor doesn't need an extra ingest-side cap.
//
// maxBlockTxs caps GetTxs's per-call return; pairs with the channel
// size so a reaper poll can fully drain a 100 MiB-block-worth of
// queued txs in a single call instead of needing 20× cycles.
func newInMemExecutor() *inMemExecutor {
	return &inMemExecutor{
		txChan:      make(chan []byte, 10000),
		maxBlockTxs: 10000,
	}
}

func (e *inMemExecutor) InjectTx(tx []byte) {
	e.txChan <- tx
}

type execStats struct {
	BlocksProduced   uint64
	TotalExecutedTxs uint64
}

func (e *inMemExecutor) Stats() execStats {
	return execStats{
		BlocksProduced:   e.blocks.Load(),
		TotalExecutedTxs: e.totalTxs.Load(),
	}
}

func (e *inMemExecutor) InitChain(_ context.Context, _ time.Time, _ uint64, _ string) ([]byte, error) {
	return []byte("evnode-fibre-genesis-root"), nil
}

func (e *inMemExecutor) GetTxs(_ context.Context) ([][]byte, error) {
	var txs [][]byte
	for len(txs) < e.maxBlockTxs {
		select {
		case tx := <-e.txChan:
			txs = append(txs, tx)
		default:
			return txs, nil
		}
	}
	return txs, nil
}

func (e *inMemExecutor) ExecuteTxs(_ context.Context, txs [][]byte, height uint64, _ time.Time, _ []byte) ([]byte, error) {
	e.blocks.Add(1)
	e.totalTxs.Add(uint64(len(txs)))
	root := make([]byte, 32)
	binary.BigEndian.PutUint64(root, height)
	binary.BigEndian.PutUint64(root[8:], uint64(len(txs)))
	return root, nil
}

func (e *inMemExecutor) SetFinal(_ context.Context, _ uint64) error            { return nil }
func (e *inMemExecutor) Rollback(_ context.Context, _ uint64) error            { return nil }
func (e *inMemExecutor) GetExecutionInfo(_ context.Context) (coreexecution.ExecutionInfo, error) {
	return coreexecution.ExecutionInfo{MaxGas: 0}, nil
}

// FilterTxs admits txs in arrival order until the maxBytes budget is
// reached, then postpones the rest back to the sequencer queue so they
// land in a future batch. Skipping this enforcement (a previous version
// returned FilterOK unconditionally) lets a single block sweep up the
// entire mempool — under sustained txsim load that produced 369 MiB
// blocks that exceeded Fibre's per-upload cap and crashed the node
// with `single item exceeds DA blob size limit`.
func (e *inMemExecutor) FilterTxs(_ context.Context, txs [][]byte, maxBytes, _ uint64, _ bool) ([]coreexecution.FilterStatus, error) {
	st := make([]coreexecution.FilterStatus, len(txs))
	var used uint64
	for i, tx := range txs {
		size := uint64(len(tx))
		if maxBytes > 0 && used+size > maxBytes {
			st[i] = coreexecution.FilterPostpone
			continue
		}
		st[i] = coreexecution.FilterOK
		used += size
	}
	return st, nil
}

var _ coreexecution.Executor = (*inMemExecutor)(nil)
