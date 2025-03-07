package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/releases"
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

		// Use tabwriter to format table output.
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		// Print header.
		fmt.Fprintln(w, "TAG\tTYPE\tDETAILS")
		fmt.Fprintln(w, "----\t----\t-------")

		for _, r := range releasesResult {
			// For prereleases (nightly builds)
			if r.Prerelease {
				if r.TagName == "nightly" {
					shortCommit := ""
					if len(r.CommitHash) >= 10 {
						shortCommit = r.CommitHash[:10]
					}
					details := fmt.Sprintf("Published: %s, Commit: %s", timeFormat(r.PublishedAt), shortCommit)
					fmt.Fprintf(w, "%s\tNightly\t%s\n", r.TagName, details)
				} else {
					// Fallback for any other prerelease tag format.
					fmt.Fprintf(w, "%s\tNightly\t\n", r.TagName)
				}
			} else {
				details := ""
				// For stable releases: annotate only if the tag is exactly "stable"
				if r.TagName == "stable" {
					details = fmt.Sprintf("Stable version: %s", stableTag)
				}
				// For specific version releases, just print the tag name.
				fmt.Fprintf(w, "%s\tStable\t%s\n", r.TagName, details)
			}
		}

		w.Flush()
	},
}

// timeFormat is a helper to format the published date in a more user-friendly way.
func timeFormat(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	return t.Format("2006-01-02")
}

func init() {
	rootCmd.AddCommand(listRemoteCmd)
}
