//go:build evm

package e2e

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

// SpamoorAPI is a thin HTTP client for the spamoor-daemon API
type SpamoorAPI struct {
    BaseURL string // e.g., http://127.0.0.1:8080
    client  *http.Client
}

func NewSpamoorAPI(baseURL string) *SpamoorAPI {
    return &SpamoorAPI{BaseURL: baseURL, client: &http.Client{Timeout: 2 * time.Second}}
}

type createSpammerReq struct {
    Name             string `json:"name"`
    Description      string `json:"description"`
    Scenario         string `json:"scenario"`
    ConfigYAML       string `json:"config"`
    StartImmediately bool   `json:"startImmediately"`
}

// CreateSpammer posts a new spammer; returns its ID.
func (api *SpamoorAPI) CreateSpammer(name, scenario, configYAML string, start bool) (int, error) {
    reqBody := createSpammerReq{Name: name, Description: name, Scenario: scenario, ConfigYAML: configYAML, StartImmediately: start}
    b, _ := json.Marshal(reqBody)
    url := fmt.Sprintf("%s/api/spammer", api.BaseURL)
    resp, err := api.client.Post(url, "application/json", bytes.NewReader(b))
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        body, _ := io.ReadAll(resp.Body)
        return 0, fmt.Errorf("create spammer failed: %s", string(body))
    }
    var id int
    dec := json.NewDecoder(resp.Body)
    if err := dec.Decode(&id); err != nil {
        return 0, fmt.Errorf("decode id: %w", err)
    }
    return id, nil
}

// ValidateScenarioConfig attempts to create a spammer with the provided scenario/config
// without starting it, and deletes it immediately if creation succeeds. It returns
// a descriptive error when the daemon rejects the config.
func (api *SpamoorAPI) ValidateScenarioConfig(name, scenario, configYAML string) error {
    id, err := api.CreateSpammer(name, scenario, configYAML, false)
    if err != nil {
        return fmt.Errorf("invalid scenario config: %w", err)
    }
    // Best-effort cleanup of the temporary spammer
    _ = api.DeleteSpammer(id)
    return nil
}

// DeleteSpammer deletes an existing spammer by ID.
func (api *SpamoorAPI) DeleteSpammer(id int) error {
    url := fmt.Sprintf("%s/api/spammer/%d", api.BaseURL, id)
    req, _ := http.NewRequest(http.MethodDelete, url, http.NoBody)
    resp, err := api.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("delete spammer failed: %s", string(body))
    }
    return nil
}

// StartSpammer sends a start request for a given spammer ID.
func (api *SpamoorAPI) StartSpammer(id int) error {
    url := fmt.Sprintf("%s/api/spammer/%d/start", api.BaseURL, id)
    req, _ := http.NewRequest(http.MethodPost, url, http.NoBody)
    resp, err := api.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("start spammer failed: %s", string(body))
    }
    return nil
}

// GetMetrics fetches the Prometheus /metrics endpoint from the daemon.
func (api *SpamoorAPI) GetMetrics() (string, error) {
    url := fmt.Sprintf("%s/metrics", api.BaseURL)
    resp, err := api.client.Get(url)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        body, _ := io.ReadAll(resp.Body)
        return "", fmt.Errorf("metrics request failed: %s", string(body))
    }
    b, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }
    return string(b), nil
}

// Spammer represents a spammer resource minimally for status checks.
type Spammer struct {
    ID      int    `json:"id"`
    Name    string `json:"name"`
    Scenario string `json:"scenario"`
    Status  int    `json:"status"`
}

// GetSpammer retrieves a spammer by ID.
func (api *SpamoorAPI) GetSpammer(id int) (*Spammer, error) {
    url := fmt.Sprintf("%s/api/spammer/%d", api.BaseURL, id)
    resp, err := api.client.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("get spammer failed: %s", string(body))
    }
    var s Spammer
    dec := json.NewDecoder(resp.Body)
    if err := dec.Decode(&s); err != nil {
        return nil, err
    }
    return &s, nil
}
