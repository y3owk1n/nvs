package ui_test

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/briandowns/spinner"
	"github.com/y3owk1n/nvs/internal/ui"
)

func TestSafeSpinnerSetSuffixConcurrent(t *testing.T) {
	t.Parallel()

	sp := spinner.New(spinner.CharSets[14], 50*time.Millisecond)
	sp.Writer = &safeWriter{t: t}
	safe := ui.NewSafeSpinner(sp)
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

func TestSafeSpinnerSetPrefixConcurrent(t *testing.T) {
	t.Parallel()

	sp := spinner.New(spinner.CharSets[14], 50*time.Millisecond)
	sp.Writer = &safeWriter{t: t}
	safe := ui.NewSafeSpinner(sp)
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

func TestSafeSpinnerStartStopIdempotent(t *testing.T) {
	t.Parallel()

	sp := spinner.New(spinner.CharSets[14], 50*time.Millisecond)
	sp.Writer = &safeWriter{t: t}
	safe := ui.NewSafeSpinner(sp)

	safe.Start()
	safe.Stop()
	safe.Stop() // double-stop must not panic
}

// safeWriter discards spinner output during tests so we don't corrupt the
// real terminal.
type safeWriter struct {
	t *testing.T
}

func (sw *safeWriter) Write(data []byte) (int, error) {
	sw.t.Logf("spinner output suppressed (%d bytes)", len(data))

	return len(data), nil
}
