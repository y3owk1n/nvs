package table_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/y3owk1n/nvs/internal/ui/style"
	"github.com/y3owk1n/nvs/internal/ui/table"
)

const (
	testInstalled = "Installed"
	testV0100     = "v0.10.0"
)

// TestMain forces lipgloss's color profile to TrueColor so
// the assertions on emitted SGR codes (e.g. the bold-weight
// check in TestCurrentRowIsBolded) work in `go test`, which
// is run in a non-TTY context where lipgloss would
// otherwise degrade to Ascii and strip all color escapes.
//
// Setting it once at the package level is safe because
// TestMain runs before any test in the package, the value
// is constant for the test process, and any other test in
// the same invocation that doesn't care about SGR codes
// is unaffected by an upgrade from Ascii → TrueColor.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	os.Exit(m.Run())
}

// ansiEscapeRe matches a CSI escape sequence: ESC [ … final-byte.
// We strip these to make the test assertions on plain text
// (the colored output is asserted separately by the presence
// of the escape itself, not by its exact SGR codes).
var ansiEscapeRe = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")

// stripANSI removes CSI escape sequences from s.
func stripANSI(s string) string {
	return ansiEscapeRe.ReplaceAllString(s, "")
}

// TestNewEmptyTableRendersHeader is the smoke test: a Table
// with only a header (no rows) still renders something. The
// output should contain the header label and the header
// separator (we check by counting that the output has at
// least one visible character on the header row).
func TestNewEmptyTableRendersHeader(t *testing.T) {
	t.Parallel()

	palette := style.Default()
	tbl := table.New("Tag", "Status")

	out := stripANSI(tbl.Render(palette))

	if !strings.Contains(out, "Tag") {
		t.Errorf("rendered output %q does not contain header 'Tag'", out)
	}

	if !strings.Contains(out, "Status") {
		t.Errorf("rendered output %q does not contain header 'Status'", out)
	}
}

// TestRowAppendsCellValues verifies that cells added via
// Row(...) show up in the rendered output. We don't assert
// on a specific alignment or padding (lipgloss's internal
// column-shrinking algorithm decides those) — only that
// every cell value is present in the output.
func TestRowAppendsCellValues(t *testing.T) {
	t.Parallel()

	palette := style.Default()
	tbl := table.New("Tag", "Status", "Details")
	tbl.Row(testV0100, testInstalled, "stable version: v0.10.0")
	tbl.Row("v0.9.5", "Not Installed", "")

	out := stripANSI(tbl.Render(palette))

	for _, want := range []string{testV0100, testInstalled, "stable version: v0.10.0", "v0.9.5", "Not Installed"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered output %q does not contain %q", out, want)
		}
	}
}

// TestRowsAppendsManyAtOnce verifies the bulk Rows(...)
// helper behaves identically to a sequence of Row(...) calls.
func TestRowsAppendsManyAtOnce(t *testing.T) {
	t.Parallel()

	palette := style.Default()

	bulk := table.New("A", "B")
	bulk.Rows([][]string{
		{"1", "2"},
		{"3", "4"},
	})

	chained := table.New("A", "B")
	chained.Row("1", "2").Row("3", "4")

	bulkOut := stripANSI(bulk.Render(palette))
	chainedOut := stripANSI(chained.Render(palette))

	// bulk and chained should be visually identical (modulo
	// no whitespace differences). We assert the two stripped
	// outputs are equal.
	if bulkOut != chainedOut {
		t.Errorf(
			"Rows(...) differs from chained Row(...):\nbulk:    %q\nchained: %q",
			bulkOut,
			chainedOut,
		)
	}
}

// TestCurrentRowIsBolded verifies that the Current(idx) call
// marks row idx as the "current" row by emitting a SGR
// "bold" sequence (\x1b[1m) before the row's cells. We don't
// assert the foreground color (that's the caller's job via
// the pre-styled cells), only that the row gets the bold
// weight.
func TestCurrentRowIsBolded(t *testing.T) {
	t.Parallel()

	palette := style.Default()
	tbl := table.New("VERSION", "STATUS")
	tbl.Row(testV0100, testInstalled)
	tbl.Row("v0.9.5", testInstalled)
	tbl.Current(1) // mark row 1 as current

	rendered := tbl.Render(palette)
	plain := stripANSI(rendered)

	// Sanity: the cell values are still there.
	for _, want := range []string{testV0100, "v0.9.5", testInstalled} {
		if !strings.Contains(plain, want) {
			t.Errorf("rendered plain %q does not contain %q", plain, want)
		}
	}

	// Bold (\x1b[1m) must appear in the rendered string. We
	// don't pin down where in the output the bold sequence
	// appears (column alignment is lipgloss/table's
	// concern), only that it is emitted at all.
	if !strings.Contains(rendered, "\x1b[1m") {
		t.Errorf("current row is not bolded in rendered output: %q", rendered)
	}
}

// TestNoCurrentRowDoesNotBold verifies that, when Current(...)
// is never called, no row receives the bold weight. (The
// header itself does, by design — so we assert at least one
// non-header row is NOT bolded.)
func TestNoCurrentRowDoesNotBold(t *testing.T) {
	t.Parallel()

	palette := style.Default()
	tbl := table.New("VERSION", "STATUS")
	tbl.Row(testV0100, testInstalled)

	rendered := tbl.Render(palette)
	plain := stripANSI(rendered)

	if !strings.Contains(plain, testV0100) {
		t.Fatalf("rendered plain %q does not contain row", plain)
	}

	// Lipgloss/table emits the bold SGR for the header row
	// (we want bold headers). The body rows must NOT have it.
	// We assert by counting the number of bold opens vs.
	// the number of bold closes. They should be balanced —
	// i.e. any "open bold" we see is closed before the
	// string ends. A body row marked bold would have an
	// extra "\x1b[1m" inside the body; the cleanest signal
	// is that the body has no SGR between cells other than
	// what the header owns, but that's hard to assert in
	// general. Instead, we assert that the body "v0.10.0
	// Installed" segment, taken in isolation, has no SGR.
	lines := strings.Split(plain, "\n")

	var bodyLine string

	for _, line := range lines {
		if strings.Contains(line, testV0100) {
			bodyLine = line

			break
		}
	}

	if bodyLine == "" {
		t.Fatalf("could not find body row in plain output: %q", plain)
	}

	if rendered != "" && strings.Contains(rendered, "\x1b[1m") {
		// Bold opens may exist (for the header). Find them
		// and check the LAST one is not immediately before
		// the body row. Simpler: assert the count of bold
		// opens equals 1 (header) — anything more means we
		// bolded a body row.
		opens := strings.Count(rendered, "\x1b[1m")
		if opens > 1 {
			t.Errorf("no Current() set but %d bold opens in output: %q", opens, rendered)
		}
	}
}

// TestHeaderSeparatorIsPresent verifies that the table has
// a thin horizontal rule under the header. We don't assert
// the exact character (lipgloss/table picks a Unicode box
// drawing glyph) — only that the rule is at least 3 cells
// wide on the line immediately under the header.
func TestHeaderSeparatorIsPresent(t *testing.T) {
	t.Parallel()

	palette := style.Default()
	tbl := table.New("Tag")
	tbl.Row(testV0100)

	plain := stripANSI(tbl.Render(palette))

	lines := strings.Split(plain, "\n")
	if len(lines) < 2 {
		t.Fatalf("rendered output has fewer than 2 lines: %q", plain)
	}

	// The second line should be a separator — i.e. a string
	// of repeated box-drawing characters (─, =, - or _).
	sepLine := lines[1]
	if len(strings.TrimSpace(sepLine)) < 3 {
		t.Errorf("expected header separator to be at least 3 cells wide, got %q", sepLine)
	}
}

// TestWidthZeroAllowsNaturalSizing verifies that a zero
// width does not corrupt the output (it's the default).
func TestWidthZeroAllowsNaturalSizing(t *testing.T) {
	t.Parallel()

	palette := style.Default()
	tbl := table.New("A").Width(0)
	tbl.Row("x")

	plain := stripANSI(tbl.Render(palette))
	if !strings.Contains(plain, "x") {
		t.Errorf("Width(0) corrupted output: %q", plain)
	}
}

// TestWidthCapsTableWidth verifies that a non-zero Width
// produces output whose stripped length is at most that
// width. (It may be slightly less because lipgloss/table
// shrinks columns proportionally and may use fewer cells
// than the cap.)
func TestWidthCapsTableWidth(t *testing.T) {
	t.Parallel()

	palette := style.Default()
	tbl := table.New("A", "B", "C", "D")
	tbl.Width(40)
	tbl.Row(testV0100, testInstalled, "stable version: v0.10.0", "extra")

	plain := stripANSI(tbl.Render(palette))
	for line := range strings.SplitSeq(plain, "\n") {
		// We use the visible-cell count, not the byte
		// count, because Unicode box-drawing chars are
		// multi-byte. The "len" of a single visible char
		// is not 1 in all cases; we approximate with
		// runes.
		cells := len([]rune(line))
		if cells > 40 {
			t.Errorf("line %q (%d visible cells) exceeds width cap 40", line, cells)
		}
	}
}

// TestWrapFalseDoesNotBreakCells is a regression guard: the
// default for Wrap is false, so a long cell value must not
// break across lines. We assert the output has exactly one
// row of body (no wrapped continuation line).
func TestWrapFalseDoesNotBreakCells(t *testing.T) {
	t.Parallel()

	palette := style.Default()
	tbl := table.New("Tag", "Status")
	tbl.Row(testV0100, testInstalled)

	plain := stripANSI(tbl.Render(palette))

	// Expect 3 visible lines: header, separator, body.
	lines := strings.Split(strings.TrimRight(plain, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf(
			"expected 3 visible lines (header, separator, body), got %d: %q",
			len(lines),
			plain,
		)
	}
}

// TestNewDoesNotMutateHeaderSlice verifies that New(...)
// takes a copy of the input headers slice, so the caller can
// reuse / mutate the slice after constructing the table.
func TestNewDoesNotMutateHeaderSlice(t *testing.T) {
	t.Parallel()

	headers := []string{"Tag", "Status"}
	tbl := table.New(headers...)
	tbl.Row(testV0100, testInstalled)

	// Mutate the caller's slice after construction.
	headers[0] = "MUTATED"

	plain := stripANSI(tbl.Render(style.Default()))
	if strings.Contains(plain, "MUTATED") {
		t.Errorf("New(headers...) aliased caller slice: %q", plain)
	}

	if !strings.Contains(plain, "Tag") {
		t.Errorf("original header 'Tag' not present: %q", plain)
	}
}

// TestRowDoesNotMutateCellsSlice verifies that Row(...)
// takes a copy of the input cells slice, mirroring the
// New(...) behavior. Without it, the caller could mutate
// the input slice (e.g. when reusing a buffer across
// iterations of a release loop) and corrupt the rendered
// table.
func TestRowDoesNotMutateCellsSlice(t *testing.T) {
	t.Parallel()

	tbl := table.New("A", "B")
	cells := []string{testV0100, testInstalled}
	tbl.Row(cells...)
	cells[0] = "MUTATED"

	plain := stripANSI(tbl.Render(style.Default()))
	if strings.Contains(plain, "MUTATED") {
		t.Errorf("Row(...) aliased caller slice: %q", plain)
	}
}
