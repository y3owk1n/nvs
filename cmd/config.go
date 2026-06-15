// Package cmd contains the CLI commands for nvs.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/ui"
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

		ui.Message.Infof(
			"Launching Neovim with configuration: %s",
			ui.Message.Accent(args[0]),
		)

		err := GetConfigService().Launch(cmd.Context(), args[0])
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

		ui.Message.Warnf("No Neovim configurations found")

		return nil
	}

	logrus.Debugf("Available Neovim configurations: %v", configs)

	// Build the picker items from the configs slice. The
	// picker façade handles the TTY / non-TTY split — it
	// returns ErrNoTTY for non-interactive input, which the
	// caller treats as a graceful "selection canceled" exit
	// (matching the previous promptui.ErrInterrupt behavior).
	items := make([]ui.SelectItem, 0, len(configs))
	for _, configName := range configs {
		items = append(items, ui.SelectItem{Label: configName})
	}

	selectedConfig, err := ui.Picker.NewPicker(os.Stdin, os.Stdout).
		Select("Select Neovim configuration", items)
	if err != nil {
		if ui.Picker.IsNoTTY(err) {
			// Non-TTY: tell the user the command needs an
			// interactive terminal. Returning an error here
			// (rather than silently aborting) is the right
			// answer for `nvs config` because there is no
			// scriptable alternative — the only inputs are
			// the config name (which goes via args[0] and
			// was just handled above) and the picker.
			ui.Message.Warnf(
				"Selection canceled: stdin is not a TTY. Pass a config name as an argument (e.g. 'nvs config myconfig').",
			)

			return nil
		}

		if errors.Is(err, ui.Picker.ErrCanceled()) {
			ui.Message.Warnf("Selection canceled.")

			return nil
		}

		return fmt.Errorf("picker: %w", err)
	}

	logrus.Debugf("User selected configuration: %s", selectedConfig)

	ui.Message.Infof(
		"Launching Neovim with configuration: %s",
		ui.Message.Accent(selectedConfig),
	)

	err = GetConfigService().Launch(cmd.Context(), selectedConfig)
	if err != nil {
		return fmt.Errorf("failed to launch nvim with config: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(configCmd)
}
