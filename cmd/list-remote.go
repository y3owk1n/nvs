package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/domain/release"
	"github.com/y3owk1n/nvs/internal/ui"
	"github.com/y3owk1n/nvs/internal/ui/table"
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

	ui.Message.Infof("Fetching available versions...")

	// Retrieve the remote releases via version service
	releasesResult, err := GetVersionService().ListRemote(cmd.Context(), force)
	if err != nil {
		return fmt.Errorf("error fetching releases: %w", err)
	}

	logrus.Debugf("Fetched %d releases", len(releasesResult))

	// Group releases into nightly, stable, and Others.
	var groupNightly, groupStable, groupOthers []release.Release
	for _, rel := range releasesResult {
		switch {
		case rel.Prerelease() && strings.HasPrefix(strings.ToLower(rel.TagName()), "nightly"):
			groupNightly = append(groupNightly, rel)
		case rel.TagName() == constants.Stable:
			groupStable = append(groupStable, rel)
		default:
			groupOthers = append(groupOthers, rel)
		}
	}

	logrus.Debugf(
		"nightly: %d, stable: %d, Others: %d",
		len(groupNightly),
		len(groupStable),
		len(groupOthers),
	)

	// Combine all groups into one slice for display.
	combined := slices.Concat(groupNightly, groupStable, groupOthers)

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

	// Get the version service once. The rest of this function uses
	// it for per-release lookups (IsVersionInstalled,
	// GetInstalledVersionIdentifier).
	svc := GetVersionService()

	// Build a set of installed version names AND a map of
	// installed name -> identifier in a single pass. The previous
	// code:
	//   - called svc.IsVersionInstalled(key) per release (os.Stat
	//     per release);
	//   - called svc.GetInstalledVersionIdentifier(key) per
	//     INSTALLED release (os.ReadFile per installed release).
	// Both were N+1 patterns. The new code issues a single
	// os.ReadDir (in InstalledVersionIdentifiers -> List) and one
	// os.ReadFile per installed version — the minimum work to
	// answer "is this installed" and "what's its identifier".
	installedSet := make(map[string]struct{})
	installedIdentifiers := make(map[string]string)
	{
		details, listErr := svc.InstalledVersionIdentifiers()
		if listErr != nil {
			logrus.Debugf("failed to enumerate installed versions: %v", listErr)
		} else {
			for name, identifier := range details {
				installedSet[name] = struct{}{}
				installedIdentifiers[name] = identifier
			}
		}
	}

	// Resolve the "stable" pseudo-release once before the loop. Each
	// call below used to invoke FindStable independently, which
	// re-decodes the GitHub releases cache (or, on a cache miss, hits
	// the network) for every release in the list.
	var stableReleaseTag string

	stableRelease, stableErr := svc.FindStable(cmd.Context())
	if stableErr == nil {
		stableReleaseTag = stableRelease.TagName()
	}

	// releaseInfo is the row schema for --json. The struct is kept
	// local to RunListRemote (where it is built) and the table path
	// below inlines its fields into the table cell, since
	// ui.Table consumes plain strings, not structs.
	type releaseInfo struct {
		Tag        string `json:"tag"`
		Status     string `json:"status"`
		Details    string `json:"details"`
		Prerelease bool   `json:"prerelease"`
	}

	var (
		infos []releaseInfo
		tbl   *table.Table
	)

	if !jsonOutput {
		tbl = ui.Table.New("Tag", "Status", "Details")
	}

	// Iterate over the releases and build table rows with appropriate details and color-coding.
	for _, rel := range combined {
		details := buildReleaseDetails(rel, stableReleaseTag)

		key := rel.TagName()

		baseStatus, upgradeIndicator := classifyReleaseStatus(
			rel, key, currentName, installedSet, installedIdentifiers, stableReleaseTag,
		)

		localStatus := baseStatus + upgradeIndicator

		logrus.Debugf("Version: %s, Status: %s", key, localStatus)

		tag := rel.TagName()
		if tag == "" {
			tag = "(no tag)"
		}

		if jsonOutput {
			infos = append(infos, releaseInfo{
				Tag:        tag,
				Status:     localStatus,
				Details:    details,
				Prerelease: rel.Prerelease(),
			})
		} else {
			tbl.Row(styleReleaseRow(tag, baseStatus, localStatus, details)...)
		}
	}

	if jsonOutput {
		return outputJSON(map[string]any{"releases": infos})
	}

	_, _ = fmt.Fprint(os.Stdout, ui.Banner.Logo())
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprint(os.Stdout, tbl.Render(ui.Style.Palette()))

	return nil
}

// buildReleaseDetails produces the "Details" cell string for a
// given release. For nightlies it surfaces the publish date and
// a short commit hash; for the "stable" alias it surfaces the
// real stable tag. All other releases get an empty details
// string.
func buildReleaseDetails(rel release.Release, stableReleaseTag string) string {
	switch {
	case rel.Prerelease() && strings.HasPrefix(strings.ToLower(rel.TagName()), "nightly"):
		shortCommit := rel.CommitHash()
		if len(shortCommit) > constants.ShortCommitLen {
			shortCommit = shortCommit[:constants.ShortCommitLen]
		}

		return fmt.Sprintf(
			"Published: %s, Commit: %s",
			ui.TimeFormat(rel.PublishedAt().Format(time.RFC3339)),
			shortCommit,
		)
	case rel.TagName() == "stable":
		if stableReleaseTag != "" {
			return "stable version: " + stableReleaseTag
		}

		return "stable version: " + constants.Stable
	default:
		return ""
	}
}

// classifyReleaseStatus returns the human-readable base status
// for a release (Current / Installed / Not Installed) and, if
// an upgrade is available, a parenthesized " (upgrade)" suffix
// appended to the base status. The combination is the
// user-facing "Status" cell.
//
// The pre-loop arguments (currentName, installedSet,
// installedIdentifiers, stableReleaseTag) are hoisted so we
// don't re-fetch them per iteration.
func classifyReleaseStatus(
	rel release.Release,
	key string,
	currentName string,
	installedSet map[string]struct{},
	installedIdentifiers map[string]string,
	stableReleaseTag string,
) (string, string) {
	if _, isInstalled := installedSet[key]; !isInstalled {
		return "Not Installed", ""
	}

	// Resolve the "remote identifier" we compare against the
	// installed one. Nightlies compare commit hashes; the
	// "stable" alias compares against the real stable tag;
	// everything else uses the release's own tag.
	var remoteIdentifier string
	switch {
	case rel.Prerelease() && strings.HasPrefix(strings.ToLower(rel.TagName()), "nightly"):
		remoteIdentifier = rel.CommitHash()
	case rel.TagName() == constants.Stable:
		remoteIdentifier = stableReleaseTag
	default:
		remoteIdentifier = rel.TagName()
	}

	upgradeIndicator := ""

	installedIdentifier := installedIdentifiers[key]
	if installedIdentifier != "" && remoteIdentifier != "" &&
		installedIdentifier != remoteIdentifier {
		upgradeIndicator = " (" + constants.Upgrade + ")"
	}

	if key == currentName {
		return "Current", upgradeIndicator
	}

	return "Installed", upgradeIndicator
}

// styleReleaseRow applies the per-cell lipgloss styling for a
// non-JSON list-remote row. The status determines the color of
// the status cell and the tag cell (a "Current" row gets the
// highlight color; an "Installed" row with an upgrade
// available gets the warn color; everything else is muted).
// The details cell is always muted to keep noise low.
func styleReleaseRow(tag, baseStatus, fullStatus, details string) []string {
	styledTag, styledStatus := styledReleaseCells(tag, baseStatus, fullStatus)

	styledDetails := ui.Message.Muted(details)

	return []string{styledTag, styledStatus, styledDetails}
}

// styledReleaseCells returns the styled (tag, status) cells
// for a list-remote row, picking colors from the palette
// based on the release's status:
//
//	Current       → Highlight (primary + bold)
//	Installed     → Success (green) if no upgrade, Warn (yellow) if an upgrade is available
//	Not Installed → Muted (subtle)
func styledReleaseCells(tag, baseStatus, fullStatus string) (string, string) {
	switch baseStatus {
	case "Current":
		return ui.Message.Highlight(tag), ui.Message.Highlight(fullStatus)
	case "Installed":
		if strings.Contains(fullStatus, "("+constants.Upgrade+")") {
			return ui.Message.Warn(tag), ui.Message.Warn(fullStatus)
		}

		return ui.Message.Success(tag), ui.Message.Success(fullStatus)
	default:
		return ui.Message.Muted(tag), ui.Message.Muted(fullStatus)
	}
}

// init registers the listRemoteCmd with the root command.
func init() {
	listRemoteCmd.Flags().Bool("force", false, "Bypass cache and fetch latest releases")
	listRemoteCmd.Flags().Bool("json", false, "Output in JSON format")
	rootCmd.AddCommand(listRemoteCmd)
}
