//nolint:testpackage
package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// resetEnvValidationState clears the package-private dedup
// map used by the validation warnings so each test starts
// with a clean slate. The exported helper keeps the variable
// itself unexported and avoids leaking the implementation
// detail.
func resetEnvValidationState(t *testing.T) {
	t.Helper()

	envValidation.Range(func(key, _ any) bool {
		envValidation.Delete(key)

		return true
	})
}

// captureStderr redirects os.Stderr to a pipe for the duration
// of fn and returns whatever was written. The implementation
// lives in captureStderrImpl so the wrapper here is a single
// call: the body inside captureStderrImpl uses one statement
// per "concern" (pipe setup, redirect, drain, restore) and
// wsl_v5 is happy with the smaller per-block surface.
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

func TestValidPathAccepts(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"absolute_unix", "/Users/example/.nvs"},
		{"absolute_windows", `C:\Users\example\.nvs`},
		{"relative", "./.nvs"},
		{"with_spaces", "/Users/example/My Config/nvs"},
		{"with_unicode", "/Users/liexample/config"},
		{"trimmed_whitespace_around", "  /tmp/nvs  "},
		{"trailing_slash", "/tmp/nvs/"},
		{"home_expansion_not_resolved", "~/nvs"},
		{"dot_segment", "/tmp/../nvs"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, accepted := validPath("TEST", testCase.input)
			if !accepted {
				t.Errorf("validPath(%q) = rejected, want accepted", testCase.input)
			}

			if got == "" {
				t.Errorf("validPath(%q) = empty string with accepted=true", testCase.input)
			}
		})
	}
}

func TestValidPathRejects(t *testing.T) {
	// Two classes of input:
	//
	//   1. Empty / whitespace-only — silently rejected
	//      (return false) because the caller falls back to
	//      the default and a warning would be noise on every
	//      "unset env var" startup.
	//   2. Anything that contains a control character or
	//      null byte — rejected AND warned, because the user
	//      almost certainly intended a real path and the
	//      control char is a paste / shell quoting bug.
	silent := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"whitespace_only", "   "},
		{"tab_only", "\t"},
		{"newline_only", "\n"},
	}

	noisy := []struct {
		name  string
		input string
	}{
		{"null_byte", "/tmp/nvs\x00/etc"},
		{"bell_char", "/tmp/nvs\x07"},
		{"escape_seq", "/tmp/nvs\x1b[31m"},
		{"embedded_newline", "/tmp/nvs\nrm -rf /"},
		{"embedded_tab", "/tmp/nvs\tbad"},
		{"del_char", "/tmp/nvs\x7f"},
	}

	for _, testCase := range silent {
		t.Run("silent/"+testCase.name, func(t *testing.T) {
			resetEnvValidationState(t)

			warning := captureStderr(t, func() {
				got, accepted := validPath("NVS_TEST_PATH", testCase.input)
				if accepted {
					t.Errorf("validPath(%q) = accepted (%q), want rejected", testCase.input, got)
				}
			})

			if warning != "" {
				t.Errorf("expected no warning for %q, got %q", testCase.input, warning)
			}
		})
	}

	for _, testCase := range noisy {
		t.Run("noisy/"+testCase.name, func(t *testing.T) {
			resetEnvValidationState(t)

			var got string

			var accepted bool

			warning := captureStderr(t, func() {
				got, accepted = validPath("NVS_TEST_PATH", testCase.input)
			})

			if accepted {
				t.Errorf("validPath(%q) = accepted (%q), want rejected", testCase.input, got)
			}

			if !strings.Contains(warning, "NVS_TEST_PATH") {
				t.Errorf(
					"warning missing env var name for input %q; got %q",
					testCase.input,
					warning,
				)
			}
		})
	}
}

func TestValidPathWarningIsDeduplicated(t *testing.T) {
	resetEnvValidationState(t)

	warning := captureStderr(t, func() {
		_, _ = validPath("NVS_DEDUP", "/bad\x00path")
		_, _ = validPath("NVS_DEDUP", "/bad\x00path")
		_, _ = validPath("NVS_DEDUP", "/bad\x00path")
	})

	if got := strings.Count(warning, "NVS_DEDUP"); got != 1 {
		t.Errorf("warning appeared %d times, want 1 (dedup failed); warning=%q", got, warning)
	}
}

func TestParseBoolEnvAccepts(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"true", envTrue, true},
		{"True", "True", true},
		{"TRUE", "TRUE", true},
		{"yes", envYes, true},
		{"Yes", "Yes", true},
		{"on", "on", true},
		{"ON", "ON", true},
		{"one", "1", true},
		{"false", envFalse, false},
		{"FALSE", "FALSE", false},
		{"no", "no", false},
		{"off", envOff, false},
		{"zero", "0", false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, set := parseBoolEnv("TEST", testCase.input)
			if !set {
				t.Errorf("parseBoolEnv(%q) = !set, want set", testCase.input)
			}

			if got != testCase.want {
				t.Errorf("parseBoolEnv(%q) = %v, want %v", testCase.input, got, testCase.want)
			}
		})
	}
}

func TestParseBoolEnvRejectsUnset(t *testing.T) {
	resetEnvValidationState(t)

	got, set := parseBoolEnv("NVS_TEST", "")

	if set {
		t.Errorf("parseBoolEnv(\"\") = set, want !set")
	}

	if got {
		t.Errorf("parseBoolEnv(\"\") = true, want false")
	}
}

func TestParseBoolEnvWarnsOnTypo(t *testing.T) {
	resetEnvValidationState(t)

	warning := captureStderr(t, func() {
		_, set := parseBoolEnv("NVS_TEST", "ture")
		if set {
			t.Error("parseBoolEnv(\"ture\") = set, want !set (typo should be rejected)")
		}
	})

	if !strings.Contains(warning, "NVS_TEST") {
		t.Errorf("warning missing env var name; got %q", warning)
	}

	if !strings.Contains(warning, "ture") {
		t.Errorf("warning missing offending value; got %q", warning)
	}
}

func TestParseBoolEnvDeduplicates(t *testing.T) {
	resetEnvValidationState(t)

	warning := captureStderr(t, func() {
		_, _ = parseBoolEnv("NVS_DEDUP_BOOL", "ture")
		_, _ = parseBoolEnv("NVS_DEDUP_BOOL", "ture")
		_, _ = parseBoolEnv("NVS_DEDUP_BOOL", "ture")
	})

	if got := strings.Count(warning, "NVS_DEDUP_BOOL"); got != 1 {
		t.Errorf("warning appeared %d times, want 1 (dedup failed); warning=%q", got, warning)
	}
}
