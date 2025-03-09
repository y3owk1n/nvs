package cmd

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
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
		logrus.Debug("Starting installation command")
		alias := releases.NormalizeVersion(args[0])
		logrus.Debugf("Normalized version: %s", alias)
		fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("Resolving version %s...", alias)))

		release, err := releases.ResolveVersion(alias, cacheFilePath)
		if err != nil {
			logrus.Fatalf("Error resolving version: %v", err)
		}
		logrus.Debugf("Resolved release: %+v", release)

		installName := alias
		if alias != "stable" && alias != "nightly" {
			installName = release.TagName
		}
		logrus.Debugf("Determined install name: %s", installName)

		if utils.IsInstalled(versionsDir, installName) {
			logrus.Debugf("Version %s is already installed, skipping installation", installName)
			fmt.Printf("%s %s\n", utils.WarningIcon(), utils.WhiteText(fmt.Sprintf("Version %s is already installed.", installName)))
			return
		}

		assetURL, assetPattern, err := releases.GetAssetURL(release)
		if err != nil {
			logrus.Fatalf("Error getting asset URL: %v", err)
		}
		logrus.Debugf("Resolved asset URL: %s, asset pattern: %s", assetURL, assetPattern)

		checksumURL, err := releases.GetChecksumURL(release, assetPattern)
		if err != nil {
			logrus.Fatalf("Error getting checksum URL: %v", err)
		}
		logrus.Debugf("Resolved checksum URL: %s", checksumURL)

		releaseIdentifier := releases.GetReleaseIdentifier(release, alias)
		logrus.Debugf("Determined release identifier: %s", releaseIdentifier)
		fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("Installing Neovim %s...", alias)))

		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Suffix = " 0%"
		s.Start()

		err = installer.DownloadAndInstall(
			versionsDir,
			installName,
			assetURL,
			checksumURL,
			releaseIdentifier,
			func(progress int) {
				logrus.Debugf("Download progress: %d%%", progress)
				s.Suffix = fmt.Sprintf(" %d%%", progress)
			},
			func(phase string) {
				logrus.Debugf("Installation phase: %s", phase)
				s.Prefix = phase + " "
				s.Suffix = ""
			},
		)
		s.Stop()
		if err != nil {
			logrus.Fatalf("Installation failed: %v", err)
		}
		logrus.Debug("Installation successful")
		fmt.Printf("%s %s\n", utils.SuccessIcon(), utils.WhiteText("Installation successful!"))
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
