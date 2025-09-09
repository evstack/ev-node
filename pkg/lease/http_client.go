package lease

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var _ Lease = &HTTPLease{}

// HTTPLease implements the Lease interface over a simple HTTP API
// It is intended primarily for tests and local development.
type HTTPLease struct {
	baseURL   string
	leaseName string
	client    *http.Client
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

// NewHTTPLease creates a new HTTP-backed Lease implementation
func NewHTTPLease(baseURL, leaseName string) *HTTPLease {
	return &HTTPLease{
		baseURL:   strings.TrimRight(baseURL, "/"),
		leaseName: leaseName,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (h *HTTPLease) Acquire(ctx context.Context, nodeID string, duration time.Duration) (bool, error) {
	if duration <= 0 {
		return false, ErrInvalidLeaseTerm
	}
	req := leaseRequest{NodeID: nodeID, Duration: duration}
	b, _ := json.Marshal(req)
	url := fmt.Sprintf("%s/leases/%s/acquire", h.baseURL, h.leaseName)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(httpReq)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusConflict {
		return false, nil
	}
	return false, fmt.Errorf("unexpected status: %d", resp.StatusCode)
}

func (h *HTTPLease) Renew(ctx context.Context, nodeID string, duration time.Duration) error {
	if duration <= 0 {
		return ErrInvalidLeaseTerm
	}
	req := leaseRequest{NodeID: nodeID, Duration: duration}
	b, _ := json.Marshal(req)
	url := fmt.Sprintf("%s/leases/%s/renew", h.baseURL, h.leaseName)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusGone:
		return ErrLeaseExpired
	case http.StatusForbidden:
		return ErrLeaseNotHeld
	default:
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
}

func (h *HTTPLease) Release(ctx context.Context, nodeID string) error {
	req := leaseRequest{NodeID: nodeID}
	b, _ := json.Marshal(req)
	url := fmt.Sprintf("%s/leases/%s/release", h.baseURL, h.leaseName)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusForbidden:
		return ErrLeaseNotHeld
	default:
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
}

func (h *HTTPLease) IsHeld(ctx context.Context, nodeID string) (bool, error) {
	holder, err := h.GetHolder(ctx)
	if err != nil {
		return false, err
	}
	return holder == nodeID, nil
}

func (h *HTTPLease) GetHolder(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/leases/%s/holder", h.baseURL, h.leaseName)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := h.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	var hr holderResponse
	if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil {
		return "", err
	}
	return hr.Holder, nil
}

func (h *HTTPLease) GetExpiry(ctx context.Context) (time.Time, error) {
	url := fmt.Sprintf("%s/leases/%s/expiry", h.baseURL, h.leaseName)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := h.client.Do(httpReq)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	var er expiryResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		return time.Time{}, err
	}
	return er.Expiry, nil
}
