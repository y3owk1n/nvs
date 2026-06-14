package ui_test

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/y3owk1n/nvs/internal/ui"
)

func TestSpinnerSetSuffixConcurrent(t *testing.T) {
	t.Parallel()

	safe := ui.NewSpinner(&safeWriter{t: t}, 50*time.Millisecond)
	safe.Start()
	t.Cleanup(safe.Stop)

	const writers = 8

	const iters = 200

	var waitGroup sync.WaitGroup

	for idx := range writers {
		waitGroup.Go(func() {
			for tick := range iters {
				safe.SetSuffix(
					" writer=" + string(rune('A'+idx)) + " iter=" + strconv.Itoa(tick),
				)
			}
		})
	}

	waitGroup.Wait()
}

func TestSpinnerSetPrefixConcurrent(t *testing.T) {
	t.Parallel()

	safe := ui.NewSpinner(&safeWriter{t: t}, 50*time.Millisecond)
	safe.Start()
	t.Cleanup(safe.Stop)

	const writers = 4

	const iters = 100

	var waitGroup sync.WaitGroup

	for idx := range writers {
		waitGroup.Go(func() {
			for tick := range iters {
				safe.SetPrefix(" P" + strconv.Itoa(idx) + "-" + strconv.Itoa(tick))
			}
		})
	}

	waitGroup.Wait()
}

func TestSpinnerStartStopIdempotent(t *testing.T) {
	t.Parallel()

	safe := ui.NewSpinner(&safeWriter{t: t}, 20*time.Millisecond)

	safe.Start()
	safe.Stop()
	safe.Stop() // double-stop must not panic
}

func TestSpinnerStopWithoutStart(t *testing.T) {
	t.Parallel()

	safe := ui.NewSpinner(&safeWriter{t: t}, 20*time.Millisecond)

	// Stop on a spinner that was never started must be a
	// no-op, not a panic.
	safe.Stop()
}

// safeWriter discards spinner output during tests so we don't
// corrupt the real terminal. It also satisfies io.Writer.
type safeWriter struct {
	t *testing.T
}

func (sw *safeWriter) Write(data []byte) (int, error) {
	sw.t.Logf("spinner output suppressed (%d bytes)", len(data))

	return len(data), nil
}
