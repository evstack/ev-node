package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	goheaderstore "github.com/celestiaorg/go-header/store"
	"github.com/evstack/ev-node/execution/evm"
	rollcmd "github.com/evstack/ev-node/pkg/cmd"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

// NewRollbackCmd creates a command to rollback ev-node state by one height.
func NewRollbackCmd() *cobra.Command {
	var (
		height   uint64
		syncNode bool
	)

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "rollback ev-node state by one height. Pass --height to specify another height to rollback to.",
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeConfig, err := rollcmd.ParseConfig(cmd)
			if err != nil {
				return err
			}

			goCtx := cmd.Context()
			if goCtx == nil {
				goCtx = context.Background()
			}

			// evolve db
			rawEvolveDB, err := store.NewDefaultKVStore(nodeConfig.RootDir, nodeConfig.DBPath, evmDbName)
			if err != nil {
				return err
			}

			defer func() {
				if closeErr := rawEvolveDB.Close(); closeErr != nil {
					cmd.Printf("Warning: failed to close evolve database: %v\n", closeErr)
				}
			}()

			// prefixed evolve db
			evolveDB := store.NewEvNodeKVStore(rawEvolveDB)
			evolveStore := store.New(evolveDB)
			if height == 0 {
				currentHeight, err := evolveStore.Height(goCtx)
				if err != nil {
					return err
				}

				height = currentHeight - 1
			}

			// rollback ev-node main state
			if err := evolveStore.Rollback(goCtx, height, !syncNode); err != nil {
				return fmt.Errorf("failed to rollback ev-node state: %w", err)
			}

			// rollback execution layer via EngineClient
			engineClient, err := createRollbackEngineClient(cmd, rawEvolveDB)
			if err != nil {
				cmd.Printf("Warning: failed to create engine client, skipping EL rollback: %v\n", err)
			} else {
				if err := engineClient.Rollback(goCtx, height); err != nil {
					return fmt.Errorf("failed to rollback execution layer: %w", err)
				}
				cmd.Printf("Rolled back execution layer to height %d\n", height)
			}

			// rollback ev-node goheader state
			headerStore, err := goheaderstore.NewStore[*types.SignedHeader](
				evolveDB,
				goheaderstore.WithStorePrefix("headerSync"),
				goheaderstore.WithMetrics(),
			)
			if err != nil {
				return err
			}

			dataStore, err := goheaderstore.NewStore[*types.Data](
				evolveDB,
				goheaderstore.WithStorePrefix("dataSync"),
				goheaderstore.WithMetrics(),
			)
			if err != nil {
				return err
			}

			if err := headerStore.Start(goCtx); err != nil {
				return fmt.Errorf("failed to start header store: %w", err)
			}
			defer headerStore.Stop(goCtx)

			if err := dataStore.Start(goCtx); err != nil {
				return fmt.Errorf("failed to start data store: %w", err)
			}
			defer dataStore.Stop(goCtx)

			var errs error
			if err := headerStore.DeleteRange(goCtx, height+1, headerStore.Height()); err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to rollback header sync service state: %w", err))
			}

			if err := dataStore.DeleteRange(goCtx, height+1, dataStore.Height()); err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to rollback data sync service state: %w", err))
			}

			cmd.Printf("Rolled back ev-node state to height %d\n", height)
			if syncNode {
				fmt.Println("Restart the node with the `--evnode.clear_cache` flag")
			}

			return errs
		},
	}

	cmd.Flags().Uint64Var(&height, "height", 0, "rollback to a specific height")
	cmd.Flags().BoolVar(&syncNode, "sync-node", false, "sync node (no aggregator)")

	// EVM flags for execution layer rollback
	cmd.Flags().String(evm.FlagEvmEthURL, "http://localhost:8545", "URL of the Ethereum JSON-RPC endpoint")
	cmd.Flags().String(evm.FlagEvmEngineURL, "http://localhost:8551", "URL of the Engine API endpoint")
	cmd.Flags().String(evm.FlagEvmJWTSecretFile, "", "Path to file containing the JWT secret for authentication")

	return cmd
}

func createRollbackEngineClient(cmd *cobra.Command, db ds.Batching) (*evm.EngineClient, error) {
	ethURL, err := cmd.Flags().GetString(evm.FlagEvmEthURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get '%s' flag: %w", evm.FlagEvmEthURL, err)
	}
	engineURL, err := cmd.Flags().GetString(evm.FlagEvmEngineURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get '%s' flag: %w", evm.FlagEvmEngineURL, err)
	}

	jwtSecretFile, err := cmd.Flags().GetString(evm.FlagEvmJWTSecretFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get '%s' flag: %w", evm.FlagEvmJWTSecretFile, err)
	}

	if jwtSecretFile == "" {
		return nil, fmt.Errorf("JWT secret file must be provided via --evm.jwt-secret-file for EL rollback")
	}

	secretBytes, err := os.ReadFile(jwtSecretFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read JWT secret from file '%s': %w", jwtSecretFile, err)
	}
	jwtSecret := string(bytes.TrimSpace(secretBytes))

	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT secret file '%s' is empty", jwtSecretFile)
	}

	return evm.NewEngineExecutionClient(ethURL, engineURL, jwtSecret, common.Hash{}, common.Address{}, db, false, zerolog.Nop())
}
