// Package cmd contains the CLI commands for nvs.
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/helpers"
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
	RunE:    RunConfig,
}

// RunConfig executes the config command.
func RunConfig(cmd *cobra.Command, args []string) error {
	logrus.Debug("Executing config command")

	// If a configuration name is provided as an argument, launch Neovim with that configuration.
	if len(args) == 1 {
		logrus.Debugf("Launching Neovim with provided configuration: %s", args[0])

		_, err := fmt.Fprintf(os.Stdout,
			"%s %s\n",
			helpers.InfoIcon(),
			helpers.WhiteText(
				"Launching Neovim with configuration: "+helpers.CyanText(args[0]),
			),
		)
		if err != nil {
			logrus.Warnf("Failed to write to stdout: %v", err)
		}

		err = helpers.LaunchNvimWithConfig(args[0])
		if err != nil {
			return err
		}

		return nil
	}

	configDir, err := helpers.GetNvimConfigBaseDir()
	if err != nil {
		return fmt.Errorf("failed to determine config base dir: %w", err)
	}

	logrus.Debugf("Neovim config directory: %s", configDir)

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return fmt.Errorf("failed to read config directory %s: %w", configDir, err)
	}

	logrus.Debugf("Found %d entries in config directory (%s)", len(entries), configDir)

	var nvimConfigs []string
	for _, entry := range entries {
		entryPath := filepath.Join(configDir, entry.Name())
		logrus.Debugf("Processing entry: %s", entryPath)

		info, err := os.Lstat(entryPath)
		if err != nil {
			logrus.Warnf("Failed to lstat %s: %v", entryPath, err)

			continue
		}

		var isDir bool
		if info.Mode()&os.ModeSymlink != 0 {
			// Proper symlink
			resolvedPath, err := os.Readlink(entryPath)
			if err != nil {
				logrus.Warnf("Failed to resolve symlink %s: %v", entry.Name(), err)

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
			// Could be a real dir or a junction (Windows treats junctions as dirs)
			isDir = info.IsDir()
		}

		// Add directories whose name contains "nvim" (case-insensitive) to the list.
		if isDir {
			name := strings.ToLower(entry.Name())

			if strings.Contains(name, "nvim") {
				// Exclude nvim-data only on Windows
				if runtime.GOOS == "windows" && strings.HasSuffix(name, "-data") {
					logrus.Debugf("Skipping Windows nvim-data: %s", entry.Name())

					continue
				}

				logrus.Debugf("Adding Neovim config: %s", entry.Name())
				nvimConfigs = append(nvimConfigs, entry.Name())
			}
		}
	}

	if len(nvimConfigs) == 0 {
		logrus.Debugf("No Neovim configurations found in config directory: %s", configDir)

		_, err := fmt.Fprintf(os.Stdout,
			"%s %s\n",
			helpers.WarningIcon(),
			helpers.WhiteText(
				"No Neovim configuration found in "+helpers.CyanText(configDir),
			),
		)
		if err != nil {
			logrus.Warnf("Failed to write to stdout: %v", err)
		}

		return nil
	}

	logrus.Debugf("Available Neovim configurations: %v", nvimConfigs)
	prompt := promptui.Select{
		Label: "Select Neovim configuration",
		Items: nvimConfigs,
	}

	logrus.Debug("Displaying selection prompt")

	_, selectedConfig, err := prompt.Run()
	if err != nil {
		if errors.Is(err, promptui.ErrInterrupt) {
			logrus.Debug("User canceled selection")

			_, err := fmt.Fprintf(
				os.Stdout,
				"%s %s\n",
				helpers.WarningIcon(),
				helpers.WhiteText("Selection canceled."),
			)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			return nil
		}

		return fmt.Errorf("prompt failed: %w", err)
	}

	logrus.Debugf("User selected configuration: %s", selectedConfig)

	_, err = fmt.Fprintf(os.Stdout,
		"%s %s\n",
		helpers.InfoIcon(),
		helpers.WhiteText(
			"Launching Neovim with configuration: "+helpers.CyanText(selectedConfig),
		),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	helpers.LaunchNvimWithConfig(selectedConfig)

	return nil
}

func init() {
	rootCmd.AddCommand(configCmd)
}
