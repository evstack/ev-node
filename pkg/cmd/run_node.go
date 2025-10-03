package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	coreda "github.com/evstack/ev-node/core/da"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/node"
	rollconf "github.com/evstack/ev-node/pkg/config"
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/signer/file"
)

const DefaultMaxBlobSize = 1.5 * 1024 * 1024 // 1.5MB

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
	da coreda.DA,
	p2pClient *p2p.Client,
	datastore datastore.Batching,
	nodeConfig rollconf.Config,
	genesis genesispkg.Genesis,
	nodeOptions node.NodeOptions,
) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// create a new remote signer
	var signer signer.Signer
	if nodeConfig.Signer.SignerType == "file" && nodeConfig.Node.Aggregator {
		passphrase, err := cmd.Flags().GetString(rollconf.FlagSignerPassphrase)
		if err != nil {
			return err
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

	metrics := node.DefaultMetricsProvider(nodeConfig.Instrumentation)

	// Create and start the node
	rollnode, err := node.NewNode(
		nodeConfig,
		executor,
		sequencer,
		da,
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
				err := fmt.Errorf("node panicked: %v", r)
				logger.Error().Interface("panic", r).Msg("Recovered from panic in node")
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
