package style

// This file exposes a couple of package-private helpers for
// the test suite. The "_test" build constraint (Go's "internal
// test helpers" convention) means these symbols are only
// visible to tests in this package — production binaries do
// not see them.
//
// Both helpers are deliberately named with the "ForTest" /
// "ResetWarnedInvalidColorsForTest" suffix so a future reader
// does not mistake them for a real public API.

// IsValidColorForTest is the test-only re-export of
// isValidColor. It is used by the validation test suite to
// drive the validator through its public contract (hex /
// named / ANSI) without having to reach into the package via
// reflection.
func IsValidColorForTest(value string) bool { return isValidColor(value) }

// ResetWarnedInvalidColorsForTest clears the deduplication
// map used by warnInvalidColor. Each call to the validation
// test suite starts with a clean slate so warnings emitted
// in one test do not silently suppress warnings in the next.
//
// Safe to call from any goroutine; uses LoadAndDelete under
// the hood.
func ResetWarnedInvalidColorsForTest() {
	warnedInvalidColors.Range(func(key, _ any) bool {
		warnedInvalidColors.Delete(key)

		return true
	})
}
