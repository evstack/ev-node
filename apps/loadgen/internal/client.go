package internal

import (
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
	api *spamoor.API
}

// NewSpamoorClient creates a SpamoorClient backed by the real spamoor HTTP API.
func NewSpamoorClient(baseURL string) SpamoorClient {
	return spamoorAPIClient{api: spamoor.NewAPI(baseURL)}
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

func (c spamoorAPIClient) GetClients() ([]spamoor.Client, error) { return c.api.GetClients() }
