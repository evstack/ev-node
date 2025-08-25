package p2p

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// DAHeightResponse represents the response from a peer's DA height query
type DAHeightResponse struct {
	Height    uint64    `json:"height"`
	Timestamp time.Time `json:"timestamp"`
}

// QueryPeersDAHeight queries connected peers for their DA included height
// Returns the maximum height found across all responsive peers
func (c *Client) QueryPeersDAHeight(ctx context.Context, timeout time.Duration) (uint64, error) {
	peers, err := c.GetPeers()
	if err != nil {
		return 0, fmt.Errorf("failed to get peers: %w", err)
	}

	c.logger.Debug().Int("peer_count", len(peers)).Msg("querying peers for DA height")

	if len(peers) == 0 {
		return 0, fmt.Errorf("no connected peers available to query")
	}

	// Channel to collect results from peer queries
	type peerResult struct {
		peerID peer.ID
		height uint64
		err    error
	}
	
	resultCh := make(chan peerResult, len(peers))
	
	// Query each peer concurrently
	for _, peerInfo := range peers {
		go func(pInfo peer.AddrInfo) {
			height, err := c.queryPeerDAHeight(ctx, pInfo, timeout)
			resultCh <- peerResult{
				peerID: pInfo.ID,
				height: height,
				err:    err,
			}
		}(peerInfo)
	}
	
	// Collect results with timeout
	var maxHeight uint64
	var successCount int
	
	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	for i := 0; i < len(peers); i++ {
		select {
		case result := <-resultCh:
			if result.err != nil {
				c.logger.Debug().
					Str("peer_id", result.peerID.String()).
					Err(result.err).
					Msg("failed to query peer for DA height")
				continue
			}
			
			successCount++
			if result.height > maxHeight {
				maxHeight = result.height
			}
			
			c.logger.Debug().
				Str("peer_id", result.peerID.String()).
				Uint64("height", result.height).
				Msg("received DA height from peer")
				
		case <-queryCtx.Done():
			c.logger.Warn().
				Int("responses", successCount).
				Int("total_peers", len(peers)).
				Msg("timeout while querying peers for DA height")
			break
		}
	}
	
	if successCount == 0 {
		return 0, fmt.Errorf("no peers responded successfully to DA height query")
	}
	
	c.logger.Info().
		Uint64("max_height", maxHeight).
		Int("successful_queries", successCount).
		Int("total_peers", len(peers)).
		Msg("completed peer DA height queries")
		
	return maxHeight, nil
}

// queryPeerDAHeight queries a single peer for their DA included height
func (c *Client) queryPeerDAHeight(ctx context.Context, peerInfo peer.AddrInfo, timeout time.Duration) (uint64, error) {
	// For now, we'll try to infer the HTTP RPC endpoint from the peer's multiaddr
	// This is a simplified approach - in a production system, peers might advertise their RPC endpoints
	
	// Try common RPC ports - this is a heuristic approach
	// In practice, peers could advertise their RPC endpoints through peer discovery protocols
	rpcPorts := []string{"8080", "8081", "26657"}
	
	for _, addr := range peerInfo.Addrs {
		// Extract IP from multiaddr
		ip := extractIPFromMultiaddr(addr.String())
		if ip == "" {
			continue
		}
		
		// Try each potential RPC port
		for _, port := range rpcPorts {
			endpoint := fmt.Sprintf("http://%s:%s", ip, port)
			height, err := c.queryEndpointDAHeight(ctx, endpoint, timeout)
			if err == nil {
				return height, nil
			}
		}
	}
	
	return 0, fmt.Errorf("could not query DA height from peer %s", peerInfo.ID.String())
}

// queryEndpointDAHeight queries a specific HTTP endpoint for DA height
func (c *Client) queryEndpointDAHeight(ctx context.Context, endpoint string, timeout time.Duration) (uint64, error) {
	// Create HTTP client with timeout
	_ = &http.Client{
		Timeout: timeout,
	}
	
	// Try the store service endpoint to get DA included height
	// This uses the existing RPC infrastructure
	_ = fmt.Sprintf("%s/evnode.v1.StoreService/GetMetadata", endpoint)
	
	// TODO: Implement proper RPC call to get DA included height from store
	// For now, return 0 to indicate this is a placeholder
	// In a complete implementation, this would make a proper gRPC/Connect call
	// to the store service to retrieve the DA included height from metadata
	
	return 0, fmt.Errorf("DA height querying not fully implemented yet")
}

// extractIPFromMultiaddr extracts IP address from a multiaddr string
func extractIPFromMultiaddr(multiaddr string) string {
	// This is a simplified implementation
	// In practice, you'd use the multiaddr library to properly parse addresses
	// For now, return empty string to indicate we can't extract IP
	return ""
}