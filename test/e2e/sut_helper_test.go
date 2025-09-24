//go:build e2e

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestIterProcessesByCmd tests that iterProcessesByCmd correctly iterates over actual PIDs.
func TestIterProcessesByCmd(t *testing.T) {
	sut := NewSystemUnderTest(t)

	// simulate some processes being tracked
	testPIDs := []int{12345, 67890, 99999}
	cmdName := "testcmd"

	// manually add PIDs to the tracking maps.
	for _, pid := range testPIDs {
		sut.pids[pid] = struct{}{}
		sut.cmdToPids[cmdName] = append(sut.cmdToPids[cmdName], pid)
	}

	var iteratedPIDs []int
	for process := range sut.iterProcessesByCmd(cmdName) {
		iteratedPIDs = append(iteratedPIDs, process.Pid)
	}

	// The iterator should return the actual PIDs
	require.ElementsMatch(t, testPIDs, iteratedPIDs,
		"iterProcessesByCmd should iterate over actual PIDs")
}

// TestShutdownByCmd tests that ShutdownByCmd can properly find and signal processes.
// This test verifies that the fix allows the shutdown mechanism to work correctly.
func TestShutdownByCmd(t *testing.T) {
	sut := NewSystemUnderTest(t)

	sut.ExecCmd("sleep")

	require.Eventually(t, func() bool {
		return sut.HasProcess("sleep")
	}, time.Millisecond*100, time.Millisecond*10, "process should be tracked after starting")

	sut.ShutdownByCmd("sleep")

	require.Eventually(t, func() bool {
		return !sut.HasProcess("sleep")
	}, time.Millisecond*100, time.Millisecond*10, "process should be removed from tracking after shutdown")
}
