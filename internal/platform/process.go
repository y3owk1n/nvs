package platform

import (
	"context"
	"os/exec"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

// IsNeovimRunning checks if any Neovim process is currently running.
// Returns true if nvim is running, along with the count of running instances.
func IsNeovimRunning() (bool, int) {
	if runtime.GOOS == "windows" {
		return isNeovimRunningWindows()
	}

	return isNeovimRunningUnix()
}

// isNeovimRunningUnix checks for running nvim processes on Unix systems.
func isNeovimRunningUnix() (bool, int) {
	// Use pgrep to find nvim processes
	cmd := exec.CommandContext(context.Background(), "pgrep", "-x", "nvim")

	output, err := cmd.Output()
	if err != nil {
		// pgrep returns exit code 1 if no processes found
		logrus.Debugf("pgrep returned error (likely no processes): %v", err)

		return false, 0
	}

	// Count lines in output (each line is a PID)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	count := 0

	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}

	logrus.Debugf("Found %d nvim processes", count)

	return count > 0, count
}

// isNeovimRunningWindows checks for running nvim processes on Windows.
func isNeovimRunningWindows() (bool, int) {
	// Use tasklist to find nvim.exe processes
	cmd := exec.CommandContext(
		context.Background(),
		"tasklist",
		"/FI", "IMAGENAME eq nvim.exe",
		"/FO", "CSV",
		"/NH",
	)

	output, err := cmd.Output()
	if err != nil {
		logrus.Debugf("tasklist returned error: %v", err)

		return false, 0
	}

	// Parse CSV output
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" || strings.Contains(outputStr, "No tasks") {
		return false, 0
	}

	// Count lines (excluding empty lines and "INFO: No tasks" message)
	lines := strings.Split(outputStr, "\n")
	count := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.Contains(trimmed, "INFO:") {
			count++
		}
	}

	logrus.Debugf("Found %d nvim.exe processes", count)

	return count > 0, count
}
