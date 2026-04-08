package sync

import "sync"

// SyncerStatus is used by header and block exchange service for keeping track
// of the status of the syncer in them.
type SyncerStatus struct {
	mu      sync.Mutex
	started bool
}

func (syncerStatus *SyncerStatus) isStarted() bool {
	syncerStatus.mu.Lock()
	defer syncerStatus.mu.Unlock()

	return syncerStatus.started
}

func (syncerStatus *SyncerStatus) startOnce(startFn func() error) (bool, error) {
	syncerStatus.mu.Lock()
	defer syncerStatus.mu.Unlock()

	if syncerStatus.started {
		return false, nil
	}

	if err := startFn(); err != nil {
		return false, err
	}

	syncerStatus.started = true
	return true, nil
}

func (syncerStatus *SyncerStatus) stopIfStarted(stopFn func() error) error {
	syncerStatus.mu.Lock()
	defer syncerStatus.mu.Unlock()

	if !syncerStatus.started {
		return nil
	}

	if err := stopFn(); err != nil {
		return err
	}

	syncerStatus.started = false
	return nil
}
