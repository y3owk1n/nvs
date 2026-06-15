// Package table renders a minimal, modern table for nvs.
//
// The package is a thin wrapper around github.com/charmbracelet/lipgloss/table
// that encodes the nvs design: no outer border, a thin header
// separator, padded cells, and a "current" row that gets a
// bold + primary-color treatment. Callers add rows of
// pre-styled cells (e.g. via ui.Message.Highlight / Success /
// Muted) — the table itself does not re-style cells, it only
// handles alignment, column gutter, and the header.
//
// The wrapper exists so callers do not have to know about
// the underlying chainable API, the StyleFunc, or the
// HeaderRow sentinel; they just construct a Table, add rows
// with Row(...), optionally mark the current row with
// Current(idx), and call Render(palette).
package table

import (
	"github.com/charmbracelet/lipgloss"
	ltable "github.com/charmbracelet/lipgloss/table"
	"github.com/y3owk1n/nvs/internal/ui/style"
)

// noCurrentRow is the sentinel for "no row is the current
// row" in Table.Current. It is exported only as a constant
// for the wsl/mnd linters; callers should never set it
// directly.
const noCurrentRow = -1

// columnPadding is the horizontal cell padding (in terminal
// cells) applied to every cell. Two cells gives a clean
// gutter between columns without needing a vertical column
// border, which would add visual noise.
const columnPadding = 2

// defaultWrap is the cell-wrap policy used when the caller
// does not set Wrap(...) explicitly. We default to "no
// wrap" because nvs tables hold things like commit hashes
// and version strings that should never break across lines.
const defaultWrap = false

// Table is the nvs-flavored table primitive.
type Table struct {
	headers    []string
	rows       [][]string
	currentRow int
	width      int
	wrap       bool
}

// New returns a Table with the given column headers. A copy
// of headers is stored; the caller can mutate the input
// slice without affecting the Table.
func New(headers ...string) *Table {
	return &Table{
		headers:    append([]string(nil), headers...),
		currentRow: noCurrentRow,
		wrap:       defaultWrap,
	}
}

// Row appends a row to the Table. Cells are pre-styled by
// the caller (e.g. via ui.Message.Highlight / Success /
// Muted) — the table does not re-style them, it only
// handles alignment and the column gutter.
func (t *Table) Row(cells ...string) *Table {
	copied := append([]string(nil), cells...)
	t.rows = append(t.rows, copied)

	return t
}

// Rows appends multiple rows in one call. Each row slice
// is copied so the caller can mutate the input without
// affecting the Table.
func (t *Table) Rows(rows [][]string) *Table {
	for _, row := range rows {
		copied := append([]string(nil), row...)
		t.rows = append(t.rows, copied)
	}

	return t
}

// Current marks row index idx as the "current" row. The
// table will render that row's cells in bold (the color
// is whatever the caller pre-styled the cells with — the
// table does not impose a color on the current row, only
// the bold weight, so a cell that is already
// ui.Message.Highlight keeps its primary color).
//
// idx is zero-based and refers to the order of Row()
// calls. A negative idx (or never calling Current) means
// no row is marked current; the default of -1 covers that.
func (t *Table) Current(idx int) *Table {
	t.currentRow = idx

	return t
}

// Width sets the maximum table width. lipgloss/table
// auto-sizes columns to fit the content; setting a width
// caps the total and shrinks columns proportionally. 0
// means "no cap" (use the natural content width).
func (t *Table) Width(w int) *Table {
	t.width = w

	return t
}

// Wrap toggles whether cell content wraps inside the
// column. The default is false (commit hashes and version
// strings should never break). Callers that need wrapping
// (e.g. for free-form notes) can set this to true.
func (t *Table) Wrap(wrap bool) *Table {
	t.wrap = wrap

	return t
}

// Render returns the Table as a styled string. The string
// has no trailing newline.
func (t *Table) Render(palette style.Palette) string {
	cellPadding := lipgloss.NewStyle().Padding(0, columnPadding)

	headerStyle := cellPadding.
		Bold(true).
		Foreground(palette.Subtle)

	// The current-row style intentionally does NOT set a
	// foreground: the caller pre-styles each cell with the
	// color it wants (typically ui.Message.Highlight, which
	// is bold + primary). Imposing a foreground here would
	// either double-set it or, worse, mask the caller's
	// per-cell color choice.
	currentStyle := cellPadding.Bold(true)

	bodyStyle := cellPadding

	borderStyle := lipgloss.NewStyle().Foreground(palette.Border)

	tbl := ltable.New().
		Headers(t.headers...).
		Border(lipgloss.NormalBorder()).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(false).
		BorderRow(false).
		BorderHeader(true).
		BorderStyle(borderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch row {
			case ltable.HeaderRow:
				return headerStyle
			case t.currentRow:
				return currentStyle
			default:
				return bodyStyle
			}
		}).
		Wrap(t.wrap)

	if t.width > 0 {
		tbl.Width(t.width)
	}

	for _, row := range t.rows {
		tbl.Row(row...)
	}

	return tbl.String()
}
