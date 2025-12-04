// Package cmd contains the CLI commands for nvs.
package cmd

import (
	"errors"
	"fmt"
	"os"

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

		err = GetConfigService().Launch(args[0])
		if err != nil {
			return err
		}

		return nil
	}

	// List available configurations
	configs, err := GetConfigService().List()
	if err != nil {
		return fmt.Errorf("failed to list configurations: %w", err)
	}

	if len(configs) == 0 {
		logrus.Debug("No Neovim configurations found")

		_, err := fmt.Fprintf(os.Stdout,
			"%s %s\n",
			helpers.WarningIcon(),
			helpers.WhiteText(
				"No Neovim configurations found",
			),
		)
		if err != nil {
			logrus.Warnf("Failed to write to stdout: %v", err)
		}

		return nil
	}

	logrus.Debugf("Available Neovim configurations: %v", configs)
	prompt := promptui.Select{
		Label: "Select Neovim configuration",
		Items: configs,
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

	err = GetConfigService().Launch(selectedConfig)
	if err != nil {
		return fmt.Errorf("failed to launch nvim with config: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(configCmd)
}
