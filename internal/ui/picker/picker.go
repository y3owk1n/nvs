// Package picker is a thin wrapper around charmbracelet/huh
// that gives nvs commands a single, consistent way to ask the
// user a question.
//
// It exposes just the three operations nvs actually uses: a
// single-select list (Picker.Select), a yes/no confirmation
// (Picker.Confirm), and a TTY-aware confirmation that also
// handles piped input (Picker.ConfirmScriptable). The wrapper
// translates the "non-TTY" case into a typed error the caller
// can detect and turn into a clean "Selection canceled."
// message, matching the existing nvs UX.
package picker

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/y3owk1n/nvs/internal/ui/style"
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
		theme:  nvsTheme(style.PickerColors()),
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
		Foreground(lipgloss.Color(style.PickerColors().Muted)).
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

// nonTTYPromptIcon is the glyph used in the non-TTY fallback
// prompt. It is a plain ASCII '?' (not a styled icon) because
// the non-TTY path is meant to be parsed by scripts and CI,
// which must not have to handle an SGR-styled icon at the
// start of a one-line prompt.
const nonTTYPromptIcon = "?"

// promptAffirmatives is the set of case-insensitive, trimmed
// inputs that count as a "yes" in the non-TTY path. The set is
// intentionally short: "y" and "yes" cover every common shell
// convention (the Go module default uses "y" as a shortcut,
// POSIX getopt uses "y" as the affirmative letter, and
// `man 1 yes` is the standard "y" command on macOS/Linux).
// Anything else — including an empty line, EOF, or a typo —
// counts as a "no", matching the safe default of Confirm
// and the existing bufio.Reader behavior the picker replaces.
var promptAffirmatives = map[string]struct{}{
	"y":   {},
	"yes": {},
}

// ConfirmScriptable is a TTY-aware variant of Confirm that
// keeps the operation scriptable. It is the right method for
// destructive-operation prompts (uninstall, reset, ...) that
// need to remain usable from `echo y | nvs …` while upgrading
// the interactive UX to huh's full Confirm form.
//
// Behavior:
//
//   - TTY input: delegates to Confirm — huh renders a styled
//     Yes/No toggle with arrow-key navigation, default = "No",
//     Y / N / Ctrl-C shortcuts, and the picker theme.
//   - Non-TTY input: emits a one-line "<icon> <title> [y/N]: "
//     prompt to p.output and reads a line from p.input via a
//     fresh bufio.Reader. The answer is trimmed and lower-
//     cased; if it matches promptAffirmatives ("y" or "yes")
//     the method returns (true, nil). Anything else (empty
//     line, EOF, typo, ...) returns (false, nil). The only
//     error path is a non-EOF read error, which is wrapped
//     with the underlying cause for the caller's logs.
//
// Why not delegate the non-TTY case to the caller? Keeping
// it inside the picker means the y/yes recognition logic,
// the prompt format, and the future policy for "what counts
// as a confirmation" all live in one place — which is the
// same one place that owns the TTY form. Splitting the two
// would invite drift (one prompt saying "(y/N)", another
// saying "[Y/n]"; one accepting "yeah" and the other not).
func (p *Picker) ConfirmScriptable(title string) (bool, error) {
	if p.hasTTY {
		return p.Confirm(title)
	}

	_, _ = fmt.Fprintf(
		p.output,
		"%s %s [y/N]: ",
		nonTTYPromptIcon,
		title,
	)

	reader := bufio.NewReader(p.input)

	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}

	answer := strings.ToLower(strings.TrimSpace(line))
	if _, ok := promptAffirmatives[answer]; ok {
		return true, nil
	}

	return false, nil
}

// nvsTheme returns the huh theme that matches the rest of the
// nvs UI: a Neovim-green primary, dimmed borders, and an
// arrow cursor (▸) instead of huh's default bullet.
//
// The theme is intentionally minimal. Adding more knobs here
// is the right move only if every command needs them — having
// one style across the whole tool is the whole point of a
// design system.
//
// colors is read at Picker construction time, so it picks up
// whatever the NVS_PICKER_<NAME> environment variables were at
// the moment nvs started.
func nvsTheme(colors style.PickerPalette) *huh.Theme {
	theme := huh.ThemeBase()

	theme.Focused.Title = theme.Focused.Title.Foreground(lipgloss.Color(colors.Primary))
	theme.Focused.SelectSelector = theme.Focused.SelectSelector.Foreground(
		lipgloss.Color(colors.Primary),
	)
	theme.Focused.SelectedOption = theme.Focused.SelectedOption.Foreground(
		lipgloss.Color(colors.Primary),
	)
	theme.Focused.UnselectedOption = theme.Focused.UnselectedOption.Foreground(
		lipgloss.Color(colors.Text),
	)
	theme.Focused.FocusedButton = theme.Focused.FocusedButton.
		Background(lipgloss.Color(colors.Primary)).
		Foreground(lipgloss.Color(colors.Background))
	theme.Blurred.Title = theme.Blurred.Title.Foreground(lipgloss.Color(colors.Muted))

	return theme
}
