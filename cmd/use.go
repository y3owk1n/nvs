package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/builder"
	"github.com/y3owk1n/nvs/pkg/installer"
	"github.com/y3owk1n/nvs/pkg/releases"
	"github.com/y3owk1n/nvs/pkg/utils"
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
		logrus.Debug("Starting use command")

		// Create a context with a timeout for the operation.
		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Minute)
		defer cancel()

		// Normalize the version string (e.g. adding a "v" prefix if needed).
		alias := releases.NormalizeVersion(args[0])
		targetVersion := alias

		logrus.Debugf("Resolved target version: %s", targetVersion)

		// Check if the target version is already installed.
		if !utils.IsInstalled(versionsDir, targetVersion) {
			// Determine if the alias is a commit hash.
			isCommitHash := releases.IsCommitHash(alias)
			logrus.Debugf("isCommitHash: %t", isCommitHash)

			if isCommitHash {
				// Build from source if a commit hash is provided.
				logrus.Debugf("Building Neovim from commit %s", alias)
				fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText("Building Neovim from commit "+utils.CyanText(alias)))
				if err := builder.BuildFromCommit(ctx, alias, versionsDir); err != nil {
					logrus.Fatalf("%v", err)
				}
			} else {
				// Otherwise, install the version if it's not yet installed.
				logrus.Debugf("Start installing %s", alias)
				if err := installer.InstallVersion(ctx, alias, versionsDir, cacheFilePath); err != nil {
					logrus.Fatalf("%v", err)
				}
			}
		}

		// Determine the current symlink path.
		currentSymlink := filepath.Join(versionsDir, "current")
		// Resolve what "current" points to, whether it's a symlink or a junction.
		if info, err := os.Lstat(currentSymlink); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				// Regular symlink → use Readlink
				if target, err := os.Readlink(currentSymlink); err == nil {
					if filepath.Base(target) == targetVersion {
						fmt.Printf("%s Already using Neovim %s\n", utils.WarningIcon(), utils.CyanText(targetVersion))
						logrus.Debugf("Already using version: %s", targetVersion)
						return
					}
				}
			} else if runtime.GOOS == "windows" {
				// On Windows, junctions look like normal directories to os.Lstat.
				// So we just check if it resolves to the target path.
				absTarget := filepath.Join(versionsDir, targetVersion)
				absCurrent, _ := filepath.EvalSymlinks(currentSymlink) // works for junctions
				if absCurrent == absTarget {
					fmt.Printf("%s Already using Neovim %s\n", utils.WarningIcon(), utils.CyanText(targetVersion))
					logrus.Debugf("Already using version (junction): %s", targetVersion)
					return
				}
			}
		}

		// Switch to the target version by updating the symlink.
		if err := utils.UseVersion(targetVersion, currentSymlink, versionsDir, globalBinDir); err != nil {
			logrus.Fatalf("%v", err)
		}
	},
}

// init registers the useCmd with the root command.
func init() {
	rootCmd.AddCommand(useCmd)
}
