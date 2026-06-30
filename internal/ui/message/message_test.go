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

func TestStatusRowsIncludeIconsAndLabels(t *testing.T) {
	t.Parallel()

	icons := message.DefaultIcons()

	printer := message.New(
		style.Default(),
		style.Types(style.Default()),
		icons,
		io.Discard, io.Discard,
	)

	cases := []struct {
		name string
		row  string
		want string
	}{
		{"SuccessRow", printer.SuccessRow("Shell"), icons.Success},
		{"WarnRow", printer.WarnRow("Dependencies"), icons.Warn},
		{"ErrorRow", printer.ErrorRow("PATH"), icons.Error},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if !strings.HasSuffix(testCase.row, "\n") {
				t.Errorf("%s %q does not end with newline", testCase.name, testCase.row)
			}

			plain := strings.TrimRight(stripANSI(testCase.row), "\n")
			if !strings.Contains(plain, testCase.want) {
				t.Errorf(
					"%s plain %q does not contain icon %q",
					testCase.name,
					plain,
					testCase.want,
				)
			}

			if !strings.Contains(plain, testCase.want+" ") {
				t.Errorf("%s plain %q missing single-space gap after icon", testCase.name, plain)
			}
		})
	}
}

func TestDetailIsIndentedAndMuted(t *testing.T) {
	t.Parallel()

	printer := message.New(
		style.Default(),
		style.Types(style.Default()),
		message.DefaultIcons(),
		io.Discard, io.Discard,
	)

	got := printer.Detail("bin directory not in PATH")

	plain := strings.TrimRight(stripANSI(got), "\n")
	if !strings.HasPrefix(plain, "    ") {
		t.Errorf("Detail %q does not start with a 4-space indent", plain)
	}

	if !strings.HasSuffix(plain, "bin directory not in PATH") {
		t.Errorf("Detail %q does not end with text", plain)
	}
}

const (
	testCellS = "Success"
	testCellW = "Warn"
	testCellE = "Error"
	testCellA = "Accent"
	testCellT = "Text"
	testCellM = "Muted"
)

// TestCellHelpersReturnPlainTextAfterStrip verifies that the
// cell-color helpers (Success, Warn, Error, Accent, Text,
// Muted) are non-I/O: they return a string that, after
// stripping ANSI escapes, contains exactly the input text.
// The helpers must also never be empty (so a caller building
// a table cell can rely on the result being printable).
func TestCellHelpersReturnPlainTextAfterStrip(t *testing.T) {
	t.Parallel()

	printer := message.New(
		style.Default(),
		style.Types(style.Default()),
		message.DefaultIcons(),
		io.Discard, io.Discard,
	)

	cases := []struct {
		name string
		cell string
	}{
		{testCellS, printer.Success("Installed")},
		{testCellW, printer.Warn("Installed (upgrade)")},
		{testCellE, printer.Error("Not Installed")},
		{testCellA, printer.Accent("/Users/me/.local/bin")},
		{testCellT, printer.Text("v0.10.4")},
		{testCellM, printer.Muted("Published: 2024-05-01, Commit: abc1234")},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if testCase.cell == "" {
				t.Errorf("%s returned empty string", testCase.name)
			}

			plain := stripANSI(testCase.cell)
			// The helper must preserve the input text exactly
			// — no leading/trailing whitespace changes, no
			// icon injection (those are reserved for the
			// *Row helpers).
			wantText := map[string]string{
				testCellS: "Installed",
				testCellW: "Installed (upgrade)",
				testCellE: "Not Installed",
				testCellA: "/Users/me/.local/bin",
				testCellT: "v0.10.4",
				testCellM: "Published: 2024-05-01, Commit: abc1234",
			}[testCase.name]

			if plain != wantText {
				t.Errorf(
					"%s stripped %q does not equal input %q",
					testCase.name,
					plain,
					wantText,
				)
			}
		})
	}
}

// TestCellHelpersDoNotEmitNewlines verifies that the cell-color
// helpers do not inject trailing newlines. Table cells are
// composed side-by-side by lipgloss/table; a stray newline
// would push the next cell onto its own line and break the
// grid. The *Row helpers do emit newlines (each row is its
// own line) but the cell-color helpers must not.
func TestCellHelpersDoNotEmitNewlines(t *testing.T) {
	t.Parallel()

	printer := message.New(
		style.Default(),
		style.Types(style.Default()),
		message.DefaultIcons(),
		io.Discard, io.Discard,
	)

	cells := []struct {
		name string
		val  string
	}{
		{"Success", printer.Success("x")},
		{"Warn", printer.Warn("x")},
		{"Error", printer.Error("x")},
		{"Accent", printer.Accent("x")},
		{"Text", printer.Text("x")},
		{"Muted", printer.Muted("x")},
	}

	for _, cell := range cells {
		t.Run(cell.name, func(t *testing.T) {
			t.Parallel()

			if strings.Contains(cell.val, "\n") {
				t.Errorf("%s %q contains a newline", cell.name, cell.val)
			}
		})
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
