// Package github provides a client for fetching Neovim releases from GitHub API.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/internal/domain/release"
)

const (
	apiBaseURL       = "https://api.github.com"
	clientTimeoutSec = 15
)

// Client implements the release.Repository interface for GitHub.
type Client struct {
	httpClient *http.Client
	cache      *Cache
	minVersion string
}

// NewClient creates a new GitHub client with caching.
func NewClient(cacheFilePath string, cacheTTL time.Duration, minVersion string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: clientTimeoutSec * time.Second},
		cache:      NewCache(cacheFilePath, cacheTTL),
		minVersion: minVersion,
	}
}

// apiRelease represents a release from the GitHub API.
//
//nolint:tagliatelle
type apiRelease struct {
	TagName     string     `json:"tag_name"`
	Prerelease  bool       `json:"prerelease"`
	Assets      []apiAsset `json:"assets"`
	PublishedAt string     `json:"published_at"`
	CommitHash  string     `json:"target_commitish"`
}

// apiAsset represents an asset from the GitHub API.
//
//nolint:tagliatelle
type apiAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// GetAll fetches all available releases from GitHub.
func (c *Client) GetAll(ctx context.Context, force bool) ([]release.Release, error) {
	// Try cache first unless force is true
	if !force {
		cached, err := c.cache.Get()
		if err == nil {
			logrus.Debug("Using cached releases")

			return cached, nil
		}

		if !errors.Is(err, ErrCacheStale) {
			logrus.Warnf("Cache read failed: %v", err)
		}
	}

	logrus.Debug("Fetching fresh releases from GitHub")

	var allAPIReleases []apiRelease

	page := 1
	perPage := 100

	for {
		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			fmt.Sprintf(
				"%s/repos/neovim/neovim/releases?page=%d&per_page=%d",
				apiBaseURL,
				page,
				perPage,
			),
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("User-Agent", "nvs")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch releases: %w", err)
		}

		logrus.Debugf("GitHub API status code: %d", resp.StatusCode)

		if resp.StatusCode == http.StatusForbidden {
			_ = resp.Body.Close()

			return nil, fmt.Errorf("%w: please try again later", ErrRateLimitExceeded)
		}

		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()

			return nil, fmt.Errorf("%w: %d", ErrAPIRequestFailed, resp.StatusCode)
		}

		var apiReleases []apiRelease

		err = json.NewDecoder(resp.Body).Decode(&apiReleases)
		_ = resp.Body.Close()

		if err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		if len(apiReleases) == 0 {
			break
		}

		allAPIReleases = append(allAPIReleases, apiReleases...)

		// Check if there are more pages
		if len(apiReleases) < perPage {
			break
		}

		page++
	}

	releases := c.convertReleases(allAPIReleases)

	// Filter releases >= minVersion
	filtered, err := filterReleases(releases, c.minVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to filter releases: %w", err)
	}

	// Update cache
	err = c.cache.Set(filtered)
	if err != nil {
		logrus.Warnf("Failed to update cache: %v", err)
	}

	return filtered, nil
}

// FindStable returns the latest stable release.
func (c *Client) FindStable(ctx context.Context) (release.Release, error) {
	releases, err := c.GetAll(ctx, false)
	if err != nil {
		return release.Release{}, err
	}

	// Sort releases by published date descending (newest first)
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].PublishedAt().After(releases[j].PublishedAt())
	})

	for _, r := range releases {
		if !r.Prerelease() {
			return r, nil
		}
	}

	return release.Release{}, release.ErrNoStableRelease
}

// FindNightly returns the latest nightly release.
func (c *Client) FindNightly(ctx context.Context) (release.Release, error) {
	releases, err := c.GetAll(ctx, false)
	if err != nil {
		return release.Release{}, err
	}

	for _, r := range releases {
		if r.Prerelease() && strings.HasPrefix(strings.ToLower(r.TagName()), "nightly") {
			return r, nil
		}
	}

	return release.Release{}, release.ErrNoNightlyRelease
}

// FindByTag returns a specific release by tag.
func (c *Client) FindByTag(ctx context.Context, tag string) (release.Release, error) {
	releases, err := c.GetAll(ctx, false)
	if err != nil {
		return release.Release{}, err
	}

	for _, r := range releases {
		if r.TagName() == tag {
			return r, nil
		}
	}

	return release.Release{}, fmt.Errorf("%w: %s", release.ErrReleaseNotFound, tag)
}

// convertReleases converts API releases to domain releases.
func (c *Client) convertReleases(apiReleases []apiRelease) []release.Release {
	releases := make([]release.Release, 0, len(apiReleases))

	for _, apiRelease := range apiReleases {
		publishedAt, err := time.Parse(time.RFC3339, apiRelease.PublishedAt)
		if err != nil {
			logrus.Debugf("Failed to parse published_at for %s: %v", apiRelease.TagName, err)
		}

		assets := make([]release.Asset, 0, len(apiRelease.Assets))
		for _, aa := range apiRelease.Assets {
			assets = append(assets, release.NewAsset(aa.Name, aa.BrowserDownloadURL, aa.Size))
		}

		releases = append(releases, release.New(
			apiRelease.TagName,
			apiRelease.Prerelease,
			apiRelease.CommitHash,
			publishedAt,
			assets,
		))
	}

	return releases
}

// filterReleases filters releases by minimum version.
func filterReleases(releases []release.Release, minVersion string) ([]release.Release, error) {
	constraints, err := semver.NewConstraint(">=" + minVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid version constraint: %w", err)
	}

	filtered := make([]release.Release, 0, len(releases))

	for _, rel := range releases {
		// Always include stable and nightly
		if rel.TagName() == "stable" || rel.TagName() == "nightly" {
			filtered = append(filtered, rel)

			continue
		}

		versionStr := strings.TrimPrefix(rel.TagName(), "v")

		version, err := semver.NewVersion(versionStr)
		if err != nil {
			logrus.Debugf("Skipping invalid version: %s", rel.TagName())

			continue
		}

		if constraints.Check(version) {
			filtered = append(filtered, rel)
		}
	}

	return filtered, nil
}

// GetAssetURL returns the download URL for the current platform.
func GetAssetURL(rel release.Release) (string, string, error) {
	var patterns []string

	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			patterns = []string{"linux-x86_64.tar.gz", "linux-64.tar.gz", "linux64.tar.gz"}
		case "arm64":
			patterns = []string{"linux-arm64.tar.gz"}
		default:
			return "", "", fmt.Errorf("%w: %s", ErrUnsupportedArch, runtime.GOARCH)
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			patterns = []string{"macos-arm64.tar.gz", "macos.tar.gz"}
		} else {
			patterns = []string{"macos-x86_64.tar.gz", "macos.tar.gz"}
		}
	case "windows":
		patterns = []string{"win64.zip"}
	default:
		return "", "", fmt.Errorf("%w: %s", ErrUnsupportedOS, runtime.GOOS)
	}

	for _, asset := range rel.Assets() {
		for _, pattern := range patterns {
			if strings.Contains(asset.Name(), pattern) {
				return asset.DownloadURL(), pattern, nil
			}
		}
	}

	return "", "", fmt.Errorf(
		"%w for %s/%s",
		release.ErrNoMatchingAsset,
		runtime.GOOS,
		runtime.GOARCH,
	)
}

// GetChecksumURL returns the checksum URL for a given asset pattern.
func GetChecksumURL(r release.Release, assetPattern string) (string, error) {
	checksumPattern := assetPattern + ".sha256"

	for _, asset := range r.Assets() {
		if strings.Contains(asset.Name(), checksumPattern) {
			return asset.DownloadURL(), nil
		}
	}

	return "", fmt.Errorf("%w for pattern: %s", ErrChecksumNotFound, checksumPattern)
}
