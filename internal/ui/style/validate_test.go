package style_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/y3owk1n/nvs/internal/ui/style"
)

// resetColorValidationState clears the package-private dedup
// state used by warnInvalidColor so each test starts with a
// clean slate. Must NOT be called in parallel with other
// tests that exercise the warning path.
//
// We reach into the package via a tiny exported test hook
// (resetWarnedInvalidColorsForTest) rather than the
// warnedInvalidColors variable directly so the variable stays
// unexported and the public surface does not leak the
// implementation detail.
func resetColorValidationState(t *testing.T) {
	t.Helper()
	style.ResetWarnedInvalidColorsForTest()
}

// captureStderr redirects os.Stderr to a pipe for the duration
// of body and returns whatever was written. The
// redirect-via-pipe + drain-on-goroutine pattern would trip
// wsl_v5's "invalid statement above assign" rule on every
// line that follows a syscall-style expression, so the body
// lives in captureStderrImpl and the wrapper here is a
// single call. See the corresponding helper in
// cmd/envvar_test.go for the same pattern.
func captureStderr(t *testing.T, body func()) string {
	t.Helper()

	return captureStderrImpl(body)
}

func captureStderrImpl(body func()) string {
	original := os.Stderr

	reader, writer, pipeErr := os.Pipe()
	if pipeErr != nil {
		panic(pipeErr)
	}

	var buf bytes.Buffer
	// Single-use buffer consumed by the goroutine; we drain
	// the pipe on a goroutine and signal completion via
	// done so the main test can wait for the copy to finish
	// before reading buf.
	done := make(chan struct{})

	go drainStderr(reader, &buf, done)

	os.Stderr = writer

	body()
	// Close writer so the goroutine sees EOF and io.Copy
	// returns. (The double-close in the defer is a no-op.)
	_ = writer.Close()

	<-done

	os.Stderr = original

	return buf.String()
}

// drainStderr copies everything from reader into buf until
// reader sees EOF (after the writer is closed), then signals
// done. Pulled out of captureStderrImpl to keep the
// goroutine-launching statement on its own line, which wsl_v5
// treats as a "first statement in a block" and accepts without
// complaint.
func drainStderr(reader *os.File, buf *bytes.Buffer, done chan<- struct{}) {
	_, _ = io.Copy(buf, reader)

	close(done)
}

func TestIsValidColorHex(t *testing.T) {
	resetColorValidationState(t)

	cases := []struct {
		name  string
		value string
		want  bool
	}{
		// 3-digit hex, with and without "#".
		{"hash_3_lower", "#abc", true},
		{"no_hash_3_lower", "abc", true},
		{"hash_3_upper", "#ABC", true},
		{"hash_3_mixed", "#aBc", true},

		// 6-digit hex.
		{"hash_6", "#abcdef", true},
		{"no_hash_6", "abcdef", true},

		// 8-digit hex (RGBA, used by lipgloss for alpha-aware
		// colors).
		{"hash_8", "#abcdef12", true},
		{"no_hash_8", "abcdef12", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := style.IsValidColorForTest(c.value); got != c.want {
				t.Errorf("IsValidColor(%q) = %v, want %v", c.value, got, c.want)
			}
		})
	}
}

func TestIsValidColorNamed(t *testing.T) {
	resetColorValidationState(t)

	cases := []struct {
		name  string
		value string
		want  bool
	}{
		{"red_lower", "red", true},
		{"red_upper", "RED", true},
		{"red_title", "Red", true},
		{"grey_alt", "grey", true},
		{"gray_alt", "gray", true},
		{"unknown_name", "chartreuse", false},
		{"typo", "gren", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := style.IsValidColorForTest(c.value); got != c.want {
				t.Errorf("IsValidColor(%q) = %v, want %v", c.value, got, c.want)
			}
		})
	}
}

func TestIsValidColorANSI(t *testing.T) {
	resetColorValidationState(t)

	cases := []struct {
		name  string
		value string
		want  bool
	}{
		{"zero", "0", true},
		{"mid", "128", true},
		{"max", "255", true},
		{"negative", "-1", false},
		{"too_high", "256", false},
		{"out_of_range_3digit", "999", false},
		{"with_leading_zero", "007", true},
		{"with_whitespace", "  7", false},
		// "abc" is a valid 3-digit hex color, NOT a numeric
		// ANSI value — and that is fine: numeric ANSI is
		// checked first, but "abc" fails strconv.Atoi and
		// then falls through to the hex check, which accepts
		// it. The two form-families only collide for strings
		// that happen to be all-hex AND parseable as
		// integers (e.g. "0a0", "123"); those are intentionally
		// treated as numeric so "256" is rejected as
		// out-of-range rather than accepted as #225566.
		{"hex_string_abc", "abc", true},
		{"hex_string_123", "123", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := style.IsValidColorForTest(c.value); got != c.want {
				t.Errorf("IsValidColor(%q) = %v, want %v", c.value, got, c.want)
			}
		})
	}
}

func TestIsValidColorRejectsGarbage(t *testing.T) {
	resetColorValidationState(t)

	cases := []struct {
		name  string
		value string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"only_hash", "#"},
		{"hash_4", "#abcd"},
		{"hash_5", "#abcde"},
		{"hash_7", "#abcdefg"},
		{"non_hex_char", "#xyzxyz"},
		{"control_char", "\x1b[31m"},
		{"shell_injection", "; rm -rf /"},
		{"path", "/etc/passwd"},
		{"with_newline", "red\n"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := style.IsValidColorForTest(c.value); got {
				t.Errorf("IsValidColor(%q) = true, want false", c.value)
			}
		})
	}
}

func TestInvalidEnvColorWarnsAndKeepsDefault(t *testing.T) {
	resetColorValidationState(t)
	unsetColorEnv(t)

	// Snapshot the default primary before the override.
	want := style.Default().Primary.Dark

	// An invalid value must (a) NOT change the resolved color
	// and (b) write a warning to stderr. We capture stderr in
	// the call to Default so the warning fires while the
	// pipe is hooked.
	var gotPrimary string

	warning := captureStderr(t, func() {
		t.Setenv("NVS_COLOR_PRIMARY", "chartreuse")

		gotPrimary = style.Default().Primary.Dark
	})

	if gotPrimary != want {
		t.Errorf(
			"Primary.Dark = %q, want %q (default preserved on invalid override)",
			gotPrimary,
			want,
		)
	}

	if !strings.Contains(warning, "NVS_COLOR_PRIMARY") {
		t.Errorf("warning missing env var name; got %q", warning)
	}

	if !strings.Contains(warning, "chartreuse") {
		t.Errorf("warning missing offending value; got %q", warning)
	}
}

func TestInvalidColorWarningIsDeduplicated(t *testing.T) {
	resetColorValidationState(t)
	unsetColorEnv(t)

	t.Setenv("NVS_COLOR_PRIMARY", "chartreuse")

	// First call should warn; subsequent calls with the same
	// (env, value) pair should be silent.
	warning := captureStderr(t, func() {
		_ = style.Default()
		_ = style.Default()
		_ = style.Default()
	})

	// Exactly one "NVS_COLOR_PRIMARY" mention.
	if got := strings.Count(warning, "NVS_COLOR_PRIMARY"); got != 1 {
		t.Errorf("warning appeared %d times, want 1 (dedup failed); warning=%q", got, warning)
	}
}

func TestInvalidColorWarningRecoversWhenValueChanges(t *testing.T) {
	resetColorValidationState(t)
	unsetColorEnv(t)

	warning := captureStderr(t, func() {
		t.Setenv("NVS_COLOR_PRIMARY", "chartreuse")

		_ = style.Default()

		// Changing the value should produce a fresh warning
		// for the new value (the dedup key is (env, value)).
		t.Setenv("NVS_COLOR_PRIMARY", "typo2")

		_ = style.Default()
	})

	if got := strings.Count(warning, "NVS_COLOR_PRIMARY"); got != 2 {
		t.Errorf(
			"warning appeared %d times, want 2 (one per distinct value); warning=%q",
			got,
			warning,
		)
	}
}
