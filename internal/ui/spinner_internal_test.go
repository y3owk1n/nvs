package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// TestSpinnerStopBlocksUntilAnimationExited verifies the
// correctness fix for the briandowns/spinner race: once Stop
// returns, no further writes will be issued to the writer, and
// the buffer ends with the line-erase sequence Stop is
// contracted to emit.
//
// The test sets isTerminal to true directly (rather than going
// through NewSpinner) because the public constructor detects
// non-terminal writers and turns the spinner into a no-op —
// which is the correct production behavior, but defeats this
// test's ability to observe the animation goroutine. The
// internal-test access is a deliberate, narrowly-scoped escape
// hatch for verifying the lifecycle contract.
func TestSpinnerStopBlocksUntilAnimationExited(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	safe := NewSpinner(buf, 5*time.Millisecond)
	safe.isTerminal = true //nolint:exposed // test-only override
	safe.SetPrefix("prefix ")
	safe.SetSuffix("suffix")
	safe.Start()

	// Let the animation run for several tick intervals so the
	// buffer is non-trivially populated before we stop.
	time.Sleep(30 * time.Millisecond)

	safe.Stop()

	// Snapshot the buffer immediately after Stop returns. Any
	// frame written by the animation goroutine after this
	// point would be a regression of the race we are guarding
	// against. Because Stop blocks on the animation
	// goroutine's WaitGroup, the goroutine is guaranteed to
	// have exited by this point.
	snapshot := buf.String()

	// Wait a bit longer than the tick interval. If the
	// animation goroutine were still alive, it would tick
	// again and write a new frame.
	time.Sleep(30 * time.Millisecond)

	if buf.String() != snapshot {
		t.Errorf(
			"spinner wrote to writer after Stop returned:\nbefore:\n%q\nafter:\n%q",
			snapshot,
			buf.String(),
		)
	}

	// The buffer should end with the line-erase sequence
	// written by Stop. The last bytes must be "\r\033[K" —
	// that is the contract Stop exposes to callers (move to
	// column 0, clear from cursor to end of line).
	const eraseSeq = "\r\x1b[K"

	if !strings.HasSuffix(snapshot, eraseSeq) {
		t.Errorf(
			"spinner buffer does not end with the line-erase sequence %q; got tail %q",
			eraseSeq,
			tail(snapshot, len(eraseSeq)),
		)
	}
}

// TestSpinnerNoTerminalWriterNoop verifies that when the
// writer is not a terminal, the spinner is a complete no-op:
// no writes are issued, Start/Stop are safe, and the line-
// erase sequence is NOT appended to the writer.
func TestSpinnerNoTerminalWriterNoop(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	safe := NewSpinner(buf, 5*time.Millisecond)
	// isTerminal is false for *bytes.Buffer (it's not an
	// *os.File pointing at a tty). Confirm that assumption
	// holds for this test, otherwise the no-op contract is
	// not being exercised.
	if safe.isTerminal {
		t.Fatal("bytes.Buffer unexpectedly detected as a terminal")
	}

	safe.Start()
	safe.SetPrefix("prefix ")
	safe.SetSuffix("suffix")
	time.Sleep(20 * time.Millisecond)
	safe.Stop()

	if buf.Len() != 0 {
		t.Errorf(
			"spinner wrote %d bytes to a non-terminal writer; want 0: %q",
			buf.Len(),
			buf.String(),
		)
	}
}

// TestSpinnerRendersExpectedFrame verifies the per-frame output
// matches the documented format: "\r\033[K" + prefix + char +
// " " + suffix. This is the contract install/upgrade code
// relies on when reading back spinner state.
func TestSpinnerRendersExpectedFrame(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	safe := NewSpinner(buf, 5*time.Millisecond)
	safe.isTerminal = true //nolint:exposed // test-only override
	safe.SetPrefix("INFO ")
	safe.SetSuffix("working")
	safe.Start()

	// Wait for at least one frame. The tick interval is 5ms,
	// so 30ms is comfortably more than enough to guarantee
	// at least one frame has been written. We deliberately
	// avoid polling buf.Len() here — bytes.Buffer is not
	// safe for concurrent use, and the animation goroutine
	// is writing to it the whole time.
	time.Sleep(30 * time.Millisecond)

	safe.Stop()

	got := buf.String()

	// The first frame is always frame index 0 (the first
	// braille char), so the buffer must START with the
	// expected first-frame bytes regardless of how many
	// additional frames were rendered.
	want := "\r\x1b[KINFO ⠋ working"

	if !strings.HasPrefix(got, want) {
		t.Errorf("buffer prefix = %q, want %q", got, want)
	}

	// And the buffer must end with the line-erase sequence
	// Stop emits after the animation goroutine exits.
	const eraseSeq = "\r\x1b[K"

	if !strings.HasSuffix(got, eraseSeq) {
		t.Errorf(
			"buffer does not end with the line-erase sequence %q; got tail %q",
			eraseSeq,
			tail(got, len(eraseSeq)),
		)
	}
}

// TestSpinnerStartWritesFirstFrameSynchronously verifies that
// the very first frame is written to the writer BEFORE Start
// returns, without waiting for the first animation tick. This
// is what makes short-lived operations (a cache-hit fetch, a
// fast local read, ...) leave a visible loading line behind,
// instead of a Stop-time `\r\033[K` clearing a never-rendered
// line.
//
// The buffer is checked between Start and any sleep: the
// animation goroutine cannot have ticked yet at that point, so
// the only way the buffer is non-empty is if Start wrote the
// frame itself.
func TestSpinnerStartWritesFirstFrameSynchronously(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	safe := NewSpinner(buf, 50*time.Millisecond)
	safe.isTerminal = true //nolint:exposed // test-only override
	safe.SetPrefix("INFO ")
	safe.SetSuffix("working")

	safe.Start()
	// Intentionally do NOT call time.Sleep. The first
	// animation tick is at t=50ms; if we observe a frame in
	// the buffer at this point, it must have been written
	// synchronously by Start.
	if buf.Len() == 0 {
		safe.Stop()
		t.Fatal("Start returned without writing the first frame")
	}

	snapshot := buf.String()
	if !strings.HasPrefix(snapshot, "\r\x1b[KINFO ⠋ working") {
		safe.Stop()
		t.Errorf("first frame prefix = %q, want %q", snapshot, "\r\x1b[KINFO ⠋ working")
	}

	safe.Stop()
}

// tail returns the last n bytes of s as a string. If s is
// shorter than n, the whole string is returned.
func tail(s string, n int) string {
	if n > len(s) {
		n = len(s)
	}

	return s[len(s)-n:]
}
