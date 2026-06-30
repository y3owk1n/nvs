package ui_test

import (
	"testing"
	"time"

	"github.com/y3owk1n/nvs/internal/ui"
)

const testDate = "2024-12-01"

func TestTimeFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid RFC3339 timestamp",
			input: "2024-12-01T10:30:00Z",
			want:  testDate,
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
			input: testDate,
			want:  testDate,
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
	now := time.Now().UTC().Format(time.RFC3339)
	result := ui.TimeFormat(now)

	if len(result) != 10 {
		t.Errorf("TimeFormat() returned %q, expected 10-character date", result)
	}
}
