package cmd

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/releases"
	"github.com/y3owk1n/nvs/pkg/utils"
)

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

		switch current {
		case "stable":
			logrus.Debug("Fetching latest stable release")
			stable, err := releases.FindLatestStable(cacheFilePath)
			if err != nil {
				logrus.Warnf("Error fetching latest stable release: %v", err)
				fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("stable (%s)", current)))
			} else {
				fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText("stable"))
				fmt.Printf("  %s\n", utils.WhiteText(fmt.Sprintf("Version: %s", stable.TagName)))
				logrus.Debugf("Latest stable version: %s", stable.TagName)
			}
		case "nightly":
			logrus.Debug("Fetching latest nightly release")
			nightly, err := releases.FindLatestNightly(cacheFilePath)
			if err != nil {
				logrus.Warnf("Error fetching latest nightly release: %v", err)
				fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("nightly (%s)", current)))
			} else {
				shortCommit := nightly.CommitHash
				if len(shortCommit) > 10 {
					shortCommit = shortCommit[:10]
				}
				publishedStr := nightly.PublishedAt
				if t, err := time.Parse(time.RFC3339, nightly.PublishedAt); err == nil {
					publishedStr = t.Format("2006-01-02")
				}

				fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText("nightly"))
				fmt.Printf("  %s\n", utils.WhiteText(fmt.Sprintf("Published: %s", publishedStr)))
				fmt.Printf("  %s\n", utils.WhiteText(fmt.Sprintf("Commit: %s", shortCommit)))
				logrus.Debugf("Latest nightly commit: %s, Published: %s", shortCommit, publishedStr)
			}
		default:
			logrus.Debugf("Displaying custom version: %s", current)
			fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(current))
		}
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
