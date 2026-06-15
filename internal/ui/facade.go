// Package ui exposes the user-facing surface of the nvs design
// system. It re-exports the most common helpers from the
// internal ui/... subpackages so call sites can write
//
//	ui.Message.Info("hello")
//	ui.Panel.Section(p, "title", "body")
//
// without having to import five sub-packages.
//
// The package deliberately keeps the original fatih/color
// helpers (CyanText, GreenText, …) and the existing icon
// helpers (InfoIcon, SuccessIcon, …) for backward compatibility
// with the legacy code paths. New code should prefer the new
// primitives (Message, Panel, Banner, Picker) — they share a
// single design system and a single source of truth (style).
package ui

import (
	"io"
	"os"

	"github.com/charmbracelet/x/term"
	"github.com/y3owk1n/nvs/internal/ui/banner"
	"github.com/y3owk1n/nvs/internal/ui/message"
	"github.com/y3owk1n/nvs/internal/ui/panel"
	"github.com/y3owk1n/nvs/internal/ui/picker"
	"github.com/y3owk1n/nvs/internal/ui/style"
	uitable "github.com/y3owk1n/nvs/internal/ui/table"
)

// Message is the canonical printer for one-line human-readable
// output. It is wired to the default palette, the default
// typographic scale, the default icon set, and os.Stdout/err.
var Message = message.Default()

// Banner exposes the wordmark helpers. Use Banner.Logo() at the
// top of a command summary, Banner.Mark() inside an interactive
// prompt, and Banner.Header() to introduce a sub-section.
var Banner = bannerAPI{}

// Panel exposes the box/card helpers. Use Panel.Panel() for
// a simple box, Panel.Section() for a box with a title row.
var Panel = panelAPI{}

// Style re-exports the design system tokens so commands can
// reach for the palette or the type scale without importing
// internal/ui/style directly.
var Style = styleAPI{}

// Picker exposes the interactive form helpers. Use
// NewPicker(stdin, stdout) to construct one; it auto-detects
// whether the input is a TTY.
var Picker = pickerAPI{}

// Table exposes the data-table primitive. Use Table.New(headers...)
// to construct one, chain Row(...) / Current(idx) / Width(w)
// / Wrap(b) calls, and finish with Render().
var Table = tableAPI{}

// bannerAPI is the public façade for the banner package. It
// is a struct rather than a few free functions so future
// banner helpers can be added without breaking call sites.
type bannerAPI struct{}

// Logo returns the multi-line wordmark used at the top of
// high-level command output.
func (bannerAPI) Logo() string { return banner.Logo(style.Default()) }

// Mark returns the inline "nvs" prefix used in prompts and
// interactive forms.
func (bannerAPI) Mark() string { return banner.Mark(style.Default()) }

// Header renders a section title with a thin underline.
func (bannerAPI) Header(text string) string {
	return banner.Header(style.Default(), text)
}

// panelAPI is the public façade for the panel package.
type panelAPI struct{}

// Panel wraps content in a rounded box with a subtle border.
func (panelAPI) Panel(content string) string { return panel.Panel(style.Default(), content) }

// Section wraps content in a rounded box with a bold title row.
func (panelAPI) Section(title, content string) string {
	return panel.Section(style.Default(), title, content)
}

// styleAPI is the public façade for the style package. It is
// re-exported so a command can read (e.g.) the current
// primary color when it needs to compose its own style.
type styleAPI struct{}

// Palette returns the current nvs palette. The returned value
// is a copy; mutating it does not affect the global theme.
func (styleAPI) Palette() style.Palette { return style.Default() }

// Type returns the current nvs typographic scale. The returned
// value is a value type, not a pointer, but the underlying
// *lipgloss.Style fields are themselves pointers and should
// be treated as read-only.
func (styleAPI) Type() style.Type { return style.Types(style.Default()) }

// ColorEnabled reports whether ANSI color escapes should be
// emitted. It honors NO_COLOR and FORCE_COLOR.
func (styleAPI) ColorEnabled() bool { return style.ColorEnabled() }

// pickerAPI is the public façade for the picker package.
type pickerAPI struct{}

// NewPicker constructs a Picker. Pass nil for input or output
// to default to os.Stdin / os.Stdout. The returned Picker
// auto-detects whether stdin is a TTY and refuses to run a
// form if it is not, returning picker.ErrNoTTY to the caller.
func (pickerAPI) NewPicker(input io.Reader, output io.Writer) *picker.Picker {
	if input == nil {
		input = os.Stdin
	}

	if output == nil {
		output = os.Stdout
	}

	hasTTY := false

	if f, ok := input.(*os.File); ok {
		hasTTY = term.IsTerminal(f.Fd())
	}

	return picker.New(input, output, hasTTY)
}

// tableAPI is the public façade for the table package. It
// is a struct rather than a free function so future
// table-wide configuration (e.g. a default width or theme
// override) can be added without breaking call sites.
type tableAPI struct{}

// New returns a new ui/table.Table with the given column
// headers. The caller chains Row(...) / Current(idx) /
// Width(w) / Wrap(b) on the returned table and finishes
// with Render().
//
//	ui.Table.New("VERSION", "STATUS").
//	    Row("v0.10.0", "Installed").
//	    Current(0).
//	    Render(ui.Style.Palette())
func (tableAPI) New(headers ...string) *uitable.Table {
	return uitable.New(headers...)
}
