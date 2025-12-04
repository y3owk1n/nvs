package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/ui"
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
