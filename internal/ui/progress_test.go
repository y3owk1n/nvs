package ui_test

import (
	"testing"

	"github.com/y3owk1n/nvs/internal/ui"
)

func TestFormatProgressBar(t *testing.T) {
	tests := []struct {
		name    string
		percent int
		want    string
	}{
		{
			name:    "0 percent",
			percent: 0,
			want:    "[░░░░░░░░░░░░░░░░░░░░]   0%",
		},
		{
			name:    "50 percent",
			percent: 50,
			want:    "[██████████░░░░░░░░░░]  50%",
		},
		{
			name:    "100 percent",
			percent: 100,
			want:    "[████████████████████] 100%",
		},
		{
			name:    "negative for indeterminate",
			percent: -10,
			want:    "",
		},
		{
			name:    "over 100 clamped to 100",
			percent: 150,
			want:    "[████████████████████] 100%",
		},
		{
			name:    "25 percent",
			percent: 25,
			want:    "[█████░░░░░░░░░░░░░░░]  25%",
		},
		{
			name:    "75 percent",
			percent: 75,
			want:    "[███████████████░░░░░]  75%",
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
			want:    "Downloading [░░░░░░░░░░░░░░░░░░░░]   0%",
		},
		{
			name:    "extracting 100 percent",
			phase:   "Extracting",
			percent: 100,
			want:    "Extracting [████████████████████] 100%",
		},
		{
			name:    "verifying 50 percent",
			phase:   "Verifying",
			percent: 50,
			want:    "Verifying [██████████░░░░░░░░░░]  50%",
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
