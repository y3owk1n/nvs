package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/platform"
	"github.com/y3owk1n/nvs/internal/ui"
)

// NightlyHistoryEntry represents a single nightly version in history.
//
//nolint:tagliatelle
type NightlyHistoryEntry struct {
	CommitHash  string    `json:"commit_hash"`
	InstalledAt time.Time `json:"installed_at"`
	TagName     string    `json:"tag_name"`
}

// noRollbackCurrentRow is the sentinel passed to ui.Table.Current()
// when no nightly history entry is the live nightly. The internal
// table sentinel lives in package ui/table and is unexported, so we
// mirror it here rather than reach inside. -1 is the same value the
// table uses internally.
const noRollbackCurrentRow = -1

// NightlyHistory holds the history of nightly versions.
type NightlyHistory struct {
	Entries []NightlyHistoryEntry `json:"entries"`
	Limit   int                   `json:"limit"`
}

// rollbackCmd represents the "rollback" command.
var rollbackCmd = &cobra.Command{
	Use:   "rollback [index]",
	Short: "Rollback to a previous nightly version",
	Long: `Rollback to a previous nightly version.
Without arguments, lists available nightly versions to rollback to.
With an index, rolls back to that specific version.`,
	Args: cobra.MaximumNArgs(1),
	RunE: RunRollback,
}

// RunRollback executes the rollback command.
func RunRollback(cmd *cobra.Command, args []string) error {
	history, err := loadNightlyHistory()
	if err != nil {
		return fmt.Errorf("failed to load nightly history: %w", err)
	}

	// Filter out entries that no longer exist on disk
	validEntries := make([]NightlyHistoryEntry, 0, len(history.Entries))
	for _, entry := range history.Entries {
		backupDir := filepath.Join(
			GetVersionsDir(),
			"nightly-"+shortHash(entry.CommitHash, constants.ShortHashLength),
		)

		_, err := os.Stat(backupDir)
		if err == nil {
			validEntries = append(validEntries, entry)
		} else {
			logrus.Debugf(
				"Removing orphaned history entry: %s",
				shortHash(entry.CommitHash, constants.ShortHashLength),
			)
		}
	}

	// Update history if entries were removed
	if len(validEntries) != len(history.Entries) {
		history.Entries = validEntries

		saveErr := saveNightlyHistory(history)
		if saveErr != nil {
			logrus.Warnf("Failed to save cleaned history: %v", saveErr)
		}
	}

	if len(history.Entries) == 0 {
		ui.Message.Infof("No nightly history available.")
		ui.Message.Infof("Run 'nvs upgrade nightly' to start tracking versions.")

		return nil
	}

	// If no index provided, list available versions
	if len(args) == 0 {
		return listNightlyHistory(history)
	}

	// Parse index and rollback
	index, err := strconv.Atoi(args[0])
	if err != nil || index < 0 || index >= len(history.Entries) {
		return fmt.Errorf("%w: %s (use 0-%d)", ErrInvalidIndex, args[0], len(history.Entries)-1)
	}

	entry := history.Entries[index]
	logrus.Debugf("Rolling back to nightly commit %s", entry.CommitHash)

	// The rollback is essentially switching to a stored nightly version
	// We need to check if this version directory still exists
	// Safety check against TOCTOU races (directory may have been deleted since filtering)
	nightlyDir := filepath.Join(
		GetVersionsDir(),
		"nightly-"+shortHash(entry.CommitHash, constants.ShortHashLength),
	)

	_, err = os.Stat(nightlyDir)
	if os.IsNotExist(err) {
		return fmt.Errorf(
			"%w: %s",
			ErrNightlyVersionNotExists,
			shortHash(entry.CommitHash, constants.ShortHashLength),
		)
	}

	// Create symlink to this version as "nightly"
	currentNightly := filepath.Join(GetVersionsDir(), "nightly")

	// Get current nightly's commit hash before removing (to potentially back it up)
	currentCommit, err := GetVersionService().GetInstalledVersionIdentifier("nightly")
	if err != nil {
		logrus.Debugf("Could not get current nightly identifier: %v", err)
	}

	// Remove current nightly symlink/directory if it exists
	info, statErr := os.Lstat(currentNightly)
	if statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			// It's a symlink, just remove it
			err := os.Remove(currentNightly)
			if err != nil {
				logrus.Warnf("Failed to remove symlink: %v", err)
			}
		} else if info.IsDir() {
			// It's a real directory - back it up so the user can
			// recover if rollback turns out to be the wrong move.
			//
			// If we have a commit hash, name the backup after it
			// (matches the upgrade-time layout). If we don't (e.g.
			// the version.txt read failed), fall back to a
			// timestamped name so we still preserve the directory
			// rather than silently destroying it.
			backupDir := resolveNightlyBackupDir(currentCommit)

			// Atomically claim the backup slot via os.Rename.
			// The previous code did a Stat on backupDir and
			// then either renamed (if missing) or removed
			// (if present), but the Stat + rename pair is a
			// TOCTOU window: two concurrent rollbacks could
			// both see the backup missing and both try to
			// rename the same currentNightly. Whichever
			// rename ran second would fail.
			//
			// os.Rename is itself atomic at the filesystem
			// level on POSIX, so a single attempt is enough
			// to decide: success (we created the backup),
			// or "already exists" (the backup is from a
			// previous rollback, just drop the current
			// symlink/dir), or any other error (propagate).
			var backupErr error

			renameErr := os.Rename(currentNightly, backupDir)
			switch {
			case renameErr == nil:
				logrus.Debugf("Backed up current nightly to %s", backupDir)
			case os.IsExist(renameErr):
				// Backup already exists from a previous run;
				// safe to remove the current nightly.
				rmErr := os.RemoveAll(currentNightly)
				if rmErr != nil {
					return fmt.Errorf("failed to remove current nightly: %w", rmErr)
				}
			default:
				backupErr = fmt.Errorf("failed to backup current nightly: %w", renameErr)
			}

			if backupErr != nil {
				return backupErr
			}
		}
	}

	// Create symlink from nightly -> nightly-{hash}
	err = platform.UpdateSymlink(nightlyDir, currentNightly, true)
	if err != nil {
		return fmt.Errorf("failed to create nightly symlink: %w", err)
	}

	// Add the version we're rolling back FROM to history (for roll-forward capability)
	if currentCommit != "" && currentCommit != entry.CommitHash {
		histErr := AddNightlyToHistory(currentCommit, "nightly")
		if histErr != nil {
			logrus.Warnf("Failed to add previous nightly to history: %v", histErr)
		}
	}

	ui.Message.Successf(
		"Rolled back to nightly %s (from %s)",
		shortHash(entry.CommitHash, constants.ShortHashLength),
		entry.InstalledAt.Format("2006-01-02 15:04"),
	)

	return nil
}

func listNightlyHistory(history *NightlyHistory) error {
	// Get current nightly commit to show indicator
	currentCommit, _ := GetVersionService().GetInstalledVersionIdentifier("nightly")

	tbl := ui.Table.New("Index", "Commit", "Installed At", "Status")

	currentRowIdx := noRollbackCurrentRow

	for index, entry := range history.Entries {
		short := shortHash(entry.CommitHash, constants.ShortHashLength)

		isCurrent := currentCommit != "" &&
			shortHash(currentCommit, constants.ShortHashLength) == short

		indexCell := ui.Message.Text(strconv.Itoa(index))
		statusCell := ""

		if isCurrent {
			indexCell = ui.Message.Highlight("→ " + strconv.Itoa(index))
			statusCell = ui.Message.Highlight("← current")
			// ui.Table.Current() can only highlight one row at a time, so
			// record the last current match (in practice the newest
			// history entry is the live nightly, so a single match is
			// the norm; the design still tolerates duplicates by
			// per-cell highlighting the older ones).
			currentRowIdx = index
		}

		tbl.Row(indexCell, short, entry.InstalledAt.Format("2006-01-02 15:04"), statusCell)
	}

	if currentRowIdx != noRollbackCurrentRow {
		tbl.Current(currentRowIdx)
	}

	_, _ = fmt.Fprint(os.Stdout, ui.Banner.Logo())
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprint(os.Stdout, tbl.Render(ui.Style.Palette()))

	_, _ = fmt.Fprintln(os.Stdout)

	ui.Message.Mutedf("Use 'nvs rollback <index>' to rollback to a specific version.")

	return nil
}

// AddNightlyToHistory adds a nightly version to the history.
// It automatically trims the history to the configured limit and removes old nightly directories.
func AddNightlyToHistory(commitHash, tagName string) error {
	history, err := loadNightlyHistory()
	if err != nil {
		// Create new history if it doesn't exist
		history = &NightlyHistory{
			Entries: []NightlyHistoryEntry{},
			Limit:   constants.DefaultRollbackLimit,
		}
	}

	// Remove any existing entry with the same commit hash (to avoid
	// duplicates). Compare on the full commit hash: shortHash() (the
	// first ShortHashLength hex chars) has a ~1-in-16M collision
	// probability and would silently drop the wrong entry if two
	// distinct commits ever shared their leading 7 chars. Callers
	// have already passed the trimmed full hash, so a direct
	// string compare is sufficient.
	dedupedEntries := make([]NightlyHistoryEntry, 0, len(history.Entries))
	for _, entry := range history.Entries {
		if entry.CommitHash != commitHash {
			dedupedEntries = append(dedupedEntries, entry)
		}
	}

	history.Entries = dedupedEntries

	// Add new entry at the beginning
	entry := NightlyHistoryEntry{
		CommitHash:  commitHash,
		InstalledAt: time.Now(),
		TagName:     tagName,
	}
	history.Entries = append([]NightlyHistoryEntry{entry}, history.Entries...)

	// Trim to limit
	if len(history.Entries) > history.Limit {
		// Clean up old nightly directories
		for i := history.Limit; i < len(history.Entries); i++ {
			oldDir := filepath.Join(
				GetVersionsDir(),
				"nightly-"+shortHash(history.Entries[i].CommitHash, constants.ShortHashLength),
			)

			logrus.Debugf("Removing old nightly backup: %s", oldDir)

			err := os.RemoveAll(oldDir)
			if err != nil {
				logrus.Warnf("Failed to remove old nightly %s: %v", oldDir, err)
			}
		}

		history.Entries = history.Entries[:history.Limit]
	}

	return saveNightlyHistory(history)
}

// GetNightlyHistory returns the nightly history.
func GetNightlyHistory() (*NightlyHistory, error) {
	return loadNightlyHistory()
}

func loadNightlyHistory() (*NightlyHistory, error) {
	historyPath := filepath.Join(filepath.Dir(GetVersionsDir()), constants.NightlyHistoryFile)

	data, err := os.ReadFile(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &NightlyHistory{
				Entries: []NightlyHistoryEntry{},
				Limit:   constants.DefaultRollbackLimit,
			}, nil
		}

		return nil, err
	}

	var history NightlyHistory

	err = json.Unmarshal(data, &history)
	if err != nil {
		return nil, err
	}

	// Sort by installed_at descending (most recent first)
	slices.SortFunc(history.Entries, func(a, b NightlyHistoryEntry) int {
		return b.InstalledAt.Compare(a.InstalledAt)
	})

	return &history, nil
}

func saveNightlyHistory(history *NightlyHistory) error {
	historyPath := filepath.Join(filepath.Dir(GetVersionsDir()), constants.NightlyHistoryFile)

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file first for atomicity. Using a sibling
	// .tmp file ensures os.Rename is atomic on POSIX (no
	// cross-filesystem move required).
	tempPath := historyPath + ".tmp"

	err = os.WriteFile(tempPath, data, constants.FilePerm)
	if err != nil {
		return err
	}

	// Best-effort cleanup of the temp file on any failure from
	// here on. The os.Stat guard avoids an error log when the
	// rename has already succeeded (the .tmp file no longer
	// exists at that path).
	defer func() {
		_, statErr := os.Stat(tempPath)
		if statErr == nil {
			_ = os.Remove(tempPath)
		}
	}()

	// fsync before rename so a power loss between WriteFile
	// returning and the rename completing cannot leave the
	// renamed file with zero bytes (or unflushed data) on disk.
	tempFile, openErr := os.OpenFile(tempPath, os.O_RDWR, constants.FilePerm)
	if openErr == nil {
		syncErr := tempFile.Sync()
		if syncErr != nil {
			logrus.Warnf("Failed to fsync temp history file: %v", syncErr)
		}

		_ = tempFile.Close()
	}

	// Atomic rename.
	renameErr := os.Rename(tempPath, historyPath)
	if renameErr != nil {
		return fmt.Errorf("rename temp history file: %w", renameErr)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(rollbackCmd)
}

// resolveNightlyBackupDir returns the path that should be used as the
// backup slot for the current nightly. When a commit hash is known,
// the path is the upgrade-time layout (nightly-{shortHash}); when the
// hash can't be read, the path is timestamped so the user still has
// something to recover from rather than losing the directory entirely.
func resolveNightlyBackupDir(currentCommit string) string {
	if currentCommit != "" {
		return filepath.Join(
			GetVersionsDir(),
			"nightly-"+shortHash(currentCommit, constants.ShortHashLength),
		)
	}

	timestamped := filepath.Join(
		GetVersionsDir(),
		"nightly-"+time.Now().UTC().Format("20060102-150405"),
	)

	logrus.Warnf(
		"Current nightly has no readable identifier; backing up to timestamped directory %s",
		timestamped,
	)

	return timestamped
}
