// Package message is the canonical way to print human-readable
// output from nvs commands. It centralizes the icon + color +
// indentation rules so every command line in the binary looks
// the same.
//
// The package is designed to be used as:
//
//	ui.Message.Info("Switched to %s", version)
//	ui.Message.Success("Installed %s", version)
//	ui.Message.Warn("Neovim is running; switch may misbehave")
//	ui.Message.Error("Install failed: %v", err)
//
// All helpers write to os.Stdout (Info/Success/Warn) or
// os.Stderr (Error) and respect lipgloss's color detection
// (NO_COLOR, FORCE_COLOR, TTY).
package message

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/y3owk1n/nvs/internal/ui/style"
)

// Icons is the set of glyphs the message package uses. Keeping
// them as a struct rather than free-floating constants means a
// caller cannot accidentally use a different icon set in one
// place vs another.
type Icons struct {
	Info    string
	Success string
	Warn    string
	Error   string
	Step    string
	Bullet  string
	Arrow   string
}

// DefaultIcons returns the standard glyph set. The values are
// plain Unicode (no Nerd Font dependence) so the UI looks the
// same on every terminal.
func DefaultIcons() Icons {
	return Icons{
		Info:    "ℹ",
		Success: "✓",
		Warn:    "⚠",
		Error:   "✖",
		Step:    "▸",
		Bullet:  "•",
		Arrow:   "→",
	}
}

// Printer is the entry point. It is safe to call from any
// goroutine; the underlying io.Write calls are atomic for
// short strings on POSIX, which is the only platform that
// matters for terminal output.
type Printer struct {
	palette style.Palette
	types   style.Type
	icons   Icons
	out     io.Writer
	errOut  io.Writer
}

// New constructs a Printer. If out or errOut are nil, os.Stdout
// and os.Stderr are used.
func New(palette style.Palette, types style.Type, icons Icons, out, errOut io.Writer) *Printer {
	if out == nil {
		out = os.Stdout
	}

	if errOut == nil {
		errOut = os.Stderr
	}

	return &Printer{
		palette: palette,
		types:   types,
		icons:   icons,
		out:     out,
		errOut:  errOut,
	}
}

// Default returns a Printer wired to the default palette, the
// default typographic scale, the default icon set, os.Stdout,
// and os.Stderr. Most callers should use this.
func Default() *Printer {
	palette := style.Default()
	types := style.Types(palette)

	return New(palette, types, DefaultIcons(), os.Stdout, os.Stderr)
}

// Icons returns the icon set this Printer uses. Callers that
// need to embed an icon glyph in a context where the Printer
// itself can't write a full line (e.g. as the prefix of an
// external spinner or a banner) can read it from here instead
// of re-deriving it from DefaultIcons().
func (p *Printer) Icons() Icons { return p.icons }

// Infof prints a neutral informational message.
//
//	ui.Message.Infof("Fetching available versions…")
func (p *Printer) Infof(format string, args ...any) {
	p.line(p.out, p.icons.Info, fmt.Sprintf(format, args...), p.palette.Accent)
}

// Successf prints a positive-outcome message.
//
//	ui.Message.Successf("Switched to %s", version)
func (p *Printer) Successf(format string, args ...any) {
	p.line(p.out, p.icons.Success, fmt.Sprintf(format, args...), p.palette.Success)
}

// Warnf prints a non-fatal warning. It always writes to
// os.Stdout (matching the existing nvs convention) so a
// redirected stdout captures both the result and the warning.
//
//	ui.Message.Warnf("Neovim is currently running (1 instance).")
func (p *Printer) Warnf(format string, args ...any) {
	p.line(p.out, p.icons.Warn, fmt.Sprintf(format, args...), p.palette.Warning)
}

// Errorf prints a fatal error. Unlike Info/Success/Warn, Error
// writes to os.Stderr so `nvs … 2>/dev/null` produces a clean
// pipeline.
//
//	ui.Message.Errorf("Install failed: %v", err)
func (p *Printer) Errorf(format string, args ...any) {
	p.line(p.errOut, p.icons.Error, fmt.Sprintf(format, args...), p.palette.Error)
}

// Stepf prints a step indicator inside a multi-step command
// (e.g. "▸ Resolving version…"). It is styled with the primary
// accent so it stands out as the "current action".
func (p *Printer) Stepf(format string, args ...any) {
	p.line(p.out, p.icons.Step, fmt.Sprintf(format, args...), p.palette.Primary)
}

// Bulletf prints a bullet-prefixed secondary line. Use it for
// follow-up facts under a section header.
//
//	ui.Message.Infof("Installed versions:")
//	ui.Message.Bulletf("stable")
//	ui.Message.Bulletf("nightly")
func (p *Printer) Bulletf(format string, args ...any) {
	styled := lipgloss.NewStyle().
		Foreground(p.palette.Muted).
		Render("  " + p.icons.Bullet + " " + fmt.Sprintf(format, args...))

	_, _ = fmt.Fprintln(p.out, styled)
}

// Mutedf prints a dimmed, secondary line. Use it for hints,
// explanations, or anything that should not compete with the
// primary message.
func (p *Printer) Mutedf(format string, args ...any) {
	_, _ = fmt.Fprintln(p.out, p.types.Muted.Render(fmt.Sprintf(format, args...)))
}

// Pair prints a "Key  value" pair with the key right-aligned
// in a fixed-width column. Use it for key/value lists (doctor
// checks, current version details, etc.).
//
//	ui.Message.Pair("Version", "v0.10.4")
//	ui.Message.Pair("Commit",  "abc1234")
func (p *Printer) Pair(key, value string) {
	styledKey := p.types.Key.Render(key)
	styledVal := p.types.Code.Render(value)

	_, _ = fmt.Fprintf(p.out, "%s  %s\n", styledKey, styledVal)
}

// PairLine is the non-printing counterpart of Pair. It returns
// the styled "key  value" string with a trailing newline, so a
// caller can embed it inside a multi-line Panel.Section body
// (or any other styled string) without doing I/O of its own.
//
//	ui.Message.PairLine("Version", "v0.10.4")
func (p *Printer) PairLine(key, value string) string {
	styledKey := p.types.Key.Render(key)
	styledVal := p.types.Code.Render(value)

	return styledKey + "  " + styledVal + "\n"
}

// Highlight returns text rendered in the brand primary color
// and bold. Use it inside a Panel.Section body for the "this
// is the version you are on" hero line.
//
//	ui.Message.Highlight("→") + " " + ui.Message.Highlight("stable")
func (p *Printer) Highlight(text string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(p.palette.Primary).
		Render(text)
}

// Dim returns text rendered in the muted color. Use it for
// secondary facts (e.g. "Details: unavailable") inside a
// Panel.Section body.
//
//	ui.Message.Dim("Details: unavailable")
func (p *Printer) Dim(text string) string {
	return p.types.Muted.Render(text)
}

// SuccessRow returns an icon-prefixed success row, suitable for
// embedding inside a multi-line Panel.Section body. The icon is
// the brand success glyph, rendered bold and in the success
// color, followed by a space and the body-styled text. The
// returned string already ends with a newline.
//
//	ui.Message.SuccessRow("Shell")
func (p *Printer) SuccessRow(text string) string {
	return p.styledIconLine(p.icons.Success, p.palette.Success) + p.types.Body.Render(text) + "\n"
}

// WarnRow is the warning counterpart of SuccessRow. Use it for
// status rows that are "ok, but be aware" (a missing optional
// dependency, a soft-configured PATH entry, etc.).
//
//	ui.Message.WarnRow("Dependencies")
func (p *Printer) WarnRow(text string) string {
	return p.styledIconLine(p.icons.Warn, p.palette.Warning) + p.types.Body.Render(text) + "\n"
}

// ErrorRow is the failure counterpart of SuccessRow. Use it for
// rows that flag a fatal problem for the command being run.
//
//	ui.Message.ErrorRow("PATH")
func (p *Printer) ErrorRow(text string) string {
	return p.styledIconLine(p.icons.Error, p.palette.Error) + p.types.Body.Render(text) + "\n"
}

// Detail returns an indented, muted "sub-line" for use directly
// under a SuccessRow / WarnRow / ErrorRow. The four-space
// indent visually nests the detail under the parent's icon
// (which occupies one cell plus a space).
//
//	ui.Message.ErrorRow("PATH")
//	ui.Message.Detail("bin dir not in PATH: /Users/you/.local/bin")
func (p *Printer) Detail(text string) string {
	return p.types.Muted.Render("    "+text) + "\n"
}

// Success returns text in the success color. It is the
// cell-content counterpart of Successf (no icon, no I/O).
// Use it to color a cell that is semantically "positive
// state" — e.g. an "Installed" status in a list table.
//
//	ui.Message.Success("Installed")
func (p *Printer) Success(text string) string {
	return lipgloss.NewStyle().Foreground(p.palette.Success).Render(text)
}

// Warn returns text in the warning color. It is the
// cell-content counterpart of Warnf (no icon, no I/O).
//
//	ui.Message.Warn("Installed (upgrade)")
func (p *Printer) Warn(text string) string {
	return lipgloss.NewStyle().Foreground(p.palette.Warning).Render(text)
}

// Error returns text in the error color. It is the
// cell-content counterpart of Errorf (no icon, no I/O).
//
//	ui.Message.Error("Not installed")
func (p *Printer) Error(text string) string {
	return lipgloss.NewStyle().Foreground(p.palette.Error).Render(text)
}

// Accent returns text in the accent color (cyan-ish in both
// light and dark backgrounds). It is the right choice for
// inline technical details — paths, commit hashes, URLs.
//
//	ui.Message.Accent("/Users/you/.local/bin")
func (p *Printer) Accent(text string) string {
	return lipgloss.NewStyle().Foreground(p.palette.Accent).Render(text)
}

// Text returns text in the default body color. It is the
// cell-content counterpart of the default terminal color;
// use it when a table cell needs to be explicitly in the
// body color rather than the terminal default (e.g. so the
// color survives a non-TTY / monochrome pipe).
//
//	ui.Message.Text("v0.10.4")
func (p *Printer) Text(text string) string {
	return lipgloss.NewStyle().Foreground(p.palette.Text).Render(text)
}

// Muted returns text in the muted color. It is the
// cell-content counterpart of Mutedf (no I/O, no leading
// indent). Use it to dim a cell that is semantically
// secondary — e.g. commit hashes, publish dates, the
// "Details" column of a remote-versions list.
//
//	ui.Message.Muted("Published: 2024-05-01, Commit: abc1234")
func (p *Printer) Muted(text string) string {
	return p.types.Muted.Render(text)
}

// styledIconLine returns the bold + colored icon followed by a
// single space. Centralizing the rule here keeps SuccessRow /
// WarnRow / ErrorRow visually identical and lets future tweaks
// (e.g. changing the icon-to-text gap) happen in one place.
func (p *Printer) styledIconLine(icon string, color lipgloss.AdaptiveColor) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(color).
		Render(icon) + " "
}

// line is the single emit path. Centralizing the newline and
// indentation rules here means every helper agrees on what a
// "message line" looks like. It is unexported because callers
// always go through one of the named helpers above.
func (p *Printer) line(
	writer io.Writer,
	icon string,
	message string,
	iconColor lipgloss.AdaptiveColor,
) {
	styledIcon := lipgloss.NewStyle().
		Bold(true).
		Foreground(iconColor).
		Render(icon)

	styledMsg := p.types.Body.Render(message)

	// One trailing space between the icon and the message and
	// a single \n at the end. This matches what every existing
	// nvs command does, so the new output preserves the
	// spacing users are used to.
	_, _ = fmt.Fprintf(writer, "%s %s\n", styledIcon, styledMsg)
}
