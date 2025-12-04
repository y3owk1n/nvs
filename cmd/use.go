package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
var useCmd = &cobra.Command{
	Use:   "use <version|stable|nightly|commit-hash>",
	Short: "Switch to a specific version or commit hash",
	Args:  cobra.ExactArgs(1),
	RunE:  RunUse,
}

// RunUse executes the use command.
func RunUse(cmd *cobra.Command, args []string) error {
	// Create a context with a timeout for the operation.
	ctx, cancel := context.WithTimeout(cmd.Context(), TimeoutMinutes*time.Minute)
	defer cancel()

	alias := args[0]
	logrus.Debugf("Requested version: %s", alias)

	// Use version service to switch
	err := GetVersionService().Use(ctx, alias)
	if err != nil {
		// If version not found, suggest installing
		if err.Error() == "version not found" { // Simplified check, should use error wrapping check
			logrus.Infof("Version %s not found. Installing...", alias)
			// Fallback to install
			return RunInstall(cmd, args, GetVersionsDir(), GetCacheFilePath())
		}
		return err
	}

	_, err = fmt.Fprintf(
		os.Stdout,
		"%s %s\n",
		ui.SuccessIcon(),
		ui.WhiteText(fmt.Sprintf("Switched to %s", alias)),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	return nil
}

// init registers the useCmd with the root command.
func init() {
	rootCmd.AddCommand(useCmd)
}
