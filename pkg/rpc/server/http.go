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

const (
	// healthCheckWarnMultiplier is the multiplier for block time to determine WARN threshold
	// If no block has been produced in (blockTime * healthCheckWarnMultiplier), return WARN
	healthCheckWarnMultiplier = 3

	// healthCheckFailMultiplier is the multiplier for block time to determine FAIL threshold
	// If no block has been produced in (blockTime * healthCheckFailMultiplier), return FAIL
	healthCheckFailMultiplier = 5
)

// BestKnownHeightProvider should return the best-known network height observed by the node
// (e.g. min(headerSyncHeight, dataSyncHeight) for full nodes, or header height for light nodes).
type BestKnownHeightProvider func() uint64

// RegisterCustomHTTPEndpoints is the designated place to add new, non-gRPC, plain HTTP handlers.
// Additional custom HTTP endpoints can be registered on the mux here.
func RegisterCustomHTTPEndpoints(mux *http.ServeMux, s store.Store, pm p2p.P2PRPC, cfg config.Config, bestKnownHeightProvider BestKnownHeightProvider, logger zerolog.Logger) {
	// Liveness endpoint - checks if block production is healthy for aggregator nodes
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		// For aggregator nodes, check if block production is healthy
		if cfg.Node.Aggregator {
			state, err := s.GetState(r.Context())
			if err != nil {
				logger.Error().Err(err).Msg("Failed to get state for health check")
				http.Error(w, "FAIL", http.StatusServiceUnavailable)
				return
			}

			// If we have blocks, check if the last block time is recent
			if state.LastBlockHeight > 0 {
				timeSinceLastBlock := time.Since(state.LastBlockTime)

				// Calculate the threshold based on block time
				blockTime := cfg.Node.BlockTime.Duration

				// For lazy mode, use the lazy block interval instead
				if cfg.Node.LazyMode {
					blockTime = cfg.Node.LazyBlockInterval.Duration
				}

				warnThreshold := blockTime * healthCheckWarnMultiplier
				failThreshold := blockTime * healthCheckFailMultiplier

				if timeSinceLastBlock > failThreshold {
					logger.Error().
						Dur("time_since_last_block", timeSinceLastBlock).
						Dur("fail_threshold", failThreshold).
						Uint64("last_block_height", state.LastBlockHeight).
						Time("last_block_time", state.LastBlockTime).
						Msg("Health check: node has stopped producing blocks (FAIL)")
					http.Error(w, "FAIL", http.StatusServiceUnavailable)
					return
				} else if timeSinceLastBlock > warnThreshold {
					logger.Warn().
						Dur("time_since_last_block", timeSinceLastBlock).
						Dur("warn_threshold", warnThreshold).
						Uint64("last_block_height", state.LastBlockHeight).
						Time("last_block_time", state.LastBlockTime).
						Msg("Health check: block production is slow (WARN)")
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, "WARN")
					return
				}
			}
		}

		// For non-aggregator nodes or if checks pass, return healthy
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	// Readiness endpoint
	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		// Peer readiness: non-aggregator nodes should have at least 1 peer
		if pm != nil && !cfg.Node.Aggregator {
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

		localHeight, err := s.Height(r.Context())
		if err != nil {
			http.Error(w, "UNREADY: state unavailable", http.StatusServiceUnavailable)
			return
		}

		// If no blocks yet, consider unready
		if localHeight == 0 {
			http.Error(w, "UNREADY: no blocks yet", http.StatusServiceUnavailable)
			return
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
