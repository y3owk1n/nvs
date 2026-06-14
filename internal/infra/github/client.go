// Package github provides a client for fetching Neovim releases from GitHub API.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver"
	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/domain/release"
	"github.com/y3owk1n/nvs/internal/infra/httpclient"
)

// Client implements the release.Repository interface for GitHub.
type Client struct {
	httpClient     *http.Client
	cache          *Cache
	minVersion     string
	mirrorURL      string // Optional mirror URL for GitHub (e.g., https://mirror.ghproxy.com)
	useGlobalCache bool   // Whether to use global cache

	// memCacheMu guards memCacheReleases and memCacheLoaded. The
	// in-memory cache mirrors the disk cache (see Cache below) so
	// that repeated GetAll calls in the same process do not
	// re-stat, re-read, and re-decode the on-disk JSON. The disk
	// cache is the source of truth across processes; the in-memory
	// cache is reset on every process restart.
	memCacheMu       sync.RWMutex
	memCacheReleases []release.Release
	memCacheLoaded   bool

	// fetchMu serializes the slow path (disk read + network fetch) so
	// that concurrent GetAll callers with no in-memory cache share
	// one fetch instead of all racing to disk and the network.
	// Callers always check the in-memory cache (read-locked) before
	// taking fetchMu, so the lock is only contended on the very
	// first concurrent call in a process.
	fetchMu sync.Mutex
}

// NewClient creates a new GitHub client with caching.
// mirrorURL is optional - pass empty string to use default GitHub URLs.
// useGlobalCache enables fetching from global cache.
func NewClient(
	cacheFilePath string,
	cacheTTL time.Duration,
	minVersion, mirrorURL string,
	useGlobalCache bool,
) *Client {
	return &Client{
		httpClient:     httpclient.NewClient(constants.ClientTimeoutSec * time.Second),
		cache:          NewCache(cacheFilePath, cacheTTL),
		minVersion:     minVersion,
		mirrorURL:      mirrorURL,
		useGlobalCache: useGlobalCache,
	}
}

// ApplyMirrorToURL replaces the default GitHub URL with the mirror URL if configured.
// This is used for download URLs (not API calls).
func ApplyMirrorToURL(url, mirrorURL string) string {
	if mirrorURL == "" {
		return url
	}

	// Replace https://github.com with the mirror URL
	return strings.Replace(url, constants.DefaultGitHubBaseURL, mirrorURL, 1)
}

// ApplyMirror replaces the default GitHub URL with the mirror URL if configured.
// This is used for download URLs (not API calls).
func (c *Client) ApplyMirror(url string) string {
	return ApplyMirrorToURL(url, c.mirrorURL)
}

// MirrorURL returns the configured mirror URL.
func (c *Client) MirrorURL() string {
	return c.mirrorURL
}

// FetchRemoteVersionsJSON fetches releases from the global cache JSON.
func (c *Client) FetchRemoteVersionsJSON(ctx context.Context) ([]release.Release, error) {
	resp, err := c.doWithRetry(ctx, constants.GlobalCacheURL)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"remote versions fetch failed with status %d: %w",
			resp.StatusCode,
			ErrAPIRequestFailed,
		)
	}

	var globalReleases []globalCacheRelease

	dec := json.NewDecoder(io.LimitReader(resp.Body, constants.MaxGitHubResponseBytes))

	err = dec.Decode(&globalReleases)
	if err != nil {
		return nil, fmt.Errorf("failed to decode remote versions: %w", err)
	}

	// Convert globalCacheRelease to apiRelease for compatibility
	apiReleases := make([]apiRelease, len(globalReleases))
	for idx, globalRelease := range globalReleases {
		assets := make([]apiAsset, len(globalRelease.Assets))
		for j, ga := range globalRelease.Assets {
			assets[j] = apiAsset(ga)
		}

		apiReleases[idx] = apiRelease{
			TagName:     globalRelease.TagName,
			Prerelease:  globalRelease.Prerelease,
			Assets:      assets,
			PublishedAt: globalRelease.PublishedAt,
			CommitHash:  globalRelease.CommitHash,
		}
	}

	releases := c.convertReleases(apiReleases)
	filtered := filterReleases(releases, c.minVersion)

	return filtered, nil
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

// globalCacheRelease represents a release from the global cache JSON.
//
//nolint:tagliatelle
type globalCacheRelease struct {
	TagName     string             `json:"TagName"`
	Prerelease  bool               `json:"Prerelease"`
	Assets      []globalCacheAsset `json:"Assets"`
	PublishedAt string             `json:"PublishedAt"`
	CommitHash  string             `json:"CommitHash"`
}

// globalCacheAsset represents an asset from the global cache JSON.
//
//nolint:tagliatelle
type globalCacheAsset struct {
	Name               string `json:"Name"`
	BrowserDownloadURL string `json:"BrowserDownloadURL"`
	Size               int64  `json:"Size"`
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
//
// Results are cached in two layers:
//
//  1. In-memory (per process). The first successful call populates
//     memCacheReleases; subsequent calls with force=false return the
//     cached slice without touching the disk or the network. This
//     avoids re-decoding the JSON cache for every call site that
//     resolves a release (FindStable, FindNightly, FindByTag, etc.),
//     which together can call GetAll three or more times in a single
//     command invocation.
//
//  2. On disk (across processes). See Cache. The disk cache is
//     consulted only when the in-memory cache is empty or force=true.
//
// The force flag bypasses both caches: it forces a fresh network
// fetch and refreshes both the in-memory and on-disk caches.
//
// Concurrency: the slow path (disk read + network fetch) is
// serialized via fetchMu, so concurrent callers with a cold
// in-memory cache share one fetch instead of N parallel disk reads
// and N parallel network requests.
//
// Resilience: if every fresh source (global cache, GitHub API) fails
// AND an on-disk cache exists, the stale cache is returned as a
// last resort so a transient network blip doesn't break the command.
func (c *Client) GetAll(ctx context.Context, force bool) ([]release.Release, error) {
	// Fast path: read in-memory cache without taking the fetch lock.
	if !force {
		if cached := c.memCacheSnapshot(); cached != nil {
			logrus.Debug("Using in-memory cached releases")

			return cached, nil
		}
	}

	// Slow path: serialize so concurrent callers share one fetch.
	// On contention, callers re-check the in-memory cache; the
	// first one populates it and the rest return early.
	c.fetchMu.Lock()
	defer c.fetchMu.Unlock()

	if !force {
		if cached := c.memCacheSnapshot(); cached != nil {
			logrus.Debug("Using in-memory cached releases")

			return cached, nil
		}
	}

	// Try local disk cache first unless force is true
	if !force {
		cached, err := c.cache.Get()
		if err == nil {
			logrus.Debug("Using on-disk cached releases")
			c.storeMemCache(cached)

			return cached, nil
		}

		if !errors.Is(err, ErrCacheStale) {
			logrus.Debugf("Cache read failed: %v", err)
		}
	}

	// Cache is stale or missing, fetch fresh data
	var (
		releases []release.Release
		err      error
	)

	if c.useGlobalCache {
		logrus.Debug("Fetching fresh releases from global cache")

		releases, err = c.FetchRemoteVersionsJSON(ctx)
		if err != nil {
			logrus.Warnf(
				"Global cache fetch failed, falling back to GitHub API: %v",
				err,
			)
			releases, err = c.fetchFromGitHubAPI(ctx)
		}
	} else {
		logrus.Debug("Fetching fresh releases from GitHub")

		releases, err = c.fetchFromGitHubAPI(ctx)
	}

	if err != nil {
		// All fresh sources failed. As a last resort, fall back to
		// whatever is on disk regardless of TTL, so a transient
		// network blip doesn't fail the entire command.
		var stale []release.Release

		stale, staleErr := c.cache.GetIgnoreStale()
		if staleErr == nil {
			logrus.Warnf(
				"Fresh fetch failed (%v); serving stale cache (%d releases)",
				err,
				len(stale),
			)
			c.storeMemCache(stale)

			return stale, nil
		}

		return nil, err
	}

	// Update cache (Set is a no-op for empty input)
	setErr := c.cache.Set(releases)
	if setErr != nil {
		logrus.Warnf("Failed to update cache: %v", setErr)
	}

	c.storeMemCache(releases)

	return releases, nil
}

// FindStable returns the latest stable release.
func (c *Client) FindStable(ctx context.Context) (release.Release, error) {
	releases, err := c.GetAll(ctx, false)
	if err != nil {
		return release.Release{}, err
	}

	// Sort releases by published date descending (newest first)
	slices.SortFunc(releases, func(a, b release.Release) int {
		return b.PublishedAt().Compare(a.PublishedAt())
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

	// Sort releases by published date descending (newest first)
	slices.SortFunc(releases, func(a, b release.Release) int {
		return b.PublishedAt().Compare(a.PublishedAt())
	})

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

// apiPageSize is the number of releases requested per page from
// the GitHub API.
const apiPageSize = 100

// fetchFromGitHubAPI fetches releases directly from the GitHub API.
func (c *Client) fetchFromGitHubAPI(ctx context.Context) ([]release.Release, error) {
	const maxPages = 50

	apiReleases := make([]apiRelease, 0, apiPageSize)

	for page := 1; page <= maxPages; page++ {
		url := fmt.Sprintf(
			"%s/repos/neovim/neovim/releases?page=%d&per_page=%d",
			constants.DefaultAPIBaseURL,
			page,
			apiPageSize,
		)

		pageReleases, lastPage, err := c.fetchGitHubAPIPage(ctx, url)
		if err != nil {
			return nil, err
		}

		apiReleases = append(apiReleases, pageReleases...)

		if lastPage {
			break
		}
	}

	releases := c.convertReleases(apiReleases)

	// Filter releases >= minVersion
	filtered := filterReleases(releases, c.minVersion)

	return filtered, nil
}

// fetchGitHubAPIPage fetches a single page of releases. lastPage is
// true when the server returned fewer results than apiPageSize (or
// zero) and there are no more pages to fetch.
func (c *Client) fetchGitHubAPIPage(
	ctx context.Context,
	url string,
) ([]apiRelease, bool, error) {
	resp, err := c.doWithRetry(ctx, url)
	if err != nil {
		return nil, false, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	logrus.Debugf("GitHub API status code: %d", resp.StatusCode)

	if resp.StatusCode == http.StatusForbidden {
		return nil, false, fmt.Errorf("%w: please try again later", ErrRateLimitExceeded)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("%w: %d", ErrAPIRequestFailed, resp.StatusCode)
	}

	var apiReleases []apiRelease

	dec := json.NewDecoder(io.LimitReader(resp.Body, constants.MaxGitHubResponseBytes))

	err = dec.Decode(&apiReleases)
	if err != nil {
		return nil, false, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiReleases) == 0 {
		return apiReleases, true, nil
	}

	lastPage := len(apiReleases) < apiPageSize

	return apiReleases, lastPage, nil
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
func filterReleases(releases []release.Release, minVersion string) []release.Release {
	if minVersion == "" {
		return releases
	}

	constraints, err := semver.NewConstraint(">=" + minVersion)
	if err != nil {
		logrus.Warnf("Invalid minVersion %s, returning all releases: %v", minVersion, err)

		return releases
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

	return filtered
}

// GetAssetURL returns the download URL for the current platform.
func GetAssetURL(rel release.Release) (string, string, error) {
	var patterns []string

	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			patterns = []string{"linux-x86_64.tar.gz", "linux-64.tar.gz", "linux64.tar.gz"}
		case constants.Arm64Arch:
			patterns = []string{"linux-arm64.tar.gz"}
		default:
			return "", "", fmt.Errorf("%w: %s", ErrUnsupportedArch, runtime.GOARCH)
		}
	case "darwin":
		if runtime.GOARCH == constants.Arm64Arch {
			patterns = []string{"macos-arm64.tar.gz", "macos.tar.gz"}
		} else {
			patterns = []string{"macos-x86_64.tar.gz", "macos.tar.gz"}
		}
	case "windows":
		switch runtime.GOARCH {
		case "amd64":
			patterns = []string{"win64.zip"}
		case constants.Arm64Arch:
			patterns = []string{"win-arm64.zip", "win64.zip"}
		default:
			return "", "", fmt.Errorf("%w: %s", ErrUnsupportedArch, runtime.GOARCH)
		}
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
func GetChecksumURL(rel release.Release, assetPattern string) (string, error) {
	// First, try per-asset .sha256 file
	checksumPattern := assetPattern + ".sha256"

	for _, asset := range rel.Assets() {
		if strings.Contains(asset.Name(), checksumPattern) {
			return asset.DownloadURL(), nil
		}
	}

	// Fallback to shasum.txt for newer releases
	for _, asset := range rel.Assets() {
		if asset.Name() == "shasum.txt" {
			return asset.DownloadURL(), nil
		}
	}

	return "", fmt.Errorf("%w for pattern: %s", ErrChecksumNotFound, assetPattern)
}

// memCacheSnapshot returns a copy of the cached release slice, or
// nil if the in-memory cache has not been populated yet. The slice
// is always cloned so that concurrent callers may freely sort,
// iterate, or otherwise transform their copy without disturbing the
// shared cache.
func (c *Client) memCacheSnapshot() []release.Release {
	c.memCacheMu.RLock()
	defer c.memCacheMu.RUnlock()

	if !c.memCacheLoaded {
		return nil
	}

	return slices.Clone(c.memCacheReleases)
}

// storeMemCache replaces the in-memory cache with releases. The slice
// is stored by reference; callers must not mutate it after handing it
// over.
func (c *Client) storeMemCache(releases []release.Release) {
	c.memCacheMu.Lock()
	defer c.memCacheMu.Unlock()

	c.memCacheReleases = releases
	c.memCacheLoaded = true
}
