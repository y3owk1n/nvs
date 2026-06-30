package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// envValidation collects per-(env var, value) deduplication
// state for the validation warnings emitted by the helpers in
// this file. The map is process-global because each env var
// is read once in InitConfig and we want the same warning to
// surface exactly once even if a future caller re-reads the
// env vars.
//
// The map is package-private; tests reset it through
// resetEnvValidationForTest (exported from envvar_test.go).
var envValidation sync.Map

// validPath returns a sanitized path and a boolean indicating
// whether the value is acceptable as a directory or file path
// for one of the NVS_*_DIR / NVS_LOG_FILE variables.
//
// envName is the name of the env var (e.g. "NVS_CONFIG_DIR").
// It is included in the warning message when the value is
// rejected, so the user immediately knows which var to fix.
//
// The check is deliberately minimal:
//
//   - empty (after TrimSpace) → ("", false) so the caller
//     falls back to the default
//   - contains a null byte → ("", false) because the OS would
//     reject the path with EINVAL anyway, but here we can give
//     a better error
//   - contains any other control character → ("", false)
//     because it almost always indicates a paste / shell
//     quoting bug
//   - otherwise → (trimmed, true); the raw value is passed
//     through and the caller runs filepath.Clean / MkdirAll /
//     OpenFile on it, which is the right place to surface
//     "too long" or "permission denied" errors with the
//     user's actual path in the message.
func validPath(envName, value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}

	if strings.ContainsRune(trimmed, 0) {
		warnInvalidPath(envName, value, "contains a null byte")

		return "", false
	}

	if hasControlChar(trimmed) {
		warnInvalidPath(envName, value, "contains a control character")

		return "", false
	}

	return trimmed, true
}

// hasControlChar reports whether s contains any byte in the
// 0x01..0x1F range (i.e. any ASCII control char except NUL,
// which is caught separately). Tab and newline are excluded
// from the check because shells frequently produce paths with
// embedded newlines from broken quoting; rejecting those here
// gives a clearer error than the OS would.
func hasControlChar(s string) bool {
	for idx := range len(s) {
		b := s[idx]

		if b < 0x20 || b == 0x7F {
			return true
		}
	}

	return false
}

const (
	envTrue  = "true"
	envFalse = "false"
	envYes   = "yes"
	envOff   = "off"
)

// parseBoolEnv parses a boolean env var value. It returns the
// resolved boolean and a "set" flag; the flag is false when
// the env var is unset, empty, or invalid. Invalid values
// (typos, the empty string, or anything not in the recognized
// set) emit a one-line warning to stderr and resolve to
// false, matching the existing lenient behavior of
// NVS_USE_GLOBAL_CACHE.
//
// Recognized true values (case-insensitive): "1", "true",
// "yes", "on". Recognized false values: "0", "false", "no",
// "off". Anything else is treated as invalid.
func parseBoolEnv(envName, value string) (bool, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false, false
	}

	switch strings.ToLower(trimmed) {
	case "1", envTrue, envYes, "on":
		return true, true

	case "0", envFalse, "no", envOff:
		return false, true

	default:
		warnInvalidBool(envName, trimmed)

		return false, false
	}
}

// warnInvalidPath writes a one-line warning to stderr about a
// path env var that was set to something the validator
// rejected. Deduped by (env var, value) so the same bad value
// only generates one warning.
func warnInvalidPath(envName, value, reason string) {
	key := envName + "\x00path\x00" + value
	if _, already := envValidation.LoadOrStore(key, struct{}{}); already {
		return
	}

	fmt.Fprintf(
		os.Stderr,
		"nvs: %s=%q is not a valid path (%s); using default\n",
		envName, value, reason,
	)
}

// warnInvalidBool writes a one-line warning to stderr about a
// boolean env var that was set to a value outside the
// recognized set. Deduped by (env var, value).
func warnInvalidBool(envName, value string) {
	key := envName + "\x00bool\x00" + value
	if _, already := envValidation.LoadOrStore(key, struct{}{}); already {
		return
	}

	fmt.Fprintf(
		os.Stderr,
		"nvs: %s=%q is not a recognized boolean (expected 1/true/yes/on or 0/false/no/off); using false\n",
		envName,
		value,
	)
}
