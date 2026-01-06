package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/evstack/ev-node/block"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/node"
	rollconf "github.com/evstack/ev-node/pkg/config"
	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/signer/file"
	"github.com/evstack/ev-node/pkg/telemetry"
)

// ParseConfig is an helpers that loads the node configuration and validates it.
func ParseConfig(cmd *cobra.Command) (rollconf.Config, error) {
	nodeConfig, err := rollconf.Load(cmd)
	if err != nil {
		return rollconf.Config{}, fmt.Errorf("failed to load node config: %w", err)
	}

	if err := nodeConfig.Validate(); err != nil {
		return rollconf.Config{}, fmt.Errorf("failed to validate node config: %w", err)
	}

	return nodeConfig, nil
}

// SetupLogger configures and returns a logger based on the provided configuration.
// It applies the following settings from the config:
//   - Log format (text or JSON)
//   - Log level (debug, info, warn, error)
//   - Stack traces for error logs
//
// The returned logger is already configured with the "module" field set to "main".
func SetupLogger(config rollconf.LogConfig) zerolog.Logger {
	// Configure output
	var output = os.Stderr

	// Configure logger format
	var logger zerolog.Logger
	if config.Format == "json" {
		logger = zerolog.New(output)
	} else {
		logger = zerolog.New(zerolog.ConsoleWriter{Out: output})
	}

	// Configure logger level
	level, err := zerolog.ParseLevel(config.Level)
	if err != nil {
		// Default to info if parsing fails
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Add timestamp and set up logger with component
	logger = logger.With().Timestamp().Str("component", "main").Logger()

	return logger
}

// StartNode handles the node startup logic
func StartNode(
	logger zerolog.Logger,
	cmd *cobra.Command,
	executor coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	p2pClient *p2p.Client,
	datastore datastore.Batching,
	nodeConfig rollconf.Config,
	genesis genesispkg.Genesis,
	nodeOptions node.NodeOptions,
) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	if nodeConfig.Instrumentation.IsTracingEnabled() {
		shutdownTracing, err := telemetry.InitTracing(ctx, nodeConfig.Instrumentation, logger)
		if err != nil {
			return fmt.Errorf("failed to initialize tracing: %w", err)
		}
		defer func() {
			// best-effort shutdown within a short timeout
			c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = shutdownTracing(c)
		}()
	}

	blobClient, err := blobrpc.NewClient(ctx, nodeConfig.DA.Address, nodeConfig.DA.AuthToken, "")
	if err != nil {
		return fmt.Errorf("failed to create blob client: %w", err)
	}
	defer blobClient.Close()
	daClient := block.NewDAClient(blobClient, nodeConfig, logger)

	// create a new remote signer
	var signer signer.Signer
	if nodeConfig.Signer.SignerType == "file" && (nodeConfig.Node.Aggregator && !nodeConfig.Node.BasedSequencer) {
		// Get passphrase file path
		passphraseFile, err := cmd.Flags().GetString(rollconf.FlagSignerPassphraseFile)
		if err != nil {
			return fmt.Errorf("failed to get '%s' flag: %w", rollconf.FlagSignerPassphraseFile, err)
		}

		if passphraseFile == "" {
			return fmt.Errorf("passphrase file must be provided via --evnode.signer.passphrase_file")
		}

		// Read passphrase from file
		passphraseBytes, err := os.ReadFile(passphraseFile)
		if err != nil {
			return fmt.Errorf("failed to read passphrase from file '%s': %w", passphraseFile, err)
		}
		passphrase := strings.TrimSpace(string(passphraseBytes))

		if passphrase == "" {
			return fmt.Errorf("passphrase file '%s' is empty", passphraseFile)
		}

		// Resolve signer path; allow absolute, relative to node root, or relative to CWD if resolution fails
		signerPath, err := filepath.Abs(strings.TrimSuffix(nodeConfig.Signer.SignerPath, "signer.json"))
		if err != nil {
			return err
		}

		signer, err = file.LoadFileSystemSigner(signerPath, []byte(passphrase))
		if err != nil {
			return err
		}
	} else if nodeConfig.Node.Aggregator && nodeConfig.Signer.SignerType != "file" {
		return fmt.Errorf("unknown signer type: %s", nodeConfig.Signer.SignerType)
	}

	// sanity check for based sequencer
	if nodeConfig.Node.BasedSequencer && genesis.DAStartHeight == 0 {
		return fmt.Errorf("based sequencing requires DAStartHeight to be set in genesis. This value should be identical for all nodes of the chain")
	}

	metrics := node.DefaultMetricsProvider(nodeConfig.Instrumentation)

	// wrap executor with tracing decorator if tracing enabled
	if nodeConfig.Instrumentation.IsTracingEnabled() {
		executor = coreexecutor.WithTracing(executor)
	}

	// Create and start the node
	rollnode, err := node.NewNode(
		nodeConfig,
		executor,
		sequencer,
		daClient,
		signer,
		p2pClient,
		genesis,
		datastore,
		metrics,
		logger,
		nodeOptions,
	)
	if err != nil {
		return fmt.Errorf("failed to create node: %w", err)
	}

	// Run the node with graceful shutdown
	errCh := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 1024)
				n := runtime.Stack(buf, false)
				err := fmt.Errorf("node panicked: %v\nstack trace:\n%s", r, buf[:n])
				logger.Error().Interface("panic", r).Str("stacktrace", string(buf[:n])).Msg("Recovered from panic in node")
				select {
				case errCh <- err:
				default:
					logger.Error().Err(err).Msg("Error channel full")
				}
			}
		}()

		err := rollnode.Run(ctx)
		select {
		case errCh <- err:
		default:
			logger.Error().Err(err).Msg("Error channel full")
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Info().Msg("shutting down node...")
		cancel()
	case err := <-errCh:
		logger.Error().Err(err).Msg("node error")
		cancel()
		return err
	}

	// Wait for node to finish shutting down
	select {
	case <-time.After(5 * time.Second):
		logger.Info().Msg("Node shutdown timed out")
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error().Err(err).Msg("Error during shutdown")
			return err
		}
	}

	return nil
}
