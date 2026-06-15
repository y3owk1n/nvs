package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/app/versionsvc"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/infra/filesystem"
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
	aliases, err := resolveUpgradeAliases(cmd, args)
	if err != nil {
		return err
	}

	// Process each alias (version) for upgrade.
	for _, alias := range aliases {
		logrus.Debugf("Processing alias: %s", alias)

		upgradeErr := runOneUpgrade(ctx, alias)
		if upgradeErr != nil {
			return upgradeErr
		}
	}

	return nil
}

// resolveUpgradeAliases implements the argument + --pick
// parsing for upgrade. It returns the list of alias names
// to upgrade in the order they should be processed, or an
// error if the input is invalid.
func resolveUpgradeAliases(cmd *cobra.Command, args []string) ([]string, error) {
	pick, _ := cmd.Flags().GetBool("pick")
	if pick {
		return pickUpgradeAliases()
	}

	// If no argument is given, upgrade both stable and "nightly".
	if len(args) == 0 {
		return []string{constants.Stable, constants.Nightly}, nil
	}

	if args[0] != constants.Stable && args[0] != constants.Nightly {
		return nil, ErrInvalidUpgradeTarget
	}

	return []string{args[0]}, nil
}

// stableNightlyAliasCount is the number of alias slots
// the upgrade --pick picker considers. Both stable and
// nightly are valid upgrade targets; nothing else is
// (the upgrade command does not accept arbitrary tags).
const stableNightlyAliasCount = 2

// pickUpgradeAliases shows the interactive picker for the
// installed stable / nightly aliases. If only one of the
// two is installed, it is used directly; if both are
// installed, the user picks one.
func pickUpgradeAliases() ([]string, error) {
	available := make([]string, 0, stableNightlyAliasCount)

	if GetVersionService().IsVersionInstalled(constants.Stable) {
		available = append(available, constants.Stable)
	}

	if GetVersionService().IsVersionInstalled(constants.Nightly) {
		available = append(available, constants.Nightly)
	}

	if len(available) == 0 {
		return nil, fmt.Errorf("%w for upgrade", ErrNoVersionsAvailable)
	}

	if len(available) == 1 {
		return available, nil
	}

	items := make([]ui.SelectItem, 0, len(available))
	for _, alias := range available {
		items = append(items, ui.SelectItem{Label: alias})
	}

	selected, err := ui.Picker.NewPicker(nil, nil).Select("Select version to upgrade", items)
	if err != nil {
		if errors.Is(err, ui.Picker.ErrCanceled()) {
			ui.Message.Warnf("Selection canceled.")

			return nil, nil
		}

		return nil, fmt.Errorf("prompt failed: %w", err)
	}

	return []string{selected}, nil
}

// runOneUpgrade performs the upgrade of a single alias:
// the optional pre-upgrade backup of nightly, the spinner-
// driven install, the post-upgrade history entry, and the
// per-alias success message.
//
// On a non-fatal error (alias not installed, already
// up-to-date) it returns nil so the caller can continue
// with the next alias. On any other error it returns the
// wrapped error so the caller can short-circuit and the
// caller can clean up the backup.
func runOneUpgrade(ctx context.Context, alias string) error {
	// For nightly, get current commit hash before upgrade (for changelog and rollback)
	var (
		oldCommitHash string
		backupDir     string
		backupCreated bool
	)

	if alias == constants.Nightly {
		oldCommitHash = prepareNightlyBackup(&backupDir, &backupCreated)
	}

	// Run the upgrade inside a closure so that a
	// `defer progressSpinner.Stop()` is scoped to this
	// iteration only. This ensures the spinner is always
	// stopped, even on panic, before the loop continues to
	// the next alias.
	err := func() error {
		progressSpinner := ui.NewSpinner(
			os.Stdout,
			constants.SpinnerSpeed*time.Millisecond,
		)
		progressSpinner.SetPrefix(ui.Message.Icons().Info + " ")
		progressSpinner.SetSuffix(" Checking for updates...")
		progressSpinner.Start()

		defer progressSpinner.Stop()

		return GetVersionService().Upgrade(ctx, alias, func(phase string, progress int) {
			progressSpinner.SetSuffix(" " + ui.FormatPhaseProgress(phase, progress))
		})
	}()
	if err != nil {
		if errors.Is(err, versionsvc.ErrNotInstalled) {
			logrus.Debugf("'%s' is not installed. Skipping upgrade.", alias)

			ui.Message.Warnf("%s is not installed. Skipping upgrade.", ui.Message.Accent(alias))

			return nil
		}

		if errors.Is(err, versionsvc.ErrAlreadyUpToDate) {
			logrus.Debugf("%s is already up-to-date", alias)

			ui.Message.Warnf("%s is already up-to-date", ui.Message.Accent(alias))

			return nil
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

	// For nightly upgrades, add OLD version to history for rollback support
	if alias == constants.Nightly && oldCommitHash != "" {
		// Add the old commit (the one we backed up) to history
		histErr := AddNightlyToHistory(oldCommitHash, constants.Nightly)
		if histErr != nil {
			logrus.Warnf("Failed to add nightly to history: %v", histErr)
		}
	}

	// Inform the user that the upgrade succeeded.
	ui.Message.Successf("%s upgraded successfully!", ui.Message.Accent(alias))

	// For nightly, show changelog
	if alias == constants.Nightly && oldCommitHash != "" {
		showUpgradeChangelog(ctx, oldCommitHash)
	}

	logrus.Debugf("%s upgraded successfully", alias)

	return nil
}

// prepareNightlyBackup reads the current nightly commit
// hash and creates a rollback backup of the nightly
// directory under the per-version lock. It returns the old
// commit hash (or "" if it could not be read) and stores
// the backup directory + a "did we actually create the
// backup?" flag via the out parameters so the caller can
// roll the backup back on a failed upgrade.
//
// The function is split out of runOneUpgrade so the
// upgrade-loop body stays readable: the lock + sentinel
// dance is its own concern.
func prepareNightlyBackup(backupDir *string, backupCreated *bool) string {
	oldCommitHash, identifierErr := GetVersionService().GetInstalledVersionIdentifier(constants.Nightly)
	if identifierErr != nil {
		// Don't silently lose rollback safety: warn loudly so
		// the user knows the upgrade will proceed without a
		// backup.
		logrus.Warnf(
			"Could not read current nightly identifier; rollback backup will be skipped: %v",
			identifierErr,
		)

		return ""
	}

	logrus.Debugf("Current nightly commit: %s", oldCommitHash)

	// Backup current nightly for rollback support
	if oldCommitHash == "" {
		return ""
	}

	nightlyDir := filepath.Join(GetVersionsDir(), constants.Nightly)
	*backupDir = filepath.Join(
		GetVersionsDir(),
		"nightly-"+shortHash(oldCommitHash, constants.ShortHashLength),
	)

	// Atomically claim the backup slot. The previous
	// implementation did Stat + copyDir, which raced
	// when two nvs processes upgraded nightly at the
	// same time: both would observe the backup dir
	// missing and both would walk the copy, with
	// partially-overlapping writes producing a
	// corrupted backup.
	//
	// MkdirAll + a sentinel file opened with
	// O_CREATE|O_EXCL gives us a single, race-free
	// "did we win the claim?" decision. The winning
	// process performs the copy; losers treat the
	// backup as already done and skip the copy.
	//
	// The copy itself runs under the same per-version
	// lock that the installer uses, so a concurrent
	// use/uninstall/reinstall of nightly cannot mutate
	// nightlyDir mid-walk.
	backupErr := backupNightlyUnderLock(
		nightlyDir,
		*backupDir,
	)
	if backupErr != nil {
		logrus.Warnf(
			"Failed to backup nightly for rollback: %v",
			backupErr,
		)
	} else {
		logrus.Debugf("Backed up nightly to %s", *backupDir)

		*backupCreated = true
	}

	return oldCommitHash
}

// showUpgradeChangelog looks up the current nightly
// release and, if the commit hash differs from the old
// one, shows the changelog between them. The function
// swallows any error: a failed changelog lookup must not
// fail the upgrade itself, since the upgrade is the
// primary action the user asked for.
func showUpgradeChangelog(ctx context.Context, oldCommitHash string) {
	nightlyRelease, findErr := GetVersionService().FindNightly(ctx)
	if findErr == nil && nightlyRelease.CommitHash() != oldCommitHash {
		_ = ShowChangelog(ctx, oldCommitHash, nightlyRelease.CommitHash())
	}
}

// init registers the upgradeCmd with the root command.
func init() {
	rootCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().
		BoolP("pick", "p", false, "Launch interactive picker to select versions to upgrade")
}

// backupNightlyUnderLock copies nightlyDir to backupDir under the
// per-version lock used by the installer service, so a concurrent
// use/install/uninstall cannot mutate nightlyDir mid-walk.
//
// The backup is staged in a sibling temp directory together with
// the sentinel file, then atomically renamed into place. The
// sentinel lives in the same dir as the copy and is published to
// backupDir in the same rename that publishes the copy, so there
// is no window in which backupDir exists with content but no
// sentinel (the failure mode of the previous MkdirAll-then-rename
// implementation, where the atomic rename always failed with
// "file exists" on any backupDir that pre-existed).
func backupNightlyUnderLock(nightlyDir, backupDir string) error {
	lockPath := filepath.Join(
		GetVersionsDir(),
		fmt.Sprintf(".nvs-version-%s.lock", constants.Nightly),
	)
	lock := filesystem.NewFileLock(lockPath)

	err := lock.LockWithDefaultTimeout()
	if err != nil {
		return fmt.Errorf("acquire nightly lock: %w", err)
	}

	defer func() {
		unlockErr := lock.Unlock()
		if unlockErr != nil {
			logrus.Warnf("Failed to unlock nightly lock: %v", unlockErr)
		}
	}()

	// The sentinel lives in the temp dir, so it only appears in
	// backupDir after a successful rename. A completed backup is
	// therefore always identified by the presence of the sentinel.
	sentinel := filepath.Join(backupDir, ".nvs-backup-owner")

	_, statErr := os.Stat(sentinel)
	if statErr == nil {
		logrus.Debugf("Backup already claimed at %s; skipping copy", backupDir)

		return nil
	}

	if !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("check existing backup: %w", statErr)
	}

	// Stage the backup in a hidden temp dir next to backupDir. The
	// dot prefix keeps it out of the way of `nvs list` and similar
	// commands; the cleanup defer removes it on every exit path
	// (success, partial copy, panic).
	tempDir, err := os.MkdirTemp(GetVersionsDir(), ".nightly-backup-")
	if err != nil {
		return fmt.Errorf("create temp backup dir: %w", err)
	}

	defer func() {
		removeErr := os.RemoveAll(tempDir)
		if removeErr != nil {
			logrus.Warnf("Failed to clean up temp backup dir %s: %v", tempDir, removeErr)
		}
	}()

	// Claim the slot by writing the sentinel into the temp dir.
	// After the rename, the sentinel lives in backupDir where it
	// signals to future invocations that the backup is finalized.
	// O_EXCL is a defense-in-depth check against a collision with
	// another concurrent backup attempt; the per-version lock
	// already prevents this, but a second layer of safety is cheap.
	tempSentinel := filepath.Join(tempDir, ".nvs-backup-owner")

	sentinelFile, openErr := os.OpenFile(
		tempSentinel,
		os.O_CREATE|os.O_EXCL|os.O_WRONLY,
		constants.FilePerm,
	)
	if openErr != nil {
		return fmt.Errorf("claim backup slot: %w", openErr)
	}

	_ = sentinelFile.Close()

	// Match the source dir's mode. MkdirTemp creates with 0700;
	// the rename at the end preserves whatever mode we set here.
	srcInfo, err := os.Stat(nightlyDir)
	if err != nil {
		return fmt.Errorf("stat nightly dir: %w", err)
	}

	err = os.Chmod(tempDir, srcInfo.Mode())
	if err != nil {
		logrus.Debugf("Failed to set temp backup dir mode: %v", err)
	}

	err = copyDirContents(nightlyDir, tempDir)
	if err != nil {
		return fmt.Errorf("copy nightly: %w", err)
	}

	// Atomic publish. If backupDir exists (stale from a previous
	// interrupted run that left a partial dir but no sentinel),
	// remove it first; the rename then moves the fully-formed
	// backup (copy + sentinel) into backupDir in one step.
	_, err = os.Stat(backupDir)
	if err == nil {
		err = os.RemoveAll(backupDir)
		if err != nil {
			return fmt.Errorf("remove stale backup: %w", err)
		}
	}

	err = os.Rename(tempDir, backupDir)
	if err != nil {
		return fmt.Errorf("finalize backup: %w", err)
	}

	return nil
}
