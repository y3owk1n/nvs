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
		versions, err := utils.ListInstalledVersions(versionsDir)
		if err != nil {
			logrus.Fatalf("Error listing versions: %v", err)
		}

		if len(versions) == 0 {
			fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText("No installed versions..."))
			return
		}

		current, err := utils.GetCurrentVersion(versionsDir)
		if err != nil {
			logrus.Warn("No current version set or unable to determine the current version")
			current = "none"
		}

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
