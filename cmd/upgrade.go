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
	appversion "github.com/y3owk1n/nvs/internal/app/version"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/ui"
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
	RunE:    RunUpgrade,
}

// RunUpgrade executes the upgrade command.
func RunUpgrade(cmd *cobra.Command, args []string) error {
	logrus.Debug("Starting upgrade command")

	// Create a context with a 30-minute timeout for the upgrade process.
	ctx, cancel := context.WithTimeout(cmd.Context(), constants.TimeoutMinutes*time.Minute)
	defer cancel()

	// Determine which aliases (versions) to upgrade.
	// If no argument is given, upgrade both stable and "nightly".
	var aliases []string
	if len(args) == 0 {
		aliases = []string{constants.Stable, constants.Nightly}
	} else {
		if args[0] != constants.Stable && args[0] != constants.Nightly {
			return ErrInvalidUpgradeTarget
		}

		aliases = []string{args[0]}
	}

	// Process each alias (version) for upgrade.
	for _, alias := range aliases {
		logrus.Debugf("Processing alias: %s", alias)

		// For nightly, get current commit hash before upgrade (for changelog and rollback)
		var (
			oldCommitHash string
			backupDir     string
			backupCreated bool
		)

		if alias == constants.Nightly {
			oldCommitHash, _ = GetVersionService().GetInstalledVersionIdentifier(constants.Nightly)
			logrus.Debugf("Current nightly commit: %s", oldCommitHash)

			// Backup current nightly for rollback support
			if oldCommitHash != "" {
				nightlyDir := filepath.Join(GetVersionsDir(), constants.Nightly)
				backupDir = filepath.Join(
					GetVersionsDir(),
					"nightly-"+shortHash(oldCommitHash, constants.ShortHashLength),
				)

				// Only backup if the backup doesn't already exist
				var statErr error

				_, statErr = os.Stat(backupDir)
				if os.IsNotExist(statErr) {
					var statErr2 error

					_, statErr2 = os.Stat(nightlyDir)
					if statErr2 == nil {
						// Copy directory (rename would break the current install)
						copyErr := copyDir(nightlyDir, backupDir)
						if copyErr != nil {
							logrus.Warnf("Failed to backup nightly for rollback: %v", copyErr)
						} else {
							logrus.Debugf("Backed up nightly to %s", backupDir)

							backupCreated = true
						}
					}
				}
			}
		}

		// Create and start a spinner to show progress.
		progressSpinner := spinner.New(
			spinner.CharSets[14],
			constants.SpinnerSpeed*time.Millisecond,
		)
		progressSpinner.Prefix = ui.InfoIcon() + " "
		progressSpinner.Suffix = " Checking for updates..."
		progressSpinner.Start()

		err := GetVersionService().Upgrade(ctx, alias, func(phase string, progress int) {
			progressSpinner.Suffix = " " + ui.FormatPhaseProgress(phase, progress)
		})
		if err != nil {
			progressSpinner.Stop()

			if errors.Is(err, appversion.ErrNotInstalled) {
				logrus.Debugf("'%s' is not installed. Skipping upgrade.", alias)

				_, printErr := fmt.Fprintf(os.Stdout,
					"%s %s %s\n",
					ui.WarningIcon(),
					ui.CyanText(alias),
					ui.WhiteText("is not installed. Skipping upgrade."),
				)
				if printErr != nil {
					logrus.Warnf("Failed to write to stdout: %v", printErr)
				}

				continue
			}

			if errors.Is(err, appversion.ErrAlreadyUpToDate) {
				logrus.Debugf("%s is already up-to-date", alias)

				_, printErr := fmt.Fprintf(os.Stdout,
					"%s %s %s\n",
					ui.WarningIcon(),
					ui.CyanText(alias),
					ui.WhiteText("is already up-to-date"),
				)
				if printErr != nil {
					logrus.Warnf("Failed to write to stdout: %v", printErr)
				}

				continue
			}

			// Clean up backup on failure
			if backupCreated {
				removeErr := os.RemoveAll(backupDir)
				if removeErr != nil {
					logrus.Warnf("Failed to clean up backup on upgrade failure: %v", removeErr)
				}
			}

			logrus.Errorf("Upgrade failed for %s: %v", alias, err)

			return fmt.Errorf("upgrade failed for %s: %w", alias, err)
		}

		progressSpinner.Stop()

		// For nightly upgrades, add OLD version to history for rollback support
		if alias == constants.Nightly && oldCommitHash != "" {
			// Add the old commit (the one we backed up) to history
			histErr := AddNightlyToHistory(oldCommitHash, constants.Nightly)
			if histErr != nil {
				logrus.Warnf("Failed to add nightly to history: %v", histErr)
			}
		}

		// Inform the user that the upgrade succeeded.
		_, printErr := fmt.Fprintf(os.Stdout,
			"%s %s %s\n",
			ui.SuccessIcon(),
			ui.CyanText(alias),
			ui.WhiteText("upgraded successfully!"),
		)
		if printErr != nil {
			logrus.Warnf("Failed to write to stdout: %v", printErr)
		}

		// For nightly, show changelog
		if alias == constants.Nightly && oldCommitHash != "" {
			nightlyRelease, findErr := GetVersionService().FindNightly(ctx)
			if findErr == nil && nightlyRelease.CommitHash() != oldCommitHash {
				_ = ShowChangelog(ctx, oldCommitHash, nightlyRelease.CommitHash())
			}
		}

		logrus.Debugf("%s upgraded successfully", alias)
	}

	return nil
}

// init registers the upgradeCmd with the root command.
func init() {
	rootCmd.AddCommand(upgradeCmd)
}
