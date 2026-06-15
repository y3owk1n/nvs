// Package style provides the centralized design tokens (palette,
// typography, spacing) used by every other nvs UI primitive.
//
// All values are computed lazily from lipgloss's terminal
// capability detection, so NO_COLOR, dark/light background, and
// limited color terminals are handled automatically. Consumers
// should never instantiate *lipgloss.Style directly; instead, use
// the exported Style() / Palette() accessors so every primitive
// shares a single source of truth.
//
// Theming:
//
//   - Palette colors can be overridden per-slot via
//     NVS_COLOR_<NAME> (e.g. NVS_COLOR_PRIMARY). When set, the
//     value replaces BOTH the light and dark variants of that
//     slot. Use the _LIGHT / _DARK suffix to target a single
//     variant.
//   - The picker (huh) draws its colors from the same palette
//     so a single NVS_COLOR_* override cascades into the
//     picker automatically.
package style

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Palette holds the semantic colors used by the nvs UI.
//
// The default palette is the "Pastel Twilight" base16 theme:
// a deep violet twilight sky (#1f1d2e) lit by soft pastel
// accents (lavender, dusk blue, pastel mint, blush pink,
// lantern yellow). The "Dark" variant is the theme's
// intended-on-dark-background colors; the "Light" variant is
// the same palette darkened to read on light backgrounds.
//
// The exact hex values are not part of the public contract —
// only the semantic names are — so every slot can be
// overridden via an NVS_COLOR_<NAME> env var.
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

// Pastel Twilight — base16 dark variant.
//
//	#1f1d2e  base00  background (deep violet twilight sky)
//	#2a2738  base01  dim background (horizon shadows)
//	#3a364d  base02  subtle highlights (distant hills)
//	#5a5672  base03  muted gray (gentle twilight haze)
//	#9a96b5  base04  faded midtone (moonlit clouds)
//	#e0def4  base05  foreground text (moonlight glow)
//	#f0ecf9  base06  brighter text (stars shimmer)
//	#faf7ff  base07  highest contrast (paper white)
//	#f28fad  base08  red (blush pink, soft warmth of sunset)
//	#f8bd96  base09  orange (amber glow on horizon)
//	#f9e2af  base0A  yellow (gentle lantern light)
//	#abe9b3  base0B  green (pastel mint, calm renewal)
//	#b5e8e0  base0C  cyan (misty teal, serene rivers)
//	#80b8e8  base0D  blue (dusk blue, quiet skies with depth)
//	#c9a0e9  base0E  purple (lavender dreams, duskier tone)
//	#d4a4b8  base0F  mauve (fading petals, special accents)
const (
	twilightBase00 = "#1f1d2e"
	twilightBase01 = "#2a2738"
	twilightBase02 = "#3a364d"
	twilightBase03 = "#5a5672"
	twilightBase04 = "#9a96b5"
	twilightBase05 = "#e0def4"
	twilightBase08 = "#f28fad" // red
	twilightBase0A = "#f9e2af" // yellow
	twilightBase0B = "#abe9b3" // green
	twilightBase0D = "#80b8e8" // blue
	twilightBase0E = "#c9a0e9" // purple
)

// basePalette returns the hardcoded Pastel Twilight palette.
// It is the single source of truth for the default colors;
// callers should use Default() to obtain a palette with
// environment variable overrides applied.
//
// "Dark" is the intended-on-dark-background variant (the base16
// spec defines a "dark" theme). "Light" is the same palette
// darkened so it reads on light backgrounds — we pick the
// dimmer base16 colors (base00-base02) for body text / muted /
// border and darken the accent colors by roughly halving the
// RGB values, keeping the hue recognizable.
func basePalette() Palette {
	return Palette{
		Primary: lipgloss.AdaptiveColor{
			Light: "#6f4d8c", // darker lavender
			Dark:  twilightBase0E,
		},
		Text: lipgloss.AdaptiveColor{
			Light: twilightBase01, // horizon shadows
			Dark:  twilightBase05, // moonlight glow
		},
		Muted: lipgloss.AdaptiveColor{
			Light: twilightBase03, // gentle twilight haze
			Dark:  twilightBase04, // moonlit clouds
		},
		Subtle: lipgloss.AdaptiveColor{
			Light: twilightBase04, // faded midtone
			Dark:  twilightBase03, // muted gray
		},
		Border: lipgloss.AdaptiveColor{
			Light: twilightBase01, // horizon shadows
			Dark:  twilightBase02, // distant hills
		},
		Accent: lipgloss.AdaptiveColor{
			Light: "#4068a0", // darker dusk blue
			Dark:  twilightBase0D,
		},
		Success: lipgloss.AdaptiveColor{
			Light: "#5a9b65", // darker pastel mint
			Dark:  twilightBase0B,
		},
		Warning: lipgloss.AdaptiveColor{
			Light: "#b89556", // darker lantern light
			Dark:  twilightBase0A,
		},
		Error: lipgloss.AdaptiveColor{
			Light: "#b86080", // darker blush pink
			Dark:  twilightBase08,
		},
	}
}

// Default returns the nvs palette with environment variable
// overrides applied. It is the single source of truth for the
// active theme; do not construct a Palette from literal colors
// outside of this package.
//
// Each palette slot can be overridden by an NVS_COLOR_<NAME>
// environment variable, where <NAME> is the uppercased field
// name (PRIMARY, TEXT, ...). The base variable sets both the
// light and dark variants; the _LIGHT and _DARK suffixed
// variables target a single variant and take precedence over
// the base.
func Default() Palette {
	palette := basePalette()

	palette.Primary = overrideAdaptiveColor(palette.Primary, "PRIMARY")
	palette.Text = overrideAdaptiveColor(palette.Text, "TEXT")
	palette.Muted = overrideAdaptiveColor(palette.Muted, "MUTED")
	palette.Subtle = overrideAdaptiveColor(palette.Subtle, "SUBTLE")
	palette.Border = overrideAdaptiveColor(palette.Border, "BORDER")
	palette.Accent = overrideAdaptiveColor(palette.Accent, "ACCENT")
	palette.Success = overrideAdaptiveColor(palette.Success, "SUCCESS")
	palette.Warning = overrideAdaptiveColor(palette.Warning, "WARNING")
	palette.Error = overrideAdaptiveColor(palette.Error, "ERROR")

	return palette
}

// overrideAdaptiveColor returns color with the corresponding
// NVS_COLOR_<NAME> environment variable overrides applied.
//
// Override precedence (highest first):
//
//  1. NVS_COLOR_<NAME>_LIGHT (light variant only)
//  2. NVS_COLOR_<NAME>_DARK  (dark variant only)
//  3. NVS_COLOR_<NAME>        (both light and dark)
//
// Each value is validated (see isValidColor). An unset, empty,
// whitespace-only, or invalid value is silently ignored: the
// slot keeps its previous value, and a one-line warning is
// written to stderr for the invalid case (deduplicated per
// (env var, value) so a long-running command does not spam
// the same message).
func overrideAdaptiveColor(color lipgloss.AdaptiveColor, name string) lipgloss.AdaptiveColor {
	if v, ok := envColor("NVS_COLOR_" + name); ok {
		color.Light = v
		color.Dark = v
	}

	if v, ok := envColor("NVS_COLOR_" + name + "_LIGHT"); ok {
		color.Light = v
	}

	if v, ok := envColor("NVS_COLOR_" + name + "_DARK"); ok {
		color.Dark = v
	}

	return color
}

// envColor returns the validated color from envName. The
// boolean is false when the env var is unset, empty, or
// invalid; in the invalid case a one-line warning has been
// written to stderr.
func envColor(envName string) (string, bool) {
	raw := strings.TrimSpace(os.Getenv(envName))
	if raw == "" {
		return "", false
	}

	if !isValidColor(raw) {
		warnInvalidColor(envName, raw)

		return "", false
	}

	return raw, true
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
