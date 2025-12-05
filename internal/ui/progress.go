package ui

import (
	"fmt"
	"strings"
)

const (
	// ProgressBarWidth is the default width of the progress bar.
	ProgressBarWidth = 20
	// ProgressFilled is the character for filled portion of the bar.
	ProgressFilled = "█"
	// ProgressEmpty is the character for empty portion of the bar.
	ProgressEmpty = "░"
	// ProgressMax is the maximum percentage value.
	ProgressMax = 100
)

// FormatProgressBar creates a visual progress bar string.
// Example output: "[████████████░░░░░░░░] 60%".
// Returns empty string for indeterminate progress (percent < 0).
func FormatProgressBar(percent int) string {
	if percent < 0 {
		return ""
	}

	if percent > ProgressMax {
		percent = ProgressMax
	}

	filled := (percent * ProgressBarWidth) / ProgressMax
	empty := ProgressBarWidth - filled

	bar := strings.Repeat(ProgressFilled, filled) + strings.Repeat(ProgressEmpty, empty)

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
