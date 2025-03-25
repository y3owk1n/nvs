package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/builder"
	"github.com/y3owk1n/nvs/pkg/installer"
	"github.com/y3owk1n/nvs/pkg/releases"
	"github.com/y3owk1n/nvs/pkg/utils"
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
var installCmd = &cobra.Command{
	Use:     "install <version|stable|nightly|master|commit-hash>",
	Aliases: []string{"i"},
	Short:   "Install a Neovim version or commit",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Debug("Starting installation command")

		// Create a context with a timeout to prevent hanging installations.
		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Minute)
		defer cancel()

		// Normalize the input version (e.g., prefix with "v" if needed)
		alias := releases.NormalizeVersion(args[0])
		logrus.Debugf("Normalized version: %s", alias)
		fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("Resolving version %s...", utils.CyanText(alias))))

		// Check if the alias is a commit hash
		isCommitHash := releases.IsCommitHash(alias)
		logrus.Debugf("isCommitHash: %t", isCommitHash)

		// If it is a commit hash, build Neovim from that commit.
		if isCommitHash {
			logrus.Debugf("Building Neovim from commit %s", alias)
			fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText("Building Neovim from commit "+utils.CyanText(alias)))
			if err := builder.BuildFromCommit(ctx, alias, versionsDir); err != nil {
				logrus.Fatalf("%v", err)
			}
		} else {
			// Otherwise, install the pre-built version.
			logrus.Debugf("Start installing %s", alias)
			if err := installer.InstallVersion(ctx, alias, versionsDir, cacheFilePath); err != nil {
				logrus.Fatalf("%v", err)
			}
		}
	},
}

// init registers the installCmd with the root command.
func init() {
	rootCmd.AddCommand(installCmd)
}
