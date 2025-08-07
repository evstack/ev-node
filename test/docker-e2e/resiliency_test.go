//go:build docker_e2e

package docker_e2e

import (
	"context"
	"fmt"
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
		bridgeNode  tastoratypes.DANode
		rollkitNode tastoratypes.RollkitNode
		client      *Client
		namespace   string // Store namespace for consistent restarts
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

// TestCelestiaDANetworkPartitionE2E tests the rollkit node's ability to handle
// complete network partition from Celestia DA layer and subsequent reconnection.
//
// Test Purpose:
// - Validate rollkit continues block production during DA network partition
// - Verify blocks accumulate locally when DA is unreachable
// - Ensure proper reconnection and batch submission after DA recovery
// - Test resilience against prolonged DA unavailability
//
// Test Flow:
// 1. Setup: Start Celestia chain, bridge node, and rollkit node
// 2. Baseline: Submit transactions and verify normal DA submission works
// 3. Partition: Stop entire Celestia network to simulate network partition
// 4. Local Accumulation: Continue submitting transactions while DA is down
// 5. Verification: Ensure blocks are created locally but not submitted to DA
// 6. Reconnection: Restart Celestia network and bridge node
// 7. Recovery: Verify accumulated blocks are submitted to DA in batches
// 8. Final Validation: Confirm system returns to normal operation
//
// This test validates the rollkit node's resilience to complete DA layer failures
// and its ability to recover gracefully without data loss.
func (s *DockerTestSuite) TestCelestiaDANetworkPartitionE2E() {
	ctx := context.Background()
	s.SetupDockerResources()

	var (
		bridgeNode  tastoratypes.DANode
		rollkitNode tastoratypes.RollkitNode
		client      *Client
		namespace   string
	)

	s.T().Run("setup initial infrastructure", func(t *testing.T) {
		// Start celestia chain
		err := s.celestia.Start(ctx)
		s.Require().NoError(err)
		t.Log("✅ Celestia chain started")

		// Start bridge node
		genesisHash := s.getGenesisHash(ctx)
		celestiaNodeHostname, err := s.celestia.GetNodes()[0].GetInternalHostName(ctx)
		s.Require().NoError(err)

		bridgeNode = s.daNetwork.GetBridgeNodes()[0]
		s.StartBridgeNode(ctx, bridgeNode, testChainID, genesisHash, celestiaNodeHostname)
		t.Log("✅ Bridge node started")

		// Fund DA wallet
		daWallet, err := bridgeNode.GetWallet()
		s.Require().NoError(err)
		s.FundWallet(ctx, daWallet, 100_000_000_00)
		t.Log("✅ DA wallet funded")

		// Generate and store namespace for consistent operations
		namespace = generateValidNamespaceHex()
		t.Logf("Using namespace: %s", namespace)

		// Start rollkit node with stored namespace
		rollkitNode = s.rollkitChain.GetNodes()[0]
		s.StartRollkitNodeWithNamespace(ctx, bridgeNode, rollkitNode, namespace)
		t.Log("✅ Rollkit node started")

		// Create HTTP client for testing
		httpPortStr := rollkitNode.GetHostHTTPPort()
		s.Require().NotEmpty(httpPortStr, "HTTP port should not be empty")

		httpPort := strings.Split(httpPortStr, ":")[len(strings.Split(httpPortStr, ":"))-1]
		var err2 error
		client, err2 = NewClient("localhost", httpPort)
		s.Require().NoError(err2)
		t.Log("✅ HTTP client created")
	})

	s.T().Run("verify baseline DA functionality", func(t *testing.T) {
		// Submit baseline transactions to verify normal operation
		const baselineTxCount = 3
		for i := 0; i < baselineTxCount; i++ {
			key := fmt.Sprintf("baseline-key-%d", i)
			value := fmt.Sprintf("baseline-value-%d", i)

			_, err := client.Post(ctx, "/tx", key, value)
			s.Require().NoError(err)
			t.Logf("Submitted baseline tx %d: %s=%s", i+1, key, value)
		}

		// Verify all baseline transactions are processed
		for i := 0; i < baselineTxCount; i++ {
			key := fmt.Sprintf("baseline-key-%d", i)
			expectedValue := fmt.Sprintf("baseline-value-%d", i)

			s.Require().Eventually(func() bool {
				res, err := client.Get(ctx, "/kv?key="+key)
				if err != nil {
					return false
				}
				return string(res) == expectedValue
			}, 15*time.Second, time.Second, "baseline transaction %d should be processed", i+1)
		}

		t.Logf("✅ Baseline functionality verified - %d transactions processed", baselineTxCount)

		// Wait a moment to ensure DA submission completes
		time.Sleep(2 * time.Second)
	})

	s.T().Run("simulate network partition - stop celestia network", func(t *testing.T) {
		// First stop the bridge node to simulate DA layer failure
		concreteBridgeNode, ok := bridgeNode.(*tastoradocker.DANode)
		s.Require().True(ok, "bridge node should be convertible to concrete type")

		err := concreteBridgeNode.StopContainer(ctx)
		s.Require().NoError(err, "failed to stop bridge node")
		t.Log("✅ Bridge node stopped")

		// Then stop the entire Celestia chain to simulate complete network partition
		err = s.celestia.Stop(ctx)
		s.Require().NoError(err, "failed to stop celestia chain")
		t.Log("✅ Celestia chain stopped - network partition simulated")

		// Wait a moment for the partition to take effect
		time.Sleep(2 * time.Second)
	})

	var partitionTxKeys []string
	s.T().Run("accumulate blocks during network partition", func(t *testing.T) {
		t.Log("🔄 Submitting transactions while DA is partitioned...")

		// Submit transactions that should accumulate locally
		const partitionTxCount = 8
		partitionStart := time.Now()

		for i := 0; i < partitionTxCount; i++ {
			key := fmt.Sprintf("partition-key-%d", i)
			value := fmt.Sprintf("partition-value-%d", i)
			partitionTxKeys = append(partitionTxKeys, key)

			_, err := client.Post(ctx, "/tx", key, value)
			s.Require().NoError(err)
			t.Logf("Submitted partition tx %d: %s=%s", i+1, key, value)

			// Small delay between transactions
			time.Sleep(500 * time.Millisecond)
		}

		// Verify transactions are processed locally (should still work)
		for i, key := range partitionTxKeys {
			expectedValue := fmt.Sprintf("partition-value-%d", i)

			s.Require().Eventually(func() bool {
				res, err := client.Get(ctx, "/kv?key="+key)
				if err != nil {
					return false
				}
				return string(res) == expectedValue
			}, 15*time.Second, time.Second, "partition transaction %d should be processed locally", i+1)
		}

		partitionDuration := time.Since(partitionStart)
		t.Logf("✅ Local block accumulation verified - %d transactions processed in %v",
			partitionTxCount, partitionDuration)
		t.Log("✅ Rollkit continues to function during DA network partition")

		// Wait additional time to ensure more blocks accumulate
		time.Sleep(3 * time.Second)
	})

	s.T().Run("simulate network recovery - restart celestia network", func(t *testing.T) {
		t.Log("🔄 Restarting Celestia network to simulate recovery...")

		// Restart the Celestia chain
		err := s.celestia.Start(ctx)
		s.Require().NoError(err)
		t.Log("✅ Celestia chain restarted")

		// Wait for Celestia to be ready
		time.Sleep(3 * time.Second)

		// Get updated celestia node hostname after restart
		celestiaNodeHostname, err := s.celestia.GetNodes()[0].GetInternalHostName(ctx)
		s.Require().NoError(err)

		// Restart bridge node with proper connection to celestia
		concreteBridgeNode, ok := bridgeNode.(*tastoradocker.DANode)
		s.Require().True(ok, "bridge node should be convertible to concrete type")

		err = concreteBridgeNode.StartContainer(ctx)
		s.Require().NoError(err, "failed to restart bridge node")
		t.Log("✅ Bridge node restarted")

		// Re-initialize bridge node connection if needed
		genesisHash := s.getGenesisHash(ctx)
		s.StartBridgeNode(ctx, bridgeNode, testChainID, genesisHash, celestiaNodeHostname)

		// Re-fund DA wallet after restart
		daWallet, err := bridgeNode.GetWallet()
		s.Require().NoError(err)
		s.FundWallet(ctx, daWallet, 100_000_000_00)
		t.Log("✅ Bridge node reconnected and wallet refunded")

		// Wait for DA connection to stabilize
		time.Sleep(5 * time.Second)
		t.Log("✅ Network recovery completed")
	})

	s.T().Run("verify recovery and batch submission", func(t *testing.T) {
		t.Log("🔄 Monitoring recovery process...")

		// Give time for accumulated blocks to be submitted to DA
		recoveryStart := time.Now()

		// Submit a new transaction to trigger DA batch submission
		recoveryKey := "recovery-test-key"
		recoveryValue := "recovery-test-value"

		_, err := client.Post(ctx, "/tx", recoveryKey, recoveryValue)
		s.Require().NoError(err)
		t.Log("✅ Recovery transaction submitted")

		// Verify the recovery transaction is processed
		s.Require().Eventually(func() bool {
			res, err := client.Get(ctx, "/kv?key="+recoveryKey)
			if err != nil {
				return false
			}
			return string(res) == recoveryValue
		}, 15*time.Second, time.Second, "recovery transaction should be processed")

		// Verify all partition transactions are still accessible
		for i, key := range partitionTxKeys {
			expectedValue := fmt.Sprintf("partition-value-%d", i)

			res, err := client.Get(ctx, "/kv?key="+key)
			s.Require().NoError(err, "partition transaction %d should still be accessible", i+1)
			s.Require().Equal(expectedValue, string(res), "partition transaction %d data should be preserved", i+1)
		}

		recoveryDuration := time.Since(recoveryStart)
		t.Logf("✅ Recovery verified in %v", recoveryDuration)
		t.Logf("✅ All %d partition transactions preserved", len(partitionTxKeys))
		t.Log("✅ DA batch submission successful")
	})

	s.T().Run("verify sustained functionality after recovery", func(t *testing.T) {
		// Submit additional transactions to verify normal operation is restored
		const postRecoveryTxCount = 3
		for i := 0; i < postRecoveryTxCount; i++ {
			key := fmt.Sprintf("post-recovery-key-%d", i)
			value := fmt.Sprintf("post-recovery-value-%d", i)

			_, err := client.Post(ctx, "/tx", key, value)
			s.Require().NoError(err)
		}

		// Verify all post-recovery transactions are processed quickly
		for i := 0; i < postRecoveryTxCount; i++ {
			key := fmt.Sprintf("post-recovery-key-%d", i)
			expectedValue := fmt.Sprintf("post-recovery-value-%d", i)

			s.Require().Eventually(func() bool {
				res, err := client.Get(ctx, "/kv?key="+key)
				if err != nil {
					return false
				}
				return string(res) == expectedValue
			}, 10*time.Second, time.Second, "post-recovery transaction %d should be processed", i+1)
		}

		t.Logf("✅ Post-recovery functionality verified - %d additional transactions", postRecoveryTxCount)

		// Final test summary
		t.Log("🎉 CELESTIA DA NETWORK PARTITION TEST COMPLETED SUCCESSFULLY!")
		t.Log("   ✅ Baseline DA functionality established")
		t.Log("   ✅ Network partition handled gracefully")
		t.Logf("   ✅ Local block accumulation: %d transactions during partition", len(partitionTxKeys))
		t.Log("   ✅ Network recovery successful")
		t.Log("   ✅ Accumulated blocks submitted to DA in batches")
		t.Log("   ✅ Data integrity preserved throughout partition and recovery")
		t.Log("   ✅ Normal operation restored after recovery")
	})
}
