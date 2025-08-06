//go:build docker_e2e

package docker_e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	tastoradocker "github.com/celestiaorg/tastora/framework/docker"
	tastoratypes "github.com/celestiaorg/tastora/framework/types"
)

// TestRollkitNodeRestart tests the ability to stop and restart a Rollkit node,
// verifying that the node can recover properly and maintain functionality.
func (s *DockerTestSuite) TestRollkitNodeRestart() {
	ctx := context.Background()
	s.SetupDockerResources()

	var (
		bridgeNode tastoratypes.DANode
		rollkitNode tastoratypes.RollkitNode
		client     *Client
		namespace  string // Store namespace for consistent restarts
	)

	s.T().Run("setup initial infrastructure", func(t *testing.T) {
		// Start celestia chain
		err := s.celestia.Start(ctx)
		s.Require().NoError(err)

		// Start bridge node
		genesisHash := s.getGenesisHash(ctx)
		celestiaNodeHostname, err := s.celestia.GetNodes()[0].GetInternalHostName(ctx)
		s.Require().NoError(err)

		bridgeNode = s.daNetwork.GetBridgeNodes()[0]
		s.StartBridgeNode(ctx, bridgeNode, testChainID, genesisHash, celestiaNodeHostname)

		// Fund DA wallet
		daWallet, err := bridgeNode.GetWallet()
		s.Require().NoError(err)
		s.FundWallet(ctx, daWallet, 100_000_000_00)

		// Generate and store namespace for consistent restarts
		namespace = generateValidNamespaceHex()
		
		// Start rollkit node with stored namespace
		rollkitNode = s.rollkitChain.GetNodes()[0]
		s.StartRollkitNodeWithNamespace(ctx, bridgeNode, rollkitNode, namespace)

		// Create HTTP client for testing
		httpPortStr := rollkitNode.GetHostHTTPPort()
		s.Require().NotEmpty(httpPortStr, "HTTP port should not be empty")
		
		httpPort := strings.Split(httpPortStr, ":")[len(strings.Split(httpPortStr, ":"))-1]
		client, err = NewClient("localhost", httpPort)
		s.Require().NoError(err)
	})

	s.T().Run("verify initial functionality", func(t *testing.T) {
		// Submit a transaction to verify node is working
		key := "pre-restart-key"
		value := "pre-restart-value"
		
		_, err := client.Post(ctx, "/tx", key, value)
		s.Require().NoError(err)

		// Verify the transaction was processed
		s.Require().Eventually(func() bool {
			res, err := client.Get(ctx, "/kv?key="+key)
			if err != nil {
				return false
			}
			return string(res) == value
		}, 10*time.Second, time.Second)
		
		t.Logf("✅ Node is functional before restart")
	})

	s.T().Run("stop rollkit node", func(t *testing.T) {
		// Cast to concrete type to access lifecycle methods
		concreteNode, ok := rollkitNode.(*tastoradocker.RollkitNode)
		s.Require().True(ok, "rollkit node should be convertible to concrete type")

		// Verify node is running before stopping
		err := concreteNode.ContainerLifecycle.Running(ctx)
		s.Require().NoError(err, "rollkit node should be running before stop")

		// Stop the rollkit node
		err = concreteNode.StopContainer(ctx)
		s.Require().NoError(err, "failed to stop rollkit node")

		// Verify node is stopped (Running() should return error when stopped)
		err = concreteNode.ContainerLifecycle.Running(ctx)
		s.Require().Error(err, "rollkit node should be stopped")
		
		t.Logf("✅ Node successfully stopped")
	})

	s.T().Run("restart rollkit node", func(t *testing.T) {
		// Cast to concrete type to access lifecycle methods
		concreteNode, ok := rollkitNode.(*tastoradocker.RollkitNode)
		s.Require().True(ok, "rollkit node should be convertible to concrete type")

		// Use StartContainer() instead of Start() to restart the existing stopped container
		err := concreteNode.StartContainer(ctx)
		s.Require().NoError(err, "failed to restart rollkit node")

		// Verify node is running again
		err = concreteNode.ContainerLifecycle.Running(ctx)
		s.Require().NoError(err, "rollkit node should be running after restart")
		
		t.Logf("✅ Node successfully restarted")
	})

	s.T().Run("verify functionality after restart", func(t *testing.T) {
		// Verify the old data is still accessible (persistence test)
		res, err := client.Get(ctx, "/kv?key=pre-restart-key")
		s.Require().NoError(err)
		s.Require().Equal("pre-restart-value", string(res), "data should persist after restart")

		// Submit a new transaction to verify node is functional
		key := "post-restart-key"
		value := "post-restart-value"
		
		_, err = client.Post(ctx, "/tx", key, value)
		s.Require().NoError(err)

		// Verify the new transaction was processed
		s.Require().Eventually(func() bool {
			res, err := client.Get(ctx, "/kv?key="+key)
			if err != nil {
				return false
			}
			return string(res) == value
		}, 10*time.Second, time.Second)
		
		t.Logf("✅ Node is fully functional after restart")
		t.Logf("✅ Data persistence verified")
	})
}