package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/briandowns/spinner"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/helpers"
	"github.com/y3owk1n/nvs/pkg/installer"
	"github.com/y3owk1n/nvs/pkg/releases"
)

// Constants for upgrade types.
const (
	stable  = "stable"
	nightly = "nightly"
)

// upgradeCmd represents the "upgrade" command (aliases: up).
// It upgrades the installed stable and/or nightly versions of Neovim.
// If no argument is provided, both stable and nightly versions are upgraded (if installed).
// Only stable or "nightly" are accepted as arguments.
// The command fetches the latest release data, compares remote and installed identifiers,
// and if an upgrade is available, it downloads and installs the new version.
//
// Example usage:
//
//	nvs upgrade
//	nvs upgrade stable
//	nvs up nightly
var upgradeCmd = &cobra.Command{
	Use:     "upgrade [stable|nightly]",
	Aliases: []string{"up"},
	Short:   "Upgrade installed stable and/or nightly versions",
	Long:    "Upgrades the installed stable and/or nightly versions. If no argument is provided, both stable and nightly are upgraded (if installed).",
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		const TimeoutMin = 30
		const SpinnerSpeed = 100
		const InitialSuffix = " 0%"
		logrus.Debug("Starting upgrade command")

		// Create a context with a 30-minute timeout for the upgrade process.
		ctx, cancel := context.WithTimeout(cmd.Context(), TimeoutMin*time.Minute)
		defer cancel()

		// Determine which aliases (versions) to upgrade.
		// If no argument is given, upgrade both stable and "nightly".
		var aliases []string
		if len(args) == 0 {
			aliases = []string{stable, nightly}
		} else {
			if args[0] != stable && args[0] != nightly {
				logrus.Fatalf("Upgrade can only be performed for 'stable' or 'nightly'")
			}
			aliases = []string{args[0]}
		}

		// Process each alias (version) for upgrade.
		for _, alias := range aliases {
			logrus.Debugf("Processing alias: %s", alias)

			var printErr error

			// Check if the alias is installed.
			if !helpers.IsInstalled(versionsDir, alias) {
				logrus.Debugf("'%s' is not installed. Skipping upgrade.", alias)
				_, printErr = fmt.Fprintf(os.Stdout,
					"%s %s %s\n",
					helpers.WarningIcon(),
					helpers.CyanText(alias),
					helpers.WhiteText("is not installed. Skipping upgrade."),
				)
				if printErr != nil {
					logrus.Warnf("Failed to write to stdout: %v", printErr)
				}

				continue
			}

			// Resolve the remote release for the given alias.
			release, err := releases.ResolveVersion(alias, cacheFilePath)
			if err != nil {
				logrus.Errorf("Error resolving %s: %v", alias, err)

				continue
			}
			logrus.Debugf("Resolved version for %s: %+v", alias, release)

			// Compare installed and remote identifiers.
			remoteIdentifier := releases.GetReleaseIdentifier(release, alias)
			installedIdentifier, err := releases.GetInstalledReleaseIdentifier(versionsDir, alias)
			if err == nil && installedIdentifier == remoteIdentifier {
				logrus.Debugf("%s is already up-to-date (%s)", alias, installedIdentifier)
				_, printErr = fmt.Fprintf(os.Stdout,
					"%s %s %s %s\n",
					helpers.WarningIcon(),
					helpers.CyanText(alias),
					helpers.WhiteText("is already up-to-date"),
					helpers.CyanText("("+installedIdentifier+")"),
				)
				if printErr != nil {
					logrus.Warnf("Failed to write to stdout: %v", printErr)
				}

				continue
			}

			// Retrieve asset and checksum URLs for the upgrade.
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

			// Notify the user about the upgrade.
			_, printErr = fmt.Fprintf(os.Stdout,
				"%s %s %s %s...\n",
				helpers.InfoIcon(),
				helpers.CyanText(alias),
				helpers.WhiteText("upgrading to new identifier"),
				helpers.CyanText(remoteIdentifier),
			)
			if printErr != nil {
				logrus.Warnf("Failed to write to stdout: %v", printErr)
			}
			logrus.Debugf("Starting upgrade for %s to identifier %s", alias, remoteIdentifier)

			// Create and start a spinner to show progress.
			spinner := spinner.New(spinner.CharSets[14], SpinnerSpeed*time.Millisecond)
			spinner.Suffix = InitialSuffix
			spinner.Start()

			// Compute the path where the version is installed.
			versionPath := filepath.Join(versionsDir, alias)
			logrus.Debug("Computed version path: ", versionPath)

			logrus.Debug("Removing the old version")
			err = os.RemoveAll(versionPath)
			if err != nil {
				logrus.Fatalf("Failed to uninstall version %s: %v", alias, err)
			}

			// Download and install the upgrade.
			err = installer.DownloadAndInstall(
				ctx,
				versionsDir,
				alias,
				assetURL,
				checksumURL,
				remoteIdentifier,
				func(progress int) {
					spinner.Suffix = fmt.Sprintf(" %d%%", progress)
				},
				func(phase string) {
					if phase != "" {
						spinner.Prefix = phase + " "
						spinner.Suffix = ""
					}
				},
			)
			spinner.Stop()
			if err != nil {
				logrus.Errorf("Upgrade failed for %s: %v", alias, err)

				continue
			}
			// Inform the user that the upgrade succeeded.
			_, printErr = fmt.Fprintf(os.Stdout,
				"%s %s %s\n",
				helpers.SuccessIcon(),
				helpers.CyanText(alias),
				helpers.WhiteText("upgraded successfully!"),
			)
			if printErr != nil {
				logrus.Warnf("Failed to write to stdout: %v", printErr)
			}
			logrus.Debugf("%s upgraded successfully", alias)
		}
	},
}

// init registers the upgradeCmd with the root command.
func init() {
	rootCmd.AddCommand(upgradeCmd)
}
