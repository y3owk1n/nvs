package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/ui"
)

// resetCmd represents the "reset" command.
// It removes all data from your configuration and cache directories and removes the symlinked nvim binary.
// **WARNING:** This command is destructive. It deletes all configuration data, cache, and the global nvim symlink.
// Use with caution.
//
// Example usage:
//
//	nvs reset
//
// When executed, the command will prompt you to confirm before performing the reset.
var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset all data (remove symlinks, downloaded versions, cache, etc.)",
	Long:  "WARNING: This command will remove all data in your configuration and cache directories and remove the symlinked nvim binary. Use with caution.",
	RunE:  RunReset,
}

// RunReset executes the reset command.
func RunReset(_ *cobra.Command, _ []string) error {
	logrus.Debug("Starting reset command")

	var err error

	// Determine directories using getter functions
	baseConfigDir := filepath.Dir(GetVersionsDir())
	logrus.Debugf("Resolved configDir: %s", baseConfigDir)

	baseCacheDir := filepath.Dir(GetCacheFilePath())
	logrus.Debugf("Resolved cacheDir: %s", baseCacheDir)

	baseBinDir := GetGlobalBinDir()
	logrus.Debugf("Resolved binDir: %s", baseBinDir)

	// Display a warning about the destructive nature of this command.
	_, err = fmt.Fprintf(
		os.Stdout,
		"%s %s\n",
		ui.WarningIcon(),
		ui.RedText(
			"WARNING: This will remove all NVS data, including downloaded versions and cache.",
		),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	_, err = fmt.Fprintf(
		os.Stdout,
		"%s %s\n",
		ui.InfoIcon(),
		ui.WhiteText("Directories to be removed:"),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	_, err = fmt.Fprintf(os.Stdout, "  - %s\n", ui.CyanText(baseConfigDir))
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	_, err = fmt.Fprintf(os.Stdout, "  - %s\n", ui.CyanText(baseCacheDir))
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	_, err = fmt.Fprintf(
		os.Stdout,
		"  - %s (if it exists)\n",
		ui.CyanText(filepath.Join(baseBinDir, "nvim")),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	// Prompt the user for confirmation.
	_, err = fmt.Fprintf(
		os.Stdout,
		"\n%s %s ",
		ui.PromptIcon(),
		"Are you sure you want to proceed? (y/N): ",
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	reader := bufio.NewReader(os.Stdin)

	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(strings.ToLower(input))
	logrus.Debugf("User input: %q", input)

	if input != "y" {
		_, err = fmt.Fprintf(
			os.Stdout,
			"%s %s\n",
			ui.InfoIcon(),
			ui.WhiteText("Aborted by user."),
		)
		if err != nil {
			logrus.Warnf("Failed to write to stdout: %v", err)
		}

		return nil
	}

	// Remove the configuration directory.
	logrus.Debugf("Removing config directory: %s", baseConfigDir)

	assertErr := assertSafeToRemovePath(baseConfigDir)
	if assertErr != nil {
		return assertErr
	}

	err = os.RemoveAll(baseConfigDir)
	if err != nil {
		return fmt.Errorf("failed to remove config directory: %w", err)
	}

	// Remove the cache directory.
	logrus.Debugf("Removing cache directory: %s", baseCacheDir)

	assertErr = assertSafeToRemovePath(baseCacheDir)
	if assertErr != nil {
		return assertErr
	}

	err = os.RemoveAll(baseCacheDir)
	if err != nil {
		return fmt.Errorf("failed to remove cache directory: %w", err)
	}

	// Remove the global nvim symlink if it exists.
	nvimSymlink := filepath.Join(baseBinDir, "nvim")
	logrus.Debugf("Removing nvim symlink: %s", nvimSymlink)

	// Use os.Remove (not os.RemoveAll) so we never recurse into
	// a directory by accident; the symlink target is always a
	// single file path. We still guard the parent directory
	// against being a top-level system dir: deleting
	// <some-root>/nvim is at best confusing, at worst destructive
	// if 'nvim' happens to be a real binary there.
	assertErr = assertSafeToRemoveParent(nvimSymlink)
	if assertErr != nil {
		return assertErr
	}

	err = os.Remove(nvimSymlink)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove nvim symlink: %w", err)
	}

	_, err = fmt.Fprintf(
		os.Stdout,
		"%s %s\n",
		ui.SuccessIcon(),
		ui.WhiteText("Reset complete. All NVS data has been removed."),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	return nil
}

// init registers the resetCmd with the root command.
func init() {
	rootCmd.AddCommand(resetCmd)
}

// errUnsafeResetPath is returned by assertSafeToRemovePath and
// assertSafeToRemoveParent when the path under consideration
// could not possibly be a valid NVS data directory. The check
// protects against a misconfigured (or hostile)
// NVS_CONFIG_DIR / NVS_CACHE_DIR / NVS_BIN_DIR environment
// variable that points at a filesystem root or a top-level
// system directory: a 'nvs reset' on such a setup would
// otherwise devolve into a recursive removal of the entire
// disk.
//
// We refuse the operation before the user is even prompted for
// confirmation so the danger is surfaced immediately and
// unambiguously, regardless of whether the user reads the
// "Directories to be removed:" warning that is printed above
// the prompt.
var errUnsafeResetPath = errors.New("refusing to remove unsafe path")

// assertSafeToRemovePath returns errUnsafeResetPath if path
// refers to a location that os.RemoveAll would be catastrophically
// destructive on. It is intended as a guard for the config
// directory and cache directory removals in RunReset.
//
// The check rejects:
//   - Empty strings
//   - The current directory (".")
//   - The platform's root separator, after filepath.Clean has
//     normalized repeated separators (e.g. "/" on Unix, "\\" on
//     Windows; any number of leading or repeated separators
//     clean to the same form on each platform)
//   - Windows drive roots ("C:\\", "D:\\", ...)
//   - Top-level system directories whose parent is a root
//     (e.g. "/etc" with parent "/", "C:\\Users" with parent
//     "C:\\")
//   - Paths that are so degenerate they have no parent
//     (e.g. "." in some corner cases)
//
// A valid NVS config or cache directory is always at least
// three levels deep (e.g. "/Users/jane/Library/Application
// Support/nvs"), so the "top-level" rejection never fires for
// legitimate setups.
//
// The checks use filepath.Separator (rather than hard-coded
// '/' and '\\') so the same source code works on Unix and
// Windows. Without the platform-separator comparison, the
// "root separator" branch would only catch "/" on Unix and
// miss "\\" on Windows (and vice versa), which is exactly the
// mismatch that produced the original Windows CI failure
// where pathListContains's test data hard-coded ':'.
func assertSafeToRemovePath(path string) error {
	if path == "" {
		return fmt.Errorf("%w: empty path", errUnsafeResetPath)
	}

	cleaned := filepath.Clean(path)
	sep := string(filepath.Separator)

	// Reject the platform root separator after Clean has
	// collapsed any number of leading or repeated separators.
	// This catches "/", "//", "///" on Unix and "\", "\\",
	// "\\\\" on Windows (all of which clean to the same single
	// separator on the respective platform).
	if cleaned == sep {
		return fmt.Errorf(
			"%w: %q is a filesystem root",
			errUnsafeResetPath,
			path,
		)
	}

	// Reject the current directory.
	if cleaned == "." {
		return fmt.Errorf(
			"%w: %q is the current directory",
			errUnsafeResetPath,
			path,
		)
	}

	if isDriveRoot(cleaned) {
		return fmt.Errorf(
			"%w: %q is a drive root",
			errUnsafeResetPath,
			path,
		)
	}

	parent := filepath.Dir(cleaned)
	if parent == cleaned {
		return fmt.Errorf(
			"%w: %q has no parent",
			errUnsafeResetPath,
			path,
		)
	}

	if isFilesystemRoot(parent) {
		return fmt.Errorf(
			"%w: %q is a top-level system directory (parent %q)",
			errUnsafeResetPath,
			path,
			parent,
		)
	}

	return nil
}

// assertSafeToRemoveParent is the single-file analog of
// assertSafeToRemovePath. It is used for the nvim symlink
// removal, where os.Remove is used (not os.RemoveAll) so the
// risk is much lower, but we still want to refuse to operate
// on <some-root>/nvim or <top-level-dir>/nvim since neither
// is a legitimate nvs symlink target.
//
// The check delegates to assertSafeToRemovePath on the
// symlink's parent directory, so it rejects the same set of
// dangerous parents (roots, top-level system directories,
// drive roots).
func assertSafeToRemoveParent(path string) error {
	parent := filepath.Dir(path)

	err := assertSafeToRemovePath(parent)
	if err != nil {
		return fmt.Errorf(
			"%w: %q parent is unsafe: %w",
			errUnsafeResetPath,
			path,
			err,
		)
	}

	return nil
}

// driveRootLen is the byte length of a Windows drive root
// such as "C:\\" or "D:/". Used by isDriveRoot.
const driveRootLen = 3

// isDriveRoot reports whether path is a Windows drive root
// like "C:\\" or "D:/". The check is unconditional — on Unix
// the format does not match any real path so the answer is
// always false, and the function is safe to call on any OS.
func isDriveRoot(path string) bool {
	if runtime.GOOS != constants.WindowsOS {
		return false
	}

	if len(path) != driveRootLen {
		return false
	}

	if path[1] != ':' {
		return false
	}

	if path[2] != '\\' && path[2] != '/' {
		return false
	}

	return true
}

// isFilesystemRoot reports whether path is a filesystem root:
// "/" on Unix, "\\" on Windows. The check uses
// filepath.Separator (not a hard-coded "/") so the same source
// works on every platform — on Windows, "/" is intentionally
// not treated as a root, matching the convention that "\\" is
// the only Windows root path. The drive check delegates to
// isDriveRoot, which already handles "C:\\" / "D:/".
func isFilesystemRoot(path string) bool {
	if path == string(filepath.Separator) {
		return true
	}

	return isDriveRoot(path)
}
