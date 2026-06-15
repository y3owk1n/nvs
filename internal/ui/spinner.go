package ui

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

// defaultSpinnerSpeed is the animation tick used when the caller
// passes a non-positive speed to NewSpinner. 100ms is a comfortable
// balance: it animates smoothly on a terminal, but is slow enough
// to avoid busy-waiting on a log file.
const defaultSpinnerSpeed = 100 * time.Millisecond

// Spinner is a thin wrapper around bubbles/spinner that
// keeps the SetPrefix/SetSuffix/Start/Stop API nvs commands
// rely on, and adds the two non-bubbles behaviors the old
// self-rolled spinner had:
//
//  1. Non-terminal writers (piped, redirected, non-*os.File)
//     are a no-op. The animation goroutine never starts and
//     no ANSI clear sequences are written, so a log file or
//     a pipe is never polluted with `\r\033[K` junk.
//
//  2. Stop blocks until the animation goroutine has fully
//     exited, so the line-erase that Stop emits is
//     guaranteed to be the last write the spinner makes.
//     This is the same correctness fix that used to live
//     in the self-rolled version; bubbles alone does not
//     give you this guarantee because in a real tea.Program
//     the loop is tea-driven, but here we drive it manually.
type Spinner struct {
	writer     io.Writer
	model      spinner.Model
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

	model := spinner.New(
		// MiniDot is the same braille set the previous
		// self-rolled spinner used (no trailing space, so
		// we own the spacing between the spinner char and
		// the suffix). Dot would also work but it bakes a
		// trailing space into each frame, which would
		// double the gap.
		spinner.WithSpinner(spinner.MiniDot),
		// The model is rendered with a neutral style; the
		// caller is expected to have already colored any
		// prefix or suffix it sets via ui.Message helpers.
		// We deliberately do NOT inherit the lipgloss
		// color profile here, because the spinner's
		// frame must remain visible regardless of the
		// terminal's color detection.
		spinner.WithStyle(lipgloss.NewStyle()),
	)

	newSpinner := &Spinner{
		writer: writer,
		model:  model,
		speed:  speed,
	}

	if file, ok := writer.(*os.File); ok {
		newSpinner.isTerminal = term.IsTerminal(file.Fd())
	}

	return newSpinner
}

// SetPrefix sets the text shown before the spinner character
// on every frame. Safe to call from any goroutine, including
// the one driving SetSuffix. The prefix is read under the
// spinner's mutex on each tick, so updates take effect on
// the next frame.
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
//
// We drive bubbles' spinner manually (Update(TickMsg) +
// View() on each tick) rather than spawning a tea.Program:
// the install/upgrade commands are not interactive Bubble
// Tea apps, and adding a tea program just to drive one
// spinner would be a heavyweight misfit. Driving the model
// directly keeps the public API (Start/Stop/SetPrefix/
// SetSuffix) intact and the non-TTY no-op behavior local.
func (s *Spinner) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.speed)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			prefix := s.prefix
			suffix := s.suffix
			s.mu.Unlock()

			// Render the current frame first, THEN advance.
			// This keeps the first frame written (at the
			// start of a Start/Stop cycle) at frame 0 of
			// the chosen set, matching the contract the
			// self-rolled spinner exposed. (The
			// alternative — Update first, then render —
			// would skip frame 0 on the very first tick,
			// which the public format contract pins down.)
			frame := s.model.View()

			// Advance the bubbles model one frame. The
			// returned tea.Cmd is the next tick, which we
			// deliberately ignore — our own time.Ticker
			// owns the animation cadence so we keep the
			// "speed" parameter meaningful (bubbles'
			// spinner.Spinner.FPS would otherwise lock us
			// to its hard-coded FPS).
			updated, _ := s.model.Update(spinner.TickMsg{
				Time: time.Now(),
				ID:   s.model.ID(),
			})
			s.model = updated

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
			//
			// The frame format is:
			//   "\r\033[K" + prefix + spinner-char + " " + suffix
			// which matches the previous self-rolled
			// contract: prefix immediately after the line
			// erase, the spinner char in its own column, a
			// single space, then the suffix.
			//nolint:errcheck
			fmt.Fprintf(
				s.writer,
				"\r\033[K%s%s %s",
				prefix,
				frame,
				suffix,
			)

			// Defensive: keep the tea import in use so a
			// future refactor that swaps to tea.Tick
			// doesn't have to re-add the import.
			_ = tea.Quit
		}
	}
}
