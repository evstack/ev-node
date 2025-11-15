package cmd

import (
	"bytes"
	"context"
	cryptoRand "crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	goheaderstore "github.com/celestiaorg/go-header/store"
	ds "github.com/ipfs/go-datastore"
	kt "github.com/ipfs/go-datastore/keytransform"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/node"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/signer/noop"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

func TestUnsafeCleanDataDir(t *testing.T) {
	tempDir := t.TempDir()

	// Create some test files and directories
	subDir := filepath.Join(tempDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0755))
	testFile := filepath.Join(tempDir, "testfile.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0o600))

	// Ensure the files and directories exist
	require.DirExists(t, subDir)
	require.FileExists(t, testFile)

	// Call the function to clean the directory
	err := UnsafeCleanDataDir(tempDir)
	require.NoError(t, err)

	// Ensure the directory is empty
	entries, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestStoreUnsafeCleanCmd(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	configDir := filepath.Join(tempDir, "config")
	configFile := filepath.Join(configDir, "evnode.yml")

	// Create necessary directories
	require.NoError(t, os.Mkdir(configDir, 0755))
	require.NoError(t, os.Mkdir(dataDir, 0755))

	// Create a dummy config file
	dummyConfig := `
	RootDir = "` + tempDir + `"
	DBPath = "data"
	`
	require.NoError(t, os.WriteFile(configFile, []byte(dummyConfig), 0o600))

	// Create some test files and directories inside dataDir
	testFile := filepath.Join(dataDir, "testfile.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0o600))

	// Ensure the files and directories exist
	require.FileExists(t, testFile)

	// Create a root command and add the subcommand
	rootCmd := &cobra.Command{Use: "root"}
	rootCmd.PersistentFlags().String("home", tempDir, "root directory")
	rootCmd.AddCommand(StoreUnsafeCleanCmd)

	// Capture output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Execute the command
	rootCmd.SetArgs([]string{"unsafe-clean"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	// Ensure the data directory is empty
	entries, err := os.ReadDir(dataDir)
	require.NoError(t, err)
	require.Empty(t, entries, "Data directory should be empty after clean")

	// Ensure the data directory itself still exists
	_, err = os.Stat(dataDir)
	require.NoError(t, err, "Data directory itself should still exist")

	// Check output message (optional)
	require.Contains(t, buf.String(), fmt.Sprintf("All contents of the data directory at %s have been removed.", dataDir))
}

func TestStoreP2PInspectCmd(t *testing.T) {
	tempDir := t.TempDir()
	const appName = "testapp"

	// Seed the header store with a couple of entries.
	seedHeaderStore(t, tempDir, appName)

	rootCmd := &cobra.Command{Use: appName}
	rootCmd.PersistentFlags().String(config.FlagRootDir, tempDir, "root directory")
	rootCmd.AddCommand(StoreP2PInspectCmd)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"store-info"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	require.Contains(t, output, "Inspecting go-header stores")
	require.Contains(t, output, "[Header Store]")
	require.Contains(t, output, "tail:  height=1")
	require.Contains(t, output, "head:  height=2")
	require.Contains(t, output, "[Data Store]")
	require.Contains(t, output, "status: empty")
}

func seedHeaderStore(t *testing.T, rootDir, dbName string) {
	t.Helper()

	rawStore, err := store.NewDefaultKVStore(rootDir, "data", dbName)
	require.NoError(t, err)

	mainStore := kt.Wrap(rawStore, &kt.PrefixTransform{
		Prefix: ds.NewKey(node.EvPrefix),
	})

	headerStore, err := goheaderstore.NewStore[*types.SignedHeader](
		mainStore,
		goheaderstore.WithStorePrefix(headerStorePrefix),
		goheaderstore.WithMetrics(),
	)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, headerStore.Start(ctx))

	defer func() {
		require.NoError(t, headerStore.Stop(ctx))
		require.NoError(t, rawStore.Close())
	}()

	pk, _, err := crypto.GenerateEd25519Key(cryptoRand.Reader)
	require.NoError(t, err)
	noopSigner, err := noop.NewNoopSigner(pk)
	require.NoError(t, err)

	chainID := "test-chain"
	headerCfg := types.HeaderConfig{
		Height:   1,
		DataHash: types.GetRandomBytes(32),
		AppHash:  types.GetRandomBytes(32),
		Signer:   noopSigner,
	}

	first, err := types.GetRandomSignedHeaderCustom(&headerCfg, chainID)
	require.NoError(t, err)
	require.NoError(t, headerStore.Append(ctx, first))

	next := &types.SignedHeader{
		Header: types.GetRandomNextHeader(first.Header, chainID),
		Signer: first.Signer,
	}
	payload, err := next.Header.MarshalBinary()
	require.NoError(t, err)
	signature, err := noopSigner.Sign(payload)
	require.NoError(t, err)
	next.Signature = signature

	require.NoError(t, headerStore.Append(ctx, next))
	require.NoError(t, headerStore.Sync(ctx))
}
