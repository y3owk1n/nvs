package style

import "github.com/charmbracelet/lipgloss"

// Type is the pre-built typographic scale. Each field is a
// lipgloss.Style the UI can apply directly.
//
// Reuse these rather than constructing ad-hoc styles — every
// title should look like every other title, etc.
type Type struct {
	// Title is the largest style, used for the wordmark and
	// for command summary headers.
	Title lipgloss.Style

	// Section is a bold, smaller header used to introduce a
	// sub-section of output (e.g. "Installed versions" inside
	// a panel).
	Section lipgloss.Style

	// Body is the default text style. Most output uses this
	// implicitly by not specifying a style at all.
	Body lipgloss.Style

	// Muted is for secondary information (descriptions under a
	// heading, captions).
	Muted lipgloss.Style

	// Subtle is for tertiary information (timestamps, hints,
	// separators).
	Subtle lipgloss.Style

	// Code is for inline values that look like code (paths,
	// commit hashes, version strings).
	Code lipgloss.Style

	// Key is for the left-hand side of a "Key: value" pair.
	Key lipgloss.Style
}

// KeyColumnWidth is the right-aligned width used for the
// "key" column in ui/message.Pair. 14 chars fits "Published"
// and "Configuration" comfortably; longer keys wrap to the
// next visual line.
const KeyColumnWidth = 14

// Types returns the pre-built typographic scale, colored with
// the given palette.
func Types(palette Palette) Type {
	return Type{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(palette.Primary).
			MarginBottom(1),

		Section: lipgloss.NewStyle().
			Bold(true).
			Foreground(palette.Text).
			MarginTop(1).
			MarginBottom(1),

		Body: lipgloss.NewStyle().
			Foreground(palette.Text),

		Muted: lipgloss.NewStyle().
			Foreground(palette.Muted),

		Subtle: lipgloss.NewStyle().
			Foreground(palette.Subtle).
			Italic(true),

		Code: lipgloss.NewStyle().
			Foreground(palette.Accent),

		Key: lipgloss.NewStyle().
			Foreground(palette.Muted).
			Width(KeyColumnWidth).
			Align(lipgloss.Right),
	}
}
