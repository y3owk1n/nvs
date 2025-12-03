package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/briandowns/spinner"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/builder"
	"github.com/y3owk1n/nvs/pkg/helpers"
	"github.com/y3owk1n/nvs/pkg/installer"
	"github.com/y3owk1n/nvs/pkg/releases"
)

// useCmd represents the "use" command.
// It switches the active Neovim version to a specific version, stable, nightly, or a commit hash.
// If the requested version is not installed, it is either built (if a commit hash) or installed.
// Finally, it updates the "current" symlink to point to the target version.
//
// Example usage:
//
//	nvs use stable
//	nvs use v0.6.0
//	nvs use nightly
//	nvs use 1a2b3c4 (a commit hash)
var useCmd = &cobra.Command{
	Use:   "use <version|stable|nightly|commit-hash>",
	Short: "Switch to a specific version or commit hash",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		const TimeoutMin = 30
		const SpinnerSpeed = 100
		const InitialSuffix = " 0%"

		// Create a context with a timeout for the operation.
		ctx, cancel := context.WithTimeout(cmd.Context(), TimeoutMin*time.Minute)
		defer cancel()

		// Normalize the version string (e.g. adding a "v" prefix if needed).
		alias := releases.NormalizeVersion(args[0])
		targetVersion := alias

		logrus.Debugf("Resolved target version: %s", targetVersion)

		// Check if the target version is already installed.
		if !helpers.IsInstalled(versionsDir, targetVersion) {
			// Determine if the alias is a commit hash.
			isCommitHash := releases.IsCommitHash(alias)
			logrus.Debugf("isCommitHash: %t", isCommitHash)

			if isCommitHash {
				// Build from source if a commit hash is provided.
				logrus.Debugf("Building Neovim from commit %s", alias)
				_, printErr := fmt.Fprintf(os.Stdout,
					"%s %s\n",
					helpers.InfoIcon(),
					helpers.WhiteText("Building Neovim from commit "+helpers.CyanText(alias)),
				)
				if printErr != nil {
					logrus.Warnf("Failed to write to stdout: %v", printErr)
				}
				err := builder.BuildFromCommit(ctx, alias, versionsDir)
				if err != nil {
					logrus.Fatalf("%v", err)
				}
			} else {
				// Otherwise, install the version if it's not yet installed.
				logrus.Debugf("Start installing %s", alias)

				// Create and start a spinner for download progress
				spinner := spinner.New(spinner.CharSets[14], SpinnerSpeed*time.Millisecond)
				spinner.Prefix = fmt.Sprintf("%s %s ", helpers.InfoIcon(), helpers.WhiteText(fmt.Sprintf("Installing Neovim %s...", alias)))
				spinner.Suffix = InitialSuffix
				spinner.Start()

				err := installer.InstallVersion(ctx, alias, versionsDir, cacheFilePath, func(progress int) {
					spinner.Suffix = fmt.Sprintf(" %d%%", progress)
				})
				if err != nil {
					logrus.Fatalf("%v", err)
				}

				spinner.Stop()

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
		}

		// Determine the current symlink path.
		currentSymlink := filepath.Join(versionsDir, "current")
		// Resolve what "current" points to, whether it's a symlink or a junction.
		var info os.FileInfo
		var err error
		info, err = os.Lstat(currentSymlink)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				// Regular symlink → use Readlink
				var target string
				target, err = os.Readlink(currentSymlink)
				if err == nil {
					if filepath.Base(target) == targetVersion {
						_, printErr := fmt.Fprintf(os.Stdout,
							"%s Already using Neovim %s\n",
							helpers.WarningIcon(),
							helpers.CyanText(targetVersion),
						)
						if printErr != nil {
							logrus.Warnf("Failed to write to stdout: %v", printErr)
						}
						logrus.Debugf("Already using version: %s", targetVersion)

						return
					}
				}
			} else if runtime.GOOS == windows {
				// On Windows, junctions look like normal directories to os.Lstat.
				// So we just check if it resolves to the target path.
				absTarget := filepath.Join(versionsDir, targetVersion)
				absCurrent, _ := filepath.EvalSymlinks(currentSymlink) // works for junctions
				if absCurrent == absTarget {
					_, printErr := fmt.Fprintf(os.Stdout, "%s Already using Neovim %s\n", helpers.WarningIcon(), helpers.CyanText(targetVersion))
					if printErr != nil {
						logrus.Warnf("Failed to write to stdout: %v", printErr)
					}
					logrus.Debugf("Already using version (junction): %s", targetVersion)

					return
				}
			}
		}

		// Switch to the target version by updating the symlink.
		err = helpers.UseVersion(targetVersion, currentSymlink, versionsDir, globalBinDir)
		if err != nil {
			logrus.Fatalf("%v", err)
		}
	},
}

// init registers the useCmd with the root command.
func init() {
	rootCmd.AddCommand(useCmd)
}
