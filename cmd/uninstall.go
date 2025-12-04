package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/ui"
)

// ErrVersionNotInstalled is returned when attempting to uninstall a version that is not installed.
var ErrVersionNotInstalled = errors.New("version not installed")

// uninstallCmd represents the "uninstall" command (aliases: rm, remove, un).
// It uninstalls a specific installed Neovim version by removing its directory.
// If the version to be uninstalled is currently active, it prompts for confirmation before
// removing the "current" symlink and then proceeds to uninstall the version.
// After uninstalling the current version, if other versions exist, it allows the user to switch
// to a different installed version.
//
// Example usage:
//
//	nvs uninstall v0.6.0
//	nvs rm stable
var uninstallCmd = &cobra.Command{
	Use:     "uninstall <version>",
	Aliases: []string{"rm", "remove", "un"},
	Short:   "Uninstall a specific version",
	Args:    cobra.ExactArgs(1),
	RunE:    RunUninstall,
}

// RunUninstall executes the uninstall command.
func RunUninstall(cmd *cobra.Command, args []string) error {
	var err error

	logrus.Debug("Running uninstall command")

	versionArg := args[0]
	logrus.Debug("Requested version: ", versionArg)

	// Check if the version to uninstall is currently active.
	isCurrent := false

	current, err := GetVersionService().Current()
	if err == nil {
		// Normalize both versions for comparison
		normalize := func(v string) string {
			if !strings.HasPrefix(v, "v") {
				return "v" + v
			}

			return v
		}
		if normalize(current.Name()) == normalize(versionArg) {
			isCurrent = true
		}
	}

	// If the version is currently active, prompt for confirmation.
	if isCurrent {
		_, printErr := fmt.Fprintf(
			os.Stdout,
			"%s The version %s is currently in use. Do you really want to uninstall it? (y/N): ",
			ui.WarningIcon(),
			ui.CyanText(versionArg),
		)
		if printErr != nil {
			logrus.Warnf("Failed to write to stdout: %v", printErr)
		}

		reader := bufio.NewReader(os.Stdin)

		var input string

		input, err = reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		if strings.ToLower(input) != "y" {
			_, printErr := fmt.Fprintln(
				os.Stdout,
				ui.InfoIcon(),
				ui.WhiteText("Aborted uninstall."),
			)
			if printErr != nil {
				logrus.Warnf("Failed to write to stdout: %v", printErr)
			}

			logrus.Debug("Uninstall canceled by user")

			return nil
		}

		logrus.Debugf("User confirmed removal of current version %s", versionArg)
	}

	// Uninstall using service
	err = GetVersionService().Uninstall(versionArg, false)
	if err != nil {
		if strings.Contains(err.Error(), "not found") { // Should use errors.Is
			return fmt.Errorf("version %s is not installed: %w", versionArg, ErrVersionNotInstalled)
		}

		return fmt.Errorf("failed to uninstall version %s: %w", versionArg, err)
	}

	successMsg := "Uninstalled version: " + ui.CyanText(versionArg)
	logrus.Debug(successMsg)

	_, printErr := fmt.Fprintf(
		os.Stdout,
		"%s %s\n",
		ui.SuccessIcon(),
		ui.WhiteText(successMsg),
	)
	if printErr != nil {
		logrus.Warnf("Failed to write to stdout: %v", printErr)
	}

	// If the uninstalled version was the current version,
	// prompt the user to switch to a different installed version.
	if isCurrent {
		versions, err := GetVersionService().List()
		if err != nil {
			return fmt.Errorf("error listing versions: %w", err)
		}

		if len(versions) == 0 {
			_, printErr := fmt.Fprintf(os.Stdout,
				"%s %s\n",
				ui.WarningIcon(),
				ui.WhiteText(
					"No other versions available. Your current version has been unset.",
				),
			)
			if printErr != nil {
				logrus.Warnf("Failed to write to stdout: %v", printErr)
			}
		} else {
			var availableVersions []string
			for _, v := range versions {
				availableVersions = append(availableVersions, v.Name())
			}

			logrus.Debugf("Switchable Installed Neovim Versions: %v", availableVersions)
			prompt := promptui.Select{
				Label: "Switchable Installed Neovim Versions",
				Items: availableVersions,
			}

			logrus.Debug("Displaying selection prompt")

			var selectedVersion string

			_, selectedVersion, err = prompt.Run()
			if err != nil {
				if errors.Is(err, promptui.ErrInterrupt) {
					logrus.Debug("User canceled selection")

					_, printErr := fmt.Fprintf(os.Stdout, "%s %s\n", ui.WarningIcon(), ui.WhiteText("Selection canceled."))
					if printErr != nil {
						logrus.Warnf("Failed to write to stdout: %v", printErr)
					}

					return nil
				}

				return fmt.Errorf("prompt failed: %w", err)
			}

			// Use the selected version as the new current version.
			err = GetVersionService().Use(context.Background(), selectedVersion)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// init registers the uninstallCmd with the root command.
func init() {
	rootCmd.AddCommand(uninstallCmd)
}
