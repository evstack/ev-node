package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	goheader "github.com/celestiaorg/go-header"
	goheaderstore "github.com/celestiaorg/go-header/store"
	ds "github.com/ipfs/go-datastore"
	kt "github.com/ipfs/go-datastore/keytransform"
	"github.com/spf13/cobra"

	"github.com/evstack/ev-node/node"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

// UnsafeCleanDataDir removes all contents of the specified data directory.
// It does not remove the data directory itself, only its contents.
func UnsafeCleanDataDir(dataDir string) error {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Data directory does not exist, nothing to clean.
			return nil
		}
		return fmt.Errorf("failed to read data directory: %w", err)
	}
	for _, entry := range entries {
		entryPath := filepath.Join(dataDir, entry.Name())
		err := os.RemoveAll(entryPath)
		if err != nil {
			return fmt.Errorf("failed to remove %s: %w", entryPath, err)
		}
	}
	return nil
}

// StoreUnsafeCleanCmd is a Cobra command that removes all contents of the data directory.
var StoreUnsafeCleanCmd = &cobra.Command{
	Use:   "unsafe-clean",
	Short: "Remove all contents of the data directory (DANGEROUS: cannot be undone)",
	Long: `Removes all files and subdirectories in the node's data directory.
This operation is unsafe and cannot be undone. Use with caution!`,
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeConfig, err := ParseConfig(cmd)
		if err != nil {
			return fmt.Errorf("error parsing config: %w", err)
		}
		dataDir := filepath.Join(nodeConfig.RootDir, nodeConfig.DBPath)
		fmt.Println("Data directory:", dataDir)
		if dataDir == "" {
			return fmt.Errorf("data directory not found in node configuration")
		}

		if err := UnsafeCleanDataDir(dataDir); err != nil {
			return err
		}
		cmd.Printf("All contents of the data directory at %s have been removed.\n", dataDir)
		return nil
	},
}

// StoreP2PInspectCmd reports head/tail information for the go-header stores used by P2P sync.
var StoreP2PInspectCmd = &cobra.Command{
	Use:   "store-info",
	Short: "Inspect the go-header (P2P) stores and display their tail/head entries",
	Long: `Opens the datastore used by the node's go-header services and reports
the current height, head, and tail information for both the header and data stores.
The datastore is opened in read-only mode so that it can run against mounted snapshots
or other read-only environments without requiring write access.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeConfig, err := ParseConfig(cmd)
		if err != nil {
			return fmt.Errorf("error parsing config: %w", err)
		}

		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		dbName := resolveDBName(cmd)

		rawStore, err := store.NewDefaultReadOnlyKVStore(nodeConfig.RootDir, nodeConfig.DBPath, dbName)
		if err != nil {
			return fmt.Errorf("failed to open datastore: %w", err)
		}
		defer func() {
			if closeErr := rawStore.Close(); closeErr != nil {
				cmd.PrintErrf("warning: failed to close datastore: %v\n", closeErr)
			}
		}()

		mainStore := kt.Wrap(rawStore, &kt.PrefixTransform{
			Prefix: ds.NewKey(node.EvPrefix),
		})

		headerSnapshot, err := inspectP2PStore[*types.SignedHeader](ctx, mainStore, headerStorePrefix, "Header Store")
		if err != nil {
			return fmt.Errorf("failed to inspect header store: %w", err)
		}

		dataSnapshot, err := inspectP2PStore[*types.Data](ctx, mainStore, dataStorePrefix, "Data Store")
		if err != nil {
			return fmt.Errorf("failed to inspect data store: %w", err)
		}

		storePath := resolveStorePath(nodeConfig.RootDir, nodeConfig.DBPath, dbName)

		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "Inspecting go-header stores at %s\n", storePath)
		printP2PStoreSnapshot(cmd, headerSnapshot)
		printP2PStoreSnapshot(cmd, dataSnapshot)

		return nil
	},
}

const (
	headerStorePrefix = "headerSync"
	dataStorePrefix   = "dataSync"
)

type p2pStoreSnapshot struct {
	Label       string
	Prefix      string
	Height      uint64
	HeadHeight  uint64
	HeadHash    string
	HeadTime    time.Time
	TailHeight  uint64
	TailHash    string
	TailTime    time.Time
	HeadPresent bool
	TailPresent bool
	Empty       bool
}

func inspectP2PStore[H goheader.Header[H]](
	ctx context.Context,
	datastore ds.Batching,
	prefix string,
	label string,
) (p2pStoreSnapshot, error) {
	storeImpl, err := goheaderstore.NewStore[H](
		datastore,
		goheaderstore.WithStorePrefix(prefix),
		goheaderstore.WithMetrics(),
	)
	if err != nil {
		return p2pStoreSnapshot{}, fmt.Errorf("failed to open %s: %w", label, err)
	}

	if err := storeImpl.Start(ctx); err != nil {
		return p2pStoreSnapshot{}, fmt.Errorf("failed to start %s: %w", label, err)
	}
	defer func() {
		_ = storeImpl.Stop(context.Background())
	}()

	snapshot := p2pStoreSnapshot{
		Label:  label,
		Prefix: prefix,
		Height: storeImpl.Height(),
	}

	if err := populateSnapshot(ctx, storeImpl, &snapshot); err != nil {
		return p2pStoreSnapshot{}, err
	}

	return snapshot, nil
}

func populateSnapshot[H goheader.Header[H]](
	ctx context.Context,
	storeImpl *goheaderstore.Store[H],
	snapshot *p2pStoreSnapshot,
) error {
	head, err := storeImpl.Head(ctx)
	switch {
	case err == nil:
		snapshot.HeadPresent = true
		snapshot.HeadHeight = head.Height()
		snapshot.HeadHash = head.Hash().String()
		snapshot.HeadTime = head.Time()
	case errors.Is(err, goheader.ErrEmptyStore), errors.Is(err, goheader.ErrNotFound):
		// store not initialized yet
	default:
		return fmt.Errorf("failed to read %s head: %w", snapshot.Label, err)
	}

	tail, err := storeImpl.Tail(ctx)
	switch {
	case err == nil:
		snapshot.TailPresent = true
		snapshot.TailHeight = tail.Height()
		snapshot.TailHash = tail.Hash().String()
		snapshot.TailTime = tail.Time()
	case errors.Is(err, goheader.ErrEmptyStore), errors.Is(err, goheader.ErrNotFound):
	default:
		return fmt.Errorf("failed to read %s tail: %w", snapshot.Label, err)
	}

	snapshot.Empty = !snapshot.HeadPresent && !snapshot.TailPresent

	return nil
}

func printP2PStoreSnapshot(cmd *cobra.Command, snapshot p2pStoreSnapshot) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "\n[%s]\n", snapshot.Label)
	fmt.Fprintf(out, "prefix: %s\n", snapshot.Prefix)
	fmt.Fprintf(out, "height: %d\n", snapshot.Height)
	if snapshot.Empty {
		fmt.Fprintln(out, "status: empty (no entries found)")
		return
	}

	if snapshot.TailPresent {
		fmt.Fprintf(out, "tail:  height=%d hash=%s%s\n", snapshot.TailHeight, snapshot.TailHash, formatTime(snapshot.TailTime))
	}
	if snapshot.HeadPresent {
		fmt.Fprintf(out, "head:  height=%d hash=%s%s\n", snapshot.HeadHeight, snapshot.HeadHash, formatTime(snapshot.HeadTime))
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return fmt.Sprintf(" time=%s", t.UTC().Format(time.RFC3339))
}

func resolveDBName(cmd *cobra.Command) string {
	if cmd == nil {
		return config.ConfigFileName
	}
	root := cmd.Root()
	if root == nil || root.Name() == "" {
		return config.ConfigFileName
	}
	return root.Name()
}

func resolveStorePath(rootDir, dbPath, dbName string) string {
	base := dbPath
	if !filepath.IsAbs(dbPath) {
		base = filepath.Join(rootDir, dbPath)
	}
	return filepath.Join(base, dbName)
}
