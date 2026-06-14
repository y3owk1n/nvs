package ui

import (
	"github.com/briandowns/spinner"
)

// SafeSpinner wraps *spinner.Spinner and synchronizes updates to its Suffix
// using the spinner's own internal mutex. The spinner reads Suffix on its
// animation goroutine under the same mutex (see briandowns/spinner spin
// loop), so all external writes must be serialized through Lock/Unlock to
// avoid data races.
type SafeSpinner struct {
	sp *spinner.Spinner
}

// NewSafeSpinner returns a SafeSpinner backed by the given spinner.
func NewSafeSpinner(sp *spinner.Spinner) *SafeSpinner {
	return &SafeSpinner{sp: sp}
}

// Start begins the spinner animation.
func (s *SafeSpinner) Start() {
	s.sp.Start()
}

// Stop ends the spinner animation. Safe to call multiple times.
func (s *SafeSpinner) Stop() {
	s.sp.Stop()
}

// SetSuffix updates the spinner's suffix text in a race-free manner.
func (s *SafeSpinner) SetSuffix(suffix string) {
	s.sp.Lock()
	s.sp.Suffix = suffix
	s.sp.Unlock()
}

// SetPrefix updates the spinner's prefix text in a race-free manner.
func (s *SafeSpinner) SetPrefix(prefix string) {
	s.sp.Lock()
	s.sp.Prefix = prefix
	s.sp.Unlock()
}
