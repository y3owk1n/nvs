package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/ui"
)

// GitHubCommit represents a commit from the GitHub Compare API.
type GitHubCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Name string `json:"name"`
			Date string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

// GitHubCompareResponse represents the response from GitHub Compare API.
//
//nolint:tagliatelle
type GitHubCompareResponse struct {
	Status       string         `json:"status"`
	AheadBy      int            `json:"ahead_by"`
	BehindBy     int            `json:"behind_by"`
	TotalCommits int            `json:"total_commits"`
	Commits      []GitHubCommit `json:"commits"`
}

// shortHash returns the first n characters of a hash, or the full hash if shorter.
func shortHash(hash string, n int) string {
	if len(hash) <= n {
		return hash
	}

	return hash[:n]
}

// ShowChangelog displays the commits between two versions.
func ShowChangelog(ctx context.Context, oldCommit, newCommit string) error {
	if oldCommit == "" || newCommit == "" {
		logrus.Debug("Cannot show changelog: missing commit hash")

		return nil
	}

	// Truncate to max constants.CommitHashLength chars (full SHA length)
	if len(oldCommit) > constants.CommitHashLength {
		oldCommit = oldCommit[:constants.CommitHashLength]
	}

	if len(newCommit) > constants.CommitHashLength {
		newCommit = newCommit[:constants.CommitHashLength]
	}

	// If they're the same, no changelog to show
	if oldCommit == newCommit {
		return nil
	}

	logrus.Debugf(
		"Fetching changelog from %s to %s",
		shortHash(oldCommit, constants.ShortHashLength),
		shortHash(newCommit, constants.ShortHashLength),
	)

	// Fetch comparison from GitHub API
	// Note: GitHub API has rate limits (60 requests/hour for unauthenticated)
	logrus.Debug("Fetching changelog from GitHub API (subject to rate limits)")

	url := fmt.Sprintf("%s/%s...%s", constants.GitHubCompareURL, oldCommit, newCommit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "nvs")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: constants.HTTPTimeoutSeconds * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		logrus.Warnf("Failed to fetch changelog: %v", err)

		return nil // Don't fail upgrade just because changelog fetch failed
	}

	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			logrus.Warnf("Failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		logrus.Debugf("GitHub API returned status %d", resp.StatusCode)

		return nil
	}

	var compareResp GitHubCompareResponse

	err = json.NewDecoder(resp.Body).Decode(&compareResp)
	if err != nil {
		logrus.Warnf("Failed to parse changelog: %v", err)

		return nil
	}

	if len(compareResp.Commits) == 0 {
		return nil
	}

	// Display changelog header
	var printErr error

	_, printErr = fmt.Fprintf(
		os.Stdout,
		"\n%s Changelog (%d commits):\n",
		ui.InfoIcon(),
		compareResp.TotalCommits,
	)
	if printErr != nil {
		logrus.Warnf("Failed to write to stdout: %v", printErr)
	}

	_, printErr = fmt.Fprintln(os.Stdout, "─────────────────────────────────────────")
	if printErr != nil {
		logrus.Warnf("Failed to write to stdout: %v", printErr)
	}

	// Show recent commits (limited)
	displayed := 0
	for i := len(compareResp.Commits) - 1; i >= 0 && displayed < constants.ChangelogLimit; i-- {
		commit := compareResp.Commits[i]

		// Get first line of commit message
		message := commit.Commit.Message
		for j, c := range message {
			if c == '\n' {
				message = message[:j]

				break
			}
		}

		// Truncate long messages
		if runeCount := len([]rune(message)); runeCount > constants.MessageTruncateLimit {
			message = string([]rune(message)[:constants.MessageTruncateLimit-3]) + "..."
		}

		_, printErr = fmt.Fprintf(os.Stdout, "  %s %s\n",
			ui.CyanText(shortHash(commit.SHA, constants.DisplayHashLength)),
			message,
		)
		if printErr != nil {
			logrus.Warnf("Failed to write to stdout: %v", printErr)
		}

		displayed++
	}

	if compareResp.TotalCommits > constants.ChangelogLimit {
		_, printErr = fmt.Fprintf(
			os.Stdout,
			"  ... and %d more commits\n",
			compareResp.TotalCommits-constants.ChangelogLimit,
		)
		if printErr != nil {
			logrus.Warnf("Failed to write to stdout: %v", printErr)
		}
	}

	_, printErr = fmt.Fprintln(os.Stdout, "─────────────────────────────────────────")
	if printErr != nil {
		logrus.Warnf("Failed to write to stdout: %v", printErr)
	}

	_, printErr = fmt.Fprintf(
		os.Stdout,
		"View full changelog: %s/neovim/neovim/compare/%s...%s\n\n",
		constants.DefaultGitHubBaseURL,
		shortHash(oldCommit, constants.DisplayHashLength),
		shortHash(newCommit, constants.DisplayHashLength),
	)
	if printErr != nil {
		logrus.Warnf("Failed to write to stdout: %v", printErr)
	}

	return nil
}
