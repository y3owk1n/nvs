package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/ui"
)

// installSpinnerSpeed is the spinner animation interval in
// milliseconds, shared between RunInstall and runInstallForAlias.
const installSpinnerSpeed = 100

// installCmd represents the "install" command.
// It installs a specified version of Neovim. The command accepts a single argument which may be:
//   - A version alias ("stable", "nightly", or "master")
//   - A specific version tag
//   - A commit hash (which triggers a build from source)
//
// Depending on whether the argument is recognized as a commit hash, it either builds Neovim from that commit
// using the builder package, or installs a pre-built version using the installer package.
//
// The installation process is bound by a 30-minute timeout.
//
// Example usage:
//
//	nvs install stable
//	nvs install nightly
//	nvs install master
//	nvs install 1a2b3c4 (for a commit hash)
//	nvs install --pick
var installCmd = &cobra.Command{
	Use:     "install [version|stable|nightly|master|commit-hash]",
	Aliases: []string{"i"},
	Short:   "Install a Neovim version or commit",
	Args:    cobra.MaximumNArgs(1),
	RunE:    RunInstall,
}

// RunInstall executes the install command.
func RunInstall(cmd *cobra.Command, args []string) error {
	logrus.Debug("Starting installation command")

	// Create a context with a timeout to prevent hanging installations.
	ctx, cancel := context.WithTimeout(cmd.Context(), constants.TimeoutMinutes*time.Minute)
	defer cancel()

	var alias string

	// Check if --pick flag is set
	pick, _ := cmd.Flags().GetBool("pick")
	if pick {
		// Launch picker for remote versions
		releases, err := GetVersionService().ListRemote(ctx, false)
		if err != nil {
			return fmt.Errorf("error fetching releases: %w", err)
		}

		if len(releases) == 0 {
			return fmt.Errorf("%w for selection", ErrNoVersionsAvailable)
		}

		availableVersions := make([]string, 0, len(releases))
		for _, release := range releases {
			availableVersions = append(availableVersions, release.TagName())
		}

		prompt := promptui.Select{
			Label: "Select version to install",
			Items: availableVersions,
		}

		_, selectedVersion, err := prompt.Run()
		if err != nil {
			if errors.Is(err, promptui.ErrInterrupt) {
				_, printErr := fmt.Fprintf(
					os.Stdout,
					"%s %s\n",
					ui.WarningIcon(),
					ui.WhiteText("Selection canceled."),
				)
				if printErr != nil {
					logrus.Warnf("Failed to write to stdout: %v", printErr)
				}

				return nil
			}

			return fmt.Errorf("prompt failed: %w", err)
		}

		alias = selectedVersion
	} else {
		if len(args) == 0 {
			return fmt.Errorf("%w", ErrVersionArgRequired)
		}

		alias = args[0]
	}

	return runInstallForAlias(ctx, cmd, alias)
}

// runInstallForAlias performs the spinner-driven install of a single
// already-resolved alias. Split out of RunInstall so that callers
// that have already resolved the alias (notably 'use --pick' falling
// through after a version-not-found error) can reuse the install
// path without re-invoking the picker.
//
// context.Context is the first parameter per the revive
// 'context-as-argument' rule; cmd is kept as a positional argument
// only to preserve a future use of cmd-derived values (e.g. IO
// streams), but is currently unused.
func runInstallForAlias(
	ctx context.Context,
	cmd *cobra.Command,
	alias string,
) error {
	_ = cmd

	logrus.Debugf("Requested version: %s", alias)

	// Create and start a spinner for progress. The spinner
	// detects non-terminal writers internally and becomes a
	// no-op when stdout is piped or redirected, so this is
	// safe in all environments.
	progressSpinner := ui.NewSpinner(
		os.Stdout,
		time.Duration(installSpinnerSpeed)*time.Millisecond,
	)
	progressSpinner.SetPrefix(ui.InfoIcon() + " ")
	progressSpinner.SetSuffix(fmt.Sprintf(" Installing %s...", alias))
	progressSpinner.Start()

	// Ensure the spinner is always stopped, even on panic, so
	// the underlying animation goroutine does not keep writing
	// to the terminal after a panic stack trace. Stop blocks
	// until the animation goroutine has fully exited, so
	// subsequent writes to stdout (such as the success
	// message below) are guaranteed to appear after the
	// spinner line has been cleared.
	defer progressSpinner.Stop()

	// Use version service to install
	err := GetVersionService().Install(ctx, alias, func(phase string, progress int) {
		progressSpinner.SetSuffix(" " + ui.FormatPhaseProgress(phase, progress))
	})
	if err != nil {
		return err
	}

	// Stop the spinner explicitly before printing the success
	// message. The explicit Stop (rather than relying on the
	// defer above) is intentional: the success message must
	// land on the cleared spinner line, not on a new line
	// below it. Stop's line-erase leaves the cursor at column
	// 0 of the (now-empty) spinner line, so the next write
	// appears right where the spinner was — producing the
	// "spinner replaced by result" UX callers expect.
	progressSpinner.Stop()

	_, err = fmt.Fprintf(
		os.Stdout,
		"%s %s\n",
		ui.SuccessIcon(),
		ui.WhiteText("Installation successful!"),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	return nil
}

// init registers the installCmd with the root command.
func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolP("pick", "p", false, "Launch interactive picker to select version")
}
