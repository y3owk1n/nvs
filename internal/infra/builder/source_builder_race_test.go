//nolint:testpackage // internal test: runCommandWithProgress is unexported
package builder

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// pipedCommander is a Commander for tests that exercises the full
// runCommandWithProgress pipeline (cmd.Run in a goroutine, stdout/stderr
// reader goroutines, output channel) with real pipes.
type pipedCommander struct {
	stdoutPipe *strings.Reader
	stderrPipe *strings.Reader
	runErr     error

	killCount atomic.Int32
	started   chan struct{} // closed by Run()
	holdRun   chan struct{} // blocks Run() until closed
}

func (p *pipedCommander) Run() error {
	close(p.started)

	// Block until the test closes holdRun (or Run is unblocked). This
	// lets the test simulate a long-running child that needs to be
	// killed on context cancellation.
	<-p.holdRun

	return p.runErr
}

func (p *pipedCommander) SetDir(string) {}

func (p *pipedCommander) SetStdout(any) {}

func (p *pipedCommander) SetStderr(any) {}

func (p *pipedCommander) StdoutPipe() (any, error) {
	return p.stdoutPipe, nil
}

func (p *pipedCommander) StderrPipe() (any, error) {
	return p.stderrPipe, nil
}

func (p *pipedCommander) Kill() error {
	p.killCount.Add(1)

	// Simulate the effect of the OS process dying: unblock Run() and
	// close the pipe readers so the surrounding pipeline can drain.
	select {
	case <-p.holdRun:
		// already closed
	default:
		close(p.holdRun)
	}

	return nil
}

func newPipedCommander(stdout, stderr string) *pipedCommander {
	return &pipedCommander{
		stdoutPipe: strings.NewReader(stdout),
		stderrPipe: strings.NewReader(stderr),
		started:    make(chan struct{}),
		holdRun:    make(chan struct{}),
	}
}

// TestRunCommandWithProgress_NoRaceOnLastMessage hammers
// runCommandWithProgress with many important output lines so that the
// outputChan case fires repeatedly. It must pass under -race.
//
// lastMessage is touched only in the main for-select goroutine. The
// output callback in runCommandWithSpinnerAndOutput only writes to
// outputChan, which provides happens-before synchronization. This test
// pins that invariant: if a future refactor introduces a real race on
// lastMessage or any other shared state in this pipeline, the race
// detector will report it.
func TestRunCommandWithProgress_NoRaceOnLastMessage(t *testing.T) {
	t.Parallel()

	lines := make([]string, 0, 200)

	for idx := range 200 {
		lines = append(lines, "-- Important line "+itoa(idx))
	}

	cmd := newPipedCommander(strings.Join(lines, "\n"), "")
	close(cmd.holdRun) // let Run() return immediately

	var progressCount int

	var progressMu sync.Mutex

	err := runCommandWithProgress(
		t.Context(),
		cmd,
		func(string, int) {
			progressMu.Lock()
			progressCount++
			progressMu.Unlock()
		},
		"Testing",
	)
	if err != nil {
		t.Fatalf("runCommandWithProgress returned error: %v", err)
	}

	progressMu.Lock()

	defer progressMu.Unlock()

	if progressCount == 0 {
		t.Error("expected progress callback to be invoked at least once")
	}
}

// TestRunCommandWithSpinnerAndOutput_KillsChildOnCancel is a regression
// test for the child-process leak in runCommandWithSpinnerAndOutput.
// Previously, when the context was canceled while the child was still
// running, the function returned ctx.Err() without killing the child.
// The child process kept running and the reader goroutines blocked on
// its pipes until the process tree eventually died on its own.
//
// The fix: call Kill on the commander before waitGroup.Wait() so the
// pipes close and the reader goroutines can exit.
func TestRunCommandWithSpinnerAndOutput_KillsChildOnCancel(t *testing.T) {
	t.Parallel()

	cmd := newPipedCommander("", "")

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)

	go func() {
		done <- runCommandWithSpinnerAndOutput(ctx, cmd, nil)
	}()

	// Wait for Run() to start so we know we are exercising the
	// ctx-cancel path (not a fast-finishing success path).
	select {
	case <-cmd.started:
	case <-time.After(2 * time.Second):
		t.Fatal("child Run() never started")
	}

	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected ctx.Err() after cancel, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runCommandWithSpinnerAndOutput did not return after cancel")
	}

	if got := cmd.killCount.Load(); got != 1 {
		t.Errorf("expected Kill to be called exactly once, got %d", got)
	}
}

// itoa converts a non-negative int to its base-10 string representation
// without depending on strconv (avoids an extra import in this file).
func itoa(value int) string {
	if value == 0 {
		return "0"
	}

	var buf [20]byte

	pos := len(buf)

	for value > 0 {
		pos--
		buf[pos] = byte('0' + value%10)
		value /= 10
	}

	return string(buf[pos:])
}
