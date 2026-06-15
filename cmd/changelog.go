package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/infra/httpclient"
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

	url := fmt.Sprintf(
		"%s/%s...%s",
		constants.GitHubCompareURL,
		url.PathEscape(oldCommit),
		url.PathEscape(newCommit),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "nvs")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := httpclient.NewClient(constants.HTTPTimeoutSeconds * time.Second)

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

	// Wrap the body with io.LimitReader before decoding. Every
	// other HTTP body decode in this codebase does the same
	// (see internal/infra/github/client.go). Without the cap, a
	// malicious or compromised upstream could return an
	// unbounded body and cause OOM.
	dec := json.NewDecoder(io.LimitReader(resp.Body, constants.MaxGitHubResponseBytes))

	err = dec.Decode(&compareResp)
	if err != nil {
		logrus.Warnf("Failed to parse changelog: %v", err)

		return nil
	}

	if len(compareResp.Commits) == 0 {
		return nil
	}

	ui.Message.Mutedf("")

	ui.Message.Infof("Changelog (%d commits)", compareResp.TotalCommits)

	ui.Message.Mutedf("")

	// Show recent commits (limited) in a table so the hash
	// column lines up exactly the way it does in
	// `nvs list` / `nvs list-remote`. The message column is
	// pre-truncated to constants.MessageTruncateLimit so the
	// table does not need to wrap.
	tbl := ui.Table.New("Hash", "Message")

	displayed := 0
	for i := len(compareResp.Commits) - 1; i >= 0 && displayed < constants.ChangelogLimit; i-- {
		commit := compareResp.Commits[i]

		message := commit.Commit.Message

		// Get first line of commit message
		for j, c := range message {
			if c == '\n' {
				message = message[:j]

				break
			}
		}

		// Truncate long messages. utf8.RuneCountInString is
		// allocation-free for the count; the full rune slice
		// is only materialized when we actually have to slice
		// the string.
		if utf8.RuneCountInString(message) > constants.MessageTruncateLimit {
			runes := []rune(message)[:constants.MessageTruncateLimit-3]
			message = string(runes) + "..."
		}

		tbl.Row(
			ui.Message.Accent(shortHash(commit.SHA, constants.DisplayHashLength)),
			message,
		)

		displayed++
	}

	_, _ = fmt.Fprint(os.Stdout, tbl.Render(ui.Style.Palette()))

	if compareResp.TotalCommits > constants.ChangelogLimit {
		ui.Message.Mutedf(
			"  ... and %d more commits",
			compareResp.TotalCommits-constants.ChangelogLimit,
		)
	}

	ui.Message.Mutedf(
		"View full changelog: %s",
		ui.Message.Accent(fmt.Sprintf(
			"%s/neovim/neovim/compare/%s...%s",
			constants.DefaultGitHubBaseURL,
			shortHash(oldCommit, constants.DisplayHashLength),
			shortHash(newCommit, constants.DisplayHashLength),
		)),
	)

	ui.Message.Mutedf("")

	return nil
}
