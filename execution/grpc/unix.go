package grpc

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"

	"connectrpc.com/connect"

	"github.com/evstack/ev-node/core/execution"
)

// ListenUnix creates a Unix domain socket listener for the gRPC execution service.
//
// If socketPath already exists, ListenUnix removes it only when it is a stale
// socket. Regular files, directories, and other path types are left untouched.
func ListenUnix(socketPath string) (net.Listener, error) {
	if socketPath == "" {
		return nil, errors.New("unix socket path is required")
	}
	if err := removeStaleUnixSocket(socketPath); err != nil {
		return nil, err
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("listen unix socket %q: %w", socketPath, err)
	}
	return listener, nil
}

// ListenAndServeUnix serves the gRPC execution service over a Unix domain socket.
//
// The NewExecutorServiceHandler handler is passed to http.Serve, so this
// function blocks until http.Serve returns an error. When it returns, deferred
// cleanup closes the listener with listener.Close and then removes the socket
// with removeStaleUnixSocket. Cleanup errors are currently ignored.
func ListenAndServeUnix(socketPath string, executor execution.Executor, opts ...connect.HandlerOption) error {
	listener, err := ListenUnix(socketPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = listener.Close()
	}()
	defer func() {
		_ = removeStaleUnixSocket(socketPath)
	}()

	return http.Serve(listener, NewExecutorServiceHandler(executor, opts...))
}

func removeStaleUnixSocket(socketPath string) error {
	info, err := os.Lstat(socketPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat unix socket %q: %w", socketPath, err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("refusing to remove non-socket path %q", socketPath)
	}
	if err := os.Remove(socketPath); err != nil {
		return fmt.Errorf("remove stale unix socket %q: %w", socketPath, err)
	}
	return nil
}
