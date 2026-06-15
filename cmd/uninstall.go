package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/domain/vtypes"
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
	logrus.Debug("Running uninstall command")

	var versionArg string

	// Check if --pick flag is set
	pick, _ := cmd.Flags().GetBool("pick")
	if pick {
		selected, err := pickUninstallVersion()
		if err != nil {
			return err
		}

		versionArg = selected
	} else {
		if len(args) == 0 {
			return fmt.Errorf("%w", ErrVersionArgRequired)
		}

		versionArg = args[0]
	}

	logrus.Debugf("Requested version: %s", versionArg)

	// Check if the version to uninstall is currently active.
	//
	// Current() can fail in two distinct ways:
	//   - os.ErrNotExist: no current symlink has ever been set
	//     (the user has installed versions but never used one),
	//     in which case the target cannot be "current" and we
	//     can safely skip the confirmation prompt.
	//   - any other error (broken symlink, junction resolve
	//     failure, permission error): we genuinely don't know
	//     whether the target is the active version, so we
	//     must require explicit confirmation rather than
	//     silently treat it as "not current" and let the
	//     service layer's internal Current() check fail
	//     the same way — leaving the current symlink
	//     dangling in the worst case.
	isCurrent := false

	current, err := GetVersionService().Current()
	switch {
	case err == nil:
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
	case errors.Is(err, os.ErrNotExist):
		// No current symlink at all — target cannot be current.
	default:
		logrus.Warnf(
			"Could not determine current version; requiring explicit confirmation: %v",
			err,
		)

		isCurrent = true
	}

	// If the version is currently active, prompt for confirmation.
	//
	// We keep the bufio.Reader y/N prompt (rather than
	// upgrading to ui.Picker.Confirm) so the command still
	// works when stdin is piped (e.g. `echo y | nvs uninstall
	// v0.6.0`). ui.Picker.Confirm refuses to run in non-TTY
	// mode by design; the y/N text path accepts piped input
	// cleanly, which is the existing behavior callers may
	// rely on.
	if isCurrent {
		ui.Message.Warnf(
			"The version %s is currently in use. Do you really want to uninstall it? (y/N): ",
			ui.Message.Accent(versionArg),
		)

		reader := bufio.NewReader(os.Stdin)

		var input string

		input, err = reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		if strings.ToLower(input) != "y" {
			ui.Message.Infof("Aborted uninstall.")

			logrus.Debug("Uninstall canceled by user")

			return nil
		}

		logrus.Debugf("User confirmed removal of current version %s", versionArg)
	}

	// Uninstall using service
	// Force uninstall if it's the current version (user already confirmed)
	err = GetVersionService().Uninstall(versionArg, isCurrent)
	if err != nil {
		if errors.Is(err, vtypes.ErrVersionNotFound) {
			return fmt.Errorf("version %s is not installed: %w", versionArg, ErrVersionNotInstalled)
		}

		return fmt.Errorf("failed to uninstall version %s: %w", versionArg, err)
	}

	logrus.Debugf("Uninstalled version: %s", versionArg)

	ui.Message.Successf("Uninstalled version: %s", ui.Message.Accent(versionArg))

	// If the uninstalled version was the current version,
	// prompt the user to switch to a different installed version.
	if isCurrent {
		return promptSwitchAfterUninstall(cmd)
	}

	return nil
}

// pickUninstallVersion shows the installed-versions picker
// and returns the version name the user chose.
func pickUninstallVersion() (string, error) {
	versions, err := GetVersionService().List()
	if err != nil {
		return "", fmt.Errorf("error listing versions: %w", err)
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("%w for selection", ErrNoVersionsAvailable)
	}

	items := make([]ui.SelectItem, 0, len(versions))
	for _, v := range versions {
		items = append(items, ui.SelectItem{Label: v.Name()})
	}

	selected, err := ui.Picker.NewPicker(nil, nil).Select("Select version to uninstall", items)
	if err != nil {
		if errors.Is(err, ui.Picker.ErrCanceled()) {
			ui.Message.Warnf("Selection canceled.")

			return "", nil
		}

		return "", fmt.Errorf("prompt failed: %w", err)
	}

	return selected, nil
}

// promptSwitchAfterUninstall runs the "pick a new current
// version" sub-flow that follows an uninstall-of-current.
// It is split out so RunUninstall's control flow stays
// readable; the sub-flow is only entered when isCurrent is
// true.
func promptSwitchAfterUninstall(cmd *cobra.Command) error {
	versions, err := GetVersionService().List()
	if err != nil {
		return fmt.Errorf("error listing versions: %w", err)
	}

	if len(versions) == 0 {
		ui.Message.Warnf("No other versions available. Your current version has been unset.")

		return nil
	}

	items := make([]ui.SelectItem, 0, len(versions))
	for _, v := range versions {
		items = append(items, ui.SelectItem{Label: v.Name()})
	}

	logrus.Debugf("Switchable installed Neovim versions: %d", len(items))

	selected, err := ui.Picker.NewPicker(nil, nil).
		Select("Switchable Installed Neovim Versions", items)
	if err != nil {
		if errors.Is(err, ui.Picker.ErrCanceled()) {
			ui.Message.Warnf("Selection canceled.")

			return nil
		}

		return fmt.Errorf("prompt failed: %w", err)
	}

	// Use the selected version as the new current version.
	_, err = GetVersionService().Use(cmd.Context(), selected)

	return err
}

// init registers the uninstallCmd with the root command.
func init() {
	rootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().BoolP("pick", "p", false, "Launch interactive picker to select version")
}
