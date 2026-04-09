package executor

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	ds "github.com/ipfs/go-datastore"
)

var acceptedResp = []byte("Transaction accepted")

type HTTPServer struct {
	executor    *KVExecutor
	server      *http.Server
	injectedTxs atomic.Uint64
}

func NewHTTPServer(executor *KVExecutor, listenAddr string) *HTTPServer {
	hs := &HTTPServer{
		executor: executor,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/tx", hs.handleTx)
	mux.HandleFunc("/tx/batch", hs.handleTxBatch)
	mux.HandleFunc("/kv", hs.handleKV)
	mux.HandleFunc("/store", hs.handleStore)
	mux.HandleFunc("/stats", hs.handleStats)

	hs.server = &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    4096,
	}

	return hs
}

func (hs *HTTPServer) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("KV Executor HTTP server starting on %s\n", hs.server.Addr)
		if err := hs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		fmt.Printf("KV Executor HTTP server shutting down on %s\n", hs.server.Addr)
		if err := hs.server.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("KV Executor HTTP server shutdown error: %v\n", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

func (hs *HTTPServer) Stop() error {
	return hs.server.Close()
}

func (hs *HTTPServer) handleTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
	r.Body.Close()
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		http.Error(w, "Empty transaction", http.StatusBadRequest)
		return
	}

	hs.executor.InjectTx(body)
	hs.injectedTxs.Add(1)
	w.WriteHeader(http.StatusAccepted)
	w.Write(acceptedResp)
}

func (hs *HTTPServer) handleTxBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	r.Body.Close()
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Empty body"))
		return
	}

	txs := splitLines(body)
	accepted := hs.executor.InjectTxs(txs)
	hs.injectedTxs.Add(uint64(accepted))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"accepted": accepted})
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	scanner := bufio.NewScanner(bytesReader(data))
	scanner.Buffer(make([]byte, 0, 256), 4096)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) > 0 {
			tx := make([]byte, len(line))
			copy(tx, line)
			lines = append(lines, tx)
		}
	}
	return lines
}

type bytesReaderImpl struct {
	data []byte
	pos  int
}

func bytesReader(data []byte) *bytesReaderImpl {
	return &bytesReaderImpl{data: data}
}

func (r *bytesReaderImpl) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (hs *HTTPServer) handleKV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Missing key parameter", http.StatusBadRequest)
		return
	}

	value, exists := hs.executor.GetStoreValue(r.Context(), key)
	if !exists {
		if _, err := hs.executor.db.Get(r.Context(), ds.NewKey(key)); errors.Is(err, ds.ErrNotFound) {
			http.Error(w, "Key not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve key", http.StatusInternalServerError)
			fmt.Printf("Error retrieving key '%s' from DB: %v\n", key, err)
		}
		return
	}

	w.Write([]byte(value))
}

func (hs *HTTPServer) handleStore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	store := hs.executor.GetAllEntries()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(store); err != nil {
		fmt.Printf("Error encoding JSON response: %v\n", err)
	}
}

func (hs *HTTPServer) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	execStats := hs.executor.GetStats()
	stats := struct {
		InjectedTxs    uint64 `json:"injected_txs"`
		ExecutedTxs    uint64 `json:"executed_txs"`
		BlocksProduced uint64 `json:"blocks_produced"`
	}{
		InjectedTxs:    hs.injectedTxs.Load(),
		ExecutedTxs:    execStats.TotalExecutedTxs,
		BlocksProduced: execStats.BlocksProduced,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
