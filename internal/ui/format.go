package ui

import "time"

// TimeFormat converts an ISO 8601 timestamp to a human-friendly
// date (YYYY-MM-DD). If the input cannot be parsed, it returns
// the original string.
func TimeFormat(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}

	return t.Format("2006-01-02")
}
