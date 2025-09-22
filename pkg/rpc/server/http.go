package server

import (
	"fmt"
	"net/http"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/store"
)

// BestKnownHeightProvider should return the best-known network height observed by the node
// (e.g. min(headerSyncHeight, dataSyncHeight) for full nodes, or header height for light nodes).
type BestKnownHeightProvider func() uint64

// RegisterCustomHTTPEndpoints is the designated place to add new, non-gRPC, plain HTTP handlers.
// Additional custom HTTP endpoints can be registered on the mux here.
func RegisterCustomHTTPEndpoints(mux *http.ServeMux, s store.Store, pm p2p.P2PRPC, cfg config.Config, bestKnownHeightProvider BestKnownHeightProvider) {
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
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
