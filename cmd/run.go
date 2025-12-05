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

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/domain/version"
)

const windowsOS = "windows"

// runCmd represents the "run" command.
// It runs a specific Neovim version without switching the global version.
// Arguments after "--" are passed directly to Neovim.
//
// Example usage:
//
//	nvs run stable
//	nvs run nightly -- --clean
//	nvs run v0.10.3 -- myfile.txt
var runCmd = &cobra.Command{
	Use:   "run <version> [-- <nvim args>...]",
	Short: "Run a specific Neovim version without switching",
	Long: `Run a specific installed Neovim version without modifying the global symlink.
Any arguments after "--" are passed directly to the Neovim instance.

Examples:
  nvs run stable
  nvs run nightly -- --clean
  nvs run v0.10.3 -- -c "checkhealth"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runRun,
}

// runRun executes the run command.
func runRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), TimeoutMinutes*time.Minute)
	defer cancel()

	versionAlias := args[0]
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

	// Get arguments to pass to nvim (everything after the version)
	var nvimArgs []string
	if len(args) > 1 {
		// Skip the "--" separator if present
		startIdx := 1
		if args[1] == "--" {
			startIdx = 2
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
	// Normalize version name
	normalized := normalizeVersionForPath(versionAlias)

	// Construct version directory path
	versionDir := filepath.Join(GetVersionsDir(), normalized)

	// Check if version directory exists
	_, statErr := os.Stat(versionDir)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return "", fmt.Errorf("%w: %s", ErrVersionDirNotFound, normalized)
		}

		return "", fmt.Errorf("failed to access version directory %s: %w", normalized, statErr)
	}

	// Find the nvim binary
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

	if runtime.GOOS == windowsOS {
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
		if runtime.GOOS == windowsOS {
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
}
