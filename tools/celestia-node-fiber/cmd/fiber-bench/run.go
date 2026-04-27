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

	"github.com/spf13/cobra"

	"github.com/evstack/ev-node/node"
	rollcmd "github.com/evstack/ev-node/pkg/cmd"
	rollconf "github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p/key"
	"github.com/evstack/ev-node/pkg/sequencers/solo"
	"github.com/evstack/ev-node/pkg/signer/file"
	"github.com/evstack/ev-node/pkg/store"
)

// Bench-local flag names. The rest come from rollconf.AddFlags
// (--evnode.da.fiber.consensus_address, --evnode.da.batching_strategy, …)
// and rollconf.AddGlobalFlags (--home, --log.level, …).
const (
	flagKeyringDir       = "keyring-dir"
	flagKeepHome         = "keep-home"
	flagDuration         = "duration"
	flagWorkers          = "workers"
	flagTxSize           = "tx-size"
	flagMempoolSize      = "mempool-size"
	flagStatsInterval    = "stats-interval"
	flagSignerPassphrase = "signer-passphrase"
)

func runCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the bench: start a single-sequencer ev-node against a Fibre network and pump load",
		RunE:  runBench,
	}

	// Canonical ev-node flags: --evnode.node.*, --evnode.da.*,
	// --evnode.da.fiber.*, --evnode.signer.*, --evnode.instrumentation.*,
	// --evnode.p2p.*, --evnode.signer.passphrase_file, etc. The bench
	// applies opinionated defaults post-parse for the ones a thoughtful
	// operator would otherwise have to flip every run (see runBench).
	rollconf.AddFlags(cmd)

	flags := cmd.Flags()
	flags.String(flagKeyringDir, defaultKeyringDir(), "directory holding the bench cosmos keyring (test backend) used to sign Fibre payment promises")
	flags.Bool(flagKeepHome, false, "do not wipe the ev-node home before starting (resumes prior state)")
	flags.Duration(flagDuration, 60*time.Second, "how long to run the bench before stopping (0 = until SIGINT)")
	flags.Int(flagWorkers, 32, "number of concurrent tx-injection goroutines")
	flags.Int(flagTxSize, 200, "size of each generated tx in bytes")
	flags.Int(flagMempoolSize, 1_000_000, "size of the in-mem executor's mempool channel (backpressure boundary)")
	flags.Duration(flagStatsInterval, time.Second, "how often to print a stats line")
	flags.String(flagSignerPassphrase, "fiber-bench-passphrase", "passphrase for the ev-node file signer (block-signing key, NOT the cosmos one). Written to a temp file consumed by --evnode.signer.passphrase_file.")

	// Fibre consensus address/chain ID don't have empty defaults
	// (DefaultConfig points at 127.0.0.1:9090 / mocha-4), but those
	// values are sentinels — running the bench against them is never
	// what the operator wants. Force them through.
	_ = cobra.MarkFlagRequired(flags, rollconf.FlagDAFiberConsensusAddress)
	_ = cobra.MarkFlagRequired(flags, rollconf.FlagDAFiberConsensusChainID)

	return cmd
}

func runBench(cobraCmd *cobra.Command, _ []string) error {
	cfg, err := rollcmd.ParseConfig(cobraCmd)
	if err != nil {
		return err
	}
	applyBenchDefaults(cobraCmd, &cfg)

	// Re-validate after the bench's overrides — ParseConfig already ran
	// once on parse, but we mutated Aggregator/Fiber/etc. afterwards.
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config invalid after bench overrides: %w", err)
	}

	logger := rollcmd.SetupLogger(cfg.Log)

	keyringDir, _ := cobraCmd.Flags().GetString(flagKeyringDir)
	keepHome, _ := cobraCmd.Flags().GetBool(flagKeepHome)
	duration, _ := cobraCmd.Flags().GetDuration(flagDuration)
	workers, _ := cobraCmd.Flags().GetInt(flagWorkers)
	txSize, _ := cobraCmd.Flags().GetInt(flagTxSize)
	mempoolSize, _ := cobraCmd.Flags().GetInt(flagMempoolSize)
	statsInterval, _ := cobraCmd.Flags().GetDuration(flagStatsInterval)
	signerPassphrase, _ := cobraCmd.Flags().GetString(flagSignerPassphrase)

	if !keepHome {
		_ = os.RemoveAll(cfg.RootDir)
	}
	if err := os.MkdirAll(cfg.RootDir, 0o755); err != nil {
		return fmt.Errorf("create home %s: %w", cfg.RootDir, err)
	}

	// 1) Cosmos keyring + bridge-bypass Fibre adapter — the two genuinely
	// fiber-bench-specific pieces. Neither lives in the production wiring
	// path.
	kr, err := openKeyring(keyringDir)
	if err != nil {
		return fmt.Errorf("open keyring at %s: %w", keyringDir, err)
	}
	rec, err := kr.Key(cfg.DA.Fiber.KeyName)
	if err != nil {
		return fmt.Errorf("key %q not found in keyring %s — run `fiber-bench keys add %s` first: %w",
			cfg.DA.Fiber.KeyName, keyringDir, cfg.DA.Fiber.KeyName, err)
	}
	addr, err := rec.GetAddress()
	if err != nil {
		return fmt.Errorf("derive key address: %w", err)
	}
	logger.Info().Str("address", addr.String()).Str("key", cfg.DA.Fiber.KeyName).Msg("loaded fibre signing key")

	logger.Info().Str("grpc", cfg.DA.Fiber.ConsensusAddress).Msg("dialing consensus gRPC")
	innerFiberClient, fiberClose, err := buildFibreAdapter(cobraCmd.Context(), cfg.DA.Fiber.ConsensusAddress, cfg.DA.Fiber.KeyName, kr)
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

	// 2) ev-node block-signing key. Created in cfg.Signer.SignerPath if
	// missing. cmd.StartNode reads the passphrase from the path stored
	// in --evnode.signer.passphrase_file; we write a temp file from
	// --signer-passphrase and inject the flag value so the canonical
	// signer-loading path works without us asking the operator to manage
	// a passphrase file by hand.
	signerDir := cfg.Signer.SignerPath
	if signerDir == "" {
		signerDir = filepath.Join(cfg.RootDir, "config")
	}
	if !filepath.IsAbs(signerDir) {
		signerDir = filepath.Join(cfg.RootDir, signerDir)
	}
	cfg.Signer.SignerPath = signerDir
	if err := os.MkdirAll(signerDir, 0o750); err != nil {
		return fmt.Errorf("create signer dir: %w", err)
	}
	signerFile := filepath.Join(signerDir, "signer.json")
	if _, statErr := os.Stat(signerFile); os.IsNotExist(statErr) {
		s, err := file.CreateFileSystemSigner(signerDir, []byte(signerPassphrase))
		if err != nil {
			return fmt.Errorf("create file signer: %w", err)
		}
		if _, err := s.GetAddress(); err != nil {
			return fmt.Errorf("signer address: %w", err)
		}
	}
	passphraseFile := filepath.Join(cfg.RootDir, "passphrase.txt")
	if err := os.WriteFile(passphraseFile, []byte(signerPassphrase), 0o600); err != nil {
		return fmt.Errorf("write passphrase file: %w", err)
	}
	if err := cobraCmd.Flags().Set(rollconf.FlagSignerPassphraseFile, passphraseFile); err != nil {
		return fmt.Errorf("set passphrase flag: %w", err)
	}

	// Reload the signer to derive the genesis proposer address.
	loaded, err := file.LoadFileSystemSigner(signerDir, []byte(signerPassphrase))
	if err != nil {
		return fmt.Errorf("load file signer: %w", err)
	}
	signerAddr, err := loaded.GetAddress()
	if err != nil {
		return fmt.Errorf("signer address: %w", err)
	}

	// 3) Genesis. Single proposer = our signer.
	gen := genesis.NewGenesis(cfg.DA.Fiber.ConsensusChainID, 1, time.Now().UTC(), signerAddr)
	if err := gen.Validate(); err != nil {
		return fmt.Errorf("invalid genesis: %w", err)
	}

	// 4) Datastore + node-key + executor + sequencer. The first three
	// look identical to what testapp/cmd/run.go does; the executor
	// is the bench-specific in-memory variant (constant state root,
	// see executor.go for rationale) and the sequencer is solo (no
	// based-sequencer / no forced inclusion machinery).
	ds, err := store.NewDefaultKVStore(cfg.RootDir, cfg.DBPath, "fiber-bench")
	if err != nil {
		return fmt.Errorf("open datastore: %w", err)
	}
	// Match canonical layout: node_key.json under <home>/config/, the
	// same dir testapp/cmd/run.go reads it from.
	nodeKey, err := key.LoadOrGenNodeKey(filepath.Dir(cfg.ConfigPath()))
	if err != nil {
		return fmt.Errorf("node key: %w", err)
	}
	exec := newInMemExecutor(mempoolSize)
	seq := solo.NewSoloSequencer(logger, []byte(gen.ChainID), exec)

	// 5) Spawn loader + stats printer BEFORE cmd.StartNode (which
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
		newLoader(exec, workers, txSize).run(bgCtx)
	}()

	printer := newStatsPrinter(exec, cfg.Instrumentation.PrometheusListenAddr, txSize, fiberClient)
	printer.start(bgCtx, statsInterval)

	logger.Info().
		Dur("duration", duration).
		Int("workers", workers).
		Int("tx_size", txSize).
		Int("mempool", mempoolSize).
		Dur("block_time", cfg.Node.BlockTime.Duration).
		Str("batching", cfg.DA.BatchingStrategy).
		Msg("bench started")

	if duration > 0 {
		go func() {
			select {
			case <-time.After(duration):
				logger.Info().Msg("duration elapsed, sending SIGINT to trigger shutdown")
				_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
			case <-bgCtx.Done():
			}
		}()
	}

	// 6) The actual node — let cmd.StartNode do all the wiring (signer
	// load, DA client, p2p, node.NewNode, run loop with shutdown). Same
	// call testapp/evm/grpc apps make.
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

// applyBenchDefaults overrides config fields that the bench needs forced
// (Aggregator, Fiber.Enabled) and the canonical defaults that are wrong
// for a throughput bench (DA block time, batching strategy, scrape
// interval, namespaces). Anything the operator passed on the command line
// is left untouched — we only override where the flag value still equals
// its canonical default.
func applyBenchDefaults(cmd *cobra.Command, cfg *rollconf.Config) {
	// Forced for the bench: aggregator-only, Fibre DA, no P2P.
	cfg.Node.Aggregator = true
	cfg.Node.BasedSequencer = false
	cfg.DA.Fiber.Enabled = true
	if cfg.DA.Fiber.BridgeAddress == "" {
		// FiberDAConfig.Validate requires a ws:// or wss:// address.
		// Bench never dials it (see fibre.go: noBridgeBlob).
		cfg.DA.Fiber.BridgeAddress = "ws://127.0.0.1:0"
	}
	cfg.P2P.ListenAddress = "/ip4/127.0.0.1/tcp/0"
	cfg.P2P.DisableConnectionGater = true
	cfg.RPC.Address = "127.0.0.1:0"
	cfg.Signer.SignerType = "file"
	cfg.Instrumentation.Pprof = false
	// The stats printer scrapes /metrics every tick — keep Prometheus on
	// even if the operator didn't pass --evnode.instrumentation.prometheus.
	cfg.Instrumentation.Prometheus = true

	// Operator-overridable bench defaults — applied only if the canonical
	// flag wasn't passed on the command line.
	overrideIfUnchanged := func(name string, set func()) {
		if !cmd.Flags().Changed(name) {
			set()
		}
	}
	overrideIfUnchanged(rollconf.FlagDABlockTime, func() {
		cfg.DA.BlockTime = rollconf.DurationWrapper{Duration: time.Second}
	})
	overrideIfUnchanged(rollconf.FlagDABatchingStrategy, func() { cfg.DA.BatchingStrategy = "immediate" })
	overrideIfUnchanged(rollconf.FlagScrapeInterval, func() {
		cfg.Node.ScrapeInterval = rollconf.DurationWrapper{Duration: 100 * time.Millisecond}
	})
	overrideIfUnchanged(rollconf.FlagDARequestTimeout, func() {
		cfg.DA.RequestTimeout = rollconf.DurationWrapper{Duration: 60 * time.Second}
	})
	overrideIfUnchanged(rollconf.FlagDANamespace, func() { cfg.DA.Namespace = "fb-bench-h" })
	overrideIfUnchanged(rollconf.FlagDADataNamespace, func() { cfg.DA.DataNamespace = "fb-bench-d" })
}
