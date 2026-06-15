package message_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/y3owk1n/nvs/internal/ui/message"
	"github.com/y3owk1n/nvs/internal/ui/style"
)

func TestPrinterInfoIncludesIcon(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	printer := message.New(
		style.Default(),
		style.Types(style.Default()),
		message.DefaultIcons(),
		&buf, &buf,
	)
	printer.Infof("hello %s", "world")

	out := buf.String()
	if !strings.Contains(out, "hello world") {
		t.Errorf("Info output %q does not contain message", out)
	}

	// The default info icon must be present even when the
	// terminal doesn't render the color escapes.
	if !strings.Contains(out, message.DefaultIcons().Info) {
		t.Errorf("Info output %q does not contain the info icon", out)
	}
}

func TestPrinterErrorGoesToErrOut(t *testing.T) {
	t.Parallel()

	var out, errOut bytes.Buffer

	printer := message.New(
		style.Default(),
		style.Types(style.Default()),
		message.DefaultIcons(),
		&out, &errOut,
	)
	printer.Errorf("boom")

	if out.Len() != 0 {
		t.Errorf("Error wrote to stdout: %q", out.String())
	}

	if !strings.Contains(errOut.String(), "boom") {
		t.Errorf("Error did not write to errOut: %q", errOut.String())
	}
}

func TestPrinterPairAlignsKey(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	printer := message.New(
		style.Default(),
		style.Types(style.Default()),
		message.DefaultIcons(),
		&buf, &buf,
	)
	printer.Pair("Version", "v0.10.4")
	printer.Pair("Commit", "abc1234")

	out := buf.String()

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	if len(lines) != 2 {
		t.Fatalf("Pair wrote %d lines, want 2: %q", len(lines), out)
	}

	// Each line should end with the value. We don't assert
	// exact width because lipgloss may add escape sequences
	// for the alignment, but the value must come last.
	for idx, want := range []string{"v0.10.4", "abc1234"} {
		if !strings.HasSuffix(stripANSI(lines[idx]), want) {
			t.Errorf("line %d %q does not end with %q", idx, lines[idx], want)
		}
	}
}

func TestPairLineReturnsStyledString(t *testing.T) {
	t.Parallel()

	printer := message.New(
		style.Default(),
		style.Types(style.Default()),
		message.DefaultIcons(),
		io.Discard, io.Discard,
	)

	got := printer.PairLine("Latest tag", "v0.10.4")

	if got == "" {
		t.Fatal("PairLine returned empty string")
	}

	plain := strings.TrimRight(stripANSI(got), "\n")

	if !strings.Contains(plain, "Latest tag") {
		t.Errorf("PairLine plain %q does not contain key", plain)
	}

	if !strings.HasSuffix(plain, "v0.10.4") {
		t.Errorf("PairLine plain %q does not end with value", plain)
	}

	// PairLine must end in a newline so it composes correctly
	// when concatenated inside a multi-line body string.
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("PairLine %q does not end with newline", got)
	}
}

func TestHighlightWrapsText(t *testing.T) {
	t.Parallel()

	printer := message.New(
		style.Default(),
		style.Types(style.Default()),
		message.DefaultIcons(),
		io.Discard, io.Discard,
	)

	got := printer.Highlight("stable")

	if !strings.Contains(stripANSI(got), "stable") {
		t.Errorf("Highlight stripped %q does not contain text", got)
	}
}

func TestDimWrapsText(t *testing.T) {
	t.Parallel()

	printer := message.New(
		style.Default(),
		style.Types(style.Default()),
		message.DefaultIcons(),
		io.Discard, io.Discard,
	)

	got := printer.Dim("Details: unavailable")

	if !strings.Contains(stripANSI(got), "Details: unavailable") {
		t.Errorf("Dim stripped %q does not contain text", got)
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
