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

// BestKnownHeightProvider returns the best-known network height observed by the node
type BestKnownHeightProvider func() uint64

// RegisterCustomHTTPEndpoints registers custom HTTP handlers on the mux.
// See docs/learn/config.md#health-endpoints for health endpoint documentation.
func RegisterCustomHTTPEndpoints(mux *http.ServeMux, s store.Store, pm p2p.P2PRPC, cfg config.Config, bestKnownHeightProvider BestKnownHeightProvider, logger zerolog.Logger) {
	// /health/live - Liveness probe: checks if process is alive and responsive
	// Should NOT check business logic (block production, sync status, etc.)
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

	// /health/ready - Readiness probe: checks if node can serve correct data
	// Failing readiness removes node from load balancer but doesn't kill process
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
			timeSinceLastBlock := time.Since(state.LastBlockTime)
			maxAllowedDelay := 5 * cfg.Node.BlockTime.Duration

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
