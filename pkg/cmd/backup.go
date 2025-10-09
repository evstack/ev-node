package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	clientrpc "github.com/evstack/ev-node/pkg/rpc/client"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// NewBackupCmd creates a cobra command that streams a datastore backup via the RPC client.
func NewBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "backup",
		Short:        "Stream a datastore backup to a local file via RPC",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeConfig, err := ParseConfig(cmd)
			if err != nil {
				return fmt.Errorf("error parsing config: %w", err)
			}

			rpcAddress := strings.TrimSpace(nodeConfig.RPC.Address)
			if rpcAddress == "" {
				return fmt.Errorf("RPC address not found in node configuration")
			}

			baseURL := rpcAddress
			if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
				baseURL = fmt.Sprintf("http://%s", baseURL)
			}

			outputPath, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}

			if outputPath == "" {
				timestamp := time.Now().UTC().Format("20060102-150405")
				outputPath = fmt.Sprintf("evnode-backup-%s.badger", timestamp)
			}

			outputPath, err = filepath.Abs(outputPath)
			if err != nil {
				return fmt.Errorf("failed to resolve output path: %w", err)
			}

			if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}

			force, err := cmd.Flags().GetBool("force")
			if err != nil {
				return err
			}

			if !force {
				if _, statErr := os.Stat(outputPath); statErr == nil {
					return fmt.Errorf("output file %s already exists (use --force to overwrite)", outputPath)
				} else if !errors.Is(statErr, os.ErrNotExist) {
					return fmt.Errorf("failed to inspect output file: %w", statErr)
				}
			}

			file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
			if err != nil {
				return fmt.Errorf("failed to open output file: %w", err)
			}
			defer file.Close()

			writer := bufio.NewWriterSize(file, 1<<20) // 1 MiB buffer for fewer syscalls.
			bytesCount := &countingWriter{}
			streamWriter := io.MultiWriter(writer, bytesCount)

			targetHeight, err := cmd.Flags().GetUint64("target-height")
			if err != nil {
				return err
			}

			sinceVersion, err := cmd.Flags().GetUint64("since-version")
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			client := clientrpc.NewClient(baseURL)

			metadata, backupErr := client.Backup(ctx, &pb.BackupRequest{
				TargetHeight: targetHeight,
				SinceVersion: sinceVersion,
			}, streamWriter)
			if backupErr != nil {
				// Remove the partial file on failure to avoid keeping corrupt snapshots.
				_ = writer.Flush()
				_ = file.Close()
				_ = os.Remove(outputPath)
				return fmt.Errorf("backup failed: %w", backupErr)
			}

			if err := writer.Flush(); err != nil {
				_ = file.Close()
				_ = os.Remove(outputPath)
				return fmt.Errorf("failed to flush backup data: %w", err)
			}

			if !metadata.GetCompleted() {
				_ = file.Close()
				_ = os.Remove(outputPath)
				return fmt.Errorf("backup stream ended without completion metadata")
			}

			cmd.Printf("Backup saved to %s (%d bytes)\n", outputPath, bytesCount.Bytes())
			cmd.Printf("Current height: %d\n", metadata.GetCurrentHeight())
			cmd.Printf("Target height: %d\n", metadata.GetTargetHeight())
			cmd.Printf("Since version: %d\n", metadata.GetSinceVersion())
			cmd.Printf("Last version: %d\n", metadata.GetLastVersion())

			return nil
		},
	}

	cmd.Flags().String("output", "", "Path to the backup file (defaults to ./evnode-backup-<timestamp>.badger)")
	cmd.Flags().Uint64("target-height", 0, "Target chain height to align the backup with (0 disables the check)")
	cmd.Flags().Uint64("since-version", 0, "Generate an incremental backup starting from the provided version")
	cmd.Flags().Bool("force", false, "Overwrite the output file if it already exists")

	return cmd
}

type countingWriter struct {
	total int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	c.total += int64(len(p))
	return len(p), nil
}

func (c *countingWriter) Bytes() int64 {
	return c.total
}
