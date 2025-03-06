package cmd

import (
	"fmt"

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

		for _, r := range releasesResult {
			// For prereleases (nightly builds)
			if r.Prerelease {
				if r.TagName == "nightly" {
					shortCommit := r.CommitHash[:10]
					fmt.Printf("%-10s (nightly: published on %s, commit: %s)\n", r.TagName, r.PublishedAt, shortCommit)
				} else {
					// Fallback for any other prerelease tag format.
					fmt.Println(r.TagName)
				}
			} else {
				// For stable releases: annotate only if the tag is exactly "stable"
				if r.TagName == "stable" {
					fmt.Printf("%-10s (stable version: %s)\n", r.TagName, stableTag)
				} else {
					// For specific version releases, just print the tag name.
					fmt.Println(r.TagName)
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(listRemoteCmd)
}
