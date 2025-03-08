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
		current, err := utils.GetCurrentVersion(versionsDir)
		if err != nil {
			logrus.Fatalf("Error getting current version: %v", err)
		}

		switch current {
		case "stable":
			stable, err := releases.FindLatestStable(cacheFilePath)
			if err != nil {
				fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(fmt.Sprintf("stable (%s)", current)))
			} else {
				fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText("stable"))
				fmt.Printf("  %s\n", utils.WhiteText(fmt.Sprintf("Version: %s", stable.TagName)))
			}
		case "nightly":
			nightly, err := releases.FindLatestNightly(cacheFilePath)
			if err != nil {
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
			}
		default:
			fmt.Printf("%s %s\n", utils.InfoIcon(), utils.WhiteText(current))
		}
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
