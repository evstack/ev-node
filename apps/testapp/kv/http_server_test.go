package executor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandleTx(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid transaction",
			method:         http.MethodPost,
			body:           "testkey=testvalue",
			expectedStatus: http.StatusAccepted,
			expectedBody:   "Transaction accepted",
		},
		{
			name:           "Empty transaction",
			method:         http.MethodPost,
			body:           "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Empty transaction\n",
		},
		{
			name:           "Invalid method",
			method:         http.MethodGet,
			body:           "testkey=testvalue",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := NewKVExecutor()
			server := NewHTTPServer(exec, ":0")

			req := httptest.NewRequest(tt.method, "/tx", strings.NewReader(tt.body))
			rr := httptest.NewRecorder()

			server.handleTx(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if rr.Body.String() != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, rr.Body.String())
			}

			if tt.method == http.MethodPost && tt.expectedStatus == http.StatusAccepted {
				time.Sleep(10 * time.Millisecond)
				ctx := context.Background()
				retrievedTxs, err := exec.GetTxs(ctx)
				if err != nil {
					t.Fatalf("GetTxs failed: %v", err)
				}
				if len(retrievedTxs) != 1 {
					t.Errorf("expected 1 transaction in channel, got %d", len(retrievedTxs))
				} else if string(retrievedTxs[0]) != tt.body {
					t.Errorf("expected channel to contain %q, got %q", tt.body, string(retrievedTxs[0]))
				}
			} else if tt.method == http.MethodPost {
				ctx := context.Background()
				retrievedTxs, err := exec.GetTxs(ctx)
				if err != nil {
					t.Fatalf("GetTxs failed: %v", err)
				}
				if len(retrievedTxs) != 0 {
					t.Errorf("expected 0 transactions in channel for failed POST, got %d", len(retrievedTxs))
				}
			}
		})
	}
}

func TestHandleKV_Get(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		value          string
		queryParam     string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Get existing key",
			key:            "mykey",
			value:          "myvalue",
			queryParam:     "mykey",
			expectedStatus: http.StatusOK,
			expectedBody:   "myvalue",
		},
		{
			name:           "Get non-existent key",
			key:            "mykey",
			value:          "myvalue",
			queryParam:     "nonexistent",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Key not found\n",
		},
		{
			name:           "Missing key parameter",
			key:            "mykey",
			value:          "myvalue",
			queryParam:     "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing key parameter\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := NewKVExecutor()
			server := NewHTTPServer(exec, ":0")

			if tt.key != "" && tt.value != "" {
				tx := fmt.Appendf(nil, "%s=%s", tt.key, tt.value)
				ctx := context.Background()
				_, err := exec.ExecuteTxs(ctx, [][]byte{tx}, 1, time.Now(), []byte(""))
				if err != nil {
					t.Fatalf("Failed to execute setup transaction: %v", err)
				}
			}

			url := "/kv"
			if tt.queryParam != "" {
				url += "?key=" + tt.queryParam
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rr := httptest.NewRecorder()

			server.handleKV(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if rr.Body.String() != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, rr.Body.String())
			}
		})
	}
}

func TestHTTPServerStartStop(t *testing.T) {
	exec := NewKVExecutor()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	httpServer := NewHTTPServer(exec, server.URL)
	if httpServer == nil {
		t.Fatal("NewHTTPServer returned nil")
	}

	if httpServer.executor != exec {
		t.Error("HTTPServer.executor not set correctly")
	}

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	testServer := &HTTPServer{
		server: &http.Server{
			Addr:              ":0",
			ReadHeaderTimeout: 10 * time.Second,
		},
		executor: exec,
	}

	_ = testServer.Start
}

func TestHTTPServerContextCancellation(t *testing.T) {
	exec := NewKVExecutor()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("Failed to close listener: %v", err)
	}

	serverAddr := fmt.Sprintf("127.0.0.1:%d", port)
	server := NewHTTPServer(exec, serverAddr)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s/store", serverAddr))
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("Failed to close response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil && errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("Server shutdown error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Server shutdown timed out")
	}

	_, err = client.Get(fmt.Sprintf("http://%s/store", serverAddr))
	if err == nil {
		t.Fatal("Expected connection error after shutdown, but got none")
	}
}

func TestHTTPIntegration_GetKVWithMultipleHeights(t *testing.T) {
	exec := NewKVExecutor()
	ctx := context.Background()

	txsHeight1 := [][]byte{[]byte("testkey=original_value")}
	_, err := exec.ExecuteTxs(ctx, txsHeight1, 1, time.Now(), []byte(""))
	if err != nil {
		t.Fatalf("ExecuteTxs failed for height 1: %v", err)
	}

	txsHeight2 := [][]byte{[]byte("testkey=updated_value")}
	_, err = exec.ExecuteTxs(ctx, txsHeight2, 2, time.Now(), []byte(""))
	if err != nil {
		t.Fatalf("ExecuteTxs failed for height 2: %v", err)
	}

	server := NewHTTPServer(exec, ":0")

	req := httptest.NewRequest(http.MethodGet, "/kv?key=testkey", nil)
	rr := httptest.NewRecorder()

	server.handleKV(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if rr.Body.String() != "updated_value" {
		t.Errorf("expected body 'updated_value', got %q", rr.Body.String())
	}
}
