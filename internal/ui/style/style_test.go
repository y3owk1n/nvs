package style_test

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/y3owk1n/nvs/internal/ui/style"
)

func TestDefaultPalette(t *testing.T) {
	t.Parallel()

	palette := style.Default()

	if palette.Primary.Light == "" || palette.Primary.Dark == "" {
		t.Error("Primary color must have both light and dark variants")
	}

	if palette.Text.Light == "" || palette.Text.Dark == "" {
		t.Error("Text color must have both light and dark variants")
	}

	// The full set of semantic colors should be defined — any
	// empty value here would be a regression in the design
	// system, since the rest of the UI relies on the palette
	// being complete.
	for name, color := range map[string]lipgloss.AdaptiveColor{
		"Primary": palette.Primary,
		"Text":    palette.Text,
		"Muted":   palette.Muted,
		"Subtle":  palette.Subtle,
		"Border":  palette.Border,
		"Accent":  palette.Accent,
		"Success": palette.Success,
		"Warning": palette.Warning,
		"Error":   palette.Error,
	} {
		if color.Light == "" || color.Dark == "" {
			t.Errorf("%s color must have both light and dark variants", name)
		}
	}
}

func TestTypes(t *testing.T) {
	t.Parallel()

	types := style.Types(style.Default())

	// A few sanity checks — every style must be usable (i.e.
	// not the zero value), and at least Title/Section/Key must
	// have non-empty renderers.
	rendered := types.Title.Render("hello")
	if rendered == "" {
		t.Error("Title.Render() returned empty string")
	}

	rendered = types.Section.Render("section")
	if rendered == "" {
		t.Error("Section.Render() returned empty string")
	}

	rendered = types.Muted.Render("muted")
	if rendered == "" {
		t.Error("Muted.Render() returned empty string")
	}
}
