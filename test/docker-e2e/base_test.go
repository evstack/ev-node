//go:build docker_e2e

package docker_e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/celestiaorg/tastora/framework/types"
	"github.com/stretchr/testify/require"
)

func (s *DockerTestSuite) TestBasicDockerE2E() {
	ctx := context.Background()
	s.SetupDockerResources()

	var (
		bridgeNode types.DANode
	)

	s.T().Run("start celestia chain", func(t *testing.T) {
		err := s.celestia.Start(ctx)
		s.Require().NoError(err)
	})

	s.T().Run("start bridge node", func(t *testing.T) {
		genesisHash := s.getGenesisHash(ctx)

		celestiaNodeHostname, err := s.celestia.GetNodes()[0].GetInternalHostName(ctx)
		s.Require().NoError(err)

		bridgeNode = s.daNetwork.GetBridgeNodes()[0]

		s.StartBridgeNode(ctx, bridgeNode, testChainID, genesisHash, celestiaNodeHostname)
	})

	s.T().Run("fund da wallet", func(t *testing.T) {
		daWallet, err := bridgeNode.GetWallet()
		s.Require().NoError(err)
		s.T().Logf("da node celestia address: %s", daWallet.GetFormattedAddress())

		s.FundWallet(ctx, daWallet, 100_000_000_00)
	})

	s.T().Run("start rollkit chain node", func(t *testing.T) {
		s.StartRollkitNode(ctx, bridgeNode, s.rollkitChain.GetNodes()[0])
	})

	s.T().Run("submit a transaction to the rollkit chain", func(t *testing.T) {
		rollkitNode := s.rollkitChain.GetNodes()[0]

		// Debug: Check if the node is running and all ports
		t.Logf("Rollkit node RPC port: %s", rollkitNode.GetHostRPCPort())
		t.Logf("Rollkit node GRPC port: %s", rollkitNode.GetHostGRPCPort())
		t.Logf("Rollkit node P2P port: %s", rollkitNode.GetHostP2PPort())

// The http port resolvable by the test runner.
		httpPortStr := rollkitNode.GetHostHTTPPort()
		t.Logf("Rollkit node HTTP port: %s", httpPortStr)

		if httpPortStr == "" {
			t.Fatal("HTTP port is empty - this indicates the HTTP server is not running or port mapping failed")
		}

		// Extract just the port number if it includes an address
		httpPort := strings.Split(httpPortStr, ":")[len(strings.Split(httpPortStr, ":"))-1]
		t.Logf("Extracted port: %s", httpPort)

		client, err := NewClient("localhost", httpPort)
		require.NoError(t, err)
		t.Logf("Created HTTP client with base URL: http://localhost:%s", httpPort)

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
