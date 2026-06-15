package panel_test

import (
	"strings"
	"testing"

	"github.com/y3owk1n/nvs/internal/ui/panel"
	"github.com/y3owk1n/nvs/internal/ui/style"
)

func TestPanelWrapsContent(t *testing.T) {
	t.Parallel()

	got := panel.Panel(style.Default(), "hello world")

	if !strings.Contains(got, "hello world") {
		t.Errorf("Panel() = %q, missing content", got)
	}

	// The default rounded border uses rounded corners on
	// modern terminals. Confirm at least one border glyph is
	// present so we know the box actually rendered.
	if !strings.Contains(got, "╭") && !strings.Contains(got, "+") {
		t.Errorf("Panel() = %q, no border glyph found", got)
	}
}

func TestSectionIncludesTitle(t *testing.T) {
	t.Parallel()

	got := panel.Section(style.Default(), "Installed versions", "stable\nnightly")

	if !strings.Contains(got, "Installed versions") {
		t.Errorf("Section() = %q, missing title", got)
	}

	if !strings.Contains(got, "stable") {
		t.Errorf("Section() = %q, missing content", got)
	}
}
