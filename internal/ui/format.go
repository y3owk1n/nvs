package ui

import (
	"time"

	"github.com/fatih/color"
)

// TimeFormat converts an ISO 8601 timestamp to a human-friendly date (YYYY-MM-DD).
// If the input cannot be parsed, it returns the original string.
func TimeFormat(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}

	return t.Format("2006-01-02")
}

// ColorizeRow applies the given color to each cell in the row and returns a new slice
// with the colorized strings.
func ColorizeRow(row []string, c *color.Color) []string {
	colored := make([]string, len(row))
	for i, cell := range row {
		colored[i] = c.Sprint(cell)
	}

	return colored
}

// FormatMessage formats a message with an icon and text.
func FormatMessage(icon, message string) string {
	return icon + " " + message
}
