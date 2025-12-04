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

// upgradeAlias handles the upgrade process for a single alias, including backup and cleanup.
func upgradeAlias(
	ctx context.Context,
	alias string,
	assetURL, checksumURL, remoteIdentifier string,
	progressCallback func(int),
	phaseCallback func(string),
) error {
	versionPath := filepath.Join(GetVersionsDir(), alias)
	backupPath := versionPath + ".backup"

	// Create backup if version exists
	_, err := os.Stat(versionPath)
	if err == nil {
		logrus.Debug("Creating backup of existing version")

		err = os.Rename(versionPath, backupPath)
		if err != nil {
			return fmt.Errorf("failed to create backup of version %s: %w", alias, err)
		}

		defer func() {
			// Cleanup: remove backup if upgrade succeeds, restore if it fails
			_, statErr := os.Stat(versionPath)
			if statErr == nil {
				// Upgrade succeeded, remove backup
				err := os.RemoveAll(backupPath)
				if err != nil {
					logrus.Warnf("Failed to remove backup directory %s: %v", backupPath, err)
				}
			} else {
				// Upgrade failed, restore backup
				err := os.Rename(backupPath, versionPath)
				if err != nil {
					logrus.Warnf("Failed to restore backup from %s to %s: %v", backupPath, versionPath, err)
				}
			}
		}()
	}

	// Download and install the upgrade.
	err = installer.DownloadAndInstall(
		ctx,
		GetVersionsDir(),
		alias,
		assetURL,
		checksumURL,
		remoteIdentifier,
		progressCallback,
		phaseCallback,
	)

	return err
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

		// Create and start a spinner to show progress.
		spinner := spinner.New(spinner.CharSets[14], SpinnerSpeed*time.Millisecond)
		spinner.Suffix = InitialSuffix
		spinner.Start()

		err := GetVersionService().Upgrade(ctx, alias, func(phase string, progress int) {
			if phase != "" {
				spinner.Prefix = phase + " "
				spinner.Suffix = ""
			}
			spinner.Suffix = fmt.Sprintf(" %d%%", progress)
		})

		if err != nil {
			spinner.Stop()
			if err.Error() == "not installed" { // Should use errors.Is
				logrus.Debugf("'%s' is not installed. Skipping upgrade.", alias)
				_, printErr := fmt.Fprintf(os.Stdout,
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
			if err.Error() == "already up-to-date" { // Should use errors.Is
				logrus.Debugf("%s is already up-to-date", alias)
				_, printErr := fmt.Fprintf(os.Stdout,
					"%s %s %s\n",
					helpers.WarningIcon(),
					helpers.CyanText(alias),
					helpers.WhiteText("is already up-to-date"),
				)
				if printErr != nil {
					logrus.Warnf("Failed to write to stdout: %v", printErr)
				}
				continue
			}

			logrus.Errorf("Upgrade failed for %s: %v", alias, err)
			return fmt.Errorf("upgrade failed for %s: %w", alias, err)
		}

		spinner.Stop()

		// Inform the user that the upgrade succeeded.
		_, printErr := fmt.Fprintf(os.Stdout,
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
