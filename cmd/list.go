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

		logrus.Debug("Fetching available versions...")
		fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText("Fetching available versions..."))

		releasesResult, err := releases.GetCachedReleases(force, cacheFilePath)
		if err != nil {
			logrus.Fatalf("Error fetching releases: %v", err)
		}
		logrus.Debugf("Fetched %d releases", len(releasesResult))

		stableRelease, err := releases.FindLatestStable(cacheFilePath)
		stableTag := "stable"
		if err == nil {
			stableTag = stableRelease.TagName
		}
		logrus.Debugf("Latest stable release: %s", stableTag)

		var groupNightly, groupStable, groupOthers []releases.Release

		for _, r := range releasesResult {
			if r.Prerelease {
				groupNightly = append(groupNightly, r)
			} else if r.TagName == "stable" {
				groupStable = append(groupStable, r)
			} else {
				groupOthers = append(groupOthers, r)
			}
		}

		logrus.Debugf("Nightly: %d, Stable: %d, Others: %d", len(groupNightly), len(groupStable), len(groupOthers))

		combined := append(append(groupNightly, groupStable...), groupOthers...)

		current, err := utils.GetCurrentVersion(versionsDir)
		if err != nil {
			current = ""
		}
		logrus.Debugf("Current version: %s", current)

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

		green := "\033[32m"
		yellow := "\033[33m"
		reset := "\033[0m"

		for _, r := range combined {
			var details string

			if r.Prerelease {
				if r.TagName == "nightly" {
					shortCommit := releases.GetReleaseIdentifier(r, "nightly")
					details = fmt.Sprintf("Published: %s, Commit: %s", utils.TimeFormat(r.PublishedAt), shortCommit)
				} else {
					row := []string{r.TagName, "Nightly", ""}
					table.Append(row)
					continue
				}
			} else if r.TagName == "stable" {
				details = fmt.Sprintf("Stable version: %s", stableTag)
			}

			key := r.TagName
			localStatus := ""
			upgradeIndicator := ""

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
				remoteIdentifier := releases.GetReleaseIdentifier(release, key)

				if installedIdentifier != "" && installedIdentifier != remoteIdentifier {
					upgradeIndicator = " (" + utils.Upgrade + ")"
				}
				if key == current {
					localStatus = "Current" + upgradeIndicator
				} else {
					localStatus = "Installed" + upgradeIndicator
				}
			} else {
				localStatus = "Not Installed"
			}

			logrus.Debugf("Version: %s, Status: %s", key, localStatus)

			row := []string{r.TagName, localStatus, details}

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
