//go:build !windows

package builder

import (
	"os"
	"syscall"
)

// isProcessAlive checks if a process with the given PID is running.
func isProcessAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	return process.Signal(syscall.Signal(0)) == nil
}
