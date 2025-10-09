package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/rpc/server"
	"github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types/pb/evnode/v1/v1connect"
)

func TestBackupCmd_Success(t *testing.T) {
	t.Parallel()

	mockStore := mocks.NewMockStore(t)

	mockStore.On("Height", mock.Anything).Return(uint64(15), nil)
	mockStore.On("Backup", mock.Anything, mock.Anything, uint64(9)).Run(func(args mock.Arguments) {
		writer := args.Get(1).(io.Writer)
		_, _ = writer.Write([]byte("chunk-1"))
		_, _ = writer.Write([]byte("chunk-2"))
	}).Return(uint64(21), nil)

	logger := zerolog.Nop()
	storeServer := server.NewStoreServer(mockStore, logger)
	mux := http.NewServeMux()
	storePath, storeHandler := v1connect.NewStoreServiceHandler(storeServer)
	mux.Handle(storePath, storeHandler)

	httpServer := httptest.NewServer(h2c.NewHandler(mux, &http2.Server{}))
	defer httpServer.Close()

	tempDir, err := os.MkdirTemp("", "evnode-backup-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})

	backupCmd := NewBackupCmd()
	config.AddFlags(backupCmd)

	rootCmd := &cobra.Command{Use: "root"}
	config.AddGlobalFlags(rootCmd, "test")
	rootCmd.AddCommand(backupCmd)

	outPath := filepath.Join(tempDir, "snapshot.badger")
	rpcAddr := strings.TrimPrefix(httpServer.URL, "http://")

	output, err := executeCommandC(
		rootCmd,
		"backup",
		"--home="+tempDir,
		"--evnode.rpc.address="+rpcAddr,
		"--output", outPath,
		"--target-height", "12",
		"--since-version", "9",
	)

	require.NoError(t, err, "command failed: %s", output)

	data, readErr := os.ReadFile(outPath)
	require.NoError(t, readErr)
	require.Equal(t, "chunk-1chunk-2", string(data))

	require.Contains(t, output, "Backup saved to")
	require.Contains(t, output, "Current height: 15")
	require.Contains(t, output, "Target height: 12")
	require.Contains(t, output, "Since version: 9")
	require.Contains(t, output, "Last version: 21")

	mockStore.AssertExpectations(t)
}

func TestBackupCmd_ExistingFileWithoutForce(t *testing.T) {
	t.Parallel()

	mockStore := mocks.NewMockStore(t)
	logger := zerolog.Nop()
	storeServer := server.NewStoreServer(mockStore, logger)
	mux := http.NewServeMux()
	storePath, storeHandler := v1connect.NewStoreServiceHandler(storeServer)
	mux.Handle(storePath, storeHandler)

	httpServer := httptest.NewServer(h2c.NewHandler(mux, &http2.Server{}))
	defer httpServer.Close()

	tempDir, err := os.MkdirTemp("", "evnode-backup-existing-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})

	outPath := filepath.Join(tempDir, "snapshot.badger")
	require.NoError(t, os.WriteFile(outPath, []byte("existing"), 0o600))

	backupCmd := NewBackupCmd()
	config.AddFlags(backupCmd)

	rootCmd := &cobra.Command{Use: "root"}
	config.AddGlobalFlags(rootCmd, "test")
	rootCmd.AddCommand(backupCmd)

	rpcAddr := strings.TrimPrefix(httpServer.URL, "http://")

	output, err := executeCommandC(
		rootCmd,
		"backup",
		"--home="+tempDir,
		"--evnode.rpc.address="+rpcAddr,
		"--output", outPath,
	)

	require.Error(t, err)
	require.Contains(t, output, "already exists (use --force to overwrite)")
}
