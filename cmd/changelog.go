package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/internal/ui"
)

const (
	githubCompareURL     = "https://api.github.com/repos/neovim/neovim/compare"
	changelogLimit       = 10
	commitHashLength     = 40
	shortHashLength      = 8
	httpTimeoutSeconds   = 30
	messageTruncateLimit = 70
	displayHashLength    = 7
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

	// Truncate to max commitHashLength chars (full SHA length)
	if len(oldCommit) > commitHashLength {
		oldCommit = oldCommit[:commitHashLength]
	}

	if len(newCommit) > commitHashLength {
		newCommit = newCommit[:commitHashLength]
	}

	// If they're the same, no changelog to show
	if oldCommit == newCommit {
		return nil
	}

	logrus.Debugf(
		"Fetching changelog from %s to %s",
		shortHash(oldCommit, shortHashLength),
		shortHash(newCommit, shortHashLength),
	)

	// Fetch comparison from GitHub API
	url := fmt.Sprintf("%s/%s...%s", githubCompareURL, oldCommit, newCommit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "nvs")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: httpTimeoutSeconds * time.Second}

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
	for i := len(compareResp.Commits) - 1; i >= 0 && displayed < changelogLimit; i-- {
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
		if len(message) > messageTruncateLimit {
			message = message[:messageTruncateLimit-3] + "..."
		}

		_, printErr = fmt.Fprintf(os.Stdout, "  %s %s\n",
			ui.CyanText(shortHash(commit.SHA, displayHashLength)),
			message,
		)
		if printErr != nil {
			logrus.Warnf("Failed to write to stdout: %v", printErr)
		}

		displayed++
	}

	if compareResp.TotalCommits > changelogLimit {
		_, printErr = fmt.Fprintf(
			os.Stdout,
			"  ... and %d more commits\n",
			compareResp.TotalCommits-changelogLimit,
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
		"View full changelog: https://github.com/neovim/neovim/compare/%s...%s\n\n",
		shortHash(oldCommit, displayHashLength),
		shortHash(newCommit, displayHashLength),
	)
	if printErr != nil {
		logrus.Warnf("Failed to write to stdout: %v", printErr)
	}

	return nil
}
