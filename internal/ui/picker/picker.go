// Package picker is a thin wrapper around charmbracelet/huh
// that gives nvs commands a single, consistent way to ask the
// user a question.
//
// The current nvs code uses manifoldco/promptui directly, which
// is plain and visually dated. huh is the modern replacement:
// it ships a polished theme, accessible default keybindings,
// and graceful Ctrl-C handling.
//
// The wrapper exposes just the two operations nvs actually
// uses: a single-select list (Picker.Select) and a yes/no
// confirmation (Picker.Confirm). The wrapper also handles the
// "non-TTY" case by returning a typed error the caller can
// detect and translate into a clean "Selection canceled."
// message, matching the existing nvs UX.
package picker

import (
	"errors"
	"fmt"
	"io"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// ErrCanceled is returned by Select and Confirm when the user
// hits Ctrl-C (or otherwise aborts the form). Callers should
// treat it as a non-fatal "user changed their mind" signal,
// not as a real error.
var ErrCanceled = errors.New("picker: canceled by user")

// ErrNoTTY is returned by Select and Confirm when stdin is not
// a terminal. This typically happens inside scripts or CI, and
// the right behavior is to fall back to a non-interactive
// error message rather than hang on input.
var ErrNoTTY = errors.New("picker: stdin is not a TTY")

// errNoItems is returned by Select when the caller passes an
// empty items slice. It is declared as a package-level
// variable so the err113 linter does not flag the package
// for using fmt.Errorf to define a static error.
var errNoItems = errors.New("picker: no items to select from")

// Colors used by nvsTheme. These are simple hex literals (not
// AdaptiveColor) because huh themes are evaluated on a single
// background; the nvs UI's auto dark/light switch happens at
// the lipgloss level for the rest of the output, and a
// fixed-mid-saturation green reads well on both.
const (
	huhPrimary    = "#80C342"
	huhMuted      = "#6B7280"
	huhText       = "#E5E7EB"
	huhBackground = "#1F2937"
)

// Picker is the entry point. The zero value is NOT usable;
// always construct one with New() so the underlying huh.Form
// has a theme and a target IO stream.
//
// Picker is single-use: one Picker instance drives one form.
// Construct a new Picker per command, not per question.
type Picker struct {
	theme  *huh.Theme
	input  io.Reader
	output io.Writer
	hasTTY bool
}

// New constructs a Picker. The nvsTheme is applied to every
// form rendered by this Picker; the input/output streams are
// where huh draws from and to; hasTTY controls whether
// interactive prompts are allowed at all.
//
// Callers should pass hasTTY = lipgloss/termenv's
// isatty(os.Stdin) result. If hasTTY is false, Select and
// Confirm return ErrNoTTY so the command can fall back to
// a non-interactive message.
func New(input io.Reader, output io.Writer, hasTTY bool) *Picker {
	return &Picker{
		theme:  nvsTheme(),
		input:  input,
		output: output,
		hasTTY: hasTTY,
	}
}

// SelectItem is one row in a Select prompt. Label is what the
// user sees; Description is appended after " — " to give
// context (a commit hash for a nightly, a published date,
// etc.). Per-option descriptions are encoded into the label
// because huh v1 does not support per-option descriptions
// outside of its dynamic DescriptionFunc API.
type SelectItem struct {
	Label       string
	Description string
}

// formattedKey returns the "Label — Description" string shown
// in the picker. If Description is empty, the Label is shown
// alone, matching the existing nvs promptui UX.
func (item SelectItem) formattedKey() string {
	if item.Description == "" {
		return item.Label
	}

	return item.Label + "  " + lipgloss.NewStyle().
		Italic(true).
		Foreground(lipgloss.Color(huhMuted)).
		Render("— "+item.Description)
}

// Select asks the user to pick one item from items. The
// returned string is the Label of the chosen item, or one of
// ErrCanceled / ErrNoTTY.
func (p *Picker) Select(title string, items []SelectItem) (string, error) {
	if !p.hasTTY {
		return "", ErrNoTTY
	}

	if len(items) == 0 {
		return "", errNoItems
	}

	opts := make([]huh.Option[string], 0, len(items))

	for _, item := range items {
		value := item.Label

		opts = append(opts, huh.NewOption(item.formattedKey(), value))
	}

	var (
		selected string
		form     = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(title).
					Options(opts...).
					Value(&selected),
			),
		).
			WithInput(p.input).
			WithOutput(p.output).
			WithTheme(p.theme)
	)

	err := form.Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return "", ErrCanceled
		}

		return "", fmt.Errorf("picker: %w", err)
	}

	return selected, nil
}

// Confirm asks the user a yes/no question. The default
// answer (when the user hits Enter) is "no", matching
// the existing nvs convention for destructive prompts.
func (p *Picker) Confirm(title string) (bool, error) {
	if !p.hasTTY {
		return false, ErrNoTTY
	}

	var (
		confirmed bool
		form      = huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(title).
					Affirmative("Yes").
					Negative("No").
					Value(&confirmed),
			),
		).
			WithInput(p.input).
			WithOutput(p.output).
			WithTheme(p.theme)
	)

	err := form.Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return false, ErrCanceled
		}

		return false, fmt.Errorf("picker: %w", err)
	}

	return confirmed, nil
}

// nvsTheme returns the huh theme that matches the rest of the
// nvs UI: a Neovim-green primary, dimmed borders, and an
// arrow cursor (▸) instead of huh's default bullet.
//
// The theme is intentionally minimal. Adding more knobs here
// is the right move only if every command needs them — having
// one style across the whole tool is the whole point of a
// design system.
func nvsTheme() *huh.Theme {
	theme := huh.ThemeBase()

	theme.Focused.Title = theme.Focused.Title.Foreground(lipgloss.Color(huhPrimary))
	theme.Focused.SelectSelector = theme.Focused.SelectSelector.Foreground(
		lipgloss.Color(huhPrimary),
	)
	theme.Focused.SelectedOption = theme.Focused.SelectedOption.Foreground(
		lipgloss.Color(huhPrimary),
	)
	theme.Focused.UnselectedOption = theme.Focused.UnselectedOption.Foreground(
		lipgloss.Color(huhText),
	)
	theme.Focused.FocusedButton = theme.Focused.FocusedButton.
		Background(lipgloss.Color(huhPrimary)).
		Foreground(lipgloss.Color(huhBackground))
	theme.Blurred.Title = theme.Blurred.Title.Foreground(lipgloss.Color(huhMuted))

	return theme
}
