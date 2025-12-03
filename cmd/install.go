package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/builder"
	"github.com/y3owk1n/nvs/pkg/helpers"
	"github.com/y3owk1n/nvs/pkg/installer"
	"github.com/y3owk1n/nvs/pkg/releases"
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
		return RunInstall(cmd, args, VersionsDir, CacheFilePath)
	},
}

// RunInstall executes the install command.
func RunInstall(cmd *cobra.Command, args []string, versionsDir, cacheFilePath string) error {
	const (
		SpinnerSpeed  = 100
		InitialSuffix = " 0%"
	)

	logrus.Debug("Starting installation command")

	// Create a context with a timeout to prevent hanging installations.
	ctx, cancel := context.WithTimeout(cmd.Context(), TimeoutMinutes*time.Minute)
	defer cancel()

	// Normalize the input version (e.g., prefix with "v" if needed)
	alias := releases.NormalizeVersion(args[0])
	logrus.Debugf("Normalized version: %s", alias)

	var err error

	_, err = fmt.Fprintf(os.Stdout,
		"%s %s\n",
		helpers.InfoIcon(),
		helpers.WhiteText(fmt.Sprintf("Resolving version %s...", helpers.CyanText(alias))),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	// Check if the alias is a commit hash
	isCommitHash := releases.IsCommitHash(alias)
	logrus.Debugf("isCommitHash: %t", isCommitHash)

	// If it is a commit hash, build Neovim from that commit.
	if isCommitHash {
		logrus.Debugf("Building Neovim from commit %s", alias)

		_, err = fmt.Fprintf(os.Stdout,
			"%s %s\n",
			helpers.InfoIcon(),
			helpers.WhiteText("Building Neovim from commit "+helpers.CyanText(alias)),
		)
		if err != nil {
			logrus.Warnf("Failed to write to stdout: %v", err)
		}

		err = builder.BuildFromCommit(ctx, alias, versionsDir)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(
			os.Stdout,
			"%s %s\n",
			helpers.SuccessIcon(),
			helpers.WhiteText("Build from commit successful!"),
		)
		if err != nil {
			logrus.Warnf("Failed to write to stdout: %v", err)
		}
	} else {
		// Otherwise, install the pre-built version.
		logrus.Debugf("Start installing %s", alias)

		// Create and start a spinner for download progress
		progressSpinner := spinner.New(spinner.CharSets[14], SpinnerSpeed*time.Millisecond)
		progressSpinner.Prefix = fmt.Sprintf("%s %s ", helpers.InfoIcon(), helpers.WhiteText(fmt.Sprintf("Installing Neovim %s...", alias)))
		progressSpinner.Suffix = InitialSuffix
		progressSpinner.Start()

		err = installer.InstallVersion(ctx, alias, versionsDir, cacheFilePath, func(progress int) {
			progressSpinner.Suffix = fmt.Sprintf(" %d%%", progress)
		})
		if err != nil {
			progressSpinner.Stop()

			return err
		}

		progressSpinner.Stop()

		_, err = fmt.Fprintf(
			os.Stdout,
			"%s %s\n",
			helpers.SuccessIcon(),
			helpers.WhiteText("Installation successful!"),
		)
		if err != nil {
			logrus.Warnf("Failed to write to stdout: %v", err)
		}
	}

	return nil
}

// init registers the installCmd with the root command.
func init() {
	rootCmd.AddCommand(installCmd)
}
