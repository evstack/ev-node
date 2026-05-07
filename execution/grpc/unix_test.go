package grpc

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestListenUnixRejectsNonSocketPath(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "executor.sock")
	if err := os.WriteFile(socketPath, []byte("not a socket"), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	listener, err := ListenUnix(socketPath)
	if err == nil {
		_ = listener.Close()
		t.Fatal("expected error for non-socket path")
	}
	if !strings.Contains(err.Error(), "refusing to remove non-socket path") {
		t.Fatalf("expected non-socket refusal, got %v", err)
	}
}

func TestListenUnixRemovesStaleSocket(t *testing.T) {
	socketPath := testUnixSocketPath(t)
	staleListener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("create stale unix socket: %v", err)
	}
	if err := staleListener.Close(); err != nil {
		t.Fatalf("close stale unix socket: %v", err)
	}

	listener, err := ListenUnix(socketPath)
	if err != nil {
		t.Fatalf("listen unix socket: %v", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("close unix socket: %v", err)
	}
}

func testUnixSocketPath(t *testing.T) string {
	t.Helper()

	socketPath := filepath.Join(
		os.TempDir(),
		fmt.Sprintf("ev-node-grpc-%d-%d.sock", os.Getpid(), time.Now().UnixNano()),
	)
	t.Cleanup(func() {
		_ = os.Remove(socketPath)
	})
	return socketPath
}
