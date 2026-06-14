//nolint:testpackage // internal test: runCommandWithProgress is unexported
package builder

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// pipedCommander is a Commander for tests that exercises the full
// runCommandWithProgress pipeline (cmd.Run in a goroutine, stdout/stderr
// reader goroutines, output channel) with real pipes.
type pipedCommander struct {
	stdoutPipe *strings.Reader
	stderrPipe *strings.Reader
	runErr     error
}

func (p *pipedCommander) Run() error { return p.runErr }

func (p *pipedCommander) SetDir(string) {}

func (p *pipedCommander) SetStdout(any) {}

func (p *pipedCommander) SetStderr(any) {}

func (p *pipedCommander) StdoutPipe() (any, error) {
	return p.stdoutPipe, nil
}

func (p *pipedCommander) StderrPipe() (any, error) {
	return p.stderrPipe, nil
}

func newPipedCommander(stdout, stderr string) *pipedCommander {
	return &pipedCommander{
		stdoutPipe: strings.NewReader(stdout),
		stderrPipe: strings.NewReader(stderr),
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
		lines = append(lines, fmt.Sprintf("-- Important line %d", idx))
	}

	cmd := newPipedCommander(strings.Join(lines, "\n"), "")

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
