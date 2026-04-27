package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/evstack/ev-node/block"
	evconfig "github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/node"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/p2p/key"
	"github.com/evstack/ev-node/pkg/sequencers/solo"
	pkgsigner "github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/signer/file"
	"github.com/evstack/ev-node/pkg/store"
)

type runFlags struct {
	// Fibre
	consensusGRPC string
	chainID       string
	keyringDir    string
	keyName       string
	headerNS      string
	dataNS        string

	// ev-node tuning
	blockTime        time.Duration
	daBlockTime      time.Duration
	batchingStrategy string
	scrapeInterval   time.Duration
	maxPending       uint64
	signerPassphrase string

	// Bench
	homeDir       string
	keepHome      bool
	duration      time.Duration
	workers       int
	txSize        int
	mempoolSize   int
	statsInterval time.Duration

	// Observability
	prometheus     bool
	prometheusAddr string
	logLevel       string
}

func runCmd() *cobra.Command {
	f := runFlags{}
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the bench: start a single-sequencer ev-node against a Fibre network and pump load",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBench(cmd.Context(), f)
		},
	}

	flags := cmd.Flags()

	flags.StringVar(&f.consensusGRPC, "consensus-grpc", "", "celestia-app gRPC address (host:port). Required.")
	flags.StringVar(&f.chainID, "chain-id", "", "celestia-app consensus chain ID. Required.")
	flags.StringVar(&f.keyringDir, "keyring-dir", defaultKeyringDir(), "directory holding the bench cosmos keyring (test backend)")
	flags.StringVar(&f.keyName, "key-name", "default", "name of the key in the keyring used to sign Fibre payment promises")
	flags.StringVar(&f.headerNS, "header-namespace", "fb-bench-h", "namespace string for ev-node block headers (10 bytes after hashing)")
	flags.StringVar(&f.dataNS, "data-namespace", "fb-bench-d", "namespace string for ev-node block data")

	flags.DurationVar(&f.blockTime, "block-time", time.Second, "ev-node block production interval")
	flags.DurationVar(&f.daBlockTime, "da-block-time", time.Second, "DA layer block time hint (controls submitter cadence)")
	flags.StringVar(&f.batchingStrategy, "batching-strategy", "immediate", "ev-node DA batching strategy: immediate|size|time|adaptive")
	flags.DurationVar(&f.scrapeInterval, "reaper-interval", 100*time.Millisecond, "how often the reaper drains the mempool")
	flags.Uint64Var(&f.maxPending, "max-pending", 0, "max pending headers/data before block production pauses (0 = unlimited)")
	flags.StringVar(&f.signerPassphrase, "signer-passphrase", "fiber-bench-passphrase", "passphrase for the ev-node file signer (block-signing key, NOT the cosmos one)")

	flags.StringVar(&f.homeDir, "home", defaultNodeHome(), "ev-node home directory (signer, store)")
	flags.BoolVar(&f.keepHome, "keep-home", false, "do not wipe the ev-node home before starting (resumes prior state)")
	flags.DurationVar(&f.duration, "duration", 60*time.Second, "how long to run the bench before stopping (0 = until SIGINT)")
	flags.IntVar(&f.workers, "workers", 32, "number of concurrent tx-injection goroutines")
	flags.IntVar(&f.txSize, "tx-size", 200, "size of each generated tx in bytes")
	flags.IntVar(&f.mempoolSize, "mempool-size", 1_000_000, "size of the in-mem executor's mempool channel (backpressure boundary)")
	flags.DurationVar(&f.statsInterval, "stats-interval", time.Second, "how often to print a stats line")

	flags.BoolVar(&f.prometheus, "prometheus", true, "enable ev-node's Prometheus metrics endpoint")
	flags.StringVar(&f.prometheusAddr, "prometheus-addr", "127.0.0.1:26660", "address for the ev-node Prometheus endpoint")
	flags.StringVar(&f.logLevel, "log-level", "info", "ev-node log level (debug|info|warn|error)")

	_ = cobra.MarkFlagRequired(flags, "consensus-grpc")
	_ = cobra.MarkFlagRequired(flags, "chain-id")

	return cmd
}

func runBench(parentCtx context.Context, f runFlags) error {
	// Single root context for everything; SIGINT cancels.
	ctx, cancel := signal.NotifyContext(parentCtx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger := setupLogger(f.logLevel)

	if !f.keepHome {
		_ = os.RemoveAll(f.homeDir)
	}
	if err := os.MkdirAll(f.homeDir, 0o755); err != nil {
		return fmt.Errorf("create home %s: %w", f.homeDir, err)
	}

	// 1) Open the cosmos keyring (must already contain --key-name; we don't
	// auto-create here so that operator-funded keys aren't accidentally
	// regenerated when bench runs are re-launched).
	kr, err := openKeyring(f.keyringDir)
	if err != nil {
		return fmt.Errorf("open keyring at %s: %w", f.keyringDir, err)
	}
	rec, err := kr.Key(f.keyName)
	if err != nil {
		return fmt.Errorf("key %q not found in keyring %s — run `fiber-bench keys add %s` first: %w",
			f.keyName, f.keyringDir, f.keyName, err)
	}
	addr, err := rec.GetAddress()
	if err != nil {
		return fmt.Errorf("derive key address: %w", err)
	}
	logger.Info().Str("address", addr.String()).Str("key", f.keyName).Msg("loaded fibre signing key")

	// 2) Build the bridge-bypass Fibre adapter.
	logger.Info().Str("grpc", f.consensusGRPC).Msg("dialing consensus gRPC")
	innerFiberClient, fiberClose, err := buildFibreAdapter(ctx, f.consensusGRPC, f.keyName, kr)
	if err != nil {
		return fmt.Errorf("build fibre adapter: %w", err)
	}
	defer func() {
		if err := fiberClose(); err != nil {
			logger.Warn().Err(err).Msg("fibre adapter close")
		}
	}()
	// Wrap in a latency-recording proxy so the stats printer can show
	// per-Upload p50/p99 — without this we can't tell whether the
	// production-vs-DA-settlement gap comes from ev-node's submitter
	// serialization (one header + one data Upload in flight at a time)
	// or from actual Fibre Upload latency.
	fiberClient := newInstrumentedAdapter(innerFiberClient)

	// 3) Build the ev-node file signer (separate key — block signing, not
	// fibre payments). Created in the home dir if missing.
	signerDir := filepath.Join(f.homeDir, "signer")
	if err := os.MkdirAll(signerDir, 0o750); err != nil {
		return fmt.Errorf("create signer dir: %w", err)
	}
	signerFile := filepath.Join(signerDir, "signer.json")
	var signer pkgsigner.Signer
	if _, err := os.Stat(signerFile); os.IsNotExist(err) {
		s, err := file.CreateFileSystemSigner(signerDir, []byte(f.signerPassphrase))
		if err != nil {
			return fmt.Errorf("create file signer: %w", err)
		}
		signer = s
	} else {
		s, err := file.LoadFileSystemSigner(signerDir, []byte(f.signerPassphrase))
		if err != nil {
			return fmt.Errorf("load file signer: %w", err)
		}
		signer = s
	}
	signerAddr, err := signer.GetAddress()
	if err != nil {
		return fmt.Errorf("signer address: %w", err)
	}

	// 4) Genesis. Single proposer = our signer.
	gen := genesis.NewGenesis(f.chainID, 1, time.Now().UTC(), signerAddr)
	if err := gen.Validate(); err != nil {
		return fmt.Errorf("invalid genesis: %w", err)
	}

	// 5) ev-node config. P2P listen on a random port; ev-node disables p2p
	// outbound when fiber is enabled, but the libp2p host is still
	// constructed, so we still need a port.
	cfg := evconfig.DefaultConfig()
	cfg.RootDir = f.homeDir
	cfg.DBPath = "data"
	cfg.Node.Aggregator = true
	cfg.Node.BlockTime = evconfig.DurationWrapper{Duration: f.blockTime}
	cfg.Node.LazyMode = false
	cfg.Node.MaxPendingHeadersAndData = f.maxPending
	cfg.Node.ScrapeInterval = evconfig.DurationWrapper{Duration: f.scrapeInterval}

	cfg.DA.BlockTime = evconfig.DurationWrapper{Duration: f.daBlockTime}
	cfg.DA.Namespace = f.headerNS
	cfg.DA.DataNamespace = f.dataNS
	cfg.DA.BatchingStrategy = f.batchingStrategy
	cfg.DA.RequestTimeout = evconfig.DurationWrapper{Duration: 60 * time.Second}
	cfg.DA.Fiber.Enabled = true
	cfg.DA.Fiber.ConsensusAddress = f.consensusGRPC
	cfg.DA.Fiber.ConsensusChainID = f.chainID
	// BridgeAddress is required by config validation when fiber enabled,
	// but we never use it. Set a syntactically-valid placeholder.
	cfg.DA.Fiber.BridgeAddress = "ws://127.0.0.1:0"
	cfg.DA.Fiber.KeyName = f.keyName

	cfg.P2P.ListenAddress = "/ip4/127.0.0.1/tcp/0"
	cfg.P2P.DisableConnectionGater = true

	cfg.Instrumentation.Prometheus = f.prometheus
	cfg.Instrumentation.PrometheusListenAddr = f.prometheusAddr
	cfg.Instrumentation.Pprof = false

	cfg.RPC.Address = "127.0.0.1:0"
	cfg.Log.Level = f.logLevel
	cfg.Signer.SignerType = "file"
	cfg.Signer.SignerPath = signerDir

	// Validate fiber config the way ev-node would.
	if err := cfg.DA.Fiber.Validate(); err != nil {
		return fmt.Errorf("fiber config: %w", err)
	}

	// 6) Datastore for ev-node's internal state. Uses the standard
	// constructor; the in-memory swap for benchmarking lives in
	// pkg/store/kv.go::NewDefaultKVStore (see HACK there).
	ds, err := store.NewDefaultKVStore(f.homeDir, cfg.DBPath, "fiber-bench")
	if err != nil {
		return fmt.Errorf("open datastore: %w", err)
	}

	// 7) Executor + sequencer.
	exec := newInMemExecutor(f.mempoolSize)
	seq := solo.NewSoloSequencer(logger, []byte(gen.ChainID), exec)

	// 8) DA client wraps our adapter as the FullDAClient ev-node expects.
	daClient := block.NewFiberDAClient(fiberClient, cfg, logger, gen.DAStartHeight)

	// 9) p2p client (required by NewNode signature; outbound is disabled
	// internally when fiber is enabled).
	nodePrivKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate node key: %w", err)
	}
	nodeKey := &key.NodeKey{PrivKey: nodePrivKey}
	p2pClient, err := p2p.NewClient(cfg.P2P, nodeKey.PrivKey, datastore.NewMapDatastore(), gen.ChainID, logger, nil)
	if err != nil {
		return fmt.Errorf("create p2p client: %w", err)
	}

	// 10) Build the node.
	rollnode, err := node.NewNode(
		cfg,
		exec,
		seq,
		daClient,
		signer,
		p2pClient,
		gen,
		ds,
		node.DefaultMetricsProvider(cfg.Instrumentation),
		logger,
		node.NodeOptions{},
	)
	if err != nil {
		return fmt.Errorf("create node: %w", err)
	}

	// 11) Start the node.
	nodeErrCh := make(chan error, 1)
	var nodeWg sync.WaitGroup
	nodeWg.Add(1)
	go func() {
		defer nodeWg.Done()
		defer func() {
			if r := recover(); r != nil {
				nodeErrCh <- fmt.Errorf("node panicked: %v", r)
			}
		}()
		nodeErrCh <- rollnode.Run(ctx)
	}()

	// 12) Start the load generator.
	loaderWg := sync.WaitGroup{}
	loaderWg.Add(1)
	go func() {
		defer loaderWg.Done()
		newLoader(exec, f.workers, f.txSize).run(ctx)
	}()

	// 13) Stats printer + duration timer.
	logger.Info().
		Dur("duration", f.duration).
		Int("workers", f.workers).
		Int("tx_size", f.txSize).
		Int("mempool", f.mempoolSize).
		Dur("block_time", f.blockTime).
		Str("batching", f.batchingStrategy).
		Msg("bench started")

	printer := newStatsPrinter(exec, f.prometheusAddr, f.txSize, fiberClient)
	printer.start(ctx, f.statsInterval)

	if f.duration > 0 {
		select {
		case <-time.After(f.duration):
			logger.Info().Msg("duration elapsed, stopping")
		case err := <-nodeErrCh:
			if err != nil && !errors.Is(err, context.Canceled) {
				logger.Error().Err(err).Msg("node exited unexpectedly")
				cancel()
				return err
			}
		case <-ctx.Done():
		}
	} else {
		select {
		case err := <-nodeErrCh:
			if err != nil && !errors.Is(err, context.Canceled) {
				logger.Error().Err(err).Msg("node exited unexpectedly")
				cancel()
				return err
			}
		case <-ctx.Done():
		}
	}

	cancel()
	loaderWg.Wait()
	nodeWg.Wait()
	printer.printFinalSummary()
	return nil
}

func setupLogger(level string) zerolog.Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	return zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
		With().Timestamp().Str("component", "fiber-bench").Logger()
}
