package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/ui/style"
)

// progressBarWidth is the bar's total width in terminal
// cells. 40 reads well on a 100-column terminal and keeps
// the percentage text to the right without crowding the
// spinner column the bar is usually rendered into.
const progressBarWidth = 40

// progressBarFull is the runes used for the filled and
// empty portions of the bar. They are the same defaults
// bubbles ships with, declared as constants here so the
// call sites can reference them and so the choice is
// visible in one place.
const (
	progressBarFullRune  = '█'
	progressBarEmptyRune = '░'
)

// FormatProgressBar creates a visual progress bar string.
// Example output: "██████████████████░░░░░░░░░░░░░░░░░  50%".
// Returns empty string for indeterminate progress (percent < 0).
//
// The implementation delegates to bubbles/progress so we
// don't have to maintain the bar math (fill width, total
// width, percentage rounding, percentage-text placement)
// ourselves. The view is rendered as a static snapshot
// (ViewAs, not View) so each call is allocation-cheap and
// the result is independent of any in-flight animation.
func FormatProgressBar(percent int) string {
	if percent < 0 {
		return ""
	}

	if percent > constants.ProgressMax {
		percent = constants.ProgressMax
	}

	// ViewAs takes a float in [0.0, 1.0]. Cast through
	// float64 to avoid integer-division truncation on the
	// small numerators that occur at low percentages.
	view := defaultProgressBar().ViewAs(float64(percent) / float64(constants.ProgressMax))

	return strings.TrimRight(view, " ")
}

// FormatPhaseProgress formats progress with a phase name and visual bar.
// Example output: "Downloading ██████████████████░░░░░░░░░░░░░░░░░ 40%".
// For indeterminate progress, shows just the phase: "Downloading".
func FormatPhaseProgress(phase string, percent int) string {
	bar := FormatProgressBar(percent)
	if bar == "" {
		return phase
	}

	return fmt.Sprintf("%s %s", phase, bar)
}

// defaultProgressBar returns a freshly-constructed
// bubbles/progress Model with the nvs look: a theme-colored
// fill, 40-cell width, no rounded border (bubbles' default
// surrounds the bar with `┃ … ┃` which adds visual weight
// we don't need for a status line), and a percentage suffix.
//
// Allocating a fresh Model per call is intentional: it
// keeps FormatProgressBar safe to call from concurrent
// goroutines (e.g. the install progress callback), and
// bubbles' ViewAs is allocation-cheap enough that the cost
// is negligible. If profiling later shows the per-call
// allocation matters, we can switch to a sync.Pool of
// pre-constructed Models.
func defaultProgressBar() progress.Model {
	palette := style.Default()

	model := progress.New(
		progress.WithWidth(progressBarWidth),
		// The default solid-fill rune is '█' and the
		// default empty rune is '░' — exactly what we
		// want, but we pass WithFillCharacters
		// explicitly so the choice is visible in one
		// place: a future refactor that switches to a
		// different rune can find it here without
		// grepping. (Caveat: do NOT index into a
		// string-literal with [0] here — that gives
		// back a byte, not a rune, and bubbles would
		// then render the first UTF-8 byte as a
		// Latin-1 character.)
		progress.WithFillCharacters(progressBarFullRune, progressBarEmptyRune),
		// Fill the completed portion with the theme's
		// primary color and the remaining portion with
		// the subtle/muted color so the bar participates
		// in the nvs design system.
		progress.WithSolidFill(palette.Primary.Dark),
		// No border: bubbles' default surrounds the bar
		// with `┃ … ┃` which adds visual weight that
		// doesn't earn its keep in a status line.
	)

	model.EmptyColor = palette.Subtle.Dark

	return model
}
