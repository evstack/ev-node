//go:build fibre

package cnfibertest_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/celestia-node/api/client"

	cnfiber "github.com/evstack/ev-node/tools/celestia-node-fiber"
	cnfibertest "github.com/evstack/ev-node/tools/celestia-node-fiber/testing"
)

// TestEvNode_FiberDA_TwoNode wires an aggregator + a full-node ev-node
// pair onto an in-process Celestia chain + Fibre + bridge and asserts
// that:
//
//   - the aggregator produces blocks at 200ms cadence and posts them
//     to the Fibre DA layer;
//   - a separate full node, sharing only the aggregator's genesis,
//     consumes those blocks via Fibre Listen + Download and applies
//     the same transactions the aggregator executed.
func TestEvNode_FiberDA_TwoNode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)

	network := cnfibertest.StartNetwork(t, ctx)
	bridge := cnfibertest.StartBridge(t, ctx, network)

	aggAdapter := newAdapter(t, ctx, network, bridge)
	fnAdapter := newAdapter(t, ctx, network, bridge)
	observer := newAdapter(t, ctx, network, bridge)

	head, err := observer.Head(ctx)
	require.NoError(t, err, "querying bridge head")
	t.Logf("bridge head at test start: %d", head)

	cnfibertest.RunEvNodeFibreTwoNodeFlow(t, ctx, aggAdapter, fnAdapter, observer, cnfibertest.EvNodeConfig{
		DAStartHeight: head,
	})
}

func newAdapter(t *testing.T, ctx context.Context, network *cnfibertest.Network, bridge *cnfibertest.Bridge) *cnfiber.Adapter {
	t.Helper()
	adapter, err := cnfiber.New(ctx, cnfiber.Config{
		Client: client.Config{
			ReadConfig: client.ReadConfig{
				BridgeDAAddr: bridge.RPCAddr(),
				DAAuthToken:  bridge.AdminToken,
				EnableDATLS:  false,
			},
			SubmitConfig: client.SubmitConfig{
				DefaultKeyName: network.ClientAccount,
				Network:        "private",
				CoreGRPCConfig: client.CoreGRPCConfig{
					Addr: network.ConsensusGRPCAddr(),
				},
			},
		},
	}, network.Consensus.Keyring)
	require.NoError(t, err, "constructing adapter")
	t.Cleanup(func() { _ = adapter.Close() })
	return adapter
}
