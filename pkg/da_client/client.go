package da_client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HeightResponse represents the response from the /da/height endpoint
type HeightResponse struct {
	Height    uint64    `json:"height"`
	Timestamp time.Time `json:"timestamp"`
}

// Client is a simple HTTP client for polling DA height from aggregator endpoints
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new DA client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetDAHeight polls the given aggregator endpoint for the current DA height
func (c *Client) GetDAHeight(ctx context.Context, aggregatorEndpoint string) (uint64, error) {
	if aggregatorEndpoint == "" {
		return 0, fmt.Errorf("aggregator endpoint is empty")
	}

	url := fmt.Sprintf("%s/da/height", aggregatorEndpoint)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to make request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("aggregator returned status %d from %s", resp.StatusCode, url)
	}

	var heightResp HeightResponse
	if err := json.NewDecoder(resp.Body).Decode(&heightResp); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return heightResp.Height, nil
}

// PollDAHeight polls the aggregator endpoint until it gets a height > 0 or the context is cancelled
func (c *Client) PollDAHeight(ctx context.Context, aggregatorEndpoint string, pollInterval time.Duration) (uint64, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-ticker.C:
			height, err := c.GetDAHeight(ctx, aggregatorEndpoint)
			if err != nil {
				// Log the error but continue polling
				continue
			}
			if height > 0 {
				return height, nil
			}
			// Height is 0, continue polling
		}
	}
}