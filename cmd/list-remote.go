package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/helpers"
	"github.com/y3owk1n/nvs/pkg/releases"
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
	RunE:    RunListRemote,
}

// RunListRemote executes the list-remote command.
func RunListRemote(cmd *cobra.Command, args []string) error {
	// Check if the user passed "force" to bypass the cache.
	force := len(args) > 0 && args[0] == "force"

	logrus.Debug("Fetching available versions...")

	var err error

	_, err = fmt.Fprintf(
		os.Stdout,
		"%s %s\n",
		helpers.InfoIcon(),
		helpers.WhiteText("Fetching available versions..."),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	// Retrieve the remote releases, using cache unless "force" is specified.
	releasesResult, err := releases.GetCachedReleases(force, CacheFilePath)
	if err != nil {
		return fmt.Errorf("error fetching releases: %w", err)
	}

	logrus.Debugf("Fetched %d releases", len(releasesResult))

	// Determine the latest stable release (if available) for reference.
	stableRelease, err := releases.FindLatestStable(CacheFilePath)

	stableTag := stable
	if err == nil {
		stableTag = stableRelease.TagName
	}

	logrus.Debugf("Latest stable release: %s", stableTag)

	// Group releases into nightly, stable, and Others.
	var groupNightly, groupStable, groupOthers []releases.Release
	for _, release := range releasesResult {
		switch {
		case release.Prerelease:
			groupNightly = append(groupNightly, release)
		case release.TagName == "stable":
			groupStable = append(groupStable, release)
		default:
			groupOthers = append(groupOthers, release)
		}
	}

	logrus.Debugf(
		"nightly: %d, stable: %d, Others: %d",
		len(groupNightly),
		len(groupStable),
		len(groupOthers),
	)

	// Combine all groups into one slice for display.
	combined := append(append(groupNightly, groupStable...), groupOthers...)

	// Determine the current installed version (if any).
	current, err := helpers.GetCurrentVersion(VersionsDir)
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
	for _, release := range combined {
		var details string

		// For nightly releases, display published date and commit hash.
		if release.Prerelease {
			if release.TagName == "nightly" {
				shortCommit := releases.GetReleaseIdentifier(release, "nightly")
				details = fmt.Sprintf(
					"Published: %s, Commit: %s",
					helpers.TimeFormat(release.PublishedAt),
					shortCommit,
				)
			} else {
				// Skip non-nightly prerelease rows if no details are available.
				row := []string{release.TagName, "nightly", ""}
				table.Append(row)

				continue
			}
		} else if release.TagName == "stable" {
			// For stable releases, reference the determined stableTag.
			details = "stable version: " + stableTag
		}

		key := release.TagName

		var localStatus string

		upgradeIndicator := ""

		// Check if the release is installed locally.
		if helpers.IsInstalled(VersionsDir, key) {
			release, err := releases.ResolveVersion(key, CacheFilePath)
			if err != nil {
				logrus.Errorf("Error resolving %s: %v", key, err)

				continue
			}

			installedIdentifier, err := helpers.GetInstalledReleaseIdentifier(VersionsDir, key)
			if err != nil {
				installedIdentifier = ""
			}

			remoteIdentifier := releases.GetReleaseIdentifier(release, key)

			// If the installed version is different from the remote, indicate an upgrade is available.
			if installedIdentifier != "" && installedIdentifier != remoteIdentifier {
				upgradeIndicator = " (" + helpers.Upgrade + ")"
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
		tag := release.TagName
		if tag == "" {
			tag = "(no tag)"
		}

		row := []string{tag, localStatus, details}

		// Colorize the row based on status.
		switch localStatus {
		case "Current" + upgradeIndicator:
			row = helpers.ColorizeRow(row, color.New(color.FgGreen))
		case "Installed" + upgradeIndicator:
			row = helpers.ColorizeRow(row, color.New(color.FgYellow))
		default:
			row = helpers.ColorizeRow(row, color.New(color.FgWhite))
		}

		table.Append(row)
	}

	// Render the table to standard output.
	table.Render()

	return nil
}

// init registers the listRemoteCmd with the root command.
func init() {
	rootCmd.AddCommand(listRemoteCmd)
}
