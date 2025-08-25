package da_client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_GetDAHeight(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse HeightResponse
		serverStatus   int
		expectError    bool
		expectedHeight uint64
	}{
		{
			name: "successful response",
			serverResponse: HeightResponse{
				Height:    42,
				Timestamp: time.Now(),
			},
			serverStatus:   http.StatusOK,
			expectError:    false,
			expectedHeight: 42,
		},
		{
			name:         "server error",
			serverStatus: http.StatusInternalServerError,
			expectError:  true,
		},
		{
			name: "zero height",
			serverResponse: HeightResponse{
				Height:    0,
				Timestamp: time.Now(),
			},
			serverStatus:   http.StatusOK,
			expectError:    false,
			expectedHeight: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/da/height" {
					http.Error(w, "not found", http.StatusNotFound)
					return
				}

				if tt.serverStatus != http.StatusOK {
					http.Error(w, "server error", tt.serverStatus)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := NewClient()
			height, err := client.GetDAHeight(context.Background(), server.URL)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if height != tt.expectedHeight {
				t.Errorf("expected height %d, got %d", tt.expectedHeight, height)
			}
		})
	}
}

func TestClient_GetDAHeight_EmptyEndpoint(t *testing.T) {
	client := NewClient()
	_, err := client.GetDAHeight(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty endpoint")
	}
}

func TestClient_PollDAHeight(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var height uint64
		if callCount == 1 {
			height = 0 // First call returns 0
		} else {
			height = 5 // Second call returns 5
		}

		resp := HeightResponse{
			Height:    height,
			Timestamp: time.Now(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	height, err := client.PollDAHeight(ctx, server.URL, 100*time.Millisecond)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if height != 5 {
		t.Errorf("expected height 5, got %d", height)
	}

	if callCount < 2 {
		t.Errorf("expected at least 2 calls, got %d", callCount)
	}
}