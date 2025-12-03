package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/helpers"
)

// listCmd represents the "list" command (aliases: ls).
// It lists all installed Neovim versions found in the versions directory and marks the current active version.
// If no versions are installed, it informs the user.
//
// Example usage:
//
//	nvs list
//	nvs ls
var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List installed versions",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunList(cmd, args, VersionsDir)
	},
}

// RunList executes the list command.
func RunList(_ *cobra.Command, _ []string, versionsDir string) error {
	logrus.Debug("Executing list command")

	// Retrieve installed versions from the versions directory.
	versions, err := helpers.ListInstalledVersions(versionsDir)
	if err != nil {
		return fmt.Errorf("error listing versions: %w", err)
	}

	logrus.Debugf("Found %d installed versions", len(versions))

	// If no versions are installed, display a message and exit.
	if len(versions) == 0 {
		_, err = fmt.Fprintf(
			os.Stdout,
			"%s %s\n",
			helpers.InfoIcon(),
			helpers.WhiteText("No installed versions..."),
		)
		if err != nil {
			logrus.Warnf("Failed to write to stdout: %v", err)
		}

		logrus.Debug("No installed versions found")

		return nil
	}

	// Get the current active version.
	current, err := helpers.GetCurrentVersion(versionsDir)
	if err != nil {
		logrus.Warn("No current version set or unable to determine the current version")

		current = "none"
	}

	logrus.Debugf("Current version: %s", current)

	// Set up a table for displaying versions and their status.
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Version", "Status"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
	)
	table.SetTablePadding("1")
	table.SetBorder(false)
	table.SetRowLine(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetAutoWrapText(false)

	// Append each version to the table.
	for _, version := range versions {
		var row []string
		if version == current {
			// Mark the current version with an arrow and use a highlighted green color.
			row = []string{
				color.New(color.Bold, color.FgHiGreen).Sprintf("→ %s", version),
				color.New(color.Bold, color.FgHiGreen).Sprintf("Current"),
			}
			logrus.Debugf("Marked version %s as current", version)
		} else {
			row = []string{version, "Installed"}
		}

		table.Append(row)
	}

	// Render the table to the standard output.
	table.Render()

	return nil
}

// init registers the listCmd with the root command.
func init() {
	rootCmd.AddCommand(listCmd)
}
