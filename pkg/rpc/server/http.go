// Package server provides HTTP endpoint handlers for the RPC server.
//
// Health Endpoints:
// This file implements health check endpoints following Kubernetes best practices.
// For comprehensive documentation on health endpoints, their differences, and usage examples,
// see: docs/learn/config.md#health-endpoints
package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/rs/zerolog"
)

// BestKnownHeightProvider should return the best-known network height observed by the node
// (e.g. min(headerSyncHeight, dataSyncHeight) for full nodes, or header height for light nodes).
type BestKnownHeightProvider func() uint64

// RegisterCustomHTTPEndpoints is the designated place to add new, non-gRPC, plain HTTP handlers.
// Additional custom HTTP endpoints can be registered on the mux here.
//
// For detailed documentation on health endpoints, see: docs/learn/config.md#health-endpoints
func RegisterCustomHTTPEndpoints(mux *http.ServeMux, s store.Store, pm p2p.P2PRPC, cfg config.Config, bestKnownHeightProvider BestKnownHeightProvider, logger zerolog.Logger) {
	// Liveness endpoint - checks if the service process is alive and responsive
	// A failing liveness check should result in killing/restarting the process
	// This endpoint should NOT check business logic (like block production or sync status)
	//
	// See docs/learn/config.md#healthlive---liveness-probe for details
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		// Basic liveness check: Can we access the store?
		// This verifies the process is alive and core dependencies are accessible
		_, err := s.Height(r.Context())
		if err != nil {
			logger.Error().Err(err).Msg("Liveness check failed: cannot access store")
			http.Error(w, "FAIL", http.StatusServiceUnavailable)
			return
		}

		// Process is alive and responsive
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	// Readiness endpoint - checks if the node can serve correct data to clients
	// A failing readiness check should result in removing the node from load balancer
	// but NOT killing the process (e.g., node is syncing, no peers, etc.)
	//
	// See docs/learn/config.md#healthready---readiness-probe for details
	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		// P2P readiness: if P2P is enabled, verify it's ready to accept connections
		if pm != nil {
			netInfo, err := pm.GetNetworkInfo()
			if err != nil {
				http.Error(w, "UNREADY: failed to query P2P network info", http.StatusServiceUnavailable)
				return
			}
			if len(netInfo.ListenAddress) == 0 {
				http.Error(w, "UNREADY: P2P not listening for connections", http.StatusServiceUnavailable)
				return
			}

			// Peer readiness: non-aggregator nodes should have at least 1 peer
			if !cfg.Node.Aggregator {
				peers, err := pm.GetPeers()
				if err != nil {
					http.Error(w, "UNREADY: failed to query peers", http.StatusServiceUnavailable)
					return
				}
				if len(peers) == 0 {
					http.Error(w, "UNREADY: no peers connected", http.StatusServiceUnavailable)
					return
				}
			}
		}

		// Get current state
		state, err := s.GetState(r.Context())
		if err != nil {
			http.Error(w, "UNREADY: state unavailable", http.StatusServiceUnavailable)
			return
		}

		localHeight := state.LastBlockHeight

		// If no blocks yet, consider unready
		if localHeight == 0 {
			http.Error(w, "UNREADY: no blocks yet", http.StatusServiceUnavailable)
			return
		}

		// Aggregator block production check: verify blocks are being produced at expected rate
		if cfg.Node.Aggregator {
			timeSinceLastBlock := time.Since(state.LastBlockTime)
			maxAllowedDelay := 5 * cfg.Node.BlockTime.Duration

			if timeSinceLastBlock > maxAllowedDelay {
				http.Error(w, "UNREADY: aggregator not producing blocks at expected rate", http.StatusServiceUnavailable)
				return
			}
		}

		// Require best-known height to make the readiness decision
		if bestKnownHeightProvider == nil {
			http.Error(w, "UNREADY: best-known height unavailable", http.StatusServiceUnavailable)
			return
		}

		bestKnownHeight := bestKnownHeightProvider()
		if bestKnownHeight == 0 {
			http.Error(w, "UNREADY: best-known height unknown", http.StatusServiceUnavailable)
			return
		}

		allowedBlocksBehind := cfg.Node.ReadinessMaxBlocksBehind
		if bestKnownHeight <= localHeight {
			// local is ahead of our observed best-known consider ready
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "READY")
			return
		}

		if bestKnownHeight-localHeight > allowedBlocksBehind {
			http.Error(w, "UNREADY: behind best-known head", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "READY")
	})

	// DA Visualization endpoints
	mux.HandleFunc("/da", func(w http.ResponseWriter, r *http.Request) {
		server := GetDAVisualizationServer()
		if server == nil {
			http.Error(w, "DA visualization not available", http.StatusServiceUnavailable)
			return
		}
		server.handleDAVisualizationHTML(w, r)
	})

	mux.HandleFunc("/da/submissions", func(w http.ResponseWriter, r *http.Request) {
		server := GetDAVisualizationServer()
		if server == nil {
			http.Error(w, "DA visualization not available", http.StatusServiceUnavailable)
			return
		}
		server.handleDASubmissions(w, r)
	})

	mux.HandleFunc("/da/blob", func(w http.ResponseWriter, r *http.Request) {
		server := GetDAVisualizationServer()
		if server == nil {
			http.Error(w, "DA visualization not available", http.StatusServiceUnavailable)
			return
		}
		server.handleDABlobDetails(w, r)
	})

	mux.HandleFunc("/da/stats", func(w http.ResponseWriter, r *http.Request) {
		server := GetDAVisualizationServer()
		if server == nil {
			http.Error(w, "DA visualization not available", http.StatusServiceUnavailable)
			return
		}
		server.handleDAStats(w, r)
	})

	mux.HandleFunc("/da/health", func(w http.ResponseWriter, r *http.Request) {
		server := GetDAVisualizationServer()
		if server == nil {
			http.Error(w, "DA visualization not available", http.StatusServiceUnavailable)
			return
		}
		server.handleDAHealth(w, r)
	})

	// Example for adding more custom endpoints:
	// mux.HandleFunc("/custom/myendpoint", func(w http.ResponseWriter, r *http.Request) {
	//     // Your handler logic here
	//     w.WriteHeader(http.StatusOK)
	//     fmt.Fprintln(w, "My custom endpoint!")
	// })
}
