//go:build evm

package benchmark

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

// gasThroughput holds the result of scanning a block range for gas usage.
type gasThroughput struct {
	totalGas      uint64
	gigagasPerSec float64
}

// measureGasThroughput scans blocks in [startBlock+1, endBlock] and computes
// gas throughput over the steady-state window (first to last non-empty block).
func measureGasThroughput(t testing.TB, ctx context.Context, client *ethclient.Client, startBlock, endBlock uint64) gasThroughput {
	t.Helper()

	var firstGasBlock, lastGasBlock uint64
	var totalGas uint64
	var emptyBlocks, nonEmptyBlocks int
	for i := startBlock + 1; i <= endBlock; i++ {
		header, err := client.HeaderByNumber(ctx, new(big.Int).SetUint64(i))
		require.NoError(t, err, "failed to get header for block %d", i)
		if header.GasUsed == 0 {
			emptyBlocks++
			continue
		}
		nonEmptyBlocks++
		if firstGasBlock == 0 {
			firstGasBlock = i
		}
		lastGasBlock = i
		totalGas += header.GasUsed
	}
	t.Logf("block summary: %d empty, %d non-empty out of %d total", emptyBlocks, nonEmptyBlocks, endBlock-startBlock)
	require.NotZero(t, firstGasBlock, "no blocks with gas found")

	firstGasHeader, err := client.HeaderByNumber(ctx, new(big.Int).SetUint64(firstGasBlock))
	require.NoError(t, err, "failed to get first gas block header")
	lastGasHeader, err := client.HeaderByNumber(ctx, new(big.Int).SetUint64(lastGasBlock))
	require.NoError(t, err, "failed to get last gas block header")

	elapsed := time.Duration(lastGasHeader.Time-firstGasHeader.Time) * time.Second
	if elapsed == 0 {
		elapsed = 1 * time.Second
	}
	t.Logf("steady-state: blocks %d-%d, elapsed %v", firstGasBlock, lastGasBlock, elapsed)

	gigagas := float64(totalGas) / 1e9
	gigagasPerSec := float64(totalGas) / elapsed.Seconds() / 1e9
	t.Logf("total gas used: %d (%.2f gigagas)", totalGas, gigagas)
	t.Logf("gas throughput: %.2f gigagas/sec", gigagasPerSec)

	return gasThroughput{totalGas: totalGas, gigagasPerSec: gigagasPerSec}
}

// waitForMetricTarget polls a metric getter function every 2s until the returned
// value >= target, or fails the test on timeout.
func waitForMetricTarget(t testing.TB, name string, poll func() (float64, error), target float64, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		v, err := poll()
		if err == nil && v >= target {
			t.Logf("metric %s reached %.0f (target %.0f)", name, v, target)
			return
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("metric %s did not reach target %.0f within %v", name, target, timeout)
}

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
