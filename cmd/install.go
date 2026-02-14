package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/manifoldco/promptui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/ui"
)

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
	const SpinnerSpeed = 100

	logrus.Debug("Starting installation command")

	// Create a context with a timeout to prevent hanging installations.
	ctx, cancel := context.WithTimeout(cmd.Context(), constants.TimeoutMinutes*time.Minute)
	defer cancel()

	var alias string

	// Check if --pick flag is set
	pick, _ := cmd.Flags().GetBool("pick")
	if pick {
		// Launch picker for remote versions
		releases, err := VersionServiceFromContext(cmd.Context()).ListRemote(ctx, false)
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

	logrus.Debugf("Requested version: %s", alias)

	// Create and start a spinner for progress
	progressSpinner := spinner.New(spinner.CharSets[14], SpinnerSpeed*time.Millisecond)
	progressSpinner.Prefix = ui.InfoIcon() + " "
	progressSpinner.Suffix = fmt.Sprintf(" Installing %s...", alias)
	progressSpinner.Start()

	// Use version service to install
	err := VersionServiceFromContext(
		cmd.Context(),
	).Install(ctx, alias, func(phase string, progress int) {
		progressSpinner.Suffix = " " + ui.FormatPhaseProgress(phase, progress)
	})
	if err != nil {
		progressSpinner.Stop()

		return err
	}

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
