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

var listCmd = &cobra.Command{
	Use:     "list [force]",
	Aliases: []string{"ls"},
	Short:   "List available remote versions with installation status (cached for 5 minutes or force)",
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
		table.SetHeader([]string{"Tag", "Status", "Details"})
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
		)
		table.SetBorder(true)
		table.SetRowLine(true)
		table.SetCenterSeparator("│")
		table.SetColumnSeparator("│")
		table.SetAutoWrapText(false)

		// ANSI color codes for status
		green := "\033[32m"
		yellow := "\033[33m"
		reset := "\033[0m"

		// Append rows with release details.
		for _, r := range combined {
			var details string
			// Check for prereleases (nightly)
			if r.Prerelease {
				if r.TagName == "nightly" {
					shortCommit := releases.GetReleaseIdentifier(r, "nightly")
					details = fmt.Sprintf("Published: %s, Commit: %s", utils.TimeFormat(r.PublishedAt), shortCommit)
				} else {
					// For other prereleases, add a simple row.
					row := []string{r.TagName, "Nightly", ""}
					table.Append(row)
					continue
				}
			} else {
				// Stable release.
				if r.TagName == "stable" {
					details = fmt.Sprintf("Stable version: %s", stableTag)
				}
			}

			key := r.TagName
			localStatus := ""
			upgradeIndicator := ""
			upgradeIcon := "(↑)"

			// Determine if version is installed.
			if utils.IsInstalled(versionsDir, key) {
				release, err := releases.ResolveVersion(key, cacheFilePath)
				if err != nil {
					logrus.Errorf("Error resolving %s: %v", key, err)
					continue
				}

				installedIdentifier, err := utils.GetInstalledReleaseIdentifier(versionsDir, key)
				if err != nil {
					installedIdentifier = ""
				}
				// Compare installed identifier with remote.
				remoteIdentifier := releases.GetReleaseIdentifier(release, key)
				if installedIdentifier != "" && installedIdentifier != remoteIdentifier {
					upgradeIndicator = " " + upgradeIcon
				}
				if key == current {
					localStatus = "Current" + upgradeIndicator
				} else {
					localStatus = "Installed" + upgradeIndicator
				}
			} else {
				localStatus = "Not Installed"
			}

			row := []string{r.TagName, localStatus, details}

			// Colorize the entire row if installed.
			switch localStatus {
			case "Current" + upgradeIndicator:
				row = utils.ColorizeRow(row, green, reset)
			case "Installed" + upgradeIndicator:
				row = utils.ColorizeRow(row, yellow, reset)
			}

			table.Append(row)
		}

		table.Render()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
