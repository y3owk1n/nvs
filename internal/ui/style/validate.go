package style

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// hexColorPattern matches a 3-, 6-, or 8-digit hex color,
// optionally prefixed with "#". The 8-digit form is RGBA and
// is the form lipgloss/termenv expects for alpha-aware
// colors. Case-insensitive.
//
// Examples that match: #abc, abc, #ABCDEF, ABCDEF, #abcdef12.
// Examples that do NOT match: #abcd (4 digits), #abcdefg (bad
// chars), "", "  ".
var hexColorPattern = regexp.MustCompile(`^#?([0-9a-fA-F]{3}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)

// namedColors is the set of 16-color names that lipgloss/termenv
// recognize out of the box. Listed explicitly here (rather than
// scanning the lipgloss API) so a user gets a clear error for
// an unknown name instead of "rendered as empty". Lookup is
// case-insensitive.
var namedColors = map[string]struct{}{
	"black":   {},
	"red":     {},
	"green":   {},
	"yellow":  {},
	"blue":    {},
	"magenta": {},
	"cyan":    {},
	"white":   {},
	"gray":    {},
	"grey":    {},
}

// isValidColor reports whether value is a color spec that
// lipgloss/termenv will render as something other than empty.
//
// Accepted forms:
//
//   - Hex:    #abc, #abcdef, #abcdef12 (with or without "#")
//   - Name:   black, red, green, yellow, blue, magenta, cyan,
//     white, gray, grey (case-insensitive)
//   - ANSI:   "0".."255" (the 256-color palette)
//
// Anything else (empty, whitespace, control characters, typos,
// unknown names, out-of-range numbers) returns false and the
// caller is expected to fall back to the default and warn the
// user.
//
// The check is intentionally permissive about hex format
// because that is what users actually paste from a color
// picker, and strict about named colors because typos in
// those are the most common silent-failure mode.
//
// Numeric values are checked as ANSI 256 first so "256" is
// rejected as out-of-range rather than silently accepted as
// a 3-digit hex color (#225566).
func isValidColor(value string) bool {
	lower := strings.ToLower(value)

	// ANSI 256: a bare integer in 0..255. strconv.Atoi is the
	// natural gate — it rejects " 7", "07 ", and any non-digit
	// (so "abc" or "0x10" don't slip through as hex).
	n, err := strconv.Atoi(lower)
	if err == nil {
		return n >= 0 && n <= 255
	}

	if hexColorPattern.MatchString(value) {
		return true
	}

	if _, ok := namedColors[lower]; ok {
		return true
	}

	return false
}

// warnedInvalidColors deduplicates the per-(env var, value)
// warning so a user who sets NVS_COLOR_PRIMARY=garbage and then
// runs a long command that touches the palette many times
// (e.g. a spinner that re-styles each frame) only sees the
// warning once per distinct value. The map is package-private;
// resetting it in tests is done by swapping in a fresh map.
var warnedInvalidColors sync.Map

// warnInvalidColor writes a one-line warning to stderr saying
// that the given env var was set to an invalid color and the
// default is being used. The warning is deduplicated by
// (envVar, value) so it surfaces the first time and is silent
// after.
//
// Writes go straight to stderr (not via the dev log) because
// this can fire before log.Init() has been called (the style
// package is consumed by the logger itself) and the user
// almost always wants to see the warning immediately, not
// after a log-level filter.
func warnInvalidColor(envVar, value string) {
	key := envVar + "\x00" + value
	if _, already := warnedInvalidColors.LoadOrStore(key, struct{}{}); already {
		return
	}

	fmt.Fprintf(
		os.Stderr,
		"nvs: %s=%q is not a valid color (expected #abc, #abcdef, #abcdef12, a named color like \"red\", or 0-255); using default\n",
		envVar,
		value,
	)
}
