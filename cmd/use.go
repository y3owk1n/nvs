package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/releases"
	"github.com/y3owk1n/nvs/pkg/utils"
)

var useCmd = &cobra.Command{
	Use:   "use <version|stable|nightly>",
	Short: "Switch to a specific version",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Debug("Starting use command")

		alias := releases.NormalizeVersion(args[0])
		targetVersion := alias

		logrus.Debugf("Resolved target version: %s", targetVersion)

		if !utils.IsInstalled(versionsDir, targetVersion) {
			logrus.Fatalf("Version %s is not installed", targetVersion)
		}

		currentSymlink := filepath.Join(versionsDir, "current")
		if current, err := os.Readlink(currentSymlink); err == nil {
			if filepath.Base(current) == targetVersion {
				fmt.Printf("%s Already using Neovim %s\n", utils.WarningIcon(), utils.CyanText(targetVersion))
				logrus.Debugf("Already using version: %s", targetVersion)
				return
			}
		}

		versionPath := filepath.Join(versionsDir, targetVersion)
		logrus.Debugf("Updating symlink to point to: %s", versionPath)
		if err := utils.UpdateSymlink(versionPath, currentSymlink); err != nil {
			logrus.Fatalf("Failed to switch version: %v", err)
		}

		nvimExec := utils.FindNvimBinary(versionPath)
		if nvimExec == "" {
			fmt.Printf("%s Could not find Neovim binary in %s. Please check the installation structure.\n", utils.ErrorIcon(), utils.CyanText(versionPath))
			logrus.Errorf("Neovim binary not found in: %s", versionPath)
			return
		}

		targetBin := filepath.Join(globalBinDir, "nvim")
		if _, err := os.Lstat(targetBin); err == nil {
			os.Remove(targetBin)
			logrus.Debugf("Removed existing global bin symlink: %s", targetBin)
		}
		if err := os.Symlink(nvimExec, targetBin); err != nil {
			logrus.Fatalf("Failed to create symlink in global bin: %v", err)
		}

		logrus.Debugf("Global Neovim binary updated: %s -> %s", targetBin, nvimExec)
		switchMsg := fmt.Sprintf("Switched to Neovim %s", utils.CyanText(targetVersion))
		fmt.Printf("%s %s\n", utils.SuccessIcon(), utils.WhiteText(switchMsg))

		if pathEnv := os.Getenv("PATH"); !strings.Contains(pathEnv, globalBinDir) {
			fmt.Printf("%s Run `nvs path` or manually add this directory to your PATH for convenience: %s\n", utils.WarningIcon(), utils.CyanText(globalBinDir))
			logrus.Debugf("Global bin directory not found in PATH: %s", globalBinDir)
		}
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
}
