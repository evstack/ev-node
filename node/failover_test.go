package node

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestCatchupStatusReady(t *testing.T) {
	tests := []struct {
		name        string
		p2pRecovery bool
		status      catchupStatus
		want        bool
	}{
		{
			name: "DA only ready",
			status: catchupStatus{
				daCaughtUp: true,
			},
			want: true,
		},
		{
			name:        "configured peers not initialized",
			p2pRecovery: true,
			status: catchupStatus{
				daCaughtUp: true,
			},
		},
		{
			name:        "only header P2P initialized",
			p2pRecovery: true,
			status: catchupStatus{
				daCaughtUp:     true,
				headerP2PReady: true,
			},
		},
		{
			name:        "only data P2P initialized",
			p2pRecovery: true,
			status: catchupStatus{
				daCaughtUp:   true,
				dataP2PReady: true,
			},
		},
		{
			name:        "observed header height ahead of store",
			p2pRecovery: true,
			status: catchupStatus{
				storeHeight:    9,
				headerHeight:   10,
				dataHeight:     9,
				headerP2PReady: true,
				dataP2PReady:   true,
				daCaughtUp:     true,
			},
		},
		{
			name:        "observed data height ahead of store",
			p2pRecovery: true,
			status: catchupStatus{
				storeHeight:    9,
				headerHeight:   9,
				dataHeight:     10,
				headerP2PReady: true,
				dataP2PReady:   true,
				daCaughtUp:     true,
			},
		},
		{
			name:        "pending catchup events",
			p2pRecovery: true,
			status: catchupStatus{
				storeHeight:    10,
				headerHeight:   10,
				dataHeight:     10,
				headerP2PReady: true,
				dataP2PReady:   true,
				daCaughtUp:     true,
				pendingEvents:  1,
			},
		},
		{
			name:        "combined DA and P2P ready",
			p2pRecovery: true,
			status: catchupStatus{
				storeHeight:    11,
				headerHeight:   10,
				dataHeight:     11,
				headerP2PReady: true,
				dataP2PReady:   true,
				daCaughtUp:     true,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.status.ready(tt.p2pRecovery))
		})
	}
}

func TestWaitForCatchupP2PTimeoutFailsClosed(t *testing.T) {
	f := &failoverState{
		logger:         zerolog.Nop(),
		p2pRecovery:    true,
		catchupTimeout: 20 * time.Millisecond,
		daBlockTime:    time.Millisecond,
		catchupStatusFn: func(context.Context) (catchupStatus, error) {
			return catchupStatus{
				storeHeight:  7,
				headerHeight: 9,
				dataHeight:   8,
			}, nil
		},
	}

	caughtUp, err := f.waitForCatchup(t.Context())
	require.False(t, caughtUp)
	require.ErrorContains(t, err, "P2P recovery timed out")
	require.ErrorContains(t, err, "store height 7")
	require.ErrorContains(t, err, "header height 9")
	require.ErrorContains(t, err, "data height 8")
	require.ErrorContains(t, err, "header P2P ready false")
	require.ErrorContains(t, err, "data P2P ready false")
	require.ErrorContains(t, err, "DA caught up false")
	require.ErrorContains(t, err, "pending events 0")
}

func TestCatchupStatusContinuityTimeoutErrorIncludesReadiness(t *testing.T) {
	err := catchupStatus{
		storeHeight:    1,
		headerHeight:   2,
		dataHeight:     3,
		headerP2PReady: true,
		daCaughtUp:     true,
		pendingEvents:  4,
	}.continuityTimeoutError(time.Second)
	require.ErrorContains(t, err, "store height 1")
	require.ErrorContains(t, err, "header height 2")
	require.ErrorContains(t, err, "data height 3")
	require.ErrorContains(t, err, "header P2P ready true")
	require.ErrorContains(t, err, "data P2P ready false")
	require.ErrorContains(t, err, "DA caught up true")
	require.ErrorContains(t, err, "pending events 4")
}

func TestWaitForCatchupContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancelCause(t.Context())
	wantErr := errors.New("operator canceled recovery")
	cancel(wantErr)

	f := &failoverState{
		logger:         zerolog.Nop(),
		p2pRecovery:    true,
		catchupTimeout: time.Hour,
		daBlockTime:    time.Hour,
	}

	caughtUp, err := f.waitForCatchup(ctx)
	require.False(t, caughtUp)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, wantErr, context.Cause(ctx))
}

func TestWaitForCatchupP2PBudgetDoesNotLimitDARecovery(t *testing.T) {
	statusCalls := 0
	f := &failoverState{
		logger:         zerolog.Nop(),
		p2pRecovery:    true,
		catchupTimeout: 10 * time.Millisecond,
		daBlockTime:    time.Millisecond,
		catchupStatusFn: func(context.Context) (catchupStatus, error) {
			statusCalls++
			return catchupStatus{
				storeHeight:    10,
				headerHeight:   10,
				dataHeight:     10,
				headerP2PReady: true,
				dataP2PReady:   true,
				daCaughtUp:     statusCalls > 15,
			}, nil
		},
	}

	caughtUp, err := f.waitForCatchup(t.Context())
	require.NoError(t, err)
	require.True(t, caughtUp)
}
