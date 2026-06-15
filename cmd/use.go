package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/domain/vtypes"
	"github.com/y3owk1n/nvs/internal/platform"
	"github.com/y3owk1n/nvs/internal/ui"
)

// useCmd represents the "use" command.
// It switches the active Neovim version to a specific version, stable, nightly, or a commit hash.
// If the requested version is not installed, it is either built (if a commit hash) or installed.
// Finally, it updates the "current" symlink to point to the target version.
//
// Example usage:
//
//	nvs use stable
//	nvs use v0.6.0
//	nvs use nightly
//	nvs use 1a2b3c4 (a commit hash)
//	nvs use        (reads from .nvs-version file)
var useCmd = &cobra.Command{
	Use:   "use [version|stable|nightly|commit-hash]",
	Short: "Switch to a specific version or commit hash",
	Long: `Switch to a specific Neovim version.
If no version is specified, reads from .nvs-version file in the current
directory or parent directories.`,
	Args: cobra.MaximumNArgs(1),
	RunE: RunUse,
}

// RunUse executes the use command.
func RunUse(cmd *cobra.Command, args []string) error {
	// Create a context with a timeout for the operation.
	ctx, cancel := context.WithTimeout(cmd.Context(), constants.TimeoutMinutes*time.Minute)
	defer cancel()

	var alias string

	// Check if --pick flag is set
	pick, _ := cmd.Flags().GetBool("pick")
	if pick {
		// Launch picker for installed versions
		versions, err := GetVersionService().List()
		if err != nil {
			return fmt.Errorf("error listing versions: %w", err)
		}

		if len(versions) == 0 {
			return fmt.Errorf("%w for selection", ErrNoVersionsAvailable)
		}

		promptItems := make([]ui.SelectItem, 0, len(versions))
		for _, v := range versions {
			promptItems = append(promptItems, ui.SelectItem{Label: v.Name()})
		}

		selectedVersion, err := ui.Picker.NewPicker(os.Stdin, os.Stdout).
			Select("Select version to use", promptItems)
		if err != nil {
			if errors.Is(err, ui.Picker.ErrCanceled()) {
				ui.Message.Warnf("Selection canceled.")

				return nil
			}

			return fmt.Errorf("picker: %w", err)
		}

		alias = selectedVersion
	} else {
		if len(args) > 0 {
			alias = args[0]
		} else {
			// Try to read from .nvs-version file
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			pinnedVersion, versionFile, err := ReadVersionFile(cwd, true)
			if err != nil {
				return fmt.Errorf("no version specified and %w", err)
			}

			alias = pinnedVersion
			logrus.Debugf("Using version %s from %s", alias, versionFile)

			ui.Message.Infof(
				"Using version from %s",
				ui.Message.Accent(versionFile),
			)
		}
	}

	logrus.Debugf("Requested version: %s", alias)

	// Check for running Neovim instances unless --force is set
	force, _ := cmd.Flags().GetBool("force")
	if !force {
		running, count := platform.IsNeovimRunning()
		if running {
			logrus.Debugf("Detected %d running Neovim instance(s)", count)

			ui.Message.Warnf(
				"Neovim is currently running (%d instance(s)). Switching versions may cause issues.",
				count,
			)

			// Use ConfirmScriptable for the same reason the
			// destructive prompts in reset/uninstall/path
			// do: TTY users get a huh Yes/No form, pipe
			// users get the scriptable "[y/N]:" fallback,
			// and there is no need to keep a hand-rolled
			// bufio.Reader path just for the non-TTY case.
			confirmed, err := ui.Picker.ConfirmScriptable(
				"Do you want to continue?",
			)
			if err != nil {
				return fmt.Errorf("failed to read confirmation: %w", err)
			}

			if !confirmed {
				ui.Message.Infof("Operation canceled.")

				return nil
			}
		}
	}

	// Use version service to switch
	resolvedVersion, err := GetVersionService().Use(ctx, alias)
	if err != nil {
		// If version not found, install it first, then try to use again
		if errors.Is(err, vtypes.ErrVersionNotFound) {
			logrus.Infof("Version %s not found. Installing...", alias)
			// Install the version using the shared install path. We
			// must NOT call RunInstall(cmd, ...) here: it would
			// re-read the --pick flag from this (use) command and
			// launch a second picker, even though the user has
			// already selected a version via 'use --pick'.
			installErr := runInstallForAlias(ctx, cmd, alias)
			if installErr != nil {
				return installErr
			}

			// Now try to use it (single retry, no recursion)
			resolvedVersion, err = GetVersionService().Use(ctx, alias)
			if err != nil {
				return fmt.Errorf("failed to activate %s: %w", alias, err)
			}
		} else {
			return err
		}
	}

	ui.Message.Successf("Switched to %s", ui.Message.Accent(resolvedVersion))

	return nil
}

// init registers the useCmd with the root command.
func init() {
	rootCmd.AddCommand(useCmd)
	useCmd.Flags().BoolP("force", "f", false, "Skip running instance check")
	useCmd.Flags().BoolP("pick", "p", false, "Launch interactive picker to select version")
}
