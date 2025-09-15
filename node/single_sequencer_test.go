//go:build !integration

package node

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test that node can start and be shutdown properly using context cancellation
func TestStartup(t *testing.T) {
	// Get the node and cleanup function
	node, cleanup := createNodeWithCleanup(t, getTestConfig(t, 1))
	defer cleanup()

	require.IsType(t, new(FullNode), node)

	// Create a context with cancel function for node operation
	ctx, cancel := context.WithCancel(context.Background())

	// Start the node in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- node.Run(ctx)
	}()

	// Wait for the node to start and verify it's healthy
	require.Eventually(t, func() bool {
		// Check if node errored out during startup
		select {
		case err := <-errChan:
			t.Fatalf("Node stopped unexpectedly during startup with error: %v", err)
			return false
		default:
			// Node hasn't errored - continue health check
		}

		// Verify the node is actually running
		return node.IsRunning()
	}, 10*time.Second, 100*time.Millisecond, "Node should start and be running")

	// Cancel the context to stop the node
	cancel()

	// Allow some time for the node to stop and check for errors
	select {
	case err := <-errChan:
		// Context cancellation should result in context.Canceled error
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(15 * time.Second):
		t.Fatal("Node did not stop after context cancellation")
	}
}
