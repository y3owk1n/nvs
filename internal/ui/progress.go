package ui

import (
	"fmt"
	"strings"

	"github.com/y3owk1n/nvs/internal/constants"
)

// FormatProgressBar creates a visual progress bar string.
// Example output: "[████████████░░░░░░░░] 60%".
// Returns empty string for indeterminate progress (percent < 0).
func FormatProgressBar(percent int) string {
	if percent < 0 {
		return ""
	}

	if percent > constants.ProgressMax {
		percent = constants.ProgressMax
	}

	filled := (percent * constants.ProgressBarWidth) / constants.ProgressMax
	empty := constants.ProgressBarWidth - filled

	bar := strings.Repeat(
		constants.ProgressFilled,
		filled,
	) + strings.Repeat(
		constants.ProgressEmpty,
		empty,
	)

	return fmt.Sprintf("[%s] %3d%%", bar, percent)
}

// FormatPhaseProgress formats progress with a phase name and visual bar.
// Example output: "Downloading [████████░░░░░░░░░░░░] 40%".
// For indeterminate progress, shows just the phase: "Downloading".
func FormatPhaseProgress(phase string, percent int) string {
	bar := FormatProgressBar(percent)
	if bar == "" {
		return phase
	}

	return fmt.Sprintf("%s %s", phase, bar)
}
