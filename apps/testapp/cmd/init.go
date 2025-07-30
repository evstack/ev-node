package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	evnodecmd "github.com/evstack/ev-node/pkg/cmd"
	evnodeconf "github.com/evstack/ev-node/pkg/config"
	evnodegenesis "github.com/evstack/ev-node/pkg/genesis"
)

// InitCmd initializes a new ev-node.yaml file in the current directory
func InitCmd() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize ev-node config",
		Long:  fmt.Sprintf("This command initializes a new %s file in the specified directory (or current directory if not specified).", evnodeconf.ConfigName),
		RunE: func(cmd *cobra.Command, args []string) error {
			homePath, err := cmd.Flags().GetString(evnodeconf.FlagRootDir)
			if err != nil {
				return fmt.Errorf("error reading home flag: %w", err)
			}

			aggregator, err := cmd.Flags().GetBool(evnodeconf.FlagAggregator)
			if err != nil {
				return fmt.Errorf("error reading aggregator flag: %w", err)
			}

			// ignore error, as we are creating a new config
			// we use load in order to parse all the flags
			cfg, _ := evnodeconf.Load(cmd)
			cfg.Node.Aggregator = aggregator
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("error validating config: %w", err)
			}

			passphrase, err := cmd.Flags().GetString(evnodeconf.FlagSignerPassphrase)
			if err != nil {
				return fmt.Errorf("error reading passphrase flag: %w", err)
			}

			proposerAddress, err := evnodecmd.CreateSigner(&cfg, homePath, passphrase)
			if err != nil {
				return err
			}

			if err := cfg.SaveAsYaml(); err != nil {
				return fmt.Errorf("error writing ev-node.yaml file: %w", err)
			}

			if err := evnodecmd.LoadOrGenNodeKey(homePath); err != nil {
				return err
			}

			// get chain ID or use default
			chainID, _ := cmd.Flags().GetString(evnodeconf.FlagChainID)
			if chainID == "" {
				chainID = "evnode-test"
			}

			// Initialize genesis without app state
			err = evnodegenesis.CreateGenesis(homePath, chainID, 1, proposerAddress)
			genesisPath := evnodegenesis.GenesisPath(homePath)
			if errors.Is(err, evnodegenesis.ErrGenesisExists) {
				// check if existing genesis file is valid
				if genesis, err := evnodegenesis.LoadGenesis(genesisPath); err == nil {
					if err := genesis.Validate(); err != nil {
						return fmt.Errorf("existing genesis file is invalid: %w", err)
					}
				} else {
					return fmt.Errorf("error loading existing genesis file: %w", err)
				}

				cmd.Printf("Genesis file already exists at %s, skipping creation.\n", genesisPath)
			} else if err != nil {
				return fmt.Errorf("error initializing genesis file: %w", err)
			}

			cmd.Printf("Successfully initialized config file at %s\n", cfg.ConfigPath())
			return nil
		},
	}

	// Add flags to the command
	evnodeconf.AddFlags(initCmd)

	return initCmd
}
