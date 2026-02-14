//go:build windows

package builder

import (
	"golang.org/x/sys/windows"
)

// isProcessAlive checks if a process with the given PID is running.
func isProcessAlive(pid int) bool {
	// On Windows, os.FindProcess always succeeds even if the process doesn't exist.
	// We need to actually try to open the process to verify it's running.
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		// Process doesn't exist or we can't access it
		return false
	}
	defer windows.CloseHandle(h) //nolint:errcheck

	// Check if process is still running by getting exit code
	var exitCode uint32
	err = windows.GetExitCodeProcess(h, &exitCode)
	if err != nil {
		return false
	}

	// STILL_ACTIVE (259) means the process is running
	return exitCode == 259
}
