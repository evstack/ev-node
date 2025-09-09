package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/pkg/lease"
)

// Demo demonstrates leader election with multiple nodes
func main() {
	fmt.Println("=== Leader Election Demo ===")
	fmt.Println("This demo shows active-active failover with leader election")
	fmt.Println()

	// Create shared lease backend (in production this would be Kubernetes Lease or etcd)
	sharedLease := lease.NewMemoryLease("demo-leader")
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	// Create multiple nodes
	nodes := []string{"node-1", "node-2", "node-3"}
	var elections []*lease.LeaderElection
	var wg sync.WaitGroup

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Starting %d nodes with leader election...\n", len(nodes))
	fmt.Println()

	// Start all nodes
	for _, nodeID := range nodes {
		election := lease.NewLeaderElection(
			sharedLease,
			nodeID,
			"demo-leader",
			5*time.Second, // 5 second lease term
			logger.With().Str("node", nodeID).Logger(),
		)
		elections = append(elections, election)

		wg.Add(1)
		go func(nodeID string, election *lease.LeaderElection) {
			defer wg.Done()
			runNode(ctx, nodeID, election)
		}(nodeID, election)

		// Start election
		if err := election.Start(ctx); err != nil {
			log.Fatalf("Failed to start election for %s: %v", nodeID, err)
		}
	}

	// Let the demo run
	fmt.Println("Demo running for 30 seconds...")
	fmt.Println("Watch the leadership transitions and failover behavior")
	fmt.Println("Press Ctrl+C to stop early")
	fmt.Println()

	// Simulate leader failure after 15 seconds
	go func() {
		time.Sleep(15 * time.Second)
		fmt.Println("\n🔥 SIMULATING LEADER FAILURE - Stopping current leader...")
		
		// Find current leader and stop it
		for i, election := range elections {
			if election.IsLeader() {
				fmt.Printf("Stopping leader: %s\n", nodes[i])
				election.Stop()
				break
			}
		}
		fmt.Println("Watch failover happen...")
		fmt.Println()
	}()

	// Wait for completion
	wg.Wait()

	// Cleanup
	for _, election := range elections {
		election.Stop()
	}

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("Leader election successfully demonstrated active-active failover!")
}

// runNode simulates a node participating in leader election
func runNode(ctx context.Context, nodeID string, election *lease.LeaderElection) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	isLeader := false
	blockCount := 0

	for {
		select {
		case <-ctx.Done():
			if isLeader {
				fmt.Printf("🔴 [%s] Stepping down as leader (context done)\n", nodeID)
			}
			return

		case leaderStatus := <-election.LeaderChan():
			if leaderStatus && !isLeader {
				fmt.Printf("👑 [%s] BECAME LEADER - Starting block production\n", nodeID)
				isLeader = true
			} else if !leaderStatus && isLeader {
				fmt.Printf("🔴 [%s] LOST LEADERSHIP - Stopping block production\n", nodeID)
				isLeader = false
			}

		case <-ticker.C:
			if isLeader {
				blockCount++
				fmt.Printf("📦 [%s] Produced block #%d (as leader)\n", nodeID, blockCount)
			} else {
				// Hot standby - maintain sync but don't produce blocks
				fmt.Printf("👁  [%s] Syncing blocks (hot standby)\n", nodeID)
			}

			// Show current leader
			if leader, err := election.GetLeader(ctx); err == nil && leader != "" {
				if leader == nodeID && !isLeader {
					// Detected leadership but not notified yet
					fmt.Printf("⚠️  [%s] Leadership detected but not yet active\n", nodeID)
				}
			}
		}
	}
}