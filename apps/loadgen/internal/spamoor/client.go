package spamoor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	dto "github.com/prometheus/client_model/go"

	spamoorapi "github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
)

const DefaultURL = "http://spamoor-daemon:8080"

// URL returns the spamoor-daemon API URL from BENCH_SPAMOOR_URL,
// falling back to DefaultURL.
func URL() string {
	if v := os.Getenv("BENCH_SPAMOOR_URL"); v != "" {
		return v
	}
	return DefaultURL
}

// Client captures the subset of spamoor-daemon operations used by loadgen.
type Client interface {
	URL() string
	ListSpammers() ([]spamoorapi.Spammer, error)
	DeleteSpammer(id int) error
	CreateSpammer(name, scenario string, config any, start bool) (int, error)
	GetSpammer(id int) (*spamoorapi.Spammer, error)
	GetMetrics() (map[string]*dto.MetricFamily, error)
	GetClients() ([]spamoorapi.Client, error)
}

type apiClient struct {
	api        *spamoorapi.API
	httpClient *http.Client
}

// NewClient creates a Client backed by the real spamoor HTTP API.
func NewClient(baseURL string) Client {
	return apiClient{
		api:        spamoorapi.NewAPI(baseURL),
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}
}

func (c apiClient) URL() string { return c.api.BaseURL }

func (c apiClient) ListSpammers() ([]spamoorapi.Spammer, error) { return c.api.ListSpammers() }

func (c apiClient) DeleteSpammer(id int) error { return c.api.DeleteSpammer(id) }

func (c apiClient) CreateSpammer(name, scenario string, config any, start bool) (int, error) {
	return c.api.CreateSpammer(name, scenario, config, start)
}

func (c apiClient) GetSpammer(id int) (*spamoorapi.Spammer, error) { return c.api.GetSpammer(id) }

func (c apiClient) GetMetrics() (map[string]*dto.MetricFamily, error) {
	return c.api.GetMetrics()
}

// clientResponse matches the actual spamoor daemon JSON response where
// the block height field is "block_height".
type clientResponse struct {
	Index       int      `json:"index"`
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Groups      []string `json:"groups"`
	Enabled     bool     `json:"enabled"`
	BlockHeight uint64   `json:"block_height"`
}

// GetClients fetches clients from spamoor, correctly mapping "block_height" to Height.
func (c apiClient) GetClients() ([]spamoorapi.Client, error) {
	url := fmt.Sprintf("%s/api/clients", c.api.BaseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("get clients: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get clients failed: %s", string(body))
	}
	var raw []clientResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode clients response: %w", err)
	}
	clients := make([]spamoorapi.Client, len(raw))
	for i, r := range raw {
		clients[i] = spamoorapi.Client{
			Index:   r.Index,
			Name:    r.Name,
			URL:     r.URL,
			Groups:  r.Groups,
			Enabled: r.Enabled,
			Height:  r.BlockHeight,
		}
	}
	return clients, nil
}
