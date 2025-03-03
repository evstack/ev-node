package commands

import (
	"context"
	"reflect"
	"testing"
	"time"

	proxy "github.com/rollkit/go-da/proxy/jsonrpc"
)

func TestParseFlags(t *testing.T) {
	flags := []string{
		"--abci", "grpc",
		"--consensus.create_empty_blocks", "true",
		"--consensus.create_empty_blocks_interval", "10s",
		"--consensus.double_sign_check_height", "10",
		"--db_backend", "cleverdb",
		"--db_dir", "data2",
		"--moniker", "yarik-playground2",
		"--p2p.external-address", "127.0.0.0:26000",
		"--p2p.laddr", "tcp://127.0.0.1:27000",
		"--p2p.pex",
		"--p2p.private_peer_ids", "1,2,3",
		"--p2p.seed_mode",
		"--p2p.unconditional_peer_ids", "4,5,6",
		"--priv_validator_laddr", "tcp://127.0.0.1:27003",
		"--proxy_app", "tcp://127.0.0.1:27004",
		"--rollkit.aggregator=false",
		"--rollkit.block_time", "2s",
		"--rollkit.da_auth_token", "token",
		"--rollkit.da_block_time", "20s",
		"--rollkit.da_gas_multiplier", "1.5",
		"--rollkit.da_gas_price", "1.5",
		"--rollkit.da_mempool_ttl", "10",
		"--rollkit.da_namespace", "namespace",
		"--rollkit.da_start_height", "100",
		"--rollkit.lazy_aggregator",
		"--rollkit.lazy_block_time", "2m",
		"--rollkit.light",
		"--rollkit.max_pending_blocks", "100",
		"--rpc.grpc_laddr", "tcp://127.0.0.1:27006",
		"--rpc.laddr", "tcp://127.0.0.1:27007",
		"--rpc.pprof_laddr", "tcp://127.0.0.1:27008",
		"--rpc.unsafe",
	}

	args := append([]string{"start"}, flags...)

	newRunNodeCmd := NewRunNodeCmd()

	if err := newRunNodeCmd.ParseFlags(args); err != nil {
		t.Errorf("Error: %v", err)
	}

	if err := parseFlags(newRunNodeCmd); err != nil {
		t.Errorf("Error: %v", err)
	}

	testCases := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"ABCI", config.ABCI, "grpc"},
		{"CreateEmptyBlocks", config.Consensus.CreateEmptyBlocks, true},
		{"CreateEmptyBlocksInterval", config.Consensus.CreateEmptyBlocksInterval, 10 * time.Second},
		{"DoubleSignCheckHeight", config.Consensus.DoubleSignCheckHeight, int64(10)},
		{"DBBackend", config.DBBackend, "cleverdb"},
		{"DBDir", config.DBDir(), "data2"},
		{"Moniker", config.Moniker, "yarik-playground2"},
		{"ExternalAddress", config.P2P.ExternalAddress, "127.0.0.0:26000"},
		{"ListenAddress", config.P2P.ListenAddress, "tcp://127.0.0.1:27000"},
		{"PexReactor", config.P2P.PexReactor, true},
		{"PrivatePeerIDs", config.P2P.PrivatePeerIDs, "1,2,3"},
		{"SeedMode", config.P2P.SeedMode, true},
		{"UnconditionalPeerIDs", config.P2P.UnconditionalPeerIDs, "4,5,6"},
		{"PrivValidatorListenAddr", config.PrivValidatorListenAddr, "tcp://127.0.0.1:27003"},
		{"ProxyApp", config.ProxyApp, "tcp://127.0.0.1:27004"},
		{"Aggregator", nodeConfig.Aggregator, false},
		{"BlockTime", nodeConfig.BlockTime, 2 * time.Second},
		{"DAAuthToken", nodeConfig.DataAvailability.AuthToken, "token"},
		{"DABlockTime", nodeConfig.BlockTime, 20 * time.Second},
		{"DAGasMultiplier", nodeConfig.DataAvailability.GasMultiplier, 1.5},
		{"DAGasPrice", nodeConfig.DataAvailability.GasPrice, 1.5},
		{"DAMempoolTTL", nodeConfig.DAMempoolTTL, uint64(10)},
		{"DANamespace", nodeConfig.DataAvailability.Namespace, "namespace"},
		{"DAStartHeight", nodeConfig.DAStartHeight, uint64(100)},
		{"LazyAggregator", nodeConfig.LazyAggregator, true},
		{"LazyBlockTime", nodeConfig.LazyBlockTime, 2 * time.Minute},
		{"Light", nodeConfig.Light, true},
		{"MaxPendingBlocks", nodeConfig.MaxPendingBlocks, uint64(100)},
		{"GRPCListenAddress", config.RPC.GRPCListenAddress, "tcp://127.0.0.1:27006"},
		{"ListenAddress", config.RPC.ListenAddress, "tcp://127.0.0.1:27007"},
		{"PprofListenAddress", config.RPC.PprofListenAddress, "tcp://127.0.0.1:27008"},
		{"Unsafe", config.RPC.Unsafe, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !reflect.DeepEqual(tc.got, tc.expected) {
				t.Errorf("Expected %v, got %v", tc.expected, tc.got)
			}
		})
	}
}

func TestAggregatorFlagInvariants(t *testing.T) {
	flagVariants := [][]string{{
		"--rollkit.aggregator=false",
	}, {
		"--rollkit.aggregator=true",
	}, {
		"--rollkit.aggregator",
	}}

	validValues := []bool{false, true, true}

	for i, flags := range flagVariants {
		args := append([]string{"start"}, flags...)

		newRunNodeCmd := NewRunNodeCmd()

		if err := newRunNodeCmd.ParseFlags(args); err != nil {
			t.Errorf("Error: %v", err)
		}

		if err := parseFlags(newRunNodeCmd); err != nil {
			t.Errorf("Error: %v", err)
		}

		if nodeConfig.Aggregator != validValues[i] {
			t.Errorf("Expected %v, got %v", validValues[i], nodeConfig.Aggregator)
		}
	}
}

// MockServer wraps proxy.Server to allow us to control its behavior in tests
type MockServer struct {
	*proxy.Server
	StartFunc func(context.Context) error
	StopFunc  func(context.Context) error
}

func (m *MockServer) Start(ctx context.Context) error {
	if m.StartFunc != nil {
		return m.StartFunc(ctx)
	}
	return m.Server.Start(ctx)
}

func (m *MockServer) Stop(ctx context.Context) error {
	if m.StopFunc != nil {
		return m.StopFunc(ctx)
	}
	return m.Server.Stop(ctx)
}
