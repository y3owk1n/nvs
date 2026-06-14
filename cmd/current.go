package cmd

import (
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

	jsonOutput, flagErr := cmd.Flags().GetBool("json")
	if flagErr != nil {
		logrus.Warnf("Failed to read json flag: %v", flagErr)
	}

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

		// findErr is kept separate from the err reused below for
		// stdout writes; the two failure modes are independent
		// (network vs. terminal) and shouldn't clobber each other.
		stable, findErr := GetVersionService().FindStable(cmd.Context())
		if findErr != nil {
			logrus.Warnf("Error fetching latest stable release: %v", findErr)

			if !jsonOutput {
				_, printErr := fmt.Fprintf(
					os.Stdout,
					"%s %s\n",
					ui.InfoIcon(),
					ui.WhiteText("stable (version details unavailable)"),
				)
				if printErr != nil {
					logrus.Warnf("Failed to write to stdout: %v", printErr)
				}
			} else {
				// In --json mode, scripts consume stdout as data
				// and rely on the exit code for status. Returning
				// here turns a fetch failure into a non-zero exit
				// so the script can detect it, rather than
				// silently emitting a partial object.
				return fmt.Errorf("failed to fetch latest stable release: %w", findErr)
			}
		} else {
			info.Version = stable.TagName()

			if !jsonOutput {
				_, err = fmt.Fprintf(
					os.Stdout,
					"%s %s\n",
					ui.InfoIcon(),
					ui.CyanText(constants.Stable),
				)
				if err != nil {
					logrus.Warnf("Failed to write to stdout: %v", err)
				}

				_, err = fmt.Fprintf(
					os.Stdout,
					"  %s\n",
					ui.WhiteText("Version: "+stable.TagName()),
				)
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

		// See the comment in the stable branch above for why
		// findErr / printErr are separate from the outer err.
		nightly, findErr := GetVersionService().FindNightly(cmd.Context())
		if findErr != nil {
			logrus.Warnf("Error fetching latest nightly release: %v", findErr)

			if !jsonOutput {
				_, printErr := fmt.Fprintf(
					os.Stdout,
					"%s %s\n",
					ui.InfoIcon(),
					ui.WhiteText("nightly (version details unavailable)"),
				)
				if printErr != nil {
					logrus.Warnf("Failed to write to stdout: %v", printErr)
				}
			} else {
				// See stable branch — emit non-zero exit so
				// --json consumers can detect the partial result.
				return fmt.Errorf("failed to fetch latest nightly release: %w", findErr)
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
				_, err = fmt.Fprintf(
					os.Stdout,
					"%s %s\n",
					ui.InfoIcon(),
					ui.CyanText("nightly"),
				)
				if err != nil {
					logrus.Warnf("Failed to write to stdout: %v", err)
				}

				_, err = fmt.Fprintf(
					os.Stdout,
					"  %s\n",
					ui.WhiteText("Published: "+publishedStr),
				)
				if err != nil {
					logrus.Warnf("Failed to write to stdout: %v", err)
				}

				_, err = fmt.Fprintf(
					os.Stdout,
					"  %s\n",
					ui.WhiteText("Commit: "+shortCommit),
				)
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

				_, err = fmt.Fprintf(
					os.Stdout,
					"%s %s\n",
					ui.InfoIcon(),
					ui.WhiteText(current.Name()),
				)
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
