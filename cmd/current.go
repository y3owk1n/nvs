package cmd

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/ui"
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
	RunE:  RunCurrent,
}

// RunCurrent executes the current command.
func RunCurrent(_ *cobra.Command, _ []string) error {
	logrus.Debug("Executing current command")

	current, err := GetVersionService().Current()
	if err != nil {
		return fmt.Errorf("error getting current version: %w", err)
	}

	logrus.Debugf("Current version detected: %s", current.Name())

	// Handle active version
	switch current.Name() {
	case stableConst:
		logrus.Debug("Fetching latest stable release")

		stable, err := GetVersionService().FindStable()
		if err != nil {
			logrus.Warnf("Error fetching latest stable release: %v", err)

			_, err = fmt.Fprintf(os.Stdout,
				"%s %s\n",
				ui.InfoIcon(),
				ui.WhiteText("stable (version details unavailable)"),
			)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}
		} else {
			_, err = fmt.Fprintf(os.Stdout, "%s %s\n", ui.InfoIcon(), ui.CyanText(stableConst))
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			_, err = fmt.Fprintf(os.Stdout, "  %s\n", ui.WhiteText("Version: "+stable.TagName()))
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			logrus.Debugf("Latest stable version: %s", stable.TagName())
		}
	case "nightly":
		logrus.Debug("Fetching latest nightly release")

		nightly, err := GetVersionService().FindNightly()
		if err != nil {
			logrus.Warnf("Error fetching latest nightly release: %v", err)

			_, err = fmt.Fprintf(os.Stdout,
				"%s %s\n",
				ui.InfoIcon(),
				ui.WhiteText(fmt.Sprintf("nightly (%s)", current.Name())),
			)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}
		} else {
			shortCommit := nightly.CommitHash()
			if len(shortCommit) > ShortCommitLen {
				shortCommit = shortCommit[:ShortCommitLen]
			}

			publishedStr := nightly.PublishedAt().Format("2006-01-02")

			_, err = fmt.Fprintf(os.Stdout, "%s %s\n", ui.InfoIcon(), ui.CyanText("nightly"))
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			_, err = fmt.Fprintf(os.Stdout, "  %s\n", ui.WhiteText("Published: "+publishedStr))
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			_, err = fmt.Fprintf(os.Stdout, "  %s\n", ui.WhiteText("Commit: "+shortCommit))
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			logrus.Debugf("Latest nightly commit: %s, Published: %s", shortCommit, publishedStr)
		}
	default:
		// Handle custom version or commit hash
		isCommitHash := GetVersionService().IsCommitHash(current.Name())
		logrus.Debugf("isCommitHash: %t", isCommitHash)

		if isCommitHash {
			logrus.Debugf("Displaying custom commit hash: %s", current.Name())

			_, err = fmt.Fprintf(os.Stdout,
				"%s %s\n",
				ui.InfoIcon(),
				ui.WhiteText(fmt.Sprintf("commit (%s)", current.Name())),
			)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}
		} else {
			logrus.Debugf("Displaying custom version: %s", current.Name())

			_, err = fmt.Fprintf(os.Stdout, "%s %s\n", ui.InfoIcon(), ui.WhiteText(current.Name()))
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
