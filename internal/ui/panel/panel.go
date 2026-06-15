// Package panel renders a rounded box around arbitrary
// content. It is the visual equivalent of a card in a GUI and
// is what every high-level nvs command should wrap its summary
// in.
//
// The package exposes two flavors:
//
//   - Panel(content)               — a simple box, content left-aligned.
//   - Section(title, content)      — a panel with a styled title
//     row at the top.
//
// Both flavors auto-size to the terminal width (capped at
// DefaultMaxWidth columns) and respect NO_COLOR via lipgloss.
package panel

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/y3owk1n/nvs/internal/ui/style"
)

// DefaultMaxWidth caps panel width to keep multi-line output
// readable on wide terminals. 80 is the historical "comfortable
// prose" width and is what the rest of the nvs UI assumes.
const DefaultMaxWidth = 80

// borderSize is the visual width of a single rounded-border
// glyph (lipgloss adds one on the left and one on the right).
const borderSize = 1

// borderSides is the number of border glyphs lipgloss adds
// around a frame (left + right). Multiplying borderSize by
// borderSides gives the total horizontal width the border
// consumes.
const borderSides = 2

// paddingX is the horizontal padding applied inside a Panel
// (not a Section). It is two cells so multi-line content
// has a little air on both sides.
const paddingX = 2

// paddingY is the vertical padding applied inside a Panel
// and a Section.
const paddingY = 1

// Panel wraps content in a rounded box with a subtle border.
// The returned string already includes a trailing newline.
func Panel(palette style.Palette, content string) string {
	outer := computeWidth()
	frameWidth := outer - borderSides*borderSize

	styled := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(palette.Border).
		Padding(paddingY, paddingX).
		Width(frameWidth)

	return styled.Render(content) + "\n"
}

// Section wraps content in a rounded box with a bold title row
// at the top. The title is rendered in the primary color so it
// reads as the "label" of the section.
//
// Example:
//
//	ui.Panel.Section(p, "Installed versions",
//	    "stable\nnightly\nv0.10.4")
//
// The returned string already includes a trailing newline.
func Section(palette style.Palette, title, content string) string {
	// `outer` is the desired visible width of the box
	// including its border. `frameWidth` is the value we hand
	// to lipgloss's Width(), which is the *content* width —
	// the border is added on top of Width() automatically.
	outer := computeWidth()
	frameWidth := outer - borderSides*borderSize
	contentWidth := frameWidth - borderSides*borderSize

	frame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(palette.Border).
		Padding(paddingY, paddingY).
		Width(frameWidth)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(palette.Primary).
		Width(contentWidth).
		Align(lipgloss.Left)

	bodyStyle := lipgloss.NewStyle().
		Foreground(palette.Text).
		Width(contentWidth)

	underline := lipgloss.NewStyle().
		Foreground(palette.Border).
		Render(strings.Repeat("─", contentWidth))

	rendered := titleStyle.Render(title) + "\n" +
		underline + "\n" +
		bodyStyle.Render(content)

	return frame.Render(rendered) + "\n"
}

// computeWidth returns the smaller of the live terminal width
// and DefaultMaxWidth. When stdout is not a terminal (for
// example when the user pipes the output to a file), it
// returns DefaultMaxWidth so the panel still renders
// correctly.
func computeWidth() int {
	width, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil || width <= 0 {
		return DefaultMaxWidth
	}

	if width < DefaultMaxWidth {
		return width
	}

	return DefaultMaxWidth
}
