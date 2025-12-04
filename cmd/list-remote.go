package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/domain/release"
	"github.com/y3owk1n/nvs/internal/ui"
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

	_, err := fmt.Fprintf(
		os.Stdout,
		"%s %s\n",
		ui.InfoIcon(),
		ui.WhiteText("Fetching available versions..."),
	)
	if err != nil {
		logrus.Warnf("Failed to write to stdout: %v", err)
	}

	// Retrieve the remote releases via version service
	releasesResult, err := GetVersionService().ListRemote(force)
	if err != nil {
		return fmt.Errorf("error fetching releases: %w", err)
	}

	logrus.Debugf("Fetched %d releases", len(releasesResult))

	// Use default stable tag
	stableTag := stableConst

	logrus.Debugf("Using stable tag: %s", stableTag)

	// Group releases into nightly, stable, and Others.
	var groupNightly, groupStable, groupOthers []release.Release
	for _, release := range releasesResult {
		switch {
		case release.Prerelease():
			groupNightly = append(groupNightly, release)
		case release.TagName() == stableConst:
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
	current, err := GetVersionService().Current()

	currentName := ""
	if err == nil {
		currentName = current.Name()
	}

	logrus.Debugf("Current version: %s", currentName)

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
		if release.Prerelease() {
			if release.TagName() == "nightly" {
				shortCommit := release.CommitHash()
				if len(shortCommit) > ShortCommitLen {
					shortCommit = shortCommit[:ShortCommitLen]
				}

				details = fmt.Sprintf(
					"Published: %s, Commit: %s",
					ui.TimeFormat(release.PublishedAt().Format(time.RFC3339)),
					shortCommit,
				)
			} else {
				// Skip non-nightly prerelease rows if no details are available.
				row := []string{release.TagName(), "nightly", ""}
				table.Append(row)

				continue
			}
		} else if release.TagName() == "stable" {
			// For stable releases, reference the determined stableTag.
			details = "stable version: " + stableTag
		}

		key := release.TagName()

		var baseStatus string

		upgradeIndicator := ""

		// Check if the release is installed locally.
		if GetVersionService().IsVersionInstalled(key) {
			// Check for upgrade availability
			// Note: This logic is simplified; ideally use service to check for updates
			installedIdentifier, err := GetVersionService().GetInstalledVersionIdentifier(key)
			if err != nil {
				installedIdentifier = ""
			}

			remoteIdentifier := release.CommitHash()
			if remoteIdentifier == "" {
				remoteIdentifier = release.TagName()
			}

			// If the installed version is different from the remote, indicate an upgrade is available.
			if installedIdentifier != "" && installedIdentifier != remoteIdentifier {
				upgradeIndicator = " (" + ui.Upgrade + ")"
			}

			if key == currentName {
				baseStatus = "Current"
			} else {
				baseStatus = "Installed"
			}
		} else {
			baseStatus = "Not Installed"
		}

		localStatus := baseStatus + upgradeIndicator

		logrus.Debugf("Version: %s, Status: %s", key, localStatus)

		// Build the row for the table.
		tag := release.TagName()
		if tag == "" {
			tag = "(no tag)"
		}

		row := []string{tag, localStatus, details}

		// Colorize the row based on status.
		switch baseStatus {
		case "Current":
			row = ui.ColorizeRow(row, color.New(color.FgGreen))
		case "Installed":
			row = ui.ColorizeRow(row, color.New(color.FgYellow))
		default:
			row = ui.ColorizeRow(row, color.New(color.FgWhite))
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
