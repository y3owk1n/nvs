// Package style provides the centralized design tokens (palette,
// typography, spacing) used by every other nvs UI primitive.
//
// All values are computed lazily from lipgloss's terminal
// capability detection, so NO_COLOR, dark/light background, and
// limited color terminals are handled automatically. Consumers
// should never instantiate *lipgloss.Style directly; instead, use
// the exported Style() / Palette() accessors so every primitive
// shares a single source of truth.
package style

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Palette holds the semantic colors used by the nvs UI.
//
// Values are inspired by popular editor themes (One Dark,
// Catppuccin) and tinted toward a Neovim-green primary so the
// tool feels at home in a Neovim-adjacent workflow. The exact
// values are not part of the public contract — only the semantic
// names are.
type Palette struct {
	// Primary is the brand accent. Used for the wordmark, "current"
	// markers, and the "active" state of any list item.
	Primary lipgloss.AdaptiveColor

	// Text is the default body text.
	Text lipgloss.AdaptiveColor

	// Muted is secondary text (descriptions, captions).
	Muted lipgloss.AdaptiveColor

	// Subtle is tertiary text (timestamps, hints, dimmed labels).
	Subtle lipgloss.AdaptiveColor

	// Border is the panel/box outline color.
	Border lipgloss.AdaptiveColor

	// Accent is a secondary highlight, used for cyan-ish values
	// (paths, commit hashes, hashes).
	Accent lipgloss.AdaptiveColor

	// Success is a positive outcome (install ok, switch ok).
	Success lipgloss.AdaptiveColor

	// Warning is a non-fatal issue (already up to date, missing
	// optional dep).
	Warning lipgloss.AdaptiveColor

	// Error is a fatal outcome (install failed, permission
	// denied).
	Error lipgloss.AdaptiveColor
}

// Default returns the nvs palette. It is the single source of
// truth; do not construct a Palette from literal colors outside
// of this package.
func Default() Palette {
	return Palette{
		Primary: lipgloss.AdaptiveColor{
			Light: "#3F8F2F",
			Dark:  "#80C342",
		},
		Text: lipgloss.AdaptiveColor{
			Light: "#1F2937",
			Dark:  "#E5E7EB",
		},
		Muted: lipgloss.AdaptiveColor{
			Light: "#4B5563",
			Dark:  "#9CA3AF",
		},
		Subtle: lipgloss.AdaptiveColor{
			Light: "#6B7280",
			Dark:  "#6B7280",
		},
		Border: lipgloss.AdaptiveColor{
			Light: "#D1D5DB",
			Dark:  "#374151",
		},
		Accent: lipgloss.AdaptiveColor{
			Light: "#0E7490",
			Dark:  "#56B6C2",
		},
		Success: lipgloss.AdaptiveColor{
			Light: "#15803D",
			Dark:  "#80C342",
		},
		Warning: lipgloss.AdaptiveColor{
			Light: "#B45309",
			Dark:  "#E5C07B",
		},
		Error: lipgloss.AdaptiveColor{
			Light: "#B91C1C",
			Dark:  "#E06C75",
		},
	}
}

// ColorEnabled reports whether the nvs UI should emit ANSI
// color escapes. The rule is: NO_COLOR wins, then FORCE_COLOR,
// then "is stdout a TTY".
//
// This is exposed (rather than letting lipgloss decide
// implicitly) so a non-UI caller (for example the doctor or
// install command wanting to log an unstyled summary) can use
// the same check.
func ColorEnabled() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}

	if _, ok := os.LookupEnv("FORCE_COLOR"); ok {
		return true
	}

	return lipgloss.DefaultRenderer().ColorProfile().Name() != "ascii"
}
