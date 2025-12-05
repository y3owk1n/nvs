package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/domain/version"
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
	ctx, cancel := context.WithTimeout(cmd.Context(), TimeoutMinutes*time.Minute)
	defer cancel()

	var alias string
	if len(args) > 0 {
		alias = args[0]
	} else {
		// Try to read from .nvs-version file
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		version, versionFile, err := ReadVersionFile(cwd, true)
		if err != nil {
			return fmt.Errorf("no version specified and %w", err)
		}

		alias = version
		logrus.Debugf("Using version %s from %s", alias, versionFile)

		_, printErr := fmt.Fprintf(os.Stdout, "%s Using version from %s\n", ui.InfoIcon(), versionFile)
		if printErr != nil {
			logrus.Warnf("Failed to write to stdout: %v", printErr)
		}
	}

	logrus.Debugf("Requested version: %s", alias)

	// Check for running Neovim instances unless --force is set
	force, _ := cmd.Flags().GetBool("force")
	if !force {
		running, count := platform.IsNeovimRunning()
		if running {
			logrus.Debugf("Detected %d running Neovim instance(s)", count)

			_, printErr := fmt.Fprintf(
				os.Stdout,
				"%s Neovim is currently running (%d instance(s)). Switching versions may cause issues.\n",
				ui.WarningIcon(),
				count,
			)
			if printErr != nil {
				logrus.Warnf("Failed to write to stdout: %v", printErr)
			}

			// Prompt for confirmation
			_, printErr = fmt.Fprint(os.Stdout, "Do you want to continue? [y/N]: ")
			if printErr != nil {
				logrus.Warnf("Failed to write to stdout: %v", printErr)
			}

			reader := bufio.NewReader(os.Stdin)

			response, readErr := reader.ReadString('\n')
			if readErr != nil {
				return fmt.Errorf("failed to read response: %w", readErr)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				_, printErr = fmt.Fprintf(os.Stdout, "%s Operation canceled.\n", ui.InfoIcon())
				if printErr != nil {
					logrus.Warnf("Failed to write to stdout: %v", printErr)
				}

				return nil
			}
		}
	}

	// Use version service to switch
	resolvedVersion, err := GetVersionService().Use(ctx, alias)
	if err != nil {
		// If version not found, install it first, then try to use again
		if errors.Is(err, version.ErrVersionNotFound) {
			logrus.Infof("Version %s not found. Installing...", alias)
			// Install the version
			err = RunInstall(cmd, []string{alias})
			if err != nil {
				return err
			}
			// Now try to use it (single retry, no recursion)
			resolvedVersion, err = GetVersionService().Use(ctx, alias)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	_, err = fmt.Fprintf(
		os.Stdout,
		"%s %s\n",
		ui.SuccessIcon(),
		ui.WhiteText("Switched to "+resolvedVersion),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	return nil
}

// init registers the useCmd with the root command.
func init() {
	rootCmd.AddCommand(useCmd)
	useCmd.Flags().BoolP("force", "f", false, "Skip running instance check")
}
