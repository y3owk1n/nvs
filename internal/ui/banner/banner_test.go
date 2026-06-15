package banner_test

import (
	"strings"
	"testing"

	"github.com/y3owk1n/nvs/internal/ui/banner"
	"github.com/y3owk1n/nvs/internal/ui/style"
)

func TestLogoIncludesWordmarkAndTagline(t *testing.T) {
	t.Parallel()

	got := banner.Logo(style.Default())

	if !strings.Contains(got, "nvs") {
		t.Errorf("Logo() = %q, missing wordmark", got)
	}

	if !strings.Contains(got, "Neovim Version Switcher") {
		t.Errorf("Logo() = %q, missing tagline", got)
	}
}

func TestMarkIsCompact(t *testing.T) {
	t.Parallel()

	got := banner.Mark(style.Default())

	if got == "" {
		t.Error("Mark() returned empty string")
	}

	if !strings.Contains(got, "nvs") {
		t.Errorf("Mark() = %q, missing wordmark", got)
	}
}

func TestHeaderHasTitleAndUnderline(t *testing.T) {
	t.Parallel()

	got := banner.Header(style.Default(), "Installed versions")

	if !strings.Contains(got, "Installed versions") {
		t.Errorf("Header() = %q, missing title", got)
	}

	// The underline is one ─ per visible character of the
	// title. Strip ANSI escapes and confirm the count.
	stripped := stripANSI(got)
	underlineCount := strings.Count(stripped, "─")

	if underlineCount != len("Installed versions") {
		t.Errorf("Header() underline has %d ─, want %d", underlineCount, len("Installed versions"))
	}
}

func stripANSI(value string) string {
	out := make([]byte, 0, len(value))
	inEscape := false

	for idx := 0; idx < len(value); idx++ {
		if inEscape {
			if value[idx] >= 0x40 && value[idx] <= 0x7E {
				inEscape = false
			}

			continue
		}

		if idx+1 < len(value) && value[idx] == 0x1b && value[idx+1] == '[' {
			inEscape = true
			idx++

			continue
		}

		out = append(out, value[idx])
	}

	return string(out)
}
