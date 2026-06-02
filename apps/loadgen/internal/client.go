package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	dto "github.com/prometheus/client_model/go"

	"github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
)

// SpamoorClient captures the subset of spamoor-daemon operations used by loadgen.
type SpamoorClient interface {
	URL() string
	ListSpammers() ([]spamoor.Spammer, error)
	DeleteSpammer(id int) error
	CreateSpammer(name, scenario string, config any, start bool) (int, error)
	GetSpammer(id int) (*spamoor.Spammer, error)
	GetMetrics() (map[string]*dto.MetricFamily, error)
	GetClients() ([]spamoor.Client, error)
}

type spamoorAPIClient struct {
	api    *spamoor.API
	client *http.Client
}

// NewSpamoorClient creates a SpamoorClient backed by the real spamoor HTTP API.
func NewSpamoorClient(baseURL string) SpamoorClient {
	return spamoorAPIClient{
		api:    spamoor.NewAPI(baseURL),
		client: &http.Client{Timeout: 2 * time.Second},
	}
}

func (c spamoorAPIClient) URL() string { return c.api.BaseURL }

func (c spamoorAPIClient) ListSpammers() ([]spamoor.Spammer, error) { return c.api.ListSpammers() }

func (c spamoorAPIClient) DeleteSpammer(id int) error { return c.api.DeleteSpammer(id) }

func (c spamoorAPIClient) CreateSpammer(name, scenario string, config any, start bool) (int, error) {
	return c.api.CreateSpammer(name, scenario, config, start)
}

func (c spamoorAPIClient) GetSpammer(id int) (*spamoor.Spammer, error) { return c.api.GetSpammer(id) }

func (c spamoorAPIClient) GetMetrics() (map[string]*dto.MetricFamily, error) {
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
func (c spamoorAPIClient) GetClients() ([]spamoor.Client, error) {
	url := fmt.Sprintf("%s/api/clients", c.api.BaseURL)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get clients failed: %s", string(body))
	}
	var raw []clientResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	clients := make([]spamoor.Client, len(raw))
	for i, r := range raw {
		clients[i] = spamoor.Client{
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
