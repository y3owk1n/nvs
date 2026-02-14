package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/ui"
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
	RunE:    RunList,
}

// RunList executes the list command.
func RunList(cmd *cobra.Command, _ []string) error {
	logrus.Debug("Executing list command")

	// Retrieve installed versions from the version service.
	versions, err := GetVersionService().List()
	if err != nil {
		return fmt.Errorf("error listing versions: %w", err)
	}

	logrus.Debugf("Found %d installed versions", len(versions))

	// If no versions are installed, display a message and exit.
	if len(versions) == 0 {
		_, err = fmt.Fprintf(
			os.Stdout,
			"%s %s\n",
			ui.InfoIcon(),
			ui.WhiteText("No installed versions..."),
		)
		if err != nil {
			logrus.Warnf("Failed to write to stdout: %v", err)
		}

		logrus.Debug("No installed versions found")

		return nil
	}

	// Get the current active version.
	current, err := GetVersionService().Current()
	if err != nil {
		logrus.Warn("No current version set or unable to determine the current version")
	} else {
		logrus.Debugf("Current version: %s", current.Name())
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		// Output in JSON format
		type VersionInfo struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Type   string `json:"type"`
		}

		var infos []VersionInfo
		for _, version := range versions {
			status := "installed"
			if current.Name() != "" && version.Name() == current.Name() {
				status = "current"
			}

			infos = append(infos, VersionInfo{
				Name:   version.Name(),
				Status: status,
				Type:   version.Type().String(),
			})
		}

		data := map[string]any{"versions": infos}

		return outputJSON(data)
	}

	// Set up a table for displaying versions and their status.
	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithRendition(tw.Rendition{
			Borders:  tw.BorderNone,
			Settings: tw.Settings{Separators: tw.Separators{BetweenRows: tw.Off}},
		}),
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{
				Alignment: tw.CellAlignment{Global: tw.AlignLeft},
			},
			Row: tw.CellConfig{
				Alignment: tw.CellAlignment{Global: tw.AlignLeft},
			},
		}),
	)
	table.Header([]string{"Version", "Status"})

	// Append each version to the table.
	for _, version := range versions {
		var row []string
		if current.Name() != "" && version.Name() == current.Name() {
			// Mark the current version with an arrow and use a highlighted green color.
			row = []string{
				color.New(color.Bold, color.FgHiGreen).Sprintf("→ %s", version.Name()),
				color.New(color.Bold, color.FgHiGreen).Sprintf("Current"),
			}
			logrus.Debugf("Marked version %s as current", version.Name())
		} else {
			row = []string{version.Name(), "Installed"}
		}

		err := table.Append(row)
		if err != nil {
			return err
		}
	}

	// Render the table to the standard output.
	err = table.Render()
	if err != nil {
		return err
	}

	return nil
}

// init registers the listCmd with the root command.
func init() {
	listCmd.Flags().Bool("json", false, "Output in JSON format")
	rootCmd.AddCommand(listCmd)
}
