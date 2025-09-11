//go:build e2e

package e2e

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"net/http/httptest"

	"github.com/evstack/ev-node/pkg/lease"
)

// testLeaseServer is a lightweight HTTP server that exposes a simple lease API
// backed by the in-memory lease implementation. It is intended for tests only.
//
// Endpoints (all under /leases/{name}):
// - POST /acquire  body: {"node_id":"...","duration_ns":<int64>}
//     -> 200 OK on acquired, 409 Conflict if already held by another node
// - POST /renew    body: same as acquire
//     -> 200 OK, 403 Forbidden if not held, 410 Gone if expired
// - POST /release  body: {"node_id":"..."}
//     -> 200 OK, 403 Forbidden if not held
// - GET  /holder   -> 204 No Content if none, 200 OK with {"holder":"..."}
// - GET  /expiry   -> 200 OK with {"expiry":"RFC3339 time"}
//
// The server keeps one MemoryLease per lease name.

type testLeaseServer struct {
	mu    sync.Mutex
	lease *lease.MemoryLease
}

type leaseRequest struct {
	NodeID   string        `json:"node_id"`
	Duration time.Duration `json:"duration_ns"`
}

type holderResponse struct {
	Holder string `json:"holder"`
}

type expiryResponse struct {
	Expiry time.Time `json:"expiry"`
}

func newTestLeaseServer(leaseName string) *testLeaseServer {
	return &testLeaseServer{lease: lease.NewMemoryLease(leaseName)}
}

func (s *testLeaseServer) getLease(name string) *lease.MemoryLease {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name != s.lease.Name() {
		return nil
	}
	return s.lease
}

func (s *testLeaseServer) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/leases/", func(w http.ResponseWriter, r *http.Request) {
		// path: /leases/{name}/action
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/leases/"), "/")
		if len(parts) < 2 {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		name, action := parts[0], parts[1]
		l := s.getLease(name)
		if l == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		switch action {
		case "acquire":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req leaseRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			ok, err := l.Acquire(r.Context(), req.NodeID, req.Duration)
			if err != nil {
				if errors.Is(err, lease.ErrInvalidLeaseTerm) {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if !ok {
				w.WriteHeader(http.StatusConflict)
				return
			}
			w.WriteHeader(http.StatusOK)
		case "renew":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req leaseRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			if err := l.Renew(r.Context(), req.NodeID, req.Duration); err != nil {
				switch err {
				case lease.ErrLeaseNotHeld:
					http.Error(w, err.Error(), http.StatusForbidden)
					return
				case lease.ErrLeaseExpired:
					http.Error(w, err.Error(), http.StatusGone)
					return
				case lease.ErrInvalidLeaseTerm:
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				default:
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			w.WriteHeader(http.StatusOK)
		case "release":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req leaseRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			if err := l.Release(r.Context(), req.NodeID); err != nil {
				if errors.Is(err, lease.ErrLeaseNotHeld) {
					http.Error(w, err.Error(), http.StatusForbidden)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		case "holder":
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			h, err := l.GetHolder(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if h == "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			_ = json.NewEncoder(w).Encode(holderResponse{Holder: h})
		case "expiry":
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			exp, err := l.GetExpiry(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(expiryResponse{Expiry: exp})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	})
	return mux
}

// startLeaseHTTPTestServer starts the lease HTTP server for tests and returns its base URL and a shutdown function.
func startLeaseHTTPTestServer(leaseName string) (baseURL string, shutdown func()) {
	ts := newTestLeaseServer(leaseName)
	h := ts.handler()
	server := httptest.NewServer(h)
	return server.URL, func() { server.Close() }
}
