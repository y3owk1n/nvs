package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/releases"
	"github.com/y3owk1n/nvs/pkg/utils"
)

// listRemoteCmd represents the "list-remote" command (aliases: ls-remote).
// It fetches available remote Neovim releases from GitHub (using a cache that expires after 5 minutes)
// and displays them in a table along with their installation status.
// If a "force" argument is provided, the cache is bypassed.
//
// Example usage:
//
//	nvs list-remote
//	nvs list-remote force
var listRemoteCmd = &cobra.Command{
	Use:     "list-remote [force]",
	Aliases: []string{"ls-remote"},
	Short:   "List available remote versions with installation status (cached for 5 minutes or force)",
	Run: func(cmd *cobra.Command, args []string) {
		// Check if the user passed "force" to bypass the cache.
		force := len(args) > 0 && args[0] == "force"

		logrus.Debug("Fetching available versions...")
		fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText("Fetching available versions..."))

		// Retrieve the remote releases, using cache unless "force" is specified.
		releasesResult, err := releases.GetCachedReleases(force, cacheFilePath)
		if err != nil {
			logrus.Fatalf("Error fetching releases: %v", err)
		}
		logrus.Debugf("Fetched %d releases", len(releasesResult))

		// Determine the latest stable release (if available) for reference.
		stableRelease, err := releases.FindLatestStable(cacheFilePath)
		stableTag := "stable"
		if err == nil {
			stableTag = stableRelease.TagName
		}
		logrus.Debugf("Latest stable release: %s", stableTag)

		// Group releases into Nightly, Stable, and Others.
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

		// Combine all groups into one slice for display.
		combined := append(append(groupNightly, groupStable...), groupOthers...)

		// Determine the current installed version (if any).
		current, err := utils.GetCurrentVersion(versionsDir)
		if err != nil {
			current = ""
		}
		logrus.Debugf("Current version: %s", current)

		// Prepare a table for displaying the remote releases and their status.
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Tag", "Status", "Details"})
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
		)
		table.SetTablePadding("1")
		table.SetBorder(false)
		table.SetRowLine(false)
		table.SetCenterSeparator("")
		table.SetColumnSeparator("")
		table.SetAutoWrapText(false)

		// Iterate over the releases and build table rows with appropriate details and color-coding.
		for _, r := range combined {
			var details string

			// For nightly releases, display published date and commit hash.
			if r.Prerelease {
				if r.TagName == "nightly" {
					shortCommit := releases.GetReleaseIdentifier(r, "nightly")
					details = fmt.Sprintf("Published: %s, Commit: %s", utils.TimeFormat(r.PublishedAt), shortCommit)
				} else {
					// Skip non-nightly prerelease rows if no details are available.
					row := []string{r.TagName, "Nightly", ""}
					table.Append(row)
					continue
				}
			} else if r.TagName == "stable" {
				// For stable releases, reference the determined stableTag.
				details = fmt.Sprintf("Stable version: %s", stableTag)
			}

			key := r.TagName
			localStatus := ""
			upgradeIndicator := ""

			// Check if the release is installed locally.
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

				// If the installed version is different from the remote, indicate an upgrade is available.
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

			// Build the row for the table.
			row := []string{r.TagName, localStatus, details}

			// Colorize the row based on status.
			switch localStatus {
			case "Current" + upgradeIndicator:
				row = utils.ColorizeRow(row, color.New(color.FgGreen))
			case "Installed" + upgradeIndicator:
				row = utils.ColorizeRow(row, color.New(color.FgYellow))
			default:
				row = utils.ColorizeRow(row, color.New(color.FgWhite))
			}

			table.Append(row)
		}

		// Render the table to standard output.
		table.Render()
	},
}

// init registers the listRemoteCmd with the root command.
func init() {
	rootCmd.AddCommand(listRemoteCmd)
}
