package node

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	seqGRPC "github.com/rollkit/go-sequencing/proxy/grpc"
	seqTest "github.com/rollkit/go-sequencing/test"

	rollkitconfig "github.com/rollkit/rollkit/config"
	coreda "github.com/rollkit/rollkit/core/da"
	coreexecutor "github.com/rollkit/rollkit/core/execution"
	coresequencer "github.com/rollkit/rollkit/core/sequencer"
	"github.com/rollkit/rollkit/types"
)

const (
	// MockDAAddress is the address used by the mock gRPC service
	// NOTE: this should be unique per test package to avoid
	// "bind: listen address already in use" because multiple packages
	// are tested in parallel
	MockDAAddress = "grpc://localhost:7990"

	// MockDANamespace is a sample namespace used by the mock DA client
	MockDANamespace = "00000000000000000000000000000000000000000000000000deadbeef"

	// MockSequencerAddress is a sample address used by the mock sequencer
	MockSequencerAddress = "127.0.0.1:50051"

	// MockExecutorAddress is a sample address used by the mock executor
	MockExecutorAddress = "127.0.0.1:40041"
)

// startMockSequencerServerGRPC starts a mock gRPC server with the given listenAddress.
func startMockSequencerServerGRPC(listenAddress string) *grpc.Server {
	dummySeq := seqTest.NewMultiRollupSequencer()
	server := seqGRPC.NewServer(dummySeq, dummySeq, dummySeq)
	lis, err := net.Listen("tcp", listenAddress)
	if err != nil {
		panic(err)
	}
	go func() {
		_ = server.Serve(lis)
	}()
	return server
}

type NodeType int

const (
	Full NodeType = iota
	Light
)

// NodeRunner contains a node and its running context
type NodeRunner struct {
	Node      Node
	Ctx       context.Context
	Cancel    context.CancelFunc
	ErrCh     chan error
	WaitGroup *sync.WaitGroup
}

// startNodeWithCleanup starts the node using the service pattern and registers cleanup
func startNodeWithCleanup(t *testing.T, node Node) *NodeRunner {
	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Create error channel and wait group
	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	// Start the node in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := node.Run(ctx)
		select {
		case errCh <- err:
		default:
			t.Logf("Error channel full, discarding error: %v", err)
		}
	}()

	// Give the node time to initialize
	time.Sleep(100 * time.Millisecond)

	// Check if the node has stopped unexpectedly
	select {
	case err := <-errCh:
		t.Fatalf("Node stopped unexpectedly with error: %v", err)
	default:
		// This is expected - node is still running
	}

	// Register cleanup function
	t.Cleanup(func() {
		cleanUpNode(cancel, &wg, errCh, t)
	})

	return &NodeRunner{
		Node:      node,
		Ctx:       ctx,
		Cancel:    cancel,
		ErrCh:     errCh,
		WaitGroup: &wg,
	}
}

// cleanUpNode stops the node using context cancellation
func cleanUpNode(cancel context.CancelFunc, wg *sync.WaitGroup, errCh chan error, t *testing.T) {
	// Cancel the context to stop the node
	cancel()

	// Wait for the node to stop with a timeout
	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		// Node stopped successfully
	case <-time.After(5 * time.Second):
		t.Log("Warning: Node did not stop gracefully within timeout")
	}

	// Check for any errors during shutdown
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Logf("Error stopping node: %v", err)
		}
	default:
		// No error
	}
}

// initAndStartNodeWithCleanup initializes and starts a node of the specified type.
func initAndStartNodeWithCleanup(ctx context.Context, t *testing.T, nodeType NodeType, chainID string) Node {
	node, _ := setupTestNode(ctx, t, nodeType, chainID)
	runner := startNodeWithCleanup(t, node)

	return runner.Node
}

// setupTestNode sets up a test node based on the NodeType.
func setupTestNode(ctx context.Context, t *testing.T, nodeType NodeType, chainID string) (Node, crypto.PrivKey) {
	node, privKey, err := newTestNode(ctx, t, nodeType, chainID)
	require.NoError(t, err)
	require.NotNil(t, node)

	return node, privKey
}

// newTestNode creates a new test node based on the NodeType.
func newTestNode(ctx context.Context, t *testing.T, nodeType NodeType, chainID string) (Node, crypto.PrivKey, error) {
	config := rollkitconfig.Config{
		RootDir: t.TempDir(),
		Node: rollkitconfig.NodeConfig{
			ExecutorAddress:  MockExecutorAddress,
			SequencerAddress: MockSequencerAddress,
			Light:            nodeType == Light,
		},
		DA: rollkitconfig.DAConfig{
			Address:   MockDAAddress,
			Namespace: MockDANamespace,
		},
	}

	genesis, genesisValidatorKey, _ := types.GetGenesisWithPrivkey(chainID)

	dummyExec := coreexecutor.NewDummyExecutor()
	dummySequencer := coresequencer.NewDummySequencer()
	dummyDA := coreda.NewDummyDA(100_000, 0, 0)
	dummyClient := coreda.NewDummyClient(dummyDA, []byte(MockDANamespace))

	err := InitFiles(config.RootDir)
	require.NoError(t, err)

	logger := log.NewTestLogger(t)

	node, err := NewNode(
		ctx,
		config,
		dummyExec,
		dummySequencer,
		dummyClient,
		genesisValidatorKey,
		genesis,
		DefaultMetricsProvider(rollkitconfig.DefaultInstrumentationConfig()),
		logger,
	)
	return node, genesisValidatorKey, err
}

func TestNewNode(t *testing.T) {
	ctx := context.Background()
	chainID := "TestNewNode"
	//ln := initAndStartNodeWithCleanup(ctx, t, Light, chainID)
	//require.IsType(t, new(LightNode), ln)
	fn := initAndStartNodeWithCleanup(ctx, t, Full, chainID)
	require.IsType(t, new(FullNode), fn)
}
