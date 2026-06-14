package ui_test

import (
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
	waitGroup.Add(writers)
	for i := range writers {
		go func(id int) {
			defer waitGroup.Done()
			for j := range iters {
				safe.SetSuffix(" writer=" + string(rune('A'+id)) + " iter=" + itoa(j))
			}
		}(i)
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
	waitGroup.Add(writers)
	for i := range writers {
		go func(id int) {
			defer waitGroup.Done()
			for j := range iters {
				safe.SetPrefix(" P" + itoa(id) + "-" + itoa(j))
			}
		}(i)
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

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}

// safeWriter discards spinner output during tests so we don't corrupt the
// real terminal.
type safeWriter struct {
	t *testing.T
}

func (sw *safeWriter) Write(p []byte) (int, error) {
	sw.t.Logf("spinner output suppressed (%d bytes)", len(p))

	return len(p), nil
}
