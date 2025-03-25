package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/utils"
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
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Debug("Executing list command")

		// Retrieve installed versions from the versions directory.
		versions, err := utils.ListInstalledVersions(versionsDir)
		if err != nil {
			logrus.Fatalf("Error listing versions: %v", err)
		}
		logrus.Debugf("Found %d installed versions", len(versions))

		// If no versions are installed, display a message and exit.
		if len(versions) == 0 {
			fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText("No installed versions..."))
			logrus.Debug("No installed versions found")
			return
		}

		// Get the current active version.
		current, err := utils.GetCurrentVersion(versionsDir)
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
		for _, v := range versions {
			var row []string
			if v == current {
				// Mark the current version with an arrow and use a highlighted green color.
				row = []string{
					color.New(color.Bold, color.FgHiGreen).Sprintf("→ %s", v),
					color.New(color.Bold, color.FgHiGreen).Sprintf("Current"),
				}
				logrus.Debugf("Marked version %s as current", v)
			} else {
				row = []string{v, "Installed"}
			}
			table.Append(row)
		}

		// Render the table to the standard output.
		table.Render()
	},
}

// init registers the listCmd with the root command.
func init() {
	rootCmd.AddCommand(listCmd)
}
