package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/utils"
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
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Debug("Starting reset command")

		// Determine the base configuration directory:
		//   If NVS_CONFIG_DIR is set, use that;
		//   Otherwise, use the system config directory, falling back to the user's home directory if needed.
		var baseConfigDir string
		if custom := os.Getenv("NVS_CONFIG_DIR"); custom != "" {
			baseConfigDir = custom
			logrus.Debugf("Using custom config directory from NVS_CONFIG_DIR: %s", baseConfigDir)
		} else {
			if configDir, err := os.UserConfigDir(); err == nil {
				baseConfigDir = filepath.Join(configDir, "nvs")
				logrus.Debugf("Using system config directory: %s", baseConfigDir)
			} else {
				home, err := os.UserHomeDir()
				if err != nil {
					logrus.Fatalf("Failed to get user home directory: %v", err)
				}
				baseConfigDir = filepath.Join(home, ".nvs")
				logrus.Debugf("Falling back to home directory for config: %s", baseConfigDir)
			}
		}

		// Determine the base cache directory:
		//   If NVS_CACHE_DIR is set, use that;
		//   Otherwise, use the system cache directory, falling back to the config directory if necessary.
		var baseCacheDir string
		if custom := os.Getenv("NVS_CACHE_DIR"); custom != "" {
			baseCacheDir = custom
			logrus.Debugf("Using custom cache directory from NVS_CACHE_DIR: %s", baseCacheDir)
		} else {
			if cacheDir, err := os.UserCacheDir(); err == nil {
				baseCacheDir = filepath.Join(cacheDir, "nvs")
				logrus.Debugf("Using system cache directory: %s", baseCacheDir)
			} else {
				baseCacheDir = filepath.Join(baseConfigDir, "cache")
				logrus.Debugf("Falling back to config directory for cache: %s", baseCacheDir)
			}
		}

		// Determine the base binary directory:
		//   If NVS_BIN_DIR is set, use that;
		//   Otherwise, default to ~/.local/bin.
		var baseBinDir string
		if custom := os.Getenv("NVS_BIN_DIR"); custom != "" {
			baseBinDir = custom
			logrus.Debugf("Using custom binary directory from NVS_BIN_DIR: %s", baseBinDir)
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				logrus.Fatalf("Failed to get user home directory: %v", err)
			}
			baseBinDir = filepath.Join(home, ".local", "bin")
			logrus.Debugf("Using default binary directory: %s", baseBinDir)
		}

		// Display a warning message showing which directories will be cleared and the binary to be removed.
		warningMsg := fmt.Sprintf(
			"WARNING: This will delete all data in the following directories:\n"+
				"- Config: %s\n"+
				"- Cache: %s\n"+
				"and remove the symlinked nvim binary in the binary directory: %s",
			utils.CyanText(baseConfigDir), utils.CyanText(baseCacheDir), utils.CyanText(baseBinDir))
		fmt.Printf("%s %s\n\n", utils.WarningIcon(), warningMsg)
		fmt.Printf("%s %s ", utils.PromptIcon(), "Are you sure? (y/N): ")

		// Prompt the user for confirmation.
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			logrus.Fatalf("Failed to read input: %v", err)
		}
		input = strings.TrimSpace(input)
		if strings.ToLower(input) != "y" {
			fmt.Println(utils.InfoIcon(), utils.WhiteText("Reset cancelled."))
			logrus.Debug("Reset cancelled by user")
			return
		}

		// Remove all contents in the configuration directory.
		logrus.Debugf("Cleaning up configuration directory: %s", baseConfigDir)
		if entries, err := os.ReadDir(baseConfigDir); err == nil {
			for _, entry := range entries {
				fullPath := filepath.Join(baseConfigDir, entry.Name())
				logrus.Debugf("Removing %s", fullPath)
				if err := os.RemoveAll(fullPath); err != nil {
					logrus.Fatalf("Failed to remove %s: %v", fullPath, err)
				}
			}
		} else {
			logrus.Warnf("Config directory not found or unreadable: %s", baseConfigDir)
		}

		// Remove all contents in the cache directory.
		logrus.Debugf("Cleaning up cache directory: %s", baseCacheDir)
		if entries, err := os.ReadDir(baseCacheDir); err == nil {
			for _, entry := range entries {
				fullPath := filepath.Join(baseCacheDir, entry.Name())
				logrus.Debugf("Removing %s", fullPath)
				if err := os.RemoveAll(fullPath); err != nil {
					logrus.Fatalf("Failed to remove %s: %v", fullPath, err)
				}
			}
		} else {
			logrus.Warnf("Cache directory not found or unreadable: %s", baseCacheDir)
		}

		// Remove the symlinked nvim binary in the binary directory.
		symlinkPath := filepath.Join(baseBinDir, "nvim")
		logrus.Debugf("Removing symlinked binary: %s", symlinkPath)
		if err := os.Remove(symlinkPath); err != nil && !os.IsNotExist(err) {
			logrus.Fatalf("Failed to remove symlink %s: %v", symlinkPath, err)
		}

		fmt.Println(utils.SuccessIcon(), utils.WhiteText("Reset successful. All data has been cleared."))
		logrus.Debug("Reset completed successfully")
	},
}

// init registers the resetCmd with the root command.
func init() {
	rootCmd.AddCommand(resetCmd)
}
