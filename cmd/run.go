package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/domain/version"
	"github.com/y3owk1n/nvs/internal/platform"
	"github.com/y3owk1n/nvs/internal/ui"
)

// runCmd represents the "run" command.
// It runs a specific Neovim version without switching the global version.
// Arguments after "--" are passed directly to Neovim.
//
// Example usage:
//
//	nvs run stable
//	nvs run nightly -- --clean
//	nvs run v0.10.3 -- myfile.txt
//	nvs run --pick
var runCmd = &cobra.Command{
	Use:   "run <version> [-- <nvim args>...] | --pick [-- <nvim args>...]",
	Short: "Run a specific Neovim version without switching",
	Long: `Run a specific installed Neovim version without modifying the global symlink.
Any arguments after "--" are passed directly to the Neovim instance.

Examples:
  nvs run stable
  nvs run nightly -- --clean
  nvs run v0.10.3 -- -c "checkhealth"
  nvs run --pick -- --clean`,
	RunE: RunRun,
}

// RunRun executes the run command.
func RunRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), constants.TimeoutMinutes*time.Minute)
	defer cancel()

	var versionAlias string

	// Check if --pick flag is set
	pick, _ := cmd.Flags().GetBool("pick")
	if pick {
		// Launch picker for installed versions
		versions, err := GetVersionService().List()
		if err != nil {
			return fmt.Errorf("error listing versions: %w", err)
		}

		if len(versions) == 0 {
			return fmt.Errorf("%w for selection", ErrNoVersionsAvailable)
		}

		availableVersions := make([]string, 0, len(versions))
		for _, v := range versions {
			availableVersions = append(availableVersions, v.Name())
		}

		prompt := promptui.Select{
			Label: "Select version to run",
			Items: availableVersions,
		}

		_, selectedVersion, err := prompt.Run()
		if err != nil {
			if errors.Is(err, promptui.ErrInterrupt) {
				_, printErr := fmt.Fprintf(
					os.Stdout,
					"%s %s\n",
					ui.WarningIcon(),
					ui.WhiteText("Selection canceled."),
				)
				if printErr != nil {
					logrus.Warnf("Failed to write to stdout: %v", printErr)
				}

				return nil
			}

			return fmt.Errorf("prompt failed: %w", err)
		}

		versionAlias = selectedVersion
	} else {
		if len(args) == 0 {
			return fmt.Errorf("%w", ErrVersionArgRequired)
		}

		versionAlias = args[0]
	}

	logrus.Debugf("Requested version to run: %s", versionAlias)

	// Check if version is installed
	if !GetVersionService().IsVersionInstalled(versionAlias) {
		return fmt.Errorf(
			"%w: %s (use 'nvs install %s' first)",
			version.ErrVersionNotFound,
			versionAlias,
			versionAlias,
		)
	}

	// Get the nvim binary path for this version
	nvimPath, err := getNvimBinaryPath(versionAlias)
	if err != nil {
		return fmt.Errorf("failed to find nvim binary: %w", err)
	}

	logrus.Debugf("Found nvim binary at: %s", nvimPath)

	// Get arguments to pass to nvim
	var (
		nvimArgs []string
		startIdx int
	)

	if pick {
		// When pick is used, all args are nvim args (possibly starting with "--")
		startIdx = 0
	} else {
		// Skip the version arg
		startIdx = 1
	}

	if startIdx < len(args) {
		// Skip the "--" separator if present
		if args[startIdx] == "--" {
			startIdx++
		}

		if startIdx < len(args) {
			nvimArgs = args[startIdx:]
		}
	}

	// Execute nvim
	//nolint:gosec // Arguments are passed through from user command line
	nvimCmd := exec.CommandContext(ctx, nvimPath, nvimArgs...)
	nvimCmd.Stdin = os.Stdin
	nvimCmd.Stdout = os.Stdout
	nvimCmd.Stderr = os.Stderr

	logrus.Debugf("Running: %s %v", nvimPath, nvimArgs)

	err = nvimCmd.Run()
	if err != nil {
		// If nvim exits with a non-zero status, we should propagate that
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Return wrapped static error with exit code info
			return fmt.Errorf("%w: code %d", ErrNvimExitNonZero, exitErr.ExitCode())
		}

		return fmt.Errorf("failed to run nvim: %w", err)
	}

	return nil
}

// getNvimBinaryPath returns the path to the nvim binary for a specific version.
func getNvimBinaryPath(versionAlias string) (string, error) {
	normalized := normalizeVersionForPath(versionAlias)

	versionDir := filepath.Join(GetVersionsDir(), normalized)

	lockFile := versionDir + ".lock"

	lockFd, lockErr := platform.NewFileLock(lockFile)
	if lockErr != nil {
		return "", fmt.Errorf("failed to open lock file: %w", lockErr)
	}

	defer func() {
		_ = lockFd.Unlock()
		_ = lockFd.Close()
	}()

	lockErr = lockFd.Lock()
	if lockErr != nil {
		return "", fmt.Errorf("failed to acquire lock: %w", lockErr)
	}

	_, statErr := os.Stat(versionDir)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return "", fmt.Errorf("%w: %s", ErrVersionDirNotFound, normalized)
		}

		return "", fmt.Errorf("failed to access version directory %s: %w", normalized, statErr)
	}

	binaryPath := findNvimBinary(versionDir)
	if binaryPath == "" {
		return "", fmt.Errorf("%w in %s", ErrNvimBinaryNotFound, versionDir)
	}

	return binaryPath, nil
}

// normalizeVersionForPath normalizes a version string for use as a directory name.
func normalizeVersionForPath(versionStr string) string {
	return version.NormalizeVersionForPath(versionStr)
}

// findNvimBinary searches for the nvim binary in a version directory.
func findNvimBinary(dir string) string {
	// Common locations to check
	var candidates []string

	if runtime.GOOS == constants.WindowsOS {
		candidates = []string{
			filepath.Join(dir, "bin", "nvim.exe"),
			filepath.Join(dir, "nvim-win64", "bin", "nvim.exe"),
			filepath.Join(dir, "Neovim", "bin", "nvim.exe"),
		}
	} else {
		candidates = []string{
			filepath.Join(dir, "bin", "nvim"),
			filepath.Join(dir, "nvim-macos-arm64", "bin", "nvim"),
			filepath.Join(dir, "nvim-macos-x86_64", "bin", "nvim"),
			filepath.Join(dir, "nvim-macos", "bin", "nvim"),
			filepath.Join(dir, "nvim-linux64", "bin", "nvim"),
			filepath.Join(dir, "nvim-linux-x86_64", "bin", "nvim"),
			filepath.Join(dir, "nvim-linux-arm64", "bin", "nvim"),
		}
	}

	// Try common locations first
	for _, candidate := range candidates {
		info, statErr := os.Stat(candidate)
		if statErr == nil && !info.IsDir() {
			return candidate
		}
	}

	// Fallback: walk the directory to find nvim binary
	var found string

	walkErr := filepath.WalkDir(dir, func(path string, dirEntry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if dirEntry.IsDir() {
			return nil
		}

		name := dirEntry.Name()
		if runtime.GOOS == constants.WindowsOS {
			if strings.EqualFold(name, "nvim.exe") {
				found = path

				return filepath.SkipAll
			}
		} else if name == "nvim" {
			info, infoErr := dirEntry.Info()
			if infoErr == nil && info.Mode()&0o111 != 0 {
				found = path

				return filepath.SkipAll
			}
		}

		return nil
	})
	if walkErr != nil {
		logrus.Debugf("Error walking directory %s: %v", dir, walkErr)
	}

	return found
}

// init registers the runCmd with the root command.
func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().BoolP("pick", "p", false, "Launch interactive picker to select version")
}
