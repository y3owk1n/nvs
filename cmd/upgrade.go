package cmd

import (
	"context"
	"errors"
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

// ErrInvalidUpgradeTarget is returned when an invalid upgrade target is specified.
var ErrInvalidUpgradeTarget = errors.New("upgrade can only be performed for 'stable' or 'nightly'")

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
	RunE:    RunUpgrade,
}

// RunUpgrade executes the upgrade command.
func RunUpgrade(cmd *cobra.Command, args []string) error {
	const (
		SpinnerSpeed  = 100
		InitialSuffix = " 0%"
	)

	logrus.Debug("Starting upgrade command")

	// Create a context with a 30-minute timeout for the upgrade process.
	ctx, cancel := context.WithTimeout(cmd.Context(), TimeoutMinutes*time.Minute)
	defer cancel()

	// Determine which aliases (versions) to upgrade.
	// If no argument is given, upgrade both stable and "nightly".
	var aliases []string
	if len(args) == 0 {
		aliases = []string{stable, nightly}
	} else {
		if args[0] != stable && args[0] != nightly {
			return ErrInvalidUpgradeTarget
		}

		aliases = []string{args[0]}
	}

	// Process each alias (version) for upgrade.
	for _, alias := range aliases {
		logrus.Debugf("Processing alias: %s", alias)

		var printErr error

		// Check if the alias is installed.
		if !helpers.IsInstalled(VersionsDir, alias) {
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
		release, err := releases.ResolveVersion(alias, CacheFilePath)
		if err != nil {
			logrus.Errorf("Error resolving %s: %v", alias, err)

			continue
		}

		logrus.Debugf("Resolved version for %s: %+v", alias, release)

		// Compare installed and remote identifiers.
		remoteIdentifier := releases.GetReleaseIdentifier(release, alias)

		installedIdentifier, err := releases.GetInstalledReleaseIdentifier(VersionsDir, alias)
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
		versionPath := filepath.Join(VersionsDir, alias)
		logrus.Debug("Computed version path: ", versionPath)

		// Create a backup of the existing version for rollback
		backupPath := versionPath + ".backup"
		if _, err := os.Stat(versionPath); err == nil {
			logrus.Debug("Creating backup of existing version")
			err = os.Rename(versionPath, backupPath)
			if err != nil {
				return fmt.Errorf("failed to create backup of version %s: %w", alias, err)
			}
			defer func() {
				// Cleanup: remove backup if upgrade succeeds, restore if it fails
				if _, statErr := os.Stat(versionPath); statErr == nil {
					// Upgrade succeeded, remove backup
					os.RemoveAll(backupPath)
				} else {
					// Upgrade failed, restore backup
					os.Rename(backupPath, versionPath)
				}
			}()
		}

		// Download and install the upgrade.
		err = installer.DownloadAndInstall(
			ctx,
			VersionsDir,
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

	return nil
}

// init registers the upgradeCmd with the root command.
func init() {
	rootCmd.AddCommand(upgradeCmd)
}
