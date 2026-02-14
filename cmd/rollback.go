package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
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
		_, printErr := fmt.Fprintf(os.Stdout, "%s No nightly history available.\n", ui.InfoIcon())
		if printErr != nil {
			logrus.Warnf("Failed to write to stdout: %v", printErr)
		}

		_, _ = fmt.Fprintf(
			os.Stdout,
			"%s Run 'nvs upgrade nightly' to start tracking versions.\n",
			ui.InfoIcon(),
		)

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

	nightlyDir := filepath.Join(
		GetVersionsDir(),
		"nightly-"+shortHash(entry.CommitHash, constants.ShortHashLength),
	)

	lockFile := nightlyDir + ".lock"

	lockFd, lockErr := platform.NewFileLock(lockFile)
	if lockErr != nil {
		return fmt.Errorf("failed to open lock file: %w", lockErr)
	}

	defer func() {
		_ = lockFd.Unlock()
		_ = lockFd.Remove()
	}()

	lockErr = lockFd.Lock()
	if lockErr != nil {
		return fmt.Errorf("failed to acquire lock: %w", lockErr)
	}

	_, err = os.Stat(nightlyDir)
	if os.IsNotExist(err) {
		return fmt.Errorf(
			"%w: %s",
			ErrNightlyVersionNotExists,
			shortHash(entry.CommitHash, constants.ShortHashLength),
		)
	}

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
			// It's a real directory - back it up if we have a commit hash
			if currentCommit != "" {
				backupDir := filepath.Join(
					GetVersionsDir(),
					"nightly-"+shortHash(currentCommit, constants.ShortHashLength),
				)

				var backupErr error

				_, backupErr = os.Stat(backupDir)
				if os.IsNotExist(backupErr) {
					// Rename current to backup
					renameErr := os.Rename(currentNightly, backupDir)
					if renameErr != nil {
						return fmt.Errorf("failed to backup current nightly: %w", renameErr)
					}

					logrus.Debugf("Backed up current nightly to %s", backupDir)
				} else {
					// Backup already exists, safe to remove current
					rmErr := os.RemoveAll(currentNightly)
					if rmErr != nil {
						return fmt.Errorf("failed to remove current nightly: %w", rmErr)
					}
				}
			} else {
				// No commit hash, just remove
				rmErr := os.RemoveAll(currentNightly)
				if rmErr != nil {
					return fmt.Errorf("failed to remove current nightly: %w", rmErr)
				}
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

	_, printErr := fmt.Fprintf(
		os.Stdout,
		"%s Rolled back to nightly %s (from %s)\n",
		ui.SuccessIcon(),
		shortHash(entry.CommitHash, constants.ShortHashLength),
		entry.InstalledAt.Format("2006-01-02 15:04"),
	)
	if printErr != nil {
		logrus.Warnf("Failed to write to stdout: %v", printErr)
	}

	return nil
}

func listNightlyHistory(history *NightlyHistory) error {
	// Get current nightly commit to show indicator
	currentCommit, _ := GetVersionService().GetInstalledVersionIdentifier("nightly")

	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithRendition(tw.Rendition{
			Borders:  tw.BorderNone,
			Settings: tw.Settings{Separators: tw.Separators{BetweenRows: tw.Off}},
		}),
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{
				Alignment: tw.CellAlignment{Global: tw.AlignLeft},
			},
			Row: tw.CellConfig{
				Alignment: tw.CellAlignment{Global: tw.AlignLeft},
			},
		}),
	)
	table.Header([]string{"Index", "Commit", "Installed At", "Status"})

	var err error
	for index, entry := range history.Entries {
		status := ""
		if shortHash(
			currentCommit,
			constants.ShortHashLength,
		) == shortHash(
			entry.CommitHash,
			constants.ShortHashLength,
		) {
			status = "← current"
		}

		err = table.Append([]string{
			strconv.Itoa(index),
			shortHash(entry.CommitHash, constants.ShortHashLength),
			entry.InstalledAt.Format("2006-01-02 15:04"),
			status,
		})
		if err != nil {
			return err
		}
	}

	err = table.Render()
	if err != nil {
		return err
	}

	var printErr error

	_, printErr = fmt.Fprintln(
		os.Stdout,
		"\nUse 'nvs rollback <index>' to rollback to a specific version.",
	)
	if printErr != nil {
		logrus.Warnf("Failed to write to stdout: %v", printErr)
	}

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

	// Remove any existing entry with the same commit hash (to avoid duplicates)
	dedupedEntries := make([]NightlyHistoryEntry, 0, len(history.Entries))
	for _, entry := range history.Entries {
		if shortHash(
			entry.CommitHash,
			constants.ShortHashLength,
		) != shortHash(
			commitHash,
			constants.ShortHashLength,
		) {
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

			lockFile := oldDir + ".lock"

			lockFd, lockErr := platform.NewFileLock(lockFile)
			if lockErr != nil {
				logrus.Warnf("Failed to acquire lock for %s: %v", oldDir, lockErr)

				continue
			}

			lockErr = lockFd.Lock()
			if lockErr != nil {
				logrus.Warnf("Failed to lock %s: %v", oldDir, lockErr)

				_ = lockFd.Close()

				continue
			}

			logrus.Debugf("Removing old nightly backup: %s", oldDir)

			err := os.RemoveAll(oldDir)
			if err != nil {
				logrus.Warnf("Failed to remove old nightly %s: %v", oldDir, err)
			}

			_ = lockFd.Unlock()
			_ = lockFd.Remove()
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
	sort.Slice(history.Entries, func(i, j int) bool {
		return history.Entries[i].InstalledAt.After(history.Entries[j].InstalledAt)
	})

	return &history, nil
}

func saveNightlyHistory(history *NightlyHistory) error {
	historyPath := filepath.Join(filepath.Dir(GetVersionsDir()), constants.NightlyHistoryFile)

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file first for atomicity
	tempPath := historyPath + ".tmp"

	err = os.WriteFile(tempPath, data, constants.FilePerm)
	if err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tempPath, historyPath)
}

func init() {
	rootCmd.AddCommand(rollbackCmd)
}
