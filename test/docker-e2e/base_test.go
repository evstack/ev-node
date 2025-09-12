//go:build docker_e2e

package docker_e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	da "github.com/celestiaorg/tastora/framework/docker/dataavailability"
	"github.com/stretchr/testify/require"
)

func (s *DockerTestSuite) TestBasicDockerE2E() {
	ctx := context.Background()
	s.SetupDockerResources()

	var (
		bridgeNode *da.Node
	)

	s.T().Run("start celestia chain", func(t *testing.T) {
		err := s.celestia.Start(ctx)
		s.Require().NoError(err)
	})

	s.T().Run("start bridge node", func(t *testing.T) {
		genesisHash := s.getGenesisHash(ctx)

		networkInfo, err := s.celestia.GetNodes()[0].GetNetworkInfo(ctx)
		s.Require().NoError(err)
		celestiaNodeHostname := networkInfo.Internal.Hostname

		bridgeNode = s.daNetwork.GetBridgeNodes()[0]

		s.StartBridgeNode(ctx, bridgeNode, testChainID, genesisHash, celestiaNodeHostname)
	})

	s.T().Run("fund da wallet", func(t *testing.T) {
		daWallet, err := bridgeNode.GetWallet()
		s.Require().NoError(err)
		s.T().Logf("da node celestia address: %s", daWallet.GetFormattedAddress())

		s.FundWallet(ctx, daWallet, 100_000_000_00)
	})

	s.T().Run("start evolve chain node", func(t *testing.T) {
		s.StartEVNode(ctx, bridgeNode, s.evNodeChain.GetNodes()[0])
	})

	s.T().Run("submit a transaction to the evolve chain", func(t *testing.T) {
		evNode := s.evNodeChain.GetNodes()[0]

		// Debug: Check if the node is running and all ports
		networkInfo, err := evNode.GetNetworkInfo(ctx)
		require.NoError(t, err)

		t.Logf("EV node RPC port: %s", networkInfo.External.RPCAddress())
		t.Logf("EV node HTTP port: %s", networkInfo.External.HTTPAddress())

		// The http port resolvable by the test runner.
		httpPortStr := networkInfo.External.HTTPAddress()
		t.Logf("EV node HTTP address: %s", httpPortStr)

		if httpPortStr == "" {
			t.Fatal("HTTP address is empty - this indicates the HTTP server is not running or port mapping failed")
		}

		// Extract the host and port from the address
		parts := strings.Split(httpPortStr, ":")
		if len(parts) != 2 {
			t.Fatalf("Invalid HTTP address format: %s", httpPortStr)
		}
		host, port := parts[0], parts[1]

		// Use localhost since this is the external address accessible from the test host
		if host == "0.0.0.0" {
			host = "localhost"
		}

		t.Logf("Extracted host: %s, port: %s", host, port)

		client, err := NewClient(host, port)
		require.NoError(t, err)
		t.Logf("Created HTTP client with base URL: http://%s:%s", host, port)

		key := "key1"
		value := "value1"
		t.Logf("Attempting to POST key=%s, value=%s to /tx", key, value)
		_, err = client.Post(ctx, "/tx", key, value)
		if err != nil {
			t.Logf("POST request failed with error: %v", err)
		}
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			res, err := client.Get(ctx, "/kv?key="+key)
			if err != nil {
				return false
			}
			return string(res) == value
		}, 10*time.Second, time.Second)
	})
}
