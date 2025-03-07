package cmd

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/releases"
	"github.com/y3owk1n/nvs/pkg/utils"
)

var listRemoteCmd = &cobra.Command{
	Use:   "list-remote [force]",
	Short: "List available remote versions (cached for 5 minutes)",
	Run: func(cmd *cobra.Command, args []string) {
		force := len(args) > 0 && args[0] == "force"
		releasesResult, err := releases.GetCachedReleases(force, cacheFilePath)
		if err != nil {
			logrus.Fatalf("Error fetching releases: %v", err)
		}

		stableRelease, err := releases.FindLatestStable(cacheFilePath)
		stableTag := ""
		if err == nil {
			stableTag = stableRelease.TagName
		} else {
			stableTag = "stable"
		}

		// Group releases into three slices:
		// - Nightly releases (prereleases)
		// - Stable release (tag equals "stable")
		// - Other versions (non-prerelease and not "stable")
		var groupNightly []releases.Release
		var groupStable []releases.Release
		var groupOthers []releases.Release

		for _, r := range releasesResult {
			if r.Prerelease {
				groupNightly = append(groupNightly, r)
			} else {
				if r.TagName == "stable" {
					groupStable = append(groupStable, r)
				} else {
					groupOthers = append(groupOthers, r)
				}
			}
		}

		// Combine the groups in order: nightly, stable, then others.
		combined := append(append(groupNightly, groupStable...), groupOthers...)

		current, err := utils.GetCurrentVersion(versionsDir)
		if err != nil {
			current = ""
		}

		// Create a modern table using tablewriter.
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Tag", "Type", "Details", "Status"})
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
		)
		table.SetBorder(true)
		table.SetRowLine(true)
		table.SetCenterSeparator("│")
		table.SetColumnSeparator("│")
		table.SetAutoWrapText(true)

		// Append rows with release details.
		for _, r := range combined {
			var typ, details string
			if r.Prerelease {
				if r.TagName == "nightly" {
					typ = "Nightly"
					shortCommit := ""
					if len(r.CommitHash) >= 10 {
						shortCommit = r.CommitHash[:10]
					}
					details = fmt.Sprintf("Published: %s, Commit: %s", utils.TimeFormat(r.PublishedAt), shortCommit)
				} else {
					table.Append([]string{r.TagName, "Nightly", ""})
				}
			} else {
				if r.TagName == "stable" {
					typ = "Stable"
					details = fmt.Sprintf("Stable version: %s", stableTag)
				} else {
					typ = "Exact"
				}
			}

			key := r.TagName
			localStatus := ""
			if utils.IsInstalled(versionsDir, key) {
				if key == current {
					localStatus = "Current"
				} else {
					localStatus = "Installed"
				}
			}

			table.Append([]string{r.TagName, typ, details, localStatus})
		}

		table.Render()
	},
}

func init() {
	rootCmd.AddCommand(listRemoteCmd)
}
