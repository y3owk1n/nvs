package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

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

		// Create a tab writer for a formatted table.
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		// Table header.
		fmt.Fprintln(w, "VERSION\tSTATUS")
		fmt.Fprintln(w, "-------\t------")
		for _, v := range versions {
			status := ""
			if v == current {
				status = "Current"
			} else {
				status = "Installed"
			}
			fmt.Fprintf(w, "%s\t%s\n", v, status)
		}
		w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(listInstalledCmd)
}
