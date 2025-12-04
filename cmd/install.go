package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/ui"
)

// TimeoutMinutes is the timeout in minutes for installation.
const TimeoutMinutes = 30

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
var installCmd = &cobra.Command{
	Use:     "install <version|stable|nightly|master|commit-hash>",
	Aliases: []string{"i"},
	Short:   "Install a Neovim version or commit",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunInstall(cmd, args)
	},
}

// RunInstall executes the install command.
func RunInstall(cmd *cobra.Command, args []string) error {
	const (
		SpinnerSpeed  = 100
		InitialSuffix = " 0%"
	)

	logrus.Debug("Starting installation command")

	// Create a context with a timeout to prevent hanging installations.
	ctx, cancel := context.WithTimeout(cmd.Context(), TimeoutMinutes*time.Minute)
	defer cancel()

	alias := args[0]
	logrus.Debugf("Requested version: %s", alias)

	// Create and start a spinner for progress
	progressSpinner := spinner.New(spinner.CharSets[14], SpinnerSpeed*time.Millisecond)
	progressSpinner.Prefix = fmt.Sprintf("%s %s ", ui.InfoIcon(), ui.WhiteText(fmt.Sprintf("Installing Neovim %s...", alias)))
	progressSpinner.Suffix = InitialSuffix
	progressSpinner.Start()

	// Use version service to install
	err := GetVersionService().Install(ctx, alias, func(phase string, progress int) {
		progressSpinner.Suffix = fmt.Sprintf(" %s %d%%", phase, progress)
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
}
