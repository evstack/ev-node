package executing

import coreexecutor "github.com/evstack/ev-node/core/execution"

// GetCoreExecutor returns the underlying core executor for testing purposes
func (e *Executor) GetCoreExecutor() coreexecutor.Executor {
	return e.exec
}

// HasPendingTxNotification checks if there's a pending transaction notification
// It is used for testing purposes
func (e *Executor) HasPendingTxNotification() bool {
	select {
	case <-e.txNotifyCh:
		return true
	default:
		return false
	}
}
