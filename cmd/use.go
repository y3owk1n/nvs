package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/utils"
)

var useCmd = &cobra.Command{
	Use:   "use <version|stable|nightly>",
	Short: "Switch to a specific version",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		alias := args[0]
		targetVersion := alias
		// For specific version names (other than "stable" or "nightly") you could resolve or validate further if needed.
		if !utils.IsInstalled(versionsDir, targetVersion) {
			logrus.Fatalf("Version %s is not installed", targetVersion)
		}

		// Check if the current symlink already points to the target version.
		currentSymlink := filepath.Join(versionsDir, "current")
		if current, err := os.Readlink(currentSymlink); err == nil {
			if filepath.Base(current) == targetVersion {
				fmt.Printf("Already using Neovim %s\n", targetVersion)
				return
			}
		}

		// Update the "current" symlink.
		symlinkPath := filepath.Join(versionsDir, "current")
		versionPath := filepath.Join(versionsDir, targetVersion)
		if err := utils.UpdateSymlink(versionPath, symlinkPath); err != nil {
			logrus.Fatalf("Failed to switch version: %v", err)
		}

		fmt.Printf("Switched to Neovim %s\n", targetVersion)
		nvimExec := utils.FindNvimBinary(versionPath)
		if nvimExec == "" {
			fmt.Printf("Warning: Could not find Neovim binary in %s. Please check the installation structure.\n", versionPath)
			return
		}

		targetBin := filepath.Join(globalBinDir, "nvim")
		if _, err := os.Lstat(targetBin); err == nil {
			os.Remove(targetBin)
		}
		if err := os.Symlink(nvimExec, targetBin); err != nil {
			logrus.Fatalf("Failed to create symlink in global bin: %v", err)
		}
		fmt.Printf("Global Neovim binary updated: %s -> %s\n", targetBin, nvimExec)

		// If the global bin directory is not in PATH, advise the user.
		pathEnv := os.Getenv("PATH")
		if !strings.Contains(pathEnv, globalBinDir) {
			fmt.Printf("Add this to your PATH: %s\n", globalBinDir)
		}
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
}
