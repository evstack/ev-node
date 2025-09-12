package submitting

import (
	"testing"
	"time"
)

func TestRetryStateNext_Table(t *testing.T) {
	pol := RetryPolicy{
		MaxAttempts:      10,
		MinBackoff:       100 * time.Millisecond,
		MaxBackoff:       1 * time.Second,
		MinGasPrice:      0.0,
		MaxGasPrice:      10.0,
		MaxBlobBytes:     2 * 1024 * 1024,
		MaxGasMultiplier: 3.0,
	}

	tests := []struct {
		name          string
		startGas      float64
		startBackoff  time.Duration
		reason        retryReason
		gm            float64
		sentinelNoGas bool
		wantGas       float64
		wantBackoff   time.Duration
	}{
		{
			name:         "success reduces gas and resets backoff",
			startGas:     9.0,
			startBackoff: 500 * time.Millisecond,
			reason:       reasonSuccess,
			gm:           3.0,
			wantGas:      3.0, // 9 / 3
			wantBackoff:  pol.MinBackoff,
		},
		{
			name:         "success clamps very small gm to 1/Max, possibly increasing gas",
			startGas:     3.0,
			startBackoff: 250 * time.Millisecond,
			reason:       reasonSuccess,
			gm:           0.01, // clamped to 1/MaxGasMultiplier = 1/3
			wantGas:      9.0,  // 3 / (1/3)
			wantBackoff:  pol.MinBackoff,
		},
		{
			name:         "mempool increases gas and sets max backoff",
			startGas:     2.0,
			startBackoff: 0,
			reason:       reasonMempool,
			gm:           2.0,
			wantGas:      4.0, // 2 * 2
			wantBackoff:  pol.MaxBackoff,
		},
		{
			name:         "mempool clamps gas to max",
			startGas:     9.5,
			startBackoff: 0,
			reason:       reasonMempool,
			gm:           3.0,
			wantGas:      10.0, // 9.5 * 3 = 28.5 -> clamp 10
			wantBackoff:  pol.MaxBackoff,
		},
		{
			name:         "failure sets initial backoff",
			startGas:     1.0,
			startBackoff: 0,
			reason:       reasonFailure,
			gm:           2.0,
			wantGas:      1.0, // unchanged
			wantBackoff:  pol.MinBackoff,
		},
		{
			name:         "failure doubles backoff capped at max",
			startGas:     1.0,
			startBackoff: 700 * time.Millisecond,
			reason:       reasonFailure,
			gm:           2.0,
			wantGas:      1.0,             // unchanged
			wantBackoff:  1 * time.Second, // 700ms*2=1400ms -> clamp 1s
		},
		{
			name:         "tooBig doubles backoff like failure",
			startGas:     1.0,
			startBackoff: 100 * time.Millisecond,
			reason:       reasonTooBig,
			gm:           2.0,
			wantGas:      1.0,
			wantBackoff:  200 * time.Millisecond,
		},
		{
			name:          "sentinel no gas keeps gas unchanged on success",
			startGas:      5.0,
			startBackoff:  0,
			reason:        reasonSuccess,
			gm:            2.0,
			sentinelNoGas: true,
			wantGas:       5.0,
			wantBackoff:   pol.MinBackoff,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rs := RetryState{Attempt: 0, Backoff: tc.startBackoff, GasPrice: tc.startGas}
			rs.Next(tc.reason, pol, tc.gm, tc.sentinelNoGas)

			if rs.GasPrice != tc.wantGas {
				t.Fatalf("gas price: got %v, want %v", rs.GasPrice, tc.wantGas)
			}
			if rs.Backoff != tc.wantBackoff {
				t.Fatalf("backoff: got %v, want %v", rs.Backoff, tc.wantBackoff)
			}
			if rs.Attempt != 1 {
				t.Fatalf("attempt: got %d, want %d", rs.Attempt, 1)
			}
		})
	}
}
