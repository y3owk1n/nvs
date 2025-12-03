package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/helpers"
	"github.com/y3owk1n/nvs/pkg/releases"
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
var uninstallCmd = &cobra.Command{
	Use:     "uninstall <version>",
	Aliases: []string{"rm", "remove", "un"},
	Short:   "Uninstall a specific version",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunUninstall(cmd, args)
	},
}

// RunUninstall executes the uninstall command.
func RunUninstall(cmd *cobra.Command, args []string) error {
	var err error
	logrus.Debug("Running uninstall command")

	// Normalize the provided version argument (e.g. add "v" prefix if missing)
	versionArg := releases.NormalizeVersion(args[0])
	logrus.Debug("Normalized version: ", versionArg)

	// Compute the path where the version is installed.
	versionPath := filepath.Join(VersionsDir, versionArg)
	logrus.Debug("Computed version path: ", versionPath)

	// Check if the version is installed.
	_, err = os.Stat(versionPath)
	if os.IsNotExist(err) {
		_, printErr := fmt.Fprintf(os.Stdout,
			"%s %s\n",
			helpers.ErrorIcon(),
			helpers.RedText(fmt.Sprintf("Version %s is not installed", versionArg)),
		)
		if printErr != nil {
			logrus.Warnf("Failed to write to stdout: %v", printErr)
		}
		logrus.Debug("Version not installed")

		return nil
	}

	currentSymlink := filepath.Join(VersionsDir, "current")

	// Check if the version to uninstall is currently active.
	isCurrent := false
	var current string
	current, err = helpers.GetCurrentVersion(VersionsDir)
	if err == nil {
		if current == versionArg {
			isCurrent = true
		}
	}

	// If the version is currently active, prompt for confirmation.
	if isCurrent {
		_, printErr := fmt.Fprintf(
			os.Stdout,
			"%s The version %s is currently in use. Do you really want to uninstall it? (y/N): ",
			helpers.WarningIcon(),
			helpers.CyanText(versionArg),
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
				helpers.InfoIcon(),
				helpers.WhiteText("Aborted uninstall."),
			)
			if printErr != nil {
				logrus.Warnf("Failed to write to stdout: %v", printErr)
			}
			logrus.Debug("Uninstall canceled by user")

			return nil
		}

		logrus.Debugf("User confirmed removal of current version %s", versionArg)
		// Remove the current symlink/junction if uninstalling the active version.
		var info os.FileInfo
		info, err = os.Lstat(currentSymlink)
		if err != nil {
			return fmt.Errorf("failed to lstat current symlink: %w", err)
		}

		if info.Mode()&os.ModeSymlink != 0 {
			// POSIX symlink
			err = os.Remove(currentSymlink)
			if err != nil {
				return fmt.Errorf("failed to remove current symlink: %w", err)
			}
			logrus.Debug("Removed current symlink")
		} else if info.IsDir() {
			// Likely a Windows junction — requires RemoveAll
			err := os.RemoveAll(currentSymlink)
			if err != nil {
				return fmt.Errorf("failed to remove current junction: %w", err)
			}
			logrus.Debug("Removed current junction")
		} else {
			logrus.Warnf("Current entry is neither a symlink nor a directory: %s", currentSymlink)
		}
	}

	logrus.Debug("Version is installed, proceeding with removal")
	// Remove the version's directory.
	err = os.RemoveAll(versionPath)
	if err != nil {
		return fmt.Errorf("failed to uninstall version %s: %w", versionArg, err)
	}

	successMsg := "Uninstalled version: " + helpers.CyanText(versionArg)
	logrus.Debug(successMsg)
	_, printErr := fmt.Fprintf(
		os.Stdout,
		"%s %s\n",
		helpers.SuccessIcon(),
		helpers.WhiteText(successMsg),
	)
	if printErr != nil {
		logrus.Warnf("Failed to write to stdout: %v", printErr)
	}

	// If the uninstalled version was the current version,
	// prompt the user to switch to a different installed version.
	if isCurrent {
		var versions []string
		versions, err = helpers.ListInstalledVersions(VersionsDir)
		if err != nil {
			return fmt.Errorf("error listing versions: %w", err)
		}

		var availableVersions []string
		// Exclude the "current" symlink entry.
		for _, entry := range versions {
			if entry != "current" {
				availableVersions = append(availableVersions, entry)
			}
		}

		if len(availableVersions) == 0 {
			_, printErr := fmt.Fprintf(os.Stdout,
				"%s %s\n",
				helpers.WarningIcon(),
				helpers.WhiteText(
					"No other versions available. Your current version has been unset.",
				),
			)
			if printErr != nil {
				logrus.Warnf("Failed to write to stdout: %v", printErr)
			}
		} else {
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
					_, printErr := fmt.Fprintf(os.Stdout, "%s %s\n", helpers.WarningIcon(), helpers.WhiteText("Selection canceled."))
					if printErr != nil {
						logrus.Warnf("Failed to write to stdout: %v", printErr)
					}

					return nil
				}
				return fmt.Errorf("prompt failed: %w", err)
			}

			// Use the selected version as the new current version.
			err = helpers.UseVersion(selectedVersion, currentSymlink, VersionsDir, GlobalBinDir)
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
