package ui_test

import (
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/y3owk1n/nvs/internal/ui"
)

func TestTimeFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid RFC3339 timestamp",
			input: "2024-12-01T10:30:00Z",
			want:  "2024-12-01",
		},
		{
			name:  "valid RFC3339 with timezone",
			input: "2024-06-15T14:45:30+08:00",
			want:  "2024-06-15",
		},
		{
			name:  "invalid timestamp returns original",
			input: "not-a-date",
			want:  "not-a-date",
		},
		{
			name:  "empty string returns empty",
			input: "",
			want:  "",
		},
		{
			name:  "partial date returns original",
			input: "2024-12-01",
			want:  "2024-12-01",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := ui.TimeFormat(testCase.input)
			if got != testCase.want {
				t.Errorf("TimeFormat(%q) = %q, want %q", testCase.input, got, testCase.want)
			}
		})
	}
}

func TestTimeFormat_RecentDates(t *testing.T) {
	// Test with a dynamically generated date
	now := time.Now().UTC().Format(time.RFC3339)
	result := ui.TimeFormat(now)

	// Result should be in YYYY-MM-DD format
	if len(result) != 10 {
		t.Errorf("TimeFormat() returned %q, expected 10-character date", result)
	}
}

func TestColorizeRow(t *testing.T) {
	tests := []struct {
		name string
		row  []string
		want int // expected length of result
	}{
		{
			name: "empty row",
			row:  []string{},
			want: 0,
		},
		{
			name: "single cell",
			row:  []string{"hello"},
			want: 1,
		},
		{
			name: "multiple cells",
			row:  []string{"cell1", "cell2", "cell3"},
			want: 3,
		},
	}

	c := color.New(color.FgGreen)

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := ui.ColorizeRow(testCase.row, c)

			if len(got) != testCase.want {
				t.Errorf("ColorizeRow() length = %d, want %d", len(got), testCase.want)
			}

			// Verify each cell is colorized (contains ANSI codes when color is enabled)
			// Note: In CI environments, color may be disabled, so we just check length
			for i, cell := range got {
				if len(cell) == 0 && len(testCase.row[i]) > 0 {
					t.Errorf("ColorizeRow() cell %d is empty, expected content", i)
				}
			}
		})
	}
}

func TestFormatMessage(t *testing.T) {
	tests := []struct {
		name    string
		icon    string
		message string
		want    string
	}{
		{
			name:    "check mark with message",
			icon:    "✓",
			message: "Switched to Neovim stable",
			want:    "✓ Switched to Neovim stable",
		},
		{
			name:    "error icon with message",
			icon:    "✗",
			message: "Version not found",
			want:    "✗ Version not found",
		},
		{
			name:    "empty icon",
			icon:    "",
			message: "Just a message",
			want:    " Just a message",
		},
		{
			name:    "empty message",
			icon:    "•",
			message: "",
			want:    "• ",
		},
		{
			name:    "both empty",
			icon:    "",
			message: "",
			want:    " ",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := ui.FormatMessage(testCase.icon, testCase.message)
			if got != testCase.want {
				t.Errorf(
					"FormatMessage(%q, %q) = %q, want %q",
					testCase.icon,
					testCase.message,
					got,
					testCase.want,
				)
			}
		})
	}
}
