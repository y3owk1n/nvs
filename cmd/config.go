package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/utils"
)

var configCmd = &cobra.Command{
	Use:     "config",
	Aliases: []string{"conf", "c"},
	Short:   "Switch Neovim configuration",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			utils.LaunchNvimWithConfig(args[0])
			return
		}

		home, err := os.UserHomeDir()
		if err != nil {
			logrus.Fatalf("Failed to get home directory: %v", err)
		}
		configDir := filepath.Join(home, ".config")

		entries, err := os.ReadDir(configDir)
		if err != nil {
			logrus.Fatalf("Failed to read config directory: %v", err)
		}

		var nvimConfigs []string
		for _, entry := range entries {
			entryPath := filepath.Join(configDir, entry.Name())
			isDir := false
			if entry.Type()&os.ModeSymlink != 0 {
				// For symlink, resolve the target.
				resolvedPath, err := os.Readlink(entryPath)
				if err != nil {
					logrus.Warnf("Failed to resolve symlink for %s: %v", entry.Name(), err)
					continue
				}
				targetInfo, err := os.Stat(resolvedPath)
				if err != nil {
					logrus.Warnf("Failed to stat resolved path for %s: %v", entry.Name(), err)
					continue
				}
				isDir = targetInfo.IsDir()
				logrus.Debugf("%s is a symlink to %s", entry.Name(), resolvedPath)
			} else {
				isDir = entry.IsDir()
			}

			if isDir && strings.Contains(strings.ToLower(entry.Name()), "nvim") {
				nvimConfigs = append(nvimConfigs, entry.Name())
			}
		}

		if len(nvimConfigs) == 0 {
			fmt.Println("No Neovim configuration directories found in ~/.config")
			return
		}

		// Use promptui for an interactive selection.
		prompt := promptui.Select{
			Label: "Select Neovim configuration",
			Items: nvimConfigs,
		}

		_, selectedConfig, err := prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				fmt.Println("Selection cancelled.")
				return
			}
			logrus.Fatalf("Prompt failed: %v", err)
		}

		utils.LaunchNvimWithConfig(selectedConfig)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
