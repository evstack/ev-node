package submitting

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryStateNext_Table(t *testing.T) {
	pol := retryPolicy{
		MaxAttempts:      10,
		MinBackoff:       100 * time.Millisecond,
		MaxBackoff:       1 * time.Second,
		MinGasPrice:      0.0,
		MaxGasPrice:      10.0,
		MaxBlobBytes:     2 * 1024 * 1024,
		MaxGasMultiplier: 3.0,
	}

	tests := map[string]struct {
		startGas      float64
		startBackoff  time.Duration
		reason        retryReason
		gasMult       float64
		sentinelNoGas bool
		wantGas       float64
		wantBackoff   time.Duration
	}{
		"success reduces gas and resets backoff": {
			startGas:     9.0,
			startBackoff: 500 * time.Millisecond,
			reason:       reasonSuccess,
			gasMult:      3.0,
			wantGas:      3.0, // 9 / 3
			wantBackoff:  pol.MinBackoff,
		},
		"success clamps very small gasMult to 1/Max, possibly increasing gas": {
			startGas:     3.0,
			startBackoff: 250 * time.Millisecond,
			reason:       reasonSuccess,
			gasMult:      0.01, // clamped to 1/MaxGasMultiplier = 1/3
			wantGas:      9.0,  // 3 / (1/3)
			wantBackoff:  pol.MinBackoff,
		},
		"mempool increases gas and sets max backoff": {
			startGas:     2.0,
			startBackoff: 0,
			reason:       reasonMempool,
			gasMult:      2.0,
			wantGas:      4.0, // 2 * 2
			wantBackoff:  pol.MaxBackoff,
		},
		"mempool clamps gas to max": {
			startGas:     9.5,
			startBackoff: 0,
			reason:       reasonMempool,
			gasMult:      3.0,
			wantGas:      10.0, // 9.5 * 3 = 28.5 -> clamp 10
			wantBackoff:  pol.MaxBackoff,
		},
		"failure sets initial backoff": {
			startGas:     1.0,
			startBackoff: 0,
			reason:       reasonFailure,
			gasMult:      2.0,
			wantGas:      1.0, // unchanged
			wantBackoff:  pol.MinBackoff,
		},
		"failure doubles backoff capped at max": {
			startGas:     1.0,
			startBackoff: 700 * time.Millisecond,
			reason:       reasonFailure,
			gasMult:      2.0,
			wantGas:      1.0,             // unchanged
			wantBackoff:  1 * time.Second, // 700ms*2=1400ms -> clamp 1s
		},
		"tooBig doubles backoff like failure": {
			startGas:     1.0,
			startBackoff: 100 * time.Millisecond,
			reason:       reasonTooBig,
			gasMult:      2.0,
			wantGas:      1.0,
			wantBackoff:  200 * time.Millisecond,
		},
		"sentinel no gas keeps gas unchanged on success": {
			startGas:      5.0,
			startBackoff:  0,
			reason:        reasonSuccess,
			gasMult:       2.0,
			sentinelNoGas: true,
			wantGas:       5.0,
			wantBackoff:   pol.MinBackoff,
		},
		"undefined reason keeps gas unchanged and uses min backoff": {
			startGas:     3.0,
			startBackoff: 500 * time.Millisecond,
			reason:       reasonUndefined,
			gasMult:      2.0,
			wantGas:      3.0,
			wantBackoff:  0,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			rs := retryState{Attempt: 0, Backoff: tc.startBackoff, GasPrice: tc.startGas}
			rs.Next(tc.reason, pol, tc.gasMult, tc.sentinelNoGas)

			assert.Equal(t, tc.wantGas, rs.GasPrice, "gas price")
			assert.Equal(t, tc.wantBackoff, rs.Backoff, "backoff")
			assert.Equal(t, 1, rs.Attempt, "attempt")
		})
	}
}
