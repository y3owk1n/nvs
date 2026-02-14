package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/domain/version"
	"github.com/y3owk1n/nvs/internal/ui"
)

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
//	nvs uninstall --pick
var uninstallCmd = &cobra.Command{
	Use:     "uninstall [version]",
	Aliases: []string{"rm", "remove", "un"},
	Short:   "Uninstall a specific version",
	Args:    cobra.MaximumNArgs(1),
	RunE:    RunUninstall,
}

// RunUninstall executes the uninstall command.
func RunUninstall(cmd *cobra.Command, args []string) error {
	var err error

	logrus.Debug("Running uninstall command")

	var versionArg string

	// Check if --pick flag is set
	pick, _ := cmd.Flags().GetBool("pick")
	if pick {
		// Launch picker for installed versions
		versions, err := VersionServiceFromContext(cmd.Context()).List()
		if err != nil {
			return fmt.Errorf("error listing versions: %w", err)
		}

		if len(versions) == 0 {
			return fmt.Errorf("%w for selection", ErrNoVersionsAvailable)
		}

		availableVersions := make([]string, 0, len(versions))
		for _, v := range versions {
			availableVersions = append(availableVersions, v.Name())
		}

		prompt := promptui.Select{
			Label: "Select version to uninstall",
			Items: availableVersions,
		}

		_, selectedVersion, err := prompt.Run()
		if err != nil {
			if errors.Is(err, promptui.ErrInterrupt) {
				_, printErr := fmt.Fprintf(
					os.Stdout,
					"%s %s\n",
					ui.WarningIcon(),
					ui.WhiteText("Selection canceled."),
				)
				if printErr != nil {
					logrus.Warnf("Failed to write to stdout: %v", printErr)
				}

				return nil
			}

			return fmt.Errorf("prompt failed: %w", err)
		}

		versionArg = selectedVersion
	} else {
		if len(args) == 0 {
			return fmt.Errorf("%w", ErrVersionArgRequired)
		}

		versionArg = args[0]
	}

	logrus.Debug("Requested version: ", versionArg)

	// Check if the version to uninstall is currently active.
	isCurrent := false

	current, err := VersionServiceFromContext(cmd.Context()).Current()
	if err == nil {
		// Normalize both versions for comparison
		normalizedCurrent := current.Name()
		normalizedArg := versionArg

		if !strings.HasPrefix(normalizedCurrent, "v") {
			normalizedCurrent = "v" + normalizedCurrent
		}

		if !strings.HasPrefix(normalizedArg, "v") {
			normalizedArg = "v" + normalizedArg
		}

		if normalizedCurrent == normalizedArg {
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
	// Force uninstall if it's the current version (user already confirmed)
	err = VersionServiceFromContext(cmd.Context()).Uninstall(versionArg, isCurrent)
	if err != nil {
		if errors.Is(err, version.ErrVersionNotFound) {
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
		versions, err := VersionServiceFromContext(cmd.Context()).List()
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
			availableVersions := make([]string, 0, len(versions))
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

					_, printErr := fmt.Fprintf(
						os.Stdout,
						"%s %s\n",
						ui.WarningIcon(),
						ui.WhiteText("Selection canceled."),
					)
					if printErr != nil {
						logrus.Warnf("Failed to write to stdout: %v", printErr)
					}

					return nil
				}

				return fmt.Errorf("prompt failed: %w", err)
			}

			// Use the selected version as the new current version.
			_, err = VersionServiceFromContext(cmd.Context()).Use(cmd.Context(), selectedVersion)
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
	uninstallCmd.Flags().BoolP("pick", "p", false, "Launch interactive picker to select version")
}
