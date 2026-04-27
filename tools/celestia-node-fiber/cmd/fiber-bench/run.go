package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	rollcmd "github.com/evstack/ev-node/pkg/cmd"
	evconfig "github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/node"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p/key"
	"github.com/evstack/ev-node/pkg/sequencers/solo"
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
			return runBench(cmd, f)
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

	// FlagSignerPassphraseFile is what cmd.StartNode reads to load the
	// file-backed signer's passphrase. We define it on the bench's run
	// command so cmd.StartNode finds it via cmd.Flags().GetString;
	// runBench writes the operator's --signer-passphrase to a temp
	// file and sets this flag's value before delegating.
	flags.String(evconfig.FlagSignerPassphraseFile, "", "(internal) populated by --signer-passphrase before cmd.StartNode runs")
	_ = cmd.Flags().MarkHidden(evconfig.FlagSignerPassphraseFile)

	_ = cobra.MarkFlagRequired(flags, "consensus-grpc")
	_ = cobra.MarkFlagRequired(flags, "chain-id")

	return cmd
}

func runBench(cobraCmd *cobra.Command, f runFlags) error {
	logger := setupLogger(f.logLevel)

	if !f.keepHome {
		_ = os.RemoveAll(f.homeDir)
	}
	if err := os.MkdirAll(f.homeDir, 0o755); err != nil {
		return fmt.Errorf("create home %s: %w", f.homeDir, err)
	}

	// 1) Open the cosmos keyring + build the bridge-bypass Fibre
	// adapter. These are the two genuinely fiber-bench-specific
	// pieces — neither lives in the production wiring path.
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

	logger.Info().Str("grpc", f.consensusGRPC).Msg("dialing consensus gRPC")
	innerFiberClient, fiberClose, err := buildFibreAdapter(cobraCmd.Context(), f.consensusGRPC, f.keyName, kr)
	if err != nil {
		return fmt.Errorf("build fibre adapter: %w", err)
	}
	defer func() {
		if err := fiberClose(); err != nil {
			logger.Warn().Err(err).Msg("fibre adapter close")
		}
	}()
	// Wrap in a latency-recording proxy so the stats printer can show
	// per-Upload p50/p99.
	fiberClient := newInstrumentedAdapter(innerFiberClient)

	// 2) ev-node block-signing key. Created in the home dir if missing.
	signerDir := filepath.Join(f.homeDir, "signer")
	if err := os.MkdirAll(signerDir, 0o750); err != nil {
		return fmt.Errorf("create signer dir: %w", err)
	}
	signerFile := filepath.Join(signerDir, "signer.json")
	if _, statErr := os.Stat(signerFile); os.IsNotExist(statErr) {
		s, err := file.CreateFileSystemSigner(signerDir, []byte(f.signerPassphrase))
		if err != nil {
			return fmt.Errorf("create file signer: %w", err)
		}
		if _, err := s.GetAddress(); err != nil {
			return fmt.Errorf("signer address: %w", err)
		}
	}
	// cmd.StartNode reads the passphrase from a file path stored in
	// FlagSignerPassphraseFile; write the in-memory string out so
	// the canonical signer-loading path works without a separate
	// passphrase-flag flow.
	passphraseFile := filepath.Join(f.homeDir, "passphrase.txt")
	if err := os.WriteFile(passphraseFile, []byte(f.signerPassphrase), 0o600); err != nil {
		return fmt.Errorf("write passphrase file: %w", err)
	}
	if err := cobraCmd.Flags().Set(evconfig.FlagSignerPassphraseFile, passphraseFile); err != nil {
		return fmt.Errorf("set passphrase flag: %w", err)
	}

	// Reload the signer to derive the genesis proposer address.
	loaded, err := file.LoadFileSystemSigner(signerDir, []byte(f.signerPassphrase))
	if err != nil {
		return fmt.Errorf("load file signer: %w", err)
	}
	signerAddr, err := loaded.GetAddress()
	if err != nil {
		return fmt.Errorf("signer address: %w", err)
	}

	// 3) Genesis. Single proposer = our signer.
	gen := genesis.NewGenesis(f.chainID, 1, time.Now().UTC(), signerAddr)
	if err := gen.Validate(); err != nil {
		return fmt.Errorf("invalid genesis: %w", err)
	}

	// 4) ev-node config.
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

	if err := cfg.DA.Fiber.Validate(); err != nil {
		return fmt.Errorf("fiber config: %w", err)
	}

	// 5) Datastore + node-key + executor + sequencer. The first three
	// look identical to what testapp/cmd/run.go does; the executor
	// is the bench-specific in-memory variant (constant state root,
	// see executor.go for rationale) and the sequencer is solo (no
	// based-sequencer / no forced inclusion machinery).
	ds, err := store.NewDefaultKVStore(f.homeDir, cfg.DBPath, "fiber-bench")
	if err != nil {
		return fmt.Errorf("open datastore: %w", err)
	}
	nodeKey, err := loadOrGenNodeKey(filepath.Join(f.homeDir, "node-key.json"))
	if err != nil {
		return fmt.Errorf("node key: %w", err)
	}
	exec := newInMemExecutor(f.mempoolSize)
	seq := solo.NewSoloSequencer(logger, []byte(gen.ChainID), exec)

	// 6) Spawn loader + stats printer BEFORE cmd.StartNode (which
	// blocks). They run for the lifetime of the bench. cmd.StartNode
	// owns its own signal-handling goroutine; we send SIGINT to
	// ourselves when the duration timer expires so it can exit
	// through its normal shutdown path.
	bgCtx, bgCancel := signal.NotifyContext(cobraCmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer bgCancel()

	var loaderWg sync.WaitGroup
	loaderWg.Add(1)
	go func() {
		defer loaderWg.Done()
		newLoader(exec, f.workers, f.txSize).run(bgCtx)
	}()

	printer := newStatsPrinter(exec, f.prometheusAddr, f.txSize, fiberClient)
	printer.start(bgCtx, f.statsInterval)

	logger.Info().
		Dur("duration", f.duration).
		Int("workers", f.workers).
		Int("tx_size", f.txSize).
		Int("mempool", f.mempoolSize).
		Dur("block_time", f.blockTime).
		Str("batching", f.batchingStrategy).
		Msg("bench started")

	if f.duration > 0 {
		go func() {
			select {
			case <-time.After(f.duration):
				logger.Info().Msg("duration elapsed, sending SIGINT to trigger shutdown")
				_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
			case <-bgCtx.Done():
			}
		}()
	}

	// 7) The actual node — let cmd.StartNode do all the wiring
	// (signer load, DA client, p2p, node.NewNode, run loop with
	// shutdown). Same call testapp/evm/grpc apps make.
	startErr := rollcmd.StartNode(
		logger, cobraCmd, exec, seq, nodeKey, ds, cfg, gen,
		node.NodeOptions{}, fiberClient,
	)

	bgCancel()
	loaderWg.Wait()
	printer.printFinalSummary()

	if startErr != nil && !errors.Is(startErr, context.Canceled) {
		return startErr
	}
	return nil
}

// loadOrGenNodeKey is a tiny shim around pkg/p2p/key.LoadOrGenNodeKey,
// kept as a package-local helper so the bench can stay decoupled from
// changes to that helper's import path. The behaviour is identical.
func loadOrGenNodeKey(path string) (*key.NodeKey, error) {
	return key.LoadOrGenNodeKey(filepath.Dir(path))
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
