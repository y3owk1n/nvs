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

var upgradeCmd = &cobra.Command{
	Use:     "upgrade [stable|nightly]",
	Aliases: []string{"up"},
	Short:   "Upgrade installed stable and/or nightly versions",
	Long:    "Upgrades the installed stable and/or nightly versions. If no argument is provided, both stable and nightly are upgraded (if installed).",
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Debug("Starting upgrade command")

		var aliases []string
		if len(args) == 0 {
			aliases = []string{"stable", "nightly"}
		} else {
			if args[0] != "stable" && args[0] != "nightly" {
				logrus.Fatalf("Upgrade can only be performed for 'stable' or 'nightly'")
			}
			aliases = []string{args[0]}
		}

		for _, alias := range aliases {
			logrus.Debugf("Processing alias: %s", alias)

			if !utils.IsInstalled(versionsDir, alias) {
				logrus.Debugf("Alias '%s' is not installed. Skipping upgrade.", alias)
				continue
			}

			release, err := releases.ResolveVersion(alias, cacheFilePath)
			if err != nil {
				logrus.Errorf("Error resolving %s: %v", alias, err)
				continue
			}
			logrus.Debugf("Resolved version for %s: %+v", alias, release)

			remoteIdentifier := releases.GetReleaseIdentifier(release, alias)
			installedIdentifier, err := releases.GetInstalledReleaseIdentifier(versionsDir, alias)
			if err == nil && installedIdentifier == remoteIdentifier {
				logrus.Debugf("%s is already up-to-date (%s)", alias, installedIdentifier)
				fmt.Printf("%s %s is already up-to-date (%s)\n", utils.WarningIcon(), utils.WhiteText(alias), utils.WhiteText(installedIdentifier))
				continue
			}

			logrus.Debugf("Fetching asset URL for %s", alias)
			assetURL, assetPattern, err := releases.GetAssetURL(release)
			if err != nil {
				logrus.Errorf("Error getting asset URL for %s: %v", alias, err)
				continue
			}

			logrus.Debugf("Fetching checksum URL for %s", alias)
			checksumURL, err := releases.GetChecksumURL(release, assetPattern)
			if err != nil {
				logrus.Errorf("Error getting checksum URL for %s: %v", alias, err)
				continue
			}

			fmt.Printf("%s %s upgrading to new identifier %s...\n", utils.InfoIcon(), utils.WhiteText(alias), utils.WhiteText(remoteIdentifier))
			logrus.Debugf("Starting upgrade for %s to identifier %s", alias, remoteIdentifier)

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			s.Suffix = " 0%"
			s.Start()

			err = installer.DownloadAndInstall(
				versionsDir,
				alias,
				assetURL,
				checksumURL,
				remoteIdentifier,
				func(progress int) {
					s.Suffix = fmt.Sprintf(" %d%%", progress)
					logrus.Debugf("%s upgrade progress: %d%%", alias, progress)
				},
				func(phase string) {
					s.Prefix = phase + " "
					s.Suffix = ""
					logrus.Debugf("%s upgrade phase: %s", alias, phase)
				},
			)
			s.Stop()
			if err != nil {
				logrus.Errorf("Upgrade failed for %s: %v", alias, err)
				continue
			}
			fmt.Printf("%s %s upgraded successfully!\n", utils.SuccessIcon(), utils.WhiteText(alias))
			logrus.Debugf("%s upgraded successfully", alias)
		}
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}
