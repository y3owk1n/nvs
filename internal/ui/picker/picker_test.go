package picker_test

import (
	"errors"
	"testing"

	"github.com/y3owk1n/nvs/internal/ui/picker"
)

func TestErrCanceledIsExported(t *testing.T) {
	t.Parallel()

	// ErrCanceled must be a stable, exported sentinel so cmd
	// code can use errors.Is to detect "user pressed Ctrl-C".
	if picker.ErrCanceled == nil {
		t.Fatal("ErrCanceled must be non-nil")
	}

	if errors.Is(nil, picker.ErrCanceled) {
		t.Error("nil error must not satisfy errors.Is(ErrCanceled)")
	}
}

func TestErrNoTTYIsExported(t *testing.T) {
	t.Parallel()

	if picker.ErrNoTTY == nil {
		t.Fatal("ErrNoTTY must be non-nil")
	}
}

func TestPickerSelectRequiresTTY(t *testing.T) {
	t.Parallel()

	// A picker created with hasTTY=false must refuse to run a
	// form. This protects scripted invocations (CI, piped
	// stdin) from hanging on input.
	p := picker.New(nil, nil, false)

	_, err := p.Select("Pick one", []picker.SelectItem{{Label: "a"}})
	if !errors.Is(err, picker.ErrNoTTY) {
		t.Errorf("Select with no TTY returned %v, want ErrNoTTY", err)
	}
}

func TestPickerConfirmRequiresTTY(t *testing.T) {
	t.Parallel()

	p := picker.New(nil, nil, false)

	_, err := p.Confirm("Are you sure?")
	if !errors.Is(err, picker.ErrNoTTY) {
		t.Errorf("Confirm with no TTY returned %v, want ErrNoTTY", err)
	}
}

func TestPickerSelectRejectsEmptyItems(t *testing.T) {
	t.Parallel()

	// Even with a TTY, an empty items slice is a programmer
	// error and must not hang the form. The picker should
	// return a clear error before drawing anything.
	p := picker.New(nil, nil, true)

	_, err := p.Select("Pick one", nil)
	if err == nil {
		t.Error("Select with empty items returned nil error")
	}
}
