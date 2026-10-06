package syncing

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/common"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	testmocks "github.com/evstack/ev-node/test/mocks"
)

func TestDAFollower_HandleEvent(t *testing.T) {
	tests := []struct {
		name          string
		isInline      bool
		blobs         [][]byte
		mockEvents    []common.DAHeightEvent
		mockPipeErr   error
		expectedError string
	}{
		{
			name:     "ignore_not_inline",
			isInline: false,
		},
		{
			name:          "error_no_blobs",
			isInline:      true,
			blobs:         [][]byte{},
			expectedError: "skip inline: no blobs",
		},
		{
			name:          "error_no_complete_events",
			isInline:      true,
			blobs:         [][]byte{[]byte("blob")},
			mockEvents:    []common.DAHeightEvent{},
			expectedError: "skip inline: no complete events",
		},
		{
			name:          "error_pipe_fails",
			isInline:      true,
			blobs:         [][]byte{[]byte("blob")},
			mockEvents:    []common.DAHeightEvent{{DaHeight: 100}},
			mockPipeErr:   errors.New("pipe error"),
			expectedError: "pipe error",
		},
		{
			name:        "success",
			isInline:    true,
			blobs:       [][]byte{[]byte("blob")},
			mockEvents:  []common.DAHeightEvent{{DaHeight: 100}},
			mockPipeErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			daRetriever := NewMockDARetriever(t)
			ctx := t.Context()

			var pipedEvents []common.DAHeightEvent
			pipeEvent := func(_ context.Context, ev common.DAHeightEvent) error {
				pipedEvents = append(pipedEvents, ev)
				return tt.mockPipeErr
			}

			follower := &daFollower{
				retriever: daRetriever,
				eventSink: common.EventSinkFunc(pipeEvent),
				logger:    zerolog.Nop(),
			}

			ev := datypes.SubscriptionEvent{Height: 100, Blobs: tt.blobs}

			if tt.isInline && len(tt.blobs) > 0 {
				daRetriever.On("ProcessBlobs", mock.Anything, tt.blobs, uint64(100)).Return(tt.mockEvents)
			}

			err := follower.HandleEvent(ctx, ev, tt.isInline)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				if tt.isInline && len(tt.blobs) > 0 {
					assert.Len(t, pipedEvents, len(tt.mockEvents))
				}
			}
		})
	}
}

func TestDAFollower_HandleCatchup(t *testing.T) {
	type spec struct {
		daHeight               uint64
		initialPriorityHeights []uint64
		pipeErr                error
		setupMock              func(m *MockDARetriever)
		wantErrIs              error
		wantPipedHeights       []uint64
		wantRemainingPriority  []uint64
	}

	newFollower := func(t *testing.T, s spec, m *MockDARetriever) (*daFollower, func() []common.DAHeightEvent) {
		t.Helper()

		var pipedEvents []common.DAHeightEvent
		pipeEvent := func(_ context.Context, ev common.DAHeightEvent) error {
			pipedEvents = append(pipedEvents, ev)
			return s.pipeErr
		}

		follower := &daFollower{
			retriever:       m,
			eventSink:       common.EventSinkFunc(pipeEvent),
			logger:          zerolog.Nop(),
			priorityHeights: append([]uint64(nil), s.initialPriorityHeights...),
		}
		return follower, func() []common.DAHeightEvent { return pipedEvents }
	}

	specs := map[string]spec{
		"seq_ok": {
			daHeight:         100,
			wantPipedHeights: []uint64{100},
			setupMock: func(m *MockDARetriever) {
				m.On("RetrieveFromDA", mock.Anything, uint64(100)).
					Return([]common.DAHeightEvent{{DaHeight: 100}}, nil).Once()
			},
		},
		"seq_blob_missing": {
			daHeight: 100,
			setupMock: func(m *MockDARetriever) {
				m.On("RetrieveFromDA", mock.Anything, uint64(100)).
					Return(nil, datypes.ErrBlobNotFound).Once()
			},
		},
		"seq_err": {
			daHeight:  100,
			wantErrIs: datypes.ErrHeightFromFuture,
			setupMock: func(m *MockDARetriever) {
				m.On("RetrieveFromDA", mock.Anything, uint64(100)).
					Return(nil, datypes.ErrHeightFromFuture).Once()
			},
		},
		"prio_first": {
			daHeight:               100,
			initialPriorityHeights: []uint64{105},
			wantPipedHeights:       []uint64{105, 100},
			setupMock: func(m *MockDARetriever) {
				m.On("RetrieveFromDA", mock.Anything, uint64(105)).
					Return([]common.DAHeightEvent{{DaHeight: 105}}, nil).Once()
				m.On("RetrieveFromDA", mock.Anything, uint64(100)).
					Return([]common.DAHeightEvent{{DaHeight: 100}}, nil).Once()
			},
		},
		"skip_stale_prio_already_included": {
			daHeight:               100,
			initialPriorityHeights: []uint64{99},
			wantPipedHeights:       []uint64{100},
			setupMock: func(m *MockDARetriever) {
				// stale priority hint (< daHeight) is discarded; only sequential height is fetched
				m.On("RetrieveFromDA", mock.Anything, uint64(100)).
					Return([]common.DAHeightEvent{{DaHeight: 100}}, nil).Once()
			},
		},
	}

	for name, s := range specs {
		t.Run(name, func(t *testing.T) {
			daRetriever := NewMockDARetriever(t)
			if s.setupMock != nil {
				s.setupMock(daRetriever)
			}

			follower, getPipedEvents := newFollower(t, s, daRetriever)
			err := follower.HandleCatchup(t.Context(), s.daHeight)

			if s.wantErrIs != nil {
				require.ErrorIs(t, err, s.wantErrIs)
			} else {
				require.NoError(t, err)
			}

			pipedEvents := getPipedEvents()
			gotHeights := make([]uint64, 0, len(pipedEvents))
			for _, ev := range pipedEvents {
				gotHeights = append(gotHeights, ev.DaHeight)
			}
			wantHeights := s.wantPipedHeights
			if wantHeights == nil {
				wantHeights = []uint64{}
			}
			assert.Equal(t, wantHeights, gotHeights)

			if s.wantRemainingPriority != nil {
				assert.Equal(t, s.wantRemainingPriority, follower.priorityHeights)
			} else {
				assert.Empty(t, follower.priorityHeights)
			}
		})
	}
}

func TestDAFollower_QueuePriorityHeight(t *testing.T) {
	specs := map[string]struct {
		initial []uint64
		queue   []uint64
		want    []uint64
	}{
		"sorts_and_deduplicates": {
			initial: []uint64{5, 10},
			queue:   []uint64{7, 10, 3},
			want:    []uint64{3, 5, 7, 10},
		},
		"bounded_drops_largest_when_smaller_arrives": {
			initial: makeRange(1, maxPriorityHeights),
			queue:   []uint64{maxPriorityHeights + 1, 0},
			want:    append([]uint64{0}, makeRange(1, maxPriorityHeights-1)...),
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			follower := &daFollower{
				logger:          zerolog.Nop(),
				priorityHeights: append([]uint64(nil), spec.initial...),
			}

			for _, daHeight := range spec.queue {
				follower.QueuePriorityHeight(daHeight)
			}

			assert.Equal(t, spec.want, follower.priorityHeights)
		})
	}
}

func makeRange(start, end uint64) []uint64 {
	if end < start {
		return nil
	}
	out := make([]uint64, 0, end-start+1)
	for v := start; v <= end; v++ {
		out = append(out, v)
	}
	return out
}

func TestDAFollowerSkipsAfterTenFailedFetchesAndContinues(t *testing.T) {
	retriever := NewMockDARetriever(t)
	fetchErr := errors.New("historical DA height no longer available")
	for _, h := range []uint64{100, 101} {
		retriever.On("RetrieveFromDA", mock.Anything, h).Return(nil, fetchErr).Times(10)
	}
	retriever.On("RetrieveFromDA", mock.Anything, uint64(102)).Return([]common.DAHeightEvent(nil), nil).Once()
	client := testmocks.NewMockClient(t)
	client.On("SupportsSubscribe").Return(false)
	client.On("GetLatestDAHeight", mock.Anything).Return(uint64(102), nil)
	var lastSkipped atomic.Uint64
	f := NewDAFollower(DAFollowerConfig{
		Client: client, Retriever: retriever, Logger: zerolog.Nop(),
		Namespace: []byte("ns"), StartDAHeight: 100, DABlockTime: time.Millisecond,
		OnSkip: func(_ context.Context, height uint64) error {
			lastSkipped.Store(height)
			return nil
		},
	}).(*daFollower)
	require.NoError(t, f.Start(t.Context()))
	t.Cleanup(f.Stop)
	require.Eventually(t, func() bool { return f.subscriber.LocalDAHeight() == 103 }, time.Second, time.Millisecond)
	require.Equal(t, uint64(101), lastSkipped.Load())
	// Stop before checking mocks so no background calls race with assertions.
	f.Stop()
	retriever.AssertExpectations(t)
}

func TestDAFollowerDoesNotSkipFutureCancellationOrDeliveryFailures(t *testing.T) {
	for _, kind := range []string{"future", "cancellation", "delivery"} {
		t.Run(kind, func(t *testing.T) {
			ctx := t.Context()
			retriever := NewMockDARetriever(t)
			var expected error
			var events []common.DAHeightEvent
			var retrievalErr error
			switch kind {
			case "future":
				expected = datypes.ErrHeightFromFuture
				retrievalErr = expected
			case "cancellation":
				canceled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = canceled
				expected = context.Canceled
				retrievalErr = expected
			case "delivery":
				expected = errors.New("event delivery failed")
				events = []common.DAHeightEvent{{DaHeight: 100}}
			}
			retriever.On("RetrieveFromDA", mock.Anything, uint64(100)).Return(events, retrievalErr).Times(12)
			f := &daFollower{retriever: retriever, logger: zerolog.Nop(),
				eventSink: common.EventSinkFunc(func(context.Context, common.DAHeightEvent) error { return expected }),
				onSkip: func(context.Context, uint64) error {
					t.Fatal("height must not be skipped")
					return nil
				},
			}
			for range 12 {
				require.ErrorIs(t, f.HandleCatchup(ctx, 100), expected)
			}
			require.Zero(t, f.fetchFailures)
		})
	}
}

func TestDAFollowerSkipWaitsForPersistence(t *testing.T) {
	retriever := NewMockDARetriever(t)
	fetchErr := errors.New("DA unavailable")
	retriever.On("RetrieveFromDA", mock.Anything, uint64(100)).Return(nil, fetchErr).Times(11)
	persistErr := errors.New("metadata write failed")
	f := &daFollower{retriever: retriever, logger: zerolog.Nop(),
		onSkip: func(context.Context, uint64) error { return persistErr },
	}
	for range 9 {
		require.ErrorIs(t, f.HandleCatchup(t.Context(), 100), fetchErr)
	}
	require.ErrorIs(t, f.HandleCatchup(t.Context(), 100), persistErr)
	f.onSkip = func(context.Context, uint64) error { return nil }
	require.NoError(t, f.HandleCatchup(t.Context(), 100))
}

func TestDAFollowerResetsFetchFailuresAfterRecovery(t *testing.T) {
	retriever := NewMockDARetriever(t)
	fetchErr := errors.New("DA unavailable")
	retriever.On("RetrieveFromDA", mock.Anything, uint64(100)).Return(nil, fetchErr).Times(9)
	retriever.On("RetrieveFromDA", mock.Anything, uint64(100)).Return([]common.DAHeightEvent(nil), nil).Once()
	retriever.On("RetrieveFromDA", mock.Anything, uint64(101)).Return(nil, fetchErr).Times(10)
	var skipped []uint64
	f := &daFollower{retriever: retriever, logger: zerolog.Nop(),
		onSkip: func(_ context.Context, h uint64) error { skipped = append(skipped, h); return nil },
	}
	for range 9 {
		require.ErrorIs(t, f.HandleCatchup(t.Context(), 100), fetchErr)
	}
	require.NoError(t, f.HandleCatchup(t.Context(), 100))
	for range 9 {
		require.ErrorIs(t, f.HandleCatchup(t.Context(), 101), fetchErr)
	}
	require.Empty(t, skipped)
	require.NoError(t, f.HandleCatchup(t.Context(), 101))
	require.Equal(t, []uint64{101}, skipped)
}

func TestDAFollowerPriorityFailuresDoNotPreventSequentialRetries(t *testing.T) {
	retriever := NewMockDARetriever(t)
	fetchErr := errors.New("DA unavailable")
	retriever.On("RetrieveFromDA", mock.Anything, uint64(100)).Return(nil, fetchErr).Times(10)
	retriever.On("RetrieveFromDA", mock.Anything, uint64(105)).Return(nil, fetchErr).Times(10)
	var skipped []uint64
	f := &daFollower{retriever: retriever, logger: zerolog.Nop(),
		onSkip: func(_ context.Context, h uint64) error { skipped = append(skipped, h); return nil },
	}
	for attempt := range 10 {
		// Repeated hints for the current and future heights must not starve
		// sequential retrieval or cause a second fetch of the current height.
		f.QueuePriorityHeight(100)
		f.QueuePriorityHeight(105)
		err := f.HandleCatchup(t.Context(), 100)
		if attempt < 9 {
			require.ErrorIs(t, err, fetchErr)
		} else {
			require.NoError(t, err)
		}
	}
	require.Equal(t, []uint64{100}, skipped)
}

func TestDAFollowerDoesNotSkipBeyondObservedHeadDuringOutage(t *testing.T) {
	retriever := NewMockDARetriever(t)
	fetchErr := errors.New("DA transport unavailable")
	retriever.On("RetrieveFromDA", mock.Anything, uint64(101)).Return(nil, fetchErr).Times(12)
	f := NewDAFollower(DAFollowerConfig{
		Retriever: retriever, Logger: zerolog.Nop(), Namespace: []byte("ns"), StartDAHeight: 100,
		OnSkip: func(context.Context, uint64) error {
			t.Fatal("must not skip beyond the observed DA head")
			return nil
		},
	}).(*daFollower)
	for range 12 {
		require.ErrorIs(t, f.HandleCatchup(t.Context(), 101), fetchErr)
	}
	require.Equal(t, maxDAFetchAttempts, f.fetchFailures)
}
