package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/ui"
)

// pinCmd represents the "pin" command.
// It writes the current or specified version to a .nvs-version file in the current directory.
var pinCmd = &cobra.Command{
	Use:   "pin [version]",
	Short: "Pin a Neovim version for the current directory",
	Long: `Write a .nvs-version file to the current directory.
If no version is specified, uses the currently active version.

This file can be used to ensure consistent Neovim versions across a team.
Use 'nvs use' in a directory with .nvs-version to automatically use that version.`,
	Args: cobra.MaximumNArgs(1),
	RunE: RunPin,
}

// RunPin executes the pin command.
func RunPin(cmd *cobra.Command, args []string) error {
	var versionToPin string

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
			Label: "Select version to pin",
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

		versionToPin = selectedVersion
	} else {
		if len(args) > 0 {
			versionToPin = args[0]
			logrus.Debugf("Pinning specified version: %s", versionToPin)
		} else {
			// Use currently active version
			current, err := VersionServiceFromContext(cmd.Context()).Current()
			if err != nil {
				return fmt.Errorf("no version specified and no current version: %w", err)
			}

			versionToPin = current.Name()
			logrus.Debugf("Pinning current version: %s", versionToPin)
		}
	}

	// Get directory to write to (current working directory by default)
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	global, _ := cmd.Flags().GetBool("global")
	if global {
		// Write to user's home directory
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return fmt.Errorf("failed to get home directory: %w", homeErr)
		}

		dir = home
	}

	versionFile := filepath.Join(dir, constants.VersionFileName)

	// Write version to file
	err = os.WriteFile(versionFile, []byte(versionToPin+"\n"), constants.FilePerm)
	if err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	_, err = fmt.Fprintf(
		os.Stdout,
		"%s %s\n",
		ui.SuccessIcon(),
		ui.WhiteText(fmt.Sprintf("Pinned version %s to %s", versionToPin, versionFile)),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	return nil
}

// ReadVersionFile reads the .nvs-version file from the directory hierarchy.
// It searches from startDir up to the root, returning the first version found.
// If global is true, also checks the user's home directory.
func ReadVersionFile(startDir string, checkGlobal bool) (string, string, error) {
	var (
		homeVisited bool
		homeDir     string
	)

	if checkGlobal {
		h, err := os.UserHomeDir()
		if err == nil {
			homeDir = h
		}
	}

	// Search up the directory tree
	dir := startDir
	for {
		if homeDir != "" && dir == homeDir {
			homeVisited = true
		}

		versionFile := filepath.Join(dir, constants.VersionFileName)

		data, err := os.ReadFile(versionFile)
		if err == nil {
			version := strings.TrimSpace(string(data))
			if version != "" {
				logrus.Debugf("Found version %s in %s", version, versionFile)

				return version, versionFile, nil
			}
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}

		dir = parent
	}

	// Check global version file in home directory if not already visited
	if checkGlobal && !homeVisited && homeDir != "" {
		globalFile := filepath.Join(homeDir, constants.VersionFileName)

		data, err := os.ReadFile(globalFile)
		if err == nil {
			version := strings.TrimSpace(string(data))
			if version != "" {
				logrus.Debugf("Found global version %s in %s", version, globalFile)

				return version, globalFile, nil
			}
		}
	}

	return "", "", ErrVersionFileNotFound
}

// init registers the pinCmd with the root command.
func init() {
	rootCmd.AddCommand(pinCmd)
	pinCmd.Flags().
		BoolP("global", "g", false, "Write to home directory instead of current directory")
	pinCmd.Flags().BoolP("pick", "p", false, "Launch interactive picker to select version")
}
