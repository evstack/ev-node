package submitting

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryStateNext_Table(t *testing.T) {
	pol := retryPolicy{
		MaxAttempts:  10,
		MinBackoff:   100 * time.Millisecond,
		MaxBackoff:   1 * time.Second,
		MaxBlobBytes: 2 * 1024 * 1024,
	}

	tests := map[string]struct {
		startBackoff time.Duration
		reason       retryReason
		wantBackoff  time.Duration
	}{
		"success resets backoff": {
			startBackoff: 500 * time.Millisecond,
			reason:       reasonSuccess,
			wantBackoff:  pol.MinBackoff,
		},
		"mempool sets max backoff": {
			startBackoff: 0,
			reason:       reasonMempool,
			wantBackoff:  pol.MaxBackoff,
		},
		"failure sets initial backoff": {
			startBackoff: 0,
			reason:       reasonFailure,
			wantBackoff:  pol.MinBackoff,
		},
		"failure doubles backoff capped at max": {
			startBackoff: 700 * time.Millisecond,
			reason:       reasonFailure,
			wantBackoff:  1 * time.Second, // 700ms*2=1400ms -> clamp 1s
		},
		"tooBig doubles backoff like failure": {
			startBackoff: 100 * time.Millisecond,
			reason:       reasonTooBig,
			wantBackoff:  200 * time.Millisecond,
		},
		"undefined reason uses zero backoff": {
			startBackoff: 500 * time.Millisecond,
			reason:       reasonUndefined,
			wantBackoff:  0,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			rs := retryState{Attempt: 0, Backoff: tc.startBackoff}
			rs.Next(tc.reason, pol)

			assert.Equal(t, tc.wantBackoff, rs.Backoff, "backoff")
			assert.Equal(t, 1, rs.Attempt, "attempt")
		})
	}
}
