package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/pkg/helpers"
	"github.com/y3owk1n/nvs/pkg/releases"
)

// ShortCommitLen is the number of characters to shorten commit hashes to.
const ShortCommitLen = 7

const stableConst = "stable"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunCurrent(cmd, args, VersionsDir, CacheFilePath)
	},
}

// RunCurrent executes the current command.
func RunCurrent(_ *cobra.Command, _ []string, versionsDir, cacheFilePath string) error {
	logrus.Debug("Fetching current active version")

	current, err := helpers.GetCurrentVersion(versionsDir)
	if err != nil {
		return fmt.Errorf("error getting current version: %w", err)
	}

	logrus.Debugf("Current version detected: %s", current)

	// Handle "stable" active version
	switch current {
	case stableConst:
		logrus.Debug("Fetching latest stable release")

		stable, err := releases.FindLatestStable(cacheFilePath)
		if err != nil {
			logrus.Warnf("Error fetching latest stable release: %v", err)

			_, err = fmt.Fprintf(os.Stdout,
				"%s %s\n",
				helpers.InfoIcon(),
				helpers.WhiteText(fmt.Sprintf("stable (%s)", current)),
			)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}
		} else {
			_, err = fmt.Fprintf(os.Stdout, "%s %s\n", helpers.InfoIcon(), helpers.CyanText(stableConst))
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			_, err = fmt.Fprintf(os.Stdout, "  %s\n", helpers.WhiteText("Version: "+stable.TagName))
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			logrus.Debugf("Latest stable version: %s", stable.TagName)
		}

	// Handle "nightly" active version
	case "nightly":
		logrus.Debug("Fetching latest nightly release")

		nightly, err := releases.FindLatestNightly(cacheFilePath)
		if err != nil {
			logrus.Warnf("Error fetching latest nightly release: %v", err)

			_, err = fmt.Fprintf(os.Stdout,
				"%s %s\n",
				helpers.InfoIcon(),
				helpers.WhiteText(fmt.Sprintf("nightly (%s)", current)),
			)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}
		} else {
			shortCommit := nightly.CommitHash
			if len(shortCommit) > ShortCommitLen {
				shortCommit = shortCommit[:ShortCommitLen]
			}

			publishedStr := nightly.PublishedAt

			t, parseErr := time.Parse(time.RFC3339, nightly.PublishedAt)
			if parseErr == nil {
				publishedStr = t.Format("2006-01-02")
			}

			_, err = fmt.Fprintf(os.Stdout, "%s %s\n", helpers.InfoIcon(), helpers.CyanText("nightly"))
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			_, err = fmt.Fprintf(os.Stdout, "  %s\n", helpers.WhiteText("Published: "+publishedStr))
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			_, err = fmt.Fprintf(os.Stdout, "  %s\n", helpers.WhiteText("Commit: "+shortCommit))
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			logrus.Debugf("Latest nightly commit: %s, Published: %s", shortCommit, publishedStr)
		}

	// Handle custom version or commit hash
	default:
		isCommitHash := releases.IsCommitHash(current)
		logrus.Debugf("isCommitHash: %t", isCommitHash)

		if isCommitHash {
			logrus.Debugf("Displaying custom commit hash: %s", current)

			_, err = fmt.Fprintf(os.Stdout,
				"%s %s\n",
				helpers.InfoIcon(),
				helpers.WhiteText(fmt.Sprintf("stable (%s)", current)),
			)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}
		} else {
			logrus.Debugf("Displaying custom version: %s", current)

			_, err = fmt.Fprintf(os.Stdout, "%s %s\n", helpers.InfoIcon(), helpers.WhiteText(current))
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
