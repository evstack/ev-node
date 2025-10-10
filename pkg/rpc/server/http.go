package server

import (
	"encoding/json"
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
func RegisterCustomHTTPEndpoints(
	mux *http.ServeMux,
	s store.Store,
	pm p2p.P2PRPC,
	cfg config.Config,
	bestKnownHeightProvider BestKnownHeightProvider,
	raftNode RaftNodeSource,
) {
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

	// optional Raft node details
	if raftNode != nil {
		mux.HandleFunc("/raft/node", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			rsp := struct {
				IsLeader bool   `json:"is_leader"`
				NodeId   string `json:"node_id"`
			}{
				IsLeader: raftNode.IsLeader(),
				NodeId:   raftNode.NodeID(),
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(rsp); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		})

		mux.HandleFunc("/raft/join", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			defer r.Body.Close()

			var rsp struct {
				NodeID  string `json:"id"`
				Address string `json:"address"`
			}
			if err := json.NewDecoder(r.Body).Decode(&rsp); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if rsp.NodeID == "" || rsp.Address == "" {
				http.Error(w, "id and address are required", http.StatusBadRequest)
				return
			}
			//if !raftNode.IsLeader() {
			//	http.Error(w, "not leader node", http.StatusMethodNotAllowed)
			//	return
			//}
			if err := raftNode.AddPeer(rsp.NodeID, rsp.Address); err != nil {
				http.Error(w, "failed to join peer", http.StatusInternalServerError)
			}
		})
		mux.HandleFunc("/raft/remove", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			defer r.Body.Close()
			var rsp struct {
				NodeID string `json:"id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&rsp); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
			}
			if rsp.NodeID == "" {
				http.Error(w, "id is required", http.StatusBadRequest)
				return
			}
			//if !raftNode.IsLeader() {
			//	http.Error(w, "not leader node", http.StatusMethodNotAllowed)
			//	return
			//}
			if err := raftNode.RemovePeer(rsp.NodeID); err != nil {
				http.Error(w, "failed to remove peer", http.StatusInternalServerError)
				return
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
