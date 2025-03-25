package cmd

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/releases"
	"github.com/y3owk1n/nvs/pkg/utils"
)

// currentCmd represents the "current" command.
// It displays details of the current active version.
// Depending on whether the active version is "stable", "nightly", or a custom version/commit hash,
// it fetches and displays additional details.
//
// Example usage:
//
//	nvs current
//
// This will output the active version information along with additional details like the latest stable
// tag or nightly commit and published date.
var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current active version with details",
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Debug("Fetching current active version")
		current, err := utils.GetCurrentVersion(versionsDir)
		if err != nil {
			logrus.Fatalf("Error getting current version: %v", err)
		}

		logrus.Debugf("Current version detected: %s", current)

		// Handle "stable" active version
		switch current {
		case "stable":
			logrus.Debug("Fetching latest stable release")
			stable, err := releases.FindLatestStable(cacheFilePath)
			if err != nil {
				logrus.Warnf("Error fetching latest stable release: %v", err)
				fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("stable (%s)", current)))
			} else {
				fmt.Printf("%s %s\n", utils.InfoIcon(), utils.CyanText("stable"))
				fmt.Printf("  %s\n", utils.WhiteText(fmt.Sprintf("Version: %s", stable.TagName)))
				logrus.Debugf("Latest stable version: %s", stable.TagName)
			}

		// Handle "nightly" active version
		case "nightly":
			logrus.Debug("Fetching latest nightly release")
			nightly, err := releases.FindLatestNightly(cacheFilePath)
			if err != nil {
				logrus.Warnf("Error fetching latest nightly release: %v", err)
				fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("nightly (%s)", current)))
			} else {
				shortCommit := nightly.CommitHash
				if len(shortCommit) > 7 {
					shortCommit = shortCommit[:7]
				}
				publishedStr := nightly.PublishedAt
				if t, err := time.Parse(time.RFC3339, nightly.PublishedAt); err == nil {
					publishedStr = t.Format("2006-01-02")
				}

				fmt.Printf("%s %s\n", utils.InfoIcon(), utils.CyanText("nightly"))
				fmt.Printf("  %s\n", utils.WhiteText(fmt.Sprintf("Published: %s", publishedStr)))
				fmt.Printf("  %s\n", utils.WhiteText(fmt.Sprintf("Commit: %s", shortCommit)))
				logrus.Debugf("Latest nightly commit: %s, Published: %s", shortCommit, publishedStr)
			}

		// Handle custom version or commit hash
		default:
			isCommitHash := releases.IsCommitHash(current)
			logrus.Debugf("isCommitHash: %t", isCommitHash)

			if isCommitHash {
				logrus.Debugf("Displaying custom commit hash: %s", current)
				fmt.Printf("%s %s %s\n", utils.InfoIcon(), utils.WhiteText("Commit Hash:"), utils.CyanText(current))
			} else {
				logrus.Debugf("Displaying custom version: %s", current)
				fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(current))
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
