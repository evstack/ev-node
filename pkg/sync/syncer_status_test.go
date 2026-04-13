package sync

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSyncerStatusStartOnce(t *testing.T) {
	t.Parallel()

	specs := map[string]struct {
		run func(*testing.T, *SyncerStatus)
	}{
		"concurrent_start_only_runs_once": {
			run: func(t *testing.T, status *SyncerStatus) {
				t.Helper()

				var calls atomic.Int32
				started := make(chan struct{})
				release := make(chan struct{})
				var wg sync.WaitGroup

				for range 8 {
					wg.Add(1)
					go func() {
						defer wg.Done()
						_, err := status.startOnce(func() error {
							if calls.Add(1) == 1 {
								close(started)
							}
							<-release
							return nil
						})
						require.NoError(t, err)
					}()
				}

				<-started
				close(release)
				wg.Wait()

				require.Equal(t, int32(1), calls.Load())
				require.True(t, status.isStarted())
			},
		},
		"failed_start_can_retry": {
			run: func(t *testing.T, status *SyncerStatus) {
				t.Helper()

				var calls atomic.Int32
				errBoom := errors.New("boom")

				startedNow, err := status.startOnce(func() error {
					calls.Add(1)
					return errBoom
				})
				require.ErrorIs(t, err, errBoom)
				require.False(t, startedNow)
				require.False(t, status.isStarted())

				startedNow, err = status.startOnce(func() error {
					calls.Add(1)
					return nil
				})
				require.NoError(t, err)
				require.True(t, startedNow)
				require.True(t, status.isStarted())
				require.Equal(t, int32(2), calls.Load())
			},
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			spec.run(t, &SyncerStatus{})
		})
	}
}

func TestSyncerStatusStopIfStarted(t *testing.T) {
	t.Parallel()

	specs := map[string]struct {
		started bool
		wantErr bool
	}{
		"no_op_when_not_started": {
			started: false,
			wantErr: false,
		},
		"stop_clears_started": {
			started: true,
			wantErr: false,
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			status := &SyncerStatus{started: spec.started}
			var stopCalls atomic.Int32

			err := status.stopIfStarted(func() error {
				stopCalls.Add(1)
				return nil
			})

			if spec.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if spec.started {
				require.Equal(t, int32(1), stopCalls.Load())
			} else {
				require.Zero(t, stopCalls.Load())
			}
			require.False(t, status.isStarted())
		})
	}
}
