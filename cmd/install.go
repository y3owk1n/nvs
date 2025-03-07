package cmd

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/installer"
	"github.com/y3owk1n/nvs/pkg/releases"
	"github.com/y3owk1n/nvs/pkg/utils"
)

var installCmd = &cobra.Command{
	Use:     "install <version|stable|nightly>",
	Aliases: []string{"i"},
	Short:   "Install a Neovim version",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Normalize the version string.
		alias := releases.NormalizeVersion(args[0])
		color.Cyan("Resolving version %s...", alias)

		// Resolve the version (using the cache file).
		release, err := releases.ResolveVersion(alias, cacheFilePath)
		if err != nil {
			logrus.Fatalf("Error resolving version: %v", err)
		}

		// Determine the installation folder name.
		installName := alias
		if alias != "stable" && alias != "nightly" {
			installName = release.TagName
		}
		if utils.IsInstalled(versionsDir, installName) {
			color.Yellow("Version %s is already installed", installName)
			return
		}

		assetURL, assetPattern, err := releases.GetAssetURL(release)
		if err != nil {
			logrus.Fatalf("Error getting asset URL: %v", err)
		}
		checksumURL, err := releases.GetChecksumURL(release, assetPattern)
		if err != nil {
			logrus.Fatalf("Error getting checksum URL: %v", err)
		}

		releaseIdentifier := releases.GetReleaseIdentifier(release, alias)
		color.Cyan("Installing Neovim %s...", alias)

		// Create a spinner with a modern look, similar to GitHub CLI.
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Suffix = " 0%"
		s.Start()

		// Call the installer function that reports progress.
		// This function is assumed to accept a callback that receives progress (0-100).
		err = installer.DownloadAndInstall(
			versionsDir,
			installName,
			assetURL,
			checksumURL,
			releaseIdentifier,
			func(progress int) {
				s.Suffix = fmt.Sprintf(" %d%%", progress)
			},
			func(phase string) {
				s.Prefix = phase + " "
				s.Suffix = ""
			},
		)
		if err != nil {
			s.Stop()
			logrus.Fatalf("Installation failed: %v", err)
		}
		s.Stop()
		color.Green("\nInstallation successful!")
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
