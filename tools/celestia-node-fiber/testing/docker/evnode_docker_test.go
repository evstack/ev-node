//go:build fibre_docker

// evnode_docker_test.go — wires an aggregator + full-node ev-node pair
// onto the docker-compose Celestia + Fibre stack and asserts that the
// full node DA-syncs the aggregator's blocks via Fibre.
//
// This is the docker counterpart to `TestEvNode_FiberDA_TwoNode` under
// `testing/`. Both share the same flow
// (`cnfibertest.RunEvNodeFibreTwoNodeFlow`); only the underlying
// Celestia + bridge plumbing differs.
//
// Prereqs are identical to docker_test.go — bring up the stack first:
//
//	cd tools/celestia-node-fiber/testing/docker
//	docker compose up -d --build
//	# wait until `docker compose logs register` says "setup.done flag written"
//
// Then run the test:
//
//	go test -tags 'fibre fibre_docker' -count=1 -timeout 5m \
//	    -run TestEvNode_FiberDA_Docker ./testing/docker/...

package docker_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/celestia-node/api/client"

	cnfiber "github.com/evstack/ev-node/tools/celestia-node-fiber"
	cnfibertest "github.com/evstack/ev-node/tools/celestia-node-fiber/testing"
)

// TestEvNode_FiberDA_Docker drives the aggregator + full-node ev-node
// pair against the 4-validator + bridge docker stack. Compared to the
// in-process variant this exercises:
//   - real consensus 2/3-quorum signature aggregation (4 validators),
//   - inter-validator P2P,
//   - 4 distinct fibre servers cooperating on Upload row distribution,
//   - dns:/// host registry resolution against an external chain,
//   - a bridge that's syncing real headers, not driving block production.
func TestEvNode_FiberDA_Docker(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	t.Cleanup(cancel)

	jwt := readBridgeJWT(t)
	kr := readClientKeyring(t)

	// Each role gets its own adapter so the bridge JSON-RPC websocket
	// connections aren't shared. celestia-node's go-jsonrpc client only
	// supports one event-stream subscription per connection — sharing
	// kills the socket the moment a second Subscribe lands on it.
	mkAdapter := func(label string) *cnfiber.Adapter {
		t.Helper()
		a, err := cnfiber.New(ctx, cnfiber.Config{
			Client: client.Config{
				ReadConfig: client.ReadConfig{
					BridgeDAAddr: envOr("FIBRE_BRIDGE_ADDR", bridgeAddr),
					DAAuthToken:  jwt,
					EnableDATLS:  false,
				},
				SubmitConfig: client.SubmitConfig{
					DefaultKeyName: clientAccount,
					Network:        chainID,
					CoreGRPCConfig: client.CoreGRPCConfig{
						Addr: envOr("FIBRE_CONSENSUS_ADDR", consensusAddr),
					},
				},
			},
		}, kr)
		require.NoError(t, err, "constructing %s adapter against docker stack", label)
		t.Cleanup(func() { _ = a.Close() })
		return a
	}

	aggAdapter := mkAdapter("aggregator")
	fnAdapter := mkAdapter("full-node")
	observer := mkAdapter("observer")

	// Pin the full node to the current bridge tip so its DA retriever
	// skips historical scans (where there are no Fibre blobs yet) and
	// jumps straight to the live-subscribe path.
	head, err := observer.Head(ctx)
	require.NoError(t, err, "querying bridge head")
	t.Logf("bridge head at test start: %d", head)

	cnfibertest.RunEvNodeFibreTwoNodeFlow(t, ctx, aggAdapter, fnAdapter, observer, cnfibertest.EvNodeConfig{
		ChainID:       "ev-fiber-docker",
		DAStartHeight: head,
	})
}
