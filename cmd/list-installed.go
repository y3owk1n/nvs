package cmd

import (
	"os"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/utils"
)

var listInstalledCmd = &cobra.Command{
	Use:     "list-installed",
	Aliases: []string{"ls"},
	Short:   "List installed versions",
	Run: func(cmd *cobra.Command, args []string) {
		versions, err := utils.ListInstalledVersions(versionsDir)
		if err != nil {
			logrus.Fatalf("Error listing versions: %v", err)
		}

		current, err := utils.GetCurrentVersion(versionsDir)
		if err != nil {
			logrus.Warn("No current version set or unable to determine the current version")
			current = "none"
		}

		// Create a modern table using tablewriter.
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Version", "Status"})

		// Set header color and styling.
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
		)
		table.SetBorder(true)
		table.SetRowLine(true)
		table.SetCenterSeparator("│")
		table.SetColumnSeparator("│")
		table.SetAutoWrapText(false)

		// Append rows with installed version details.
		for _, v := range versions {
			var row []string
			if v == current {
				// Emphasize the current version with a green arrow and bold text.
				row = []string{
					color.New(color.Bold, color.FgHiGreen).Sprintf("→ %s", v),
					color.New(color.Bold, color.FgHiGreen).Sprintf("Current"),
				}
			} else {
				row = []string{v, "Installed"}
			}
			table.Append(row)
		}

		table.Render()
	},
}

func init() {
	rootCmd.AddCommand(listInstalledCmd)
}
