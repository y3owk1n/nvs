package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/helpers"
)

// Windows is the string for Windows OS.
const Windows = "windows"

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

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Determine the base configuration directory:
	//   If NVS_CONFIG_DIR is set, use that;
	//   Otherwise, use the system config directory, falling back to the user's home directory if needed.
	var baseConfigDir string
	if custom := os.Getenv("NVS_CONFIG_DIR"); custom != "" {
		baseConfigDir = custom
		logrus.Debugf("Using custom config directory from NVS_CONFIG_DIR: %s", baseConfigDir)
	} else {
		configDir, err := os.UserConfigDir()
		if err == nil {
			baseConfigDir = filepath.Join(configDir, "nvs")
			logrus.Debugf("Using system config directory: %s", baseConfigDir)
		} else {
			baseConfigDir = filepath.Join(home, ".nvs")
			logrus.Debugf("Falling back to home directory for config: %s", baseConfigDir)
		}
	}

	// Note: We don't create the config directory here to avoid leaving empty directories
	// if the user cancels the reset operation.

	// Determine the base cache directory:
	//   If NVS_CACHE_DIR is set, use that;
	//   Otherwise, use the system cache directory, falling back to the config directory if needed.
	var baseCacheDir string
	if custom := os.Getenv("NVS_CACHE_DIR"); custom != "" {
		baseCacheDir = custom
		logrus.Debugf("Using custom cache directory from NVS_CACHE_DIR: %s", baseCacheDir)
	} else {
		cacheDir, err := os.UserCacheDir()
		if err == nil {
			baseCacheDir = filepath.Join(cacheDir, "nvs")
			logrus.Debugf("Using system cache directory: %s", baseCacheDir)
		} else {
			baseCacheDir = filepath.Join(baseConfigDir, "cache")
			logrus.Debugf("Falling back to config directory for cache: %s", baseCacheDir)
		}
	}

	// Determine the base binary directory:
	//   If NVS_BIN_DIR is set, use that;
	//   Otherwise, use the default binary directory based on the OS.
	var baseBinDir string
	if custom := os.Getenv("NVS_BIN_DIR"); custom != "" {
		baseBinDir = custom
		logrus.Debugf("Using custom binary directory from NVS_BIN_DIR: %s", baseBinDir)
	} else {
		if runtime.GOOS == Windows {
			baseBinDir = filepath.Join(home, "AppData", "Local", "Programs")
			logrus.Debugf("Using Windows binary directory: %s", baseBinDir)
		} else {
			baseBinDir = filepath.Join(home, ".local", "bin")
			logrus.Debugf("Using default binary directory: %s", baseBinDir)
		}
	}

	// Display a warning about the destructive nature of this command.
	_, err = fmt.Fprintf(
		os.Stdout,
		"%s %s\n",
		helpers.WarningIcon(),
		helpers.RedText(
			"WARNING: This will remove all NVS data, including downloaded versions and cache.",
		),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	_, err = fmt.Fprintf(
		os.Stdout,
		"%s %s\n",
		helpers.InfoIcon(),
		helpers.WhiteText("Directories to be removed:"),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	_, err = fmt.Fprintf(os.Stdout, "  - %s\n", helpers.CyanText(baseConfigDir))
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	_, err = fmt.Fprintf(os.Stdout, "  - %s\n", helpers.CyanText(baseCacheDir))
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	_, err = fmt.Fprintf(
		os.Stdout,
		"  - %s (if it exists)\n",
		helpers.CyanText(filepath.Join(baseBinDir, "nvim")),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	// Prompt the user for confirmation.
	_, err = fmt.Fprintf(
		os.Stdout,
		"\n%s %s ",
		helpers.PromptIcon(),
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
			helpers.InfoIcon(),
			helpers.WhiteText("Aborted by user."),
		)
		if err != nil {
			logrus.Warnf("Failed to write to stdout: %v", err)
		}

		return nil
	}

	// Remove the configuration directory.
	logrus.Debugf("Removing config directory: %s", baseConfigDir)

	err = os.RemoveAll(baseConfigDir)
	if err != nil {
		return fmt.Errorf("failed to remove config directory: %w", err)
	}

	// Remove the cache directory.
	logrus.Debugf("Removing cache directory: %s", baseCacheDir)

	err = os.RemoveAll(baseCacheDir)
	if err != nil {
		return fmt.Errorf("failed to remove cache directory: %w", err)
	}

	// Remove the global nvim symlink if it exists.
	nvimSymlink := filepath.Join(baseBinDir, "nvim")
	logrus.Debugf("Removing nvim symlink: %s", nvimSymlink)

	err = os.Remove(nvimSymlink)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove nvim symlink: %w", err)
	}

	_, err = fmt.Fprintf(
		os.Stdout,
		"%s %s\n",
		helpers.SuccessIcon(),
		helpers.WhiteText("Reset complete. All NVS data has been removed."),
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
