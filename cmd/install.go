package cmd

import (
	"fmt"

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
		alias := releases.NormalizeVersion(args[0])

		// Pass the cache file path to resolve the version.
		release, err := releases.ResolveVersion(alias, cacheFilePath)
		if err != nil {
			logrus.Fatalf("Error resolving version: %v", err)
		}

		installName := alias
		if alias != "stable" && alias != "nightly" {
			installName = release.TagName
		}
		if utils.IsInstalled(versionsDir, installName) {
			fmt.Printf("Version %s is already installed\n", installName)
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
		fmt.Printf("Installing Neovim %s...\n", alias)
		if err := installer.DownloadAndInstall(versionsDir, installName, assetURL, checksumURL, releaseIdentifier); err != nil {
			logrus.Fatalf("Installation failed: %v", err)
		}
		fmt.Println("\nInstallation successful!")
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
