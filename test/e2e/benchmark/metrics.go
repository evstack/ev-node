//go:build evm

package benchmark

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	dto "github.com/prometheus/client_model/go"
)

// requireHostUp polls a URL until it returns a 2xx status code or the timeout expires.
func requireHostUp(t testing.TB, url string, timeout time.Duration) {
	t.Helper()
	client := &http.Client{Timeout: 200 * time.Millisecond}
	require.Eventually(t, func() bool {
		resp, err := client.Get(url)
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode >= 200 && resp.StatusCode < 300
	}, timeout, 100*time.Millisecond, "daemon not ready at %s", url)
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
