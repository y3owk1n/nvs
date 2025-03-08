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
			if !utils.IsInstalled(versionsDir, alias) {
				logrus.Infof("Alias '%s' is not installed. Skipping upgrade.", alias)
				continue
			}

			release, err := releases.ResolveVersion(alias, cacheFilePath)
			if err != nil {
				logrus.Errorf("Error resolving %s: %v", alias, err)
				continue
			}

			remoteIdentifier := releases.GetReleaseIdentifier(release, alias)
			installedIdentifier, err := releases.GetInstalledReleaseIdentifier(versionsDir, alias)
			if err == nil && installedIdentifier == remoteIdentifier {
				fmt.Printf("%s %s is already up-to-date (%s)\n", utils.WarningIcon(), alias, installedIdentifier)
				continue
			}

			assetURL, assetPattern, err := releases.GetAssetURL(release)
			if err != nil {
				logrus.Errorf("Error getting asset URL for %s: %v", alias, err)
				continue
			}

			checksumURL, err := releases.GetChecksumURL(release, assetPattern)
			if err != nil {
				logrus.Errorf("Error getting checksum URL for %s: %v", alias, err)
				continue
			}

			fmt.Printf("%s %s upgrading to new identifier %s...\n", utils.InfoIcon(), utils.WhiteText(alias), remoteIdentifier)

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
				},
				func(phase string) {
					s.Prefix = phase + " "
					s.Suffix = ""
				},
			)
			s.Stop()
			if err != nil {
				logrus.Errorf("Upgrade failed for %s: %v", alias, err)
				continue
			}
			fmt.Printf("%s %s upgraded successfully!\n", utils.SuccessIcon(), utils.WhiteText(alias))
		}
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}
