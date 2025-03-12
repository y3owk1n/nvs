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

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List installed versions",
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Debug("Executing list command")

		versions, err := utils.ListInstalledVersions(versionsDir)
		if err != nil {
			logrus.Fatalf("Error listing versions: %v", err)
		}

		logrus.Debugf("Found %d installed versions", len(versions))

		if len(versions) == 0 {
			fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText("No installed versions..."))
			logrus.Debug("No installed versions found")
			return
		}

		current, err := utils.GetCurrentVersion(versionsDir)
		if err != nil {
			logrus.Warn("No current version set or unable to determine the current version")
			current = "none"
		}
		logrus.Debugf("Current version: %s", current)

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

		for _, v := range versions {
			var row []string
			if v == current {
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

		table.Render()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
