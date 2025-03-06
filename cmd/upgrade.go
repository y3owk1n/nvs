package cmd

import (
	"fmt"

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
		// Determine which alias/aliases to upgrade.
		var aliases []string
		if len(args) == 0 {
			aliases = []string{"stable", "nightly"}
		} else {
			if args[0] != "stable" && args[0] != "nightly" {
				logrus.Fatalf("Upgrade can only be performed for 'stable' or 'nightly'")
			}
			aliases = []string{args[0]}
		}

		// Loop over each alias and upgrade if installed.
		for _, alias := range aliases {
			if !utils.IsInstalled(versionsDir, alias) {
				logrus.Infof("Alias '%s' is not installed. Skipping upgrade.", alias)
				continue
			}

			// Resolve the remote release using the cache.
			release, err := releases.ResolveVersion(alias, cacheFilePath)
			if err != nil {
				logrus.Errorf("Error resolving %s: %v", alias, err)
				continue
			}

			remoteIdentifier := releases.GetReleaseIdentifier(release, alias)
			installedIdentifier, err := releases.GetInstalledReleaseIdentifier(versionsDir, alias)
			if err == nil && installedIdentifier == remoteIdentifier {
				fmt.Printf("%s is already up-to-date (%s)\n", alias, installedIdentifier)
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

			fmt.Printf("Upgrading %s to new identifier %s...\n", alias, remoteIdentifier)
			if err := installer.DownloadAndInstall(versionsDir, alias, assetURL, checksumURL, remoteIdentifier); err != nil {
				logrus.Errorf("Upgrade failed for %s: %v", alias, err)
				continue
			}
			fmt.Printf("Upgrade successful for %s!\n", alias)
		}
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}
