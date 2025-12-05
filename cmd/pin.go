package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/ui"
)

const (
	// VersionFileName is the name of the version sync file.
	VersionFileName = ".nvs-version"
	// filePerm is the file permission for the version file.
	filePerm = 0o644
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
	RunE: runPin,
}

// runPin executes the pin command.
func runPin(cmd *cobra.Command, args []string) error {
	var versionToPin string

	if len(args) > 0 {
		versionToPin = args[0]
		logrus.Debugf("Pinning specified version: %s", versionToPin)
	} else {
		// Use currently active version
		current, err := GetVersionService().Current()
		if err != nil {
			return fmt.Errorf("no version specified and no current version: %w", err)
		}

		versionToPin = current.Name()
		logrus.Debugf("Pinning current version: %s", versionToPin)
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

	versionFile := filepath.Join(dir, VersionFileName)

	// Write version to file
	err = os.WriteFile(versionFile, []byte(versionToPin+"\n"), filePerm)
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

		versionFile := filepath.Join(dir, VersionFileName)

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
		globalFile := filepath.Join(homeDir, VersionFileName)

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
}
