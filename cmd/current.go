package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/domain/vtypes"
	"github.com/y3owk1n/nvs/internal/log"
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

// currentInfo is the structured result of RunCurrent. The shape
// is the public JSON contract (TestRunCurrent_JSON asserts on
// it), so the struct fields and their JSON tags must not
// change shape.
type currentInfo struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Version   string `json:"version,omitempty"`
	Commit    string `json:"commit,omitempty"`
	Published string `json:"published,omitempty"`
}

// RunCurrent executes the current command.
func RunCurrent(cmd *cobra.Command, _ []string) error {
	log.Debug("Executing current command")

	current, err := GetVersionService().Current()
	if err != nil {
		return fmt.Errorf("error getting current version: %w", err)
	}

	log.Debugf("Current version detected: %s", current.Name())

	jsonOutput, flagErr := cmd.Flags().GetBool("json")
	if flagErr != nil {
		log.Warnf("Failed to read json flag: %v", flagErr)
	}

	var info currentInfo

	body, err := populateCurrentInfo(cmd, current, &info)
	if err != nil {
		// populateCurrentInfo returns a wrapped error only in
		// --json mode (so a script consumer can detect a
		// partial result from the non-zero exit code). In text
		// mode the user sees a "details unavailable" body
		// instead, and a network failure does not abort the
		// command.
		return err
	}

	if jsonOutput {
		return outputJSON(info)
	}

	_, _ = fmt.Fprint(os.Stdout, ui.Banner.Logo())
	_, _ = fmt.Fprint(os.Stdout, ui.Panel.Section("Current version", body))

	return nil
}

// populateCurrentInfo classifies the active version, populates
// the JSON-ready info struct, and returns the styled panel
// body for the text output.
//
// In --json mode a network failure is returned as an error.
// In text mode a network failure degrades gracefully to a
// "details unavailable" body and returns nil.
func populateCurrentInfo(
	cmd *cobra.Command,
	current vtypes.Version,
	info *currentInfo,
) (string, error) {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	switch current.Name() {
	case constants.Stable:
		log.Debug("Fetching latest stable release")

		info.Name = constants.Stable
		info.Type = "stable"

		stable, findErr := GetVersionService().FindStable(cmd.Context())
		if findErr != nil {
			log.Warnf("Error fetching latest stable release: %v", findErr)

			if jsonOutput {
				return "", fmt.Errorf("failed to fetch latest stable release: %w", findErr)
			}

			return renderUnavailableBody(constants.Stable), nil
		}

		info.Version = stable.TagName()

		log.Debugf("Latest stable version: %s", stable.TagName())

		return renderStableBody(stable.TagName()), nil
	case constants.Nightly:
		log.Debug("Fetching latest nightly release")

		info.Name = constants.Nightly
		info.Type = "nightly"

		nightly, findErr := GetVersionService().FindNightly(cmd.Context())
		if findErr != nil {
			log.Warnf("Error fetching latest nightly release: %v", findErr)

			if jsonOutput {
				return "", fmt.Errorf("failed to fetch latest nightly release: %w", findErr)
			}

			return renderUnavailableBody(constants.Nightly), nil
		}

		shortCommit := nightly.CommitHash()
		if len(shortCommit) > constants.ShortCommitLen {
			shortCommit = shortCommit[:constants.ShortCommitLen]
		}

		publishedStr := nightly.PublishedAt().Format("2006-01-02")

		info.Commit = shortCommit
		info.Published = publishedStr

		log.Debugf("Latest nightly commit: %s, Published: %s", shortCommit, publishedStr)

		return renderNightlyBody(shortCommit, publishedStr), nil
	default:
		isCommitHash := GetVersionService().IsCommitReference(current.Name())
		log.Debugf("isCommitHash: %t", isCommitHash)

		info.Name = current.Name()

		if isCommitHash {
			info.Type = "commit"

			log.Debugf("Displaying custom commit hash: %s", current.Name())

			return renderCommitBody(current.Name()), nil
		}

		info.Type = "tag"

		log.Debugf("Displaying custom version: %s", current.Name())

		return renderTagBody(current.Name()), nil
	}
}

// renderStableBody returns the panel body for the "stable"
// success case.
func renderStableBody(tag string) string {
	return ui.Message.Highlight("→") + " " + ui.Message.Highlight(constants.Stable) + "\n" +
		"\n" +
		ui.Message.PairLine("Latest tag", tag)
}

// renderNightlyBody returns the panel body for the "nightly"
// success case.
func renderNightlyBody(commit, published string) string {
	return ui.Message.Highlight("→") + " " + ui.Message.Highlight(constants.Nightly) + "\n" +
		"\n" +
		ui.Message.PairLine("Latest commit", commit) +
		ui.Message.PairLine("Published", published)
}

// renderTagBody returns the panel body for a custom tag (e.g.
// "v0.10.4"). There is no upstream metadata to look up, so the
// body is just the hero line.
func renderTagBody(name string) string {
	return ui.Message.Highlight("→") + " " + ui.Message.Highlight(name) + "\n"
}

// renderCommitBody returns the panel body for a custom commit
// reference (e.g. "abc1234def"). The commit hash is wrapped in
// "commit (...)" so the user knows at a glance it isn't a
// version tag.
func renderCommitBody(hash string) string {
	return ui.Message.Highlight("→") + " " +
		ui.Message.Highlight("commit ("+hash+")") + "\n"
}

// renderUnavailableBody returns the panel body shown when the
// upstream release fetch fails in text mode. It preserves the
// "this is the version" hero line so the user still sees
// which alias they are on, and appends a dim "details
// unavailable" hint so the empty body doesn't look like a
// crash.
func renderUnavailableBody(name string) string {
	return ui.Message.Highlight("→") + " " + ui.Message.Highlight(name) + "\n" +
		"\n" +
		ui.Message.Dim("Details: unavailable")
}

func init() {
	currentCmd.Flags().Bool("json", false, "Output in JSON format")
	rootCmd.AddCommand(currentCmd)
}
