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
				fmt.Printf("stable (%s)\n", current)
			} else {
				fmt.Printf("stable\n  Version: %s\n", stable.TagName)
			}
		case "nightly":
			nightly, err := releases.FindLatestNightly(cacheFilePath)
			if err != nil {
				fmt.Printf("nightly (%s)\n", current)
			} else {
				shortCommit := nightly.CommitHash
				if len(shortCommit) > 10 {
					shortCommit = shortCommit[:10]
				}
				publishedStr := nightly.PublishedAt
				if t, err := time.Parse(time.RFC3339, nightly.PublishedAt); err == nil {
					publishedStr = t.Format("2006-01-02")
				}
				fmt.Printf("nightly\n  Published: %s\n  Commit: %s\n", publishedStr, shortCommit)
			}
		default:
			fmt.Printf("%s\n", current)
		}
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
