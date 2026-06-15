// Package banner renders the nvs wordmark and section headers.
//
// The wordmark is intentionally typographic rather than
// ASCII-art: a leading "▌" mark in the primary color, followed
// by a bold "nvs" wordmark, then a one-line tagline in the
// muted color. It pairs well with a rounded panel border
// (see ui/panel) for command summaries.
package banner

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/y3owk1n/nvs/internal/ui/style"
)

// LogoPadLeft is the left padding applied to the tagline. It
// visually aligns the tagline with the wordmark, which sits
// one cell to the right of the leading mark.
const LogoPadLeft = 2

// Logo returns the multi-line wordmark used at the top of
// high-level command output ("nvs current", "nvs doctor",
// etc.). The returned string already has a trailing newline
// so the caller can print it directly with fmt.Println or a
// single write.
//
// The design is two lines:
//
//	▌ nvs
//	  Neovim Version Switcher
//
// The leading "▌" (left half block) is a strong, single-cell
// visual anchor in the primary color. The wordmark follows in
// the same color but bold, so the eye reads "nvs" as the
// brand. The tagline is a dimmer color so the hierarchy is
// unambiguous.
func Logo(palette style.Palette) string {
	mark := lipgloss.NewStyle().
		Foreground(palette.Primary).
		Render("▌")

	wordmark := lipgloss.NewStyle().
		Bold(true).
		Foreground(palette.Primary).
		Padding(0, 0, 0, 1).
		Render("nvs")

	tagline := lipgloss.NewStyle().
		Foreground(palette.Muted).
		Padding(0, 0, 0, LogoPadLeft).
		Render("Neovim Version Switcher")

	return mark + wordmark + "\n" + tagline + "\n"
}

// Mark returns the inline "nvs" prefix used in prompts and
// interactive forms ("nvs › choose a version"). It is a single
// styled cell that callers can drop into a huh.Form title or a
// custom prompt.
func Mark(palette style.Palette) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(palette.Primary).
		Render("nvs")
}

// Header renders a section title (e.g. "Installed versions")
// with a thin underline. It is the visual equivalent of an h2
// in HTML and is what every sub-section of a command's output
// should use to introduce itself.
func Header(palette style.Palette, text string) string {
	styled := lipgloss.NewStyle().
		Bold(true).
		Foreground(palette.Text).
		Render(text)

	underline := lipgloss.NewStyle().
		Foreground(palette.Border).
		Render(repeat("─", lipgloss.Width(text)))

	return styled + "\n" + underline + "\n"
}

// repeat is a tiny helper used by Header. Inlining keeps the
// dependency surface tight and avoids an extra strings import
// for a one-line call site.
func repeat(value string, count int) string {
	if count <= 0 {
		return ""
	}

	out := make([]byte, 0, len(value)*count)

	for range count {
		out = append(out, value...)
	}

	return string(out)
}
