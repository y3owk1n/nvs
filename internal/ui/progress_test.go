package ui_test

import (
	"math"
	"strings"
	"testing"

	"github.com/y3owk1n/nvs/internal/ui"
)

// TestFormatProgressBar verifies the bytes that come out of
// FormatProgressBar match what we expect from the bubbles
// model. The bar is `progressBarWidth` cells wide, with
// `█` for the filled portion, `░` for the empty portion,
// and a ` XX%` suffix.
//
// We assert on the exact bytes (not the visual glyphs) so
// the test is robust to the LANG / locale of the test
// runner; the rune values themselves are pinned by the
// formatProgressBarRunes table in the helper below.
//
// The math matches bubbles' progress.Model.barView:
// bar_width = total_width - percent_text_width
// filled    = round(bar_width * percent/100).
const (
	testBarWidth       = 40
	testPercentTextLen = 5 // " 100%" / "  50%" / "   0%" — all 5 chars
)

func TestFormatProgressBar(t *testing.T) {
	barCells := testBarWidth - testPercentTextLen

	tests := []struct {
		name    string
		percent int
		want    string
	}{
		{
			name:    "50 percent",
			percent: 50,
			want:    formatProgressBarRunes(roundPercent(50), barCells-roundPercent(50)) + "  50%",
		},
		{
			name:    "100 percent",
			percent: 100,
			want:    formatProgressBarRunes(barCells, 0) + " 100%",
		},
		{
			name:    "negative for indeterminate",
			percent: -10,
			want:    "",
		},
		{
			name:    "over 100 clamped to 100",
			percent: 150,
			want:    formatProgressBarRunes(barCells, 0) + " 100%",
		},
		{
			name:    "25 percent",
			percent: 25,
			want: formatProgressBarRunes(
				roundPercent(25),
				barCells-roundPercent(25),
			) + "  25%",
		},
		{
			name:    "75 percent",
			percent: 75,
			want: formatProgressBarRunes(
				roundPercent(75),
				barCells-roundPercent(75),
			) + "  75%",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := ui.FormatProgressBar(testCase.percent)
			if got != testCase.want {
				t.Errorf(
					"FormatProgressBar(%d) = %q, want %q",
					testCase.percent,
					got,
					testCase.want,
				)
			}
		})
	}
}

func TestFormatPhaseProgress(t *testing.T) {
	barCells := testBarWidth - testPercentTextLen

	tests := []struct {
		name    string
		phase   string
		percent int
		want    string
	}{
		{
			name:    "downloading 0 percent",
			phase:   "Downloading",
			percent: 0,
			want:    "Downloading " + formatProgressBarRunes(0, barCells) + "   0%",
		},
		{
			name:    "extracting 100 percent",
			phase:   "Extracting",
			percent: 100,
			want:    "Extracting " + formatProgressBarRunes(barCells, 0) + " 100%",
		},
		{
			name:    "verifying 50 percent",
			phase:   "Verifying",
			percent: 50,
			want: "Verifying " + formatProgressBarRunes(
				roundPercent(50),
				barCells-roundPercent(50),
			) + "  50%",
		},
		{
			name:    "cloning indeterminate",
			phase:   "Cloning repository",
			percent: -1,
			want:    "Cloning repository",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := ui.FormatPhaseProgress(testCase.phase, testCase.percent)
			if got != testCase.want {
				t.Errorf(
					"FormatPhaseProgress(%q, %d) = %q, want %q",
					testCase.phase,
					testCase.percent,
					got,
					testCase.want,
				)
			}
		})
	}
}

// TestFormatProgressBarRuneFillAndEmpty are explicit guard
// tests for the two runes bubbles' progress bar uses. We
// hit them on the literal output so a future refactor that
// silently swaps one rune (e.g. from '█' to '#') breaks
// this test loudly, before the visual change reaches a
// user.
func TestFormatProgressBarRuneFillAndEmpty(t *testing.T) {
	t.Parallel()

	full := ui.FormatProgressBar(100)
	if !strings.Contains(full, "█") {
		t.Errorf("100%% bar %q does not contain the filled rune '█'", full)
	}

	empty := ui.FormatProgressBar(0)
	if !strings.Contains(empty, "░") {
		t.Errorf("0%% bar %q does not contain the empty rune '░'", empty)
	}
}

// formatProgressBarRunes builds a reference bar string of
// `filled` '█' runes followed by `empty` '░' runes. It is
// the test-time mirror of what bubbles' progress model
// produces; declaring it here (instead of hard-coding
// the bar in every test case) makes the math obvious and
// keeps the table compact.
func formatProgressBarRunes(filled, empty int) string {
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

// roundPercent computes the same fill width bubbles'
// progress.Model.barView computes for a bar of total
// cells `barCells` at `percent` (0..100): round the
// float64 product barCells * percent / 100. The float64
// step matters because integer division of
// barCells * 25 / 100 truncates differently than the
// float math bubbles does (e.g. 35 * 25 / 100 = 8
// truncated, but math.Round(35 * 0.25) = 9).
//
// The unparam linter is happy with this signature
// because every caller in the test table uses the same
// `barCells` value (the file-scope constant); the
// parameter is here for readability, not for varying
// inputs. If a future test needs a different bar width,
// drop the file-scope constant and re-pass the
// per-case value.
func roundPercent(percent int) int {
	return int(math.Round(float64(testBarWidth-testPercentTextLen) * float64(percent) / 100))
}
