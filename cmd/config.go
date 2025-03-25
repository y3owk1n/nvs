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

// configCmd represents the "config" command.
// It allows the user to switch the active Neovim configuration. If a configuration name is provided
// as an argument, Neovim will be launched with that configuration. If no argument is provided,
// the command scans the ~/.config directory for directories containing "nvim" in their name,
// then prompts the user to select one.
//
// Example usage (with an argument):
//
//	nvs config myconfig
//
// This will launch Neovim with the configuration "myconfig".
//
// Example usage (without an argument):
//
//	nvs config
//
// This will display a selection prompt for available Neovim configurations found in ~/.config.
var configCmd = &cobra.Command{
	Use:     "config",
	Aliases: []string{"conf", "c"},
	Short:   "Switch Neovim configuration",
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Debug("Executing config command")

		// If a configuration name is provided as an argument, launch Neovim with that configuration.
		if len(args) == 1 {
			logrus.Debugf("Launching Neovim with provided configuration: %s", args[0])
			fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("Launching Neovim with configuration: %s", utils.CyanText(args[0]))))
			utils.LaunchNvimWithConfig(args[0])
			return
		}

		// No configuration provided; list available Neovim configurations from ~/.config.
		home, err := os.UserHomeDir()
		if err != nil {
			logrus.Fatalf("Failed to get home directory: %v", err)
		}
		logrus.Debugf("User home directory: %s", home)

		configDir := filepath.Join(home, ".config")
		logrus.Debugf("Neovim config directory: %s", configDir)

		entries, err := os.ReadDir(configDir)
		if err != nil {
			logrus.Fatalf("Failed to read config directory: %v", err)
		}
		logrus.Debugf("Found %d entries in config directory", len(entries))

		var nvimConfigs []string
		for _, entry := range entries {
			entryPath := filepath.Join(configDir, entry.Name())
			logrus.Debugf("Processing entry: %s", entryPath)

			isDir := false
			// If the entry is a symlink, resolve it.
			if entry.Type()&os.ModeSymlink != 0 {
				logrus.Debugf("%s is a symlink, resolving...", entry.Name())
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
				logrus.Debugf("%s resolved to %s (isDir: %t)", entry.Name(), resolvedPath, isDir)
			} else {
				isDir = entry.IsDir()
			}

			// Add directories whose name contains "nvim" (case-insensitive) to the list.
			if isDir && strings.Contains(strings.ToLower(entry.Name()), "nvim") {
				logrus.Debugf("Adding Neovim config: %s", entry.Name())
				nvimConfigs = append(nvimConfigs, entry.Name())
			}
		}

		if len(nvimConfigs) == 0 {
			logrus.Debug("No Neovim configurations found in ~/.config")
			fmt.Printf("%s %s\n", utils.WarningIcon(), utils.WhiteText(fmt.Sprintf("No Neovim configuration found in %s", utils.CyanText("~/.config"))))
			return
		}

		logrus.Debugf("Available Neovim configurations: %v", nvimConfigs)
		prompt := promptui.Select{
			Label: "Select Neovim configuration",
			Items: nvimConfigs,
		}

		logrus.Debug("Displaying selection prompt")
		_, selectedConfig, err := prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				logrus.Debug("User cancelled selection")
				fmt.Printf("%s %s\n", utils.WarningIcon(), utils.WhiteText("Selection cancelled."))
				return
			}
			logrus.Fatalf("Prompt failed: %v", err)
		}

		logrus.Debugf("User selected configuration: %s", selectedConfig)
		fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("Launching Neovim with configuration: %s", utils.CyanText(selectedConfig))))
		utils.LaunchNvimWithConfig(selectedConfig)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
