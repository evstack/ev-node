package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/evstack/ev-node/pkg/store"
)

// NewRestoreCmd creates a cobra command that restores a datastore from a Badger backup file.
func NewRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "restore",
		Short:        "Restore a datastore from a Badger backup file",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeConfig, err := ParseConfig(cmd)
			if err != nil {
				return fmt.Errorf("error parsing config: %w", err)
			}

			inputPath, err := cmd.Flags().GetString("input")
			if err != nil {
				return err
			}

			if inputPath == "" {
				return fmt.Errorf("--input flag is required")
			}

			inputPath, err = filepath.Abs(inputPath)
			if err != nil {
				return fmt.Errorf("failed to resolve input path: %w", err)
			}

			// Check if input file exists
			if _, err := os.Stat(inputPath); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("backup file not found: %s", inputPath)
				}
				return fmt.Errorf("failed to access backup file: %w", err)
			}

			// Check if datastore already exists
			force, err := cmd.Flags().GetBool("force")
			if err != nil {
				return err
			}

			dbPath := filepath.Join(nodeConfig.RootDir, nodeConfig.DBPath)
			if _, err := os.Stat(dbPath); err == nil && !force {
				return fmt.Errorf("datastore already exists at %s (use --force to overwrite)", dbPath)
			}

			// Remove existing datastore if force is enabled
			if force {
				if err := os.RemoveAll(dbPath); err != nil {
					return fmt.Errorf("failed to remove existing datastore: %w", err)
				}
			}

			// Create the datastore directory
			if err := os.MkdirAll(dbPath, 0o755); err != nil {
				return fmt.Errorf("failed to create datastore directory: %w", err)
			}

			appName, err := cmd.Flags().GetString("app-name")
			if err != nil {
				return err
			}

			if appName == "" {
				appName = "ev-node"
			}

			// Open the datastore
			kvStore, err := store.NewDefaultKVStore(nodeConfig.RootDir, nodeConfig.DBPath, appName)
			if err != nil {
				return fmt.Errorf("failed to open datastore: %w", err)
			}
			defer kvStore.Close()

			evStore := store.New(kvStore)

			// Open the backup file
			file, err := os.Open(inputPath)
			if err != nil {
				return fmt.Errorf("failed to open backup file: %w", err)
			}
			defer file.Close()

			reader := bufio.NewReaderSize(file, 1<<20) // 1 MiB buffer

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			cmd.Println("Restoring datastore from backup...")

			// Perform the restore
			if err := evStore.Restore(ctx, reader); err != nil {
				return fmt.Errorf("restore failed: %w", err)
			}

			cmd.Printf("Restore completed successfully\n")
			cmd.Printf("Datastore restored to: %s\n", dbPath)

			return nil
		},
	}

	cmd.Flags().String("input", "", "Path to the backup file (required)")
	cmd.Flags().String("app-name", "", "Application name for the datastore (default: ev-node)")
	cmd.Flags().Bool("force", false, "Overwrite existing datastore if it exists")

	return cmd
}
