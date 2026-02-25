//go:build evm

package benchmark

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
)

// requireHTTP polls a URL until it returns a 2xx status code or the timeout expires.
func requireHTTP(t testing.TB, url string, timeout time.Duration) {
	t.Helper()
	client := &http.Client{Timeout: 200 * time.Millisecond}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return
			}
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("daemon not ready at %s: %v", url, lastErr)
}

// sumCounter sums all counter values in a prometheus MetricFamily.
func sumCounter(f *dto.MetricFamily) float64 {
	if f == nil || f.GetType() != dto.MetricType_COUNTER {
		return 0
	}
	var sum float64
	for _, m := range f.GetMetric() {
		if m.GetCounter() != nil && m.GetCounter().Value != nil {
			sum += m.GetCounter().GetValue()
		}
	}
	return sum
}
