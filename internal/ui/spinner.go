package ui

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/x/term"
)

// defaultSpinnerSpeed is the animation tick used when the caller
// passes a non-positive speed to NewSpinner. 100ms is a comfortable
// balance: it animates smoothly on a terminal, but is slow enough
// to avoid busy-waiting on a log file.
const defaultSpinnerSpeed = 100 * time.Millisecond

// SpinnerChars is the default set of spinner characters. The
// braille-dot frames are a popular choice for terminal
// spinners: they animate smoothly and occupy a single terminal
// cell, so the spinner column stays put across frames.
var SpinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner is a minimal terminal spinner that rewrites a single
// line in place as its frame ticks. It is a drop-in replacement
// for the briandowns/spinner library used previously; the
// implementation is small enough to audit and avoids the
// library's lastOutputPlain race (where the animation goroutine
// can write a new frame after Stop's erase, leaving stale text
// on the line).
//
// Concurrency: SetPrefix and SetSuffix are safe to call from
// any goroutine, even while the animation loop is running. Start
// and Stop coordinate via the internal mutex; once Stop returns,
// the animation goroutine has fully exited and no further writes
// will be issued to the configured writer.
//
// Non-terminal writers: when the writer is not an *os.File
// referring to a terminal (for example, stdout is piped to a
// file), the spinner is a complete no-op. Start does nothing and
// Stop returns immediately. The caller does not need to special-
// case this — the install/upgrade code paths can always call
// Start/Stop regardless of whether the output is being captured.
type Spinner struct {
	writer     io.Writer
	chars      []string
	speed      time.Duration
	isTerminal bool

	mu      sync.Mutex
	prefix  string
	suffix  string
	done    chan struct{}
	wg      sync.WaitGroup
	started bool
}

// NewSpinner returns a Spinner that writes to writer. If writer
// is nil, os.Stdout is used. If speed is non-positive, the
// default of 100ms is used.
//
// The spinner only animates when writer is an *os.File whose
// underlying file descriptor refers to a terminal. In every
// other case (nil writer, non-file writer, pipe, redirected
// stdout, ...) the spinner becomes a no-op so the output is not
// polluted with carriage-return and ANSI clear sequences that
// would corrupt a log file or a pipe.
func NewSpinner(writer io.Writer, speed time.Duration) *Spinner {
	if writer == nil {
		writer = os.Stdout
	}

	if speed <= 0 {
		speed = defaultSpinnerSpeed
	}

	spinner := &Spinner{
		writer: writer,
		chars:  SpinnerChars,
		speed:  speed,
	}

	if file, ok := writer.(*os.File); ok {
		spinner.isTerminal = term.IsTerminal(file.Fd())
	}

	return spinner
}

// SetPrefix sets the text shown before the spinner character
// on every frame. Safe to call from any goroutine, including
// the one driving SetSuffix. The prefix is read under the
// spinner's mutex on each tick, so updates take effect on the
// next frame.
func (s *Spinner) SetPrefix(prefix string) {
	s.mu.Lock()
	s.prefix = prefix
	s.mu.Unlock()
}

// SetSuffix sets the text shown after the spinner character on
// every frame. Safe to call from any goroutine, including the
// one driving SetPrefix. The suffix is read under the spinner's
// mutex on each tick, so updates take effect on the next frame.
// This is the hook progress callbacks use to update the visible
// "Extracting [███░░░] 50%" line.
func (s *Spinner) SetSuffix(suffix string) {
	s.mu.Lock()
	s.suffix = suffix
	s.mu.Unlock()
}

// Start begins the spinner animation. If the spinner is
// already running, or the writer is not a terminal, Start is a
// no-op. Start does not block: the animation runs on a
// background goroutine.
func (s *Spinner) Start() {
	if !s.isTerminal {
		return
	}

	s.mu.Lock()
	if s.started {
		s.mu.Unlock()

		return
	}

	s.started = true
	s.done = make(chan struct{})
	s.mu.Unlock()

	s.wg.Add(1)

	go s.run()
}

// Stop ends the spinner animation and erases the line the
// spinner was drawing on. Safe to call multiple times. If the
// spinner was never started, or the writer is not a terminal,
// Stop is a no-op.
//
// Stop blocks until the animation goroutine has fully exited,
// so the line erase that follows is guaranteed to be the last
// write the spinner makes to the writer. This is the
// correctness fix for the old library: there is no window in
// which the animation goroutine can write a new frame after
// the line is cleared.
func (s *Spinner) Stop() {
	if !s.isTerminal {
		return
	}

	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()

		return
	}

	s.started = false

	close(s.done)
	s.mu.Unlock()

	s.wg.Wait()

	// Erase the spinner line. "\r" moves the cursor to column 0
	// of the current line; "\033[K" (CSI EL with argument 0)
	// erases from the cursor to the end of the line. Together
	// they clear the spinner line in place without advancing the
	// cursor, so the caller's next write starts at column 0 of
	// the freshly-emptied line — perfect for a "success" line
	// that should appear where the spinner was.
	//
	// We deliberately swallow the error: a write failure here
	// (typically a closed pipe) means the user has already
	// disconnected, and there is no caller-side recovery to do.
	//nolint:errcheck
	fmt.Fprint(s.writer, "\r\033[K")
}

// run is the animation loop. It runs until the done channel is
// closed, ticking at the configured speed and rewriting the
// spinner line on every tick.
func (s *Spinner) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.speed)
	defer ticker.Stop()

	frame := 0

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			prefix := s.prefix
			suffix := s.suffix
			s.mu.Unlock()

			// Clear the current line and write the new
			// content. The same erase sequence is used on
			// every frame, so the visible window stays a
			// single line even if the suffix changes length
			// between ticks (for example, the progress bar
			// growing from 0% to 100%).
			//
			// As in Stop, we swallow the write error: a
			// closed pipe at this point just means the
			// user is no longer watching, and the spinner
			// will exit on the next tick.
			//nolint:errcheck
			fmt.Fprintf(
				s.writer,
				"\r\033[K%s%s %s",
				prefix,
				s.chars[frame],
				suffix,
			)
			frame = (frame + 1) % len(s.chars)
		}
	}
}
