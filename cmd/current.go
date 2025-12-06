package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/ui"
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
	RunE:  RunCurrent,
}

// RunCurrent executes the current command.
func RunCurrent(cmd *cobra.Command, _ []string) error {
	logrus.Debug("Executing current command")

	current, err := GetVersionService().Current()
	if err != nil {
		return fmt.Errorf("error getting current version: %w", err)
	}

	logrus.Debugf("Current version detected: %s", current.Name())

	jsonOutput, _ := cmd.Flags().GetBool("json")

	type CurrentInfo struct {
		Name      string `json:"name"`
		Type      string `json:"type"`
		Version   string `json:"version,omitempty"`
		Commit    string `json:"commit,omitempty"`
		Published string `json:"published,omitempty"`
	}

	var info CurrentInfo

	// Handle active version
	switch current.Name() {
	case constants.Stable:
		logrus.Debug("Fetching latest stable release")

		info.Name = constants.Stable
		info.Type = "stable"

		stable, err := GetVersionService().FindStable(context.Background())
		if err != nil {
			logrus.Warnf("Error fetching latest stable release: %v", err)

			if !jsonOutput {
				_, err = fmt.Fprintf(os.Stdout,
					"%s %s\n",
					ui.InfoIcon(),
					ui.WhiteText("stable (version details unavailable)"),
				)
				if err != nil {
					logrus.Warnf("Failed to write to stdout: %v", err)
				}
			}
		} else {
			info.Version = stable.TagName()

			if !jsonOutput {
				_, err = fmt.Fprintf(os.Stdout, "%s %s\n", ui.InfoIcon(), ui.CyanText(constants.Stable))
				if err != nil {
					logrus.Warnf("Failed to write to stdout: %v", err)
				}

				_, err = fmt.Fprintf(os.Stdout, "  %s\n", ui.WhiteText("Version: "+stable.TagName()))
				if err != nil {
					logrus.Warnf("Failed to write to stdout: %v", err)
				}
			}

			logrus.Debugf("Latest stable version: %s", stable.TagName())
		}
	case constants.Nightly:
		logrus.Debug("Fetching latest nightly release")

		info.Name = constants.Nightly
		info.Type = "nightly"

		nightly, err := GetVersionService().FindNightly(context.Background())
		if err != nil {
			logrus.Warnf("Error fetching latest nightly release: %v", err)

			if !jsonOutput {
				_, err = fmt.Fprintf(os.Stdout,
					"%s %s\n",
					ui.InfoIcon(),
					ui.WhiteText("nightly (version details unavailable)"),
				)
				if err != nil {
					logrus.Warnf("Failed to write to stdout: %v", err)
				}
			}
		} else {
			shortCommit := nightly.CommitHash()
			if len(shortCommit) > constants.ShortCommitLen {
				shortCommit = shortCommit[:constants.ShortCommitLen]
			}

			publishedStr := nightly.PublishedAt().Format("2006-01-02")

			info.Commit = shortCommit
			info.Published = publishedStr

			if !jsonOutput {
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
			}

			logrus.Debugf("Latest nightly commit: %s, Published: %s", shortCommit, publishedStr)
		}
	default:
		// Handle custom version or commit hash
		isCommitHash := GetVersionService().IsCommitReference(current.Name())
		logrus.Debugf("isCommitHash: %t", isCommitHash)

		info.Name = current.Name()

		if isCommitHash {
			info.Type = "commit"

			if !jsonOutput {
				logrus.Debugf("Displaying custom commit hash: %s", current.Name())

				_, err = fmt.Fprintf(os.Stdout,
					"%s %s\n",
					ui.InfoIcon(),
					ui.WhiteText(fmt.Sprintf("commit (%s)", current.Name())),
				)
				if err != nil {
					logrus.Warnf("Failed to write to stdout: %v", err)
				}
			}
		} else {
			info.Type = "tag"

			if !jsonOutput {
				logrus.Debugf("Displaying custom version: %s", current.Name())

				_, err = fmt.Fprintf(os.Stdout, "%s %s\n", ui.InfoIcon(), ui.WhiteText(current.Name()))
				if err != nil {
					logrus.Warnf("Failed to write to stdout: %v", err)
				}
			}
		}
	}

	if jsonOutput {
		return outputJSON(info)
	}

	return nil
}

func init() {
	currentCmd.Flags().Bool("json", false, "Output in JSON format")
	rootCmd.AddCommand(currentCmd)
}
