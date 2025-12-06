package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/domain/release"
	"github.com/y3owk1n/nvs/internal/ui"
)

// listRemoteCmd represents the "list-remote" command (aliases: ls-remote).
// It fetches available remote Neovim releases from GitHub (using a cache that expires after 5 minutes)
// and displays them in a table along with their installation status.
// If --force is provided, the cache is bypassed.
//
// Example usage:
//
//	nvs list-remote
//	nvs list-remote --force
var listRemoteCmd = &cobra.Command{
	Use:     "list-remote",
	Aliases: []string{"ls-remote"},
	Short:   "List available remote versions with installation status (cached for 5 minutes or --force to bypass)",
	Args:    cobra.NoArgs,
	RunE:    RunListRemote,
}

// RunListRemote executes the list-remote command.
func RunListRemote(cmd *cobra.Command, _ []string) error {
	// Check if the user passed --force to bypass the cache.
	force, _ := cmd.Flags().GetBool("force")

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
	releasesResult, err := GetVersionService().ListRemote(cmd.Context(), force)
	if err != nil {
		return fmt.Errorf("error fetching releases: %w", err)
	}

	logrus.Debugf("Fetched %d releases", len(releasesResult))

	// Group releases into nightly, stable, and Others.
	var groupNightly, groupStable, groupOthers []release.Release
	for _, release := range releasesResult {
		switch {
		case release.Prerelease() && strings.HasPrefix(strings.ToLower(release.TagName()), "nightly"):
			groupNightly = append(groupNightly, release)
		case release.TagName() == constants.Stable:
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
	if err != nil {
		logrus.Debugf("No current version set: %v", err)
	} else {
		currentName = current.Name()
	}

	logrus.Debugf("Current version: %s", currentName)

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		logrus.Warnf("Failed to read json flag: %v", err)
	}

	type ReleaseInfo struct {
		Tag        string `json:"tag"`
		Status     string `json:"status"`
		Details    string `json:"details"`
		Prerelease bool   `json:"prerelease"`
	}

	var (
		infos []ReleaseInfo
		table *tablewriter.Table
	)

	if !jsonOutput {
		// Prepare a table for displaying the remote releases and their status.
		table = tablewriter.NewWriter(os.Stdout)
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
	}

	// Iterate over the releases and build table rows with appropriate details and color-coding.
	svc := GetVersionService()
	for _, release := range combined {
		var details string

		// For nightly releases, display published date and commit hash.
		if release.Prerelease() {
			if strings.HasPrefix(strings.ToLower(release.TagName()), "nightly") {
				shortCommit := release.CommitHash()
				if len(shortCommit) > constants.ShortCommitLen {
					shortCommit = shortCommit[:constants.ShortCommitLen]
				}

				details = fmt.Sprintf(
					"Published: %s, Commit: %s",
					ui.TimeFormat(release.PublishedAt().Format(time.RFC3339)),
					shortCommit,
				)
			}
		} else if release.TagName() == "stable" {
			// For stable releases, show the actual version tag.
			stableRelease, stableErr := svc.FindStable(cmd.Context())
			if stableErr == nil {
				details = "stable version: " + stableRelease.TagName()
			} else {
				details = "stable version: " + constants.Stable
			}
		}

		key := release.TagName()

		var baseStatus string

		upgradeIndicator := ""

		// Check if the release is installed locally.
		if svc.IsVersionInstalled(key) {
			// Check for upgrade availability
			installedIdentifier, err := svc.GetInstalledVersionIdentifier(key)
			if err != nil {
				installedIdentifier = ""
			}

			// Use the same logic as the Upgrade function:
			// For nightly, compare commit hash. For stable, fetch the actual stable release's tag.
			var remoteIdentifier string
			switch {
			case release.Prerelease() &&
				strings.HasPrefix(strings.ToLower(release.TagName()), "nightly"):
				remoteIdentifier = release.CommitHash()
			case release.TagName() == constants.Stable:
				// For the "stable" tag, fetch the actual stable release to get the real version tag
				stableRelease, stableErr := svc.FindStable(cmd.Context())
				if stableErr == nil {
					remoteIdentifier = stableRelease.TagName()
				}
			default:
				remoteIdentifier = release.TagName()
			}

			// If the installed version is different from the remote, indicate an upgrade is available.
			if installedIdentifier != "" && remoteIdentifier != "" &&
				installedIdentifier != remoteIdentifier {
				upgradeIndicator = " (" + constants.Upgrade + ")"
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

		if jsonOutput {
			infos = append(infos, ReleaseInfo{
				Tag:        tag,
				Status:     localStatus,
				Details:    details,
				Prerelease: release.Prerelease(),
			})
		} else {
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
	}

	if jsonOutput {
		data := map[string]any{"releases": infos}

		return outputJSON(data)
	} else {
		// Render the table to standard output.
		table.Render()
	}

	return nil
}

// init registers the listRemoteCmd with the root command.
func init() {
	listRemoteCmd.Flags().Bool("force", false, "Bypass cache and fetch latest releases")
	listRemoteCmd.Flags().Bool("json", false, "Output in JSON format")
	rootCmd.AddCommand(listRemoteCmd)
}
