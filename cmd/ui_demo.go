package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/ui"
	"github.com/y3owk1n/nvs/internal/ui/picker"
)

// spinnerDemoSpeed is the per-frame interval used by the demo
// spinner. 80ms is a comfortable visual cadence for the
// 1.2s animation in the demo.
const spinnerDemoSpeed = 80 * time.Millisecond

// spinnerDemoDuration is how long the demo spinner animates
// before Stop() is called and the success line is printed.
const spinnerDemoDuration = 1200 * time.Millisecond

// uiDemoCmd is a hidden command that previews the new nvs
// design system. It exists so a developer (or a curious user
// running `nvs __demo`) can see every new primitive in one
// place without having to install a Neovim version or change
// directory. It is intentionally not in `nvs help`.
var uiDemoCmd = &cobra.Command{
	Use:    "__demo",
	Short:  "Preview the nvs design system (hidden)",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE:   runUIDemo,
}

// runUIDemo renders the demo. It is split into discrete
// sections so the output reads as a tour rather than a wall
// of text.
func runUIDemo(cmd *cobra.Command, _ []string) error {
	interactive, _ := cmd.Flags().GetBool("interactive")
	ctx := cmd.Context()

	demoBanner()

	demoMessages()

	demoKeyValues()

	demoPanels()

	demoSpinner(ctx)

	demoProgress()

	if interactive {
		demoPicker(ctx)
	} else {
		ui.Message.Mutedf(
			"Tip: run `nvs __demo --interactive` to also preview the huh-based picker.",
		)
	}

	return nil
}

// demoBanner prints the wordmark + a section header.
func demoBanner() {
	_, _ = fmt.Fprintln(os.Stdout, ui.Banner.Logo())
	_, _ = fmt.Fprintln(os.Stdout, ui.Banner.Header("Design system preview"))
}

// demoMessages shows one of each severity level so a reviewer
// can see the color and icon mapping at a glance.
func demoMessages() {
	_, _ = fmt.Fprintln(os.Stdout, ui.Banner.Header("Messages"))

	ui.Message.Infof("Fetching available versions…")
	ui.Message.Successf("Switched to %s", "stable")
	ui.Message.Warnf("Neovim is currently running (1 instance).")
	ui.Message.Stepf("Building Neovim from commit %s", "abc1234")
	ui.Message.Bulletf("stable")
	ui.Message.Bulletf("nightly")
	ui.Message.Bulletf("v0.10.4")
}

// demoKeyValues shows the "Key  value" pair layout used by
// `nvs current` and `nvs doctor`.
func demoKeyValues() {
	_, _ = fmt.Fprintln(os.Stdout, ui.Banner.Header("Key / value pairs"))

	ui.Message.Pair("Version", "v0.10.4")
	ui.Message.Pair("Commit", "abc1234")
	ui.Message.Pair("Published", "2026-06-15")
	ui.Message.Pair("Path", "/Users/you/.local/share/nvs/versions/stable")
}

// demoPanels shows a plain panel and a sectioned panel.
func demoPanels() {
	_, _ = fmt.Fprintln(os.Stdout, ui.Banner.Header("Panels"))

	ui.Message.Infof("A plain panel:")

	_, _ = fmt.Fprint(
		os.Stdout,
		ui.Panel.Panel("This is a plain panel.\nIt can hold any multi-line content."),
	)

	ui.Message.Infof("A sectioned panel with a title:")

	_, _ = fmt.Fprint(
		os.Stdout,
		ui.Panel.Section(
			"Installed versions",
			"→ stable   (current)\n  nightly\n  v0.10.4",
		),
	)
}

// demoSpinner animates a short spinner so the reviewer can
// judge the new look end-to-end. The animation runs for
// ~1.2s, then a success line replaces the spinner in place.
func demoSpinner(_ context.Context) {
	_, _ = fmt.Fprintln(os.Stdout, ui.Banner.Header("Spinner"))

	spinner := ui.NewSpinner(os.Stdout, spinnerDemoSpeed)
	spinner.SetPrefix(ui.SuccessIcon() + " ")
	spinner.SetSuffix(" Installing v0.10.4…")
	spinner.Start()

	time.Sleep(spinnerDemoDuration)

	spinner.Stop()

	_, _ = fmt.Fprintf(
		os.Stdout,
		"%s %s\n",
		ui.SuccessIcon(),
		ui.Style.Type().Body.Render("Installation successful!"),
	)
}

// demoProgress shows the progress bar at 0%, 50%, and 100%
// using the existing public helpers. The output here is
// intentionally static — the spinner above already shows that
// the bar animates smoothly when wired to a real phase
// callback.
func demoProgress() {
	_, _ = fmt.Fprintln(os.Stdout, ui.Banner.Header("Progress"))

	for _, percent := range []int{0, 25, 50, 75, 100} {
		_, _ = fmt.Fprintf(
			os.Stdout,
			"  %s\n",
			ui.FormatPhaseProgress("Downloading", percent),
		)
	}
}

// demoPicker runs the interactive picker against a fixed
// list so the reviewer can see the huh-driven selection UX.
// It is only invoked when the user passes --interactive.
func demoPicker(_ context.Context) {
	_, _ = fmt.Fprintln(os.Stdout, ui.Banner.Header("Picker (interactive)"))

	p := ui.Picker.NewPicker(nil, nil)

	got, err := p.Select(
		"Choose a Neovim version to install",
		[]picker.SelectItem{
			{Label: "stable", Description: "Latest stable release"},
			{Label: "nightly", Description: "Latest nightly build"},
			{Label: "v0.10.4", Description: "Pinned release"},
			{Label: "abc1234", Description: "Build from commit"},
		},
	)
	if err != nil {
		ui.Message.Warnf("Picker canceled: %v", err)

		return
	}

	ui.Message.Successf("You chose: %s", got)
}

func init() {
	uiDemoCmd.Flags().BoolP("interactive", "i", false, "Also run the interactive picker demo")
	rootCmd.AddCommand(uiDemoCmd)
}
