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
	t.Run("normal_sequential_fetch_success", func(t *testing.T) {
		daRetriever := NewMockDARetriever(t)
		ctx := t.Context()

		var pipedEvents []common.DAHeightEvent
		pipeEvent := func(_ context.Context, ev common.DAHeightEvent) error {
			pipedEvents = append(pipedEvents, ev)
			return nil
		}

		follower := &daFollower{
			retriever:       daRetriever,
			eventSink:       common.EventSinkFunc(pipeEvent),
			logger:          zerolog.Nop(),
			priorityHeights: make([]uint64, 0),
		}

		events := []common.DAHeightEvent{{DaHeight: 100}}
		daRetriever.On("RetrieveFromDA", mock.Anything, uint64(100)).Return(events, nil)

		err := follower.HandleCatchup(ctx, 100)
		require.NoError(t, err)
		assert.Len(t, pipedEvents, 1)
	})

	t.Run("normal_sequential_fetch_blob_not_found_ignored", func(t *testing.T) {
		daRetriever := NewMockDARetriever(t)
		ctx := t.Context()

		follower := &daFollower{
			retriever:       daRetriever,
			eventSink:       common.EventSinkFunc(func(_ context.Context, _ common.DAHeightEvent) error { return nil }),
			logger:          zerolog.Nop(),
			priorityHeights: make([]uint64, 0),
		}

		daRetriever.On("RetrieveFromDA", mock.Anything, uint64(100)).Return(nil, datypes.ErrBlobNotFound)

		err := follower.HandleCatchup(ctx, 100)
		require.NoError(t, err)
	})

	t.Run("normal_sequential_fetch_error_propagated", func(t *testing.T) {
		daRetriever := NewMockDARetriever(t)
		ctx := t.Context()

		follower := &daFollower{
			retriever:       daRetriever,
			eventSink:       common.EventSinkFunc(func(_ context.Context, _ common.DAHeightEvent) error { return nil }),
			logger:          zerolog.Nop(),
			priorityHeights: make([]uint64, 0),
		}

		daRetriever.On("RetrieveFromDA", mock.Anything, uint64(100)).Return(nil, datypes.ErrHeightFromFuture)

		err := follower.HandleCatchup(ctx, 100)
		require.ErrorIs(t, err, datypes.ErrHeightFromFuture)
	})

	t.Run("priority_queue_handled_first", func(t *testing.T) {
		daRetriever := NewMockDARetriever(t)
		ctx := t.Context()

		var pipedEvents []common.DAHeightEvent
		pipeEvent := func(_ context.Context, ev common.DAHeightEvent) error {
			pipedEvents = append(pipedEvents, ev)
			return nil
		}

		follower := &daFollower{
			retriever:       daRetriever,
			eventSink:       common.EventSinkFunc(pipeEvent),
			logger:          zerolog.Nop(),
			priorityHeights: []uint64{105}, // Priority height to fetch
		}

		daRetriever.On("RetrieveFromDA", mock.Anything, uint64(105)).Return([]common.DAHeightEvent{{DaHeight: 105}}, nil).Once()
		daRetriever.On("RetrieveFromDA", mock.Anything, uint64(100)).Return([]common.DAHeightEvent{{DaHeight: 100}}, nil).Once()

		err := follower.HandleCatchup(ctx, 100)
		require.NoError(t, err)

		// It should pipe 105 then 100
		assert.Len(t, pipedEvents, 2)
		assert.Equal(t, uint64(105), pipedEvents[0].DaHeight)
		assert.Equal(t, uint64(100), pipedEvents[1].DaHeight)
		assert.Empty(t, follower.priorityHeights)
	})
}
