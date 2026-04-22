package syncing

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/common"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/types"
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
			events, err := follower.HandleCatchup(t.Context(), s.daHeight)

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

			_ = events
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

func makeHeader(height uint64) *types.SignedHeader {
	return &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: height}}}
}
