package node

import (
	"context"
	"testing"
	"time"

	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
)

func TestNewRPCOnlyNode(t *testing.T) {
	// Create test config
	conf := config.DefaultConfig
	conf.Node.RPCOnly = true
	conf.RPC.Address = "127.0.0.1:0" // Use port 0 for testing

	// Create test dependencies
	database := dssync.MutexWrap(ds.NewMapDatastore())
	logger := zerolog.New(zerolog.NewTestWriter(t))
	genesis := genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		AppHash:         []byte("test"),
		ProposerAddress: []byte("test-proposer"),
	}

	// Create mock P2P client
	p2pClient := &p2p.Client{}

	// Test node creation
	node, err := newRPCOnlyNode(conf, genesis, p2pClient, database, logger)
	require.NoError(t, err)
	assert.NotNil(t, node)
	assert.False(t, node.IsRunning())

	// Test that it implements the Node interface
	var _ Node = node
}

func TestRPCOnlyNodeValidation(t *testing.T) {
	tests := []struct {
		name        string
		aggregator  bool
		light       bool
		rpcOnly     bool
		shouldError bool
	}{
		{
			name:        "RPC-only mode alone should be valid",
			aggregator:  false,
			light:       false,
			rpcOnly:     true,
			shouldError: false,
		},
		{
			name:        "RPC-only with aggregator should be invalid",
			aggregator:  true,
			light:       false,
			rpcOnly:     true,
			shouldError: true,
		},
		{
			name:        "RPC-only with light should be invalid",
			aggregator:  false,
			light:       true,
			rpcOnly:     true,
			shouldError: true,
		},
		{
			name:        "All modes enabled should be invalid",
			aggregator:  true,
			light:       true,
			rpcOnly:     true,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := config.DefaultConfig
			conf.Node.Aggregator = tt.aggregator
			conf.Node.Light = tt.light
			conf.Node.RPCOnly = tt.rpcOnly

			err := conf.Validate()
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRPCOnlyNodeRun(t *testing.T) {
	// Skip this test if we can't run it fully due to network binding
	t.Skip("Skip full run test to avoid network binding issues in CI")

	// Create test config
	conf := config.DefaultConfig
	conf.Node.RPCOnly = true
	conf.RPC.Address = "127.0.0.1:0" // Use port 0 for testing

	// Create test dependencies
	database := dssync.MutexWrap(ds.NewMapDatastore())
	logger := zerolog.New(zerolog.NewTestWriter(t))
	genesis := genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		AppHash:         []byte("test"),
		ProposerAddress: []byte("test-proposer"),
	}

	// Create mock P2P client
	p2pClient := &p2p.Client{}

	// Create node
	node, err := newRPCOnlyNode(conf, genesis, p2pClient, database, logger)
	require.NoError(t, err)

	// Test node startup and shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = node.Run(ctx)
	// Expect context cancellation, not an error
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}