package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/store"
)

// BestKnownHeightProvider returns the best-known network height observed by the node
type BestKnownHeightProvider func() uint64

// RegisterCustomHTTPEndpoints registers custom HTTP handlers on the mux.
func RegisterCustomHTTPEndpoints(mux *http.ServeMux, s store.Store, pm p2p.P2PRPC, cfg config.Config, bestKnownHeightProvider BestKnownHeightProvider, logger zerolog.Logger, raftNode RaftNodeSource) {
	// /health/live performs a basic liveness check to determine if the process is alive and responsive.
	// Returns 200 if the process can access its store, 503 otherwise.
	// This is a lightweight check suitable for Kubernetes liveness probes.
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		_, err := s.Height(r.Context())
		if err != nil {
			logger.Error().Err(err).Msg("Liveness check failed: cannot access store")
			http.Error(w, "FAIL", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	// /health/ready performs a comprehensive readiness check to determine if the node can serve correct data.
	// Returns 200 if all checks pass, 503 otherwise.
	// Suitable for Kubernetes readiness probes and load balancer health checks.
	//
	// The following checks are performed:
	// 1. P2P network connectivity (if P2P is enabled):
	//    - Verifies P2P network info is accessible
	//    - Confirms node is listening for P2P connections
	//    - For non-aggregator nodes: ensures at least one peer is connected
	// 2. Block production/sync status:
	//    - Confirms node state is accessible
	//    - Verifies at least one block has been produced/synced
	// 3. Aggregator-specific checks (for aggregator nodes only):
	//    - Validates blocks are being produced at expected rate (within 5x block_time)
	// 4. Sync status (for all nodes):
	//    - Compares local height with best known network height
	//    - Ensures node is not falling behind by more than readiness_max_blocks_behind
	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

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

		state, err := s.GetState(r.Context())
		if err != nil {
			http.Error(w, "UNREADY: state unavailable", http.StatusServiceUnavailable)
			return
		}

		localHeight := state.LastBlockHeight
		if localHeight == 0 {
			http.Error(w, "UNREADY: no blocks yet", http.StatusServiceUnavailable)
			return
		}

		if cfg.Node.Aggregator {
			var maxAllowedDelay time.Duration
			if cfg.Node.LazyMode {
				maxAllowedDelay = 2 * cfg.Node.LazyBlockInterval.Duration
			} else {
				maxAllowedDelay = 5 * cfg.Node.BlockTime.Duration
			}
			timeSinceLastBlock := time.Since(state.LastBlockTime)
			if timeSinceLastBlock > maxAllowedDelay {
				http.Error(w, "UNREADY: aggregator not producing blocks at expected rate", http.StatusServiceUnavailable)
				return
			}
		}

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

	// optional Raft node details
	if raftNode != nil {
		mux.HandleFunc("/raft/node", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			rsp := struct {
				IsLeader bool   `json:"is_leader"`
				NodeID   string `json:"node_id"`
			}{
				IsLeader: raftNode.IsLeader(),
				NodeID:   raftNode.NodeID(),
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(rsp); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		})

	}

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
