// Package releases provides functions for fetching Neovim releases from GitHub.
package releases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/Masterminds/semver"
	"github.com/sirupsen/logrus"
)

// Constants for release types.
const (
	Stable  = "stable"
	Nightly = "nightly"
)

// Errors for releases operations.
var (
	ErrRateLimitExceeded = errors.New("GitHub API rate limit exceeded")
	ErrAPIStatus         = errors.New("API returned status")
	ErrNoStableRelease   = errors.New("no stable release found")
	ErrNoNightlyRelease  = errors.New("no nightly release found")
	ErrVersionNotFound   = errors.New("version not found")
	ErrUnsupportedArch   = errors.New("unsupported architecture")
	ErrUnsupportedOS     = errors.New("unsupported OS")
	ErrNoMatchingAsset   = errors.New("no matching asset found")
)

// Constants for releases operations.
const (
	StableLower      = "stable"
	NightlyLower     = "nightly"
	ClientTimeoutSec = 15
	CacheFilePerm    = 0o644
)

// Release represents a Neovim release.
type Release struct {
	TagName     string  `json:"tag_name"` //nolint:tagliatelle
	Prerelease  bool    `json:"prerelease"`
	Assets      []Asset `json:"assets"`
	PublishedAt string  `json:"published_at"`     //nolint:tagliatelle
	CommitHash  string  `json:"target_commitish"` //nolint:tagliatelle
}

// Asset represents an asset attached to a release.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"` //nolint:tagliatelle
}

// Variables for releases operations.
var (
	Client           = &http.Client{Timeout: ClientTimeoutSec * time.Second}
	apiBaseURL       = "https://api.github.com"
	releasesCacheTTL = 5 * time.Minute
)

// ResolveVersion resolves the given version alias (e.g. stable, "nightly", or a specific version)
// to a Release by checking cached releases or fetching them from GitHub.
//
// Example usage:
//
//	release, err := ResolveVersion(Stable, "/path/to/cache.json")
//	if err != nil {
//	    // handle error
//	}
//	fmt.Println("Resolved release:", release.TagName)
func ResolveVersion(version, cachePath string) (Release, error) {
	switch version {
	case Stable:
		return FindLatestStable(cachePath)
	case Nightly:
		return FindLatestNightly(cachePath)
	default:
		return FindSpecificVersion(version, cachePath)
	}
}

// GetCachedReleases retrieves releases from the cache (if available and fresh) or fetches them
// from GitHub if the cache is stale or forced. The releases are cached to the provided cachePath.
//
// Example usage:
//
//	releases, err := GetCachedReleases(false, "/path/to/cache.json")
//	if err != nil {
//	    // handle error
//	}
//	fmt.Println("Number of cached releases:", len(releases))
func GetCachedReleases(force bool, cachePath string) ([]Release, error) {
	if !force {
		info, err := os.Stat(cachePath)
		if err == nil {
			if time.Since(info.ModTime()) < releasesCacheTTL {
				data, err := os.ReadFile(cachePath)
				if err == nil {
					var releases []Release

					err = json.Unmarshal(data, &releases)
					if err == nil {
						logrus.Debug("Using cached releases")

						return releases, nil
					}
				}
			}
		}
	}

	logrus.Debug("Fetching fresh releases from GitHub")

	releases, err := GetReleases()
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(releases)
	if err == nil {
		err := os.WriteFile(cachePath, data, CacheFilePerm)
		if err != nil {
			return nil, fmt.Errorf("failed to write file to cache: %w", err)
		}
	}

	return releases, nil
}

// GetReleases fetches the list of releases from the GitHub API for Neovim and filters them
// based on a minimum version (hardcoded as "0.5.0").
//
// Example usage:
//
//	releases, err := GetReleases()
//	if err != nil {
//	    // handle error
//	}
//	fmt.Println("Fetched releases count:", len(releases))
func GetReleases() ([]Release, error) {
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		apiBaseURL+"/repos/neovim/neovim/releases",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "nvs")

	resp, err := Client.Do(req)
	// Return err
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			logrus.Warnf("Failed to close response body: %v", err)
		}
	}()

	logrus.Debugf("StatusCode: %d", resp.StatusCode)

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w. Please try again later", ErrRateLimitExceeded)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w %d", ErrAPIStatus, resp.StatusCode)
	}

	var releases []Release

	err = json.NewDecoder(resp.Body).Decode(&releases)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return FilterReleases(releases, "0.5.0")
}

// IsCommitHash checks if the given string is a valid commit hash (7 or 40 hexadecimal characters)
// or the literal "master".
//
// Example usage:
//
//	valid := IsCommitHash("1a2b3c4")
//	fmt.Println("Is valid commit hash?", valid)
func IsCommitHash(str string) bool {
	if str == "master" {
		return true
	}

	if len(str) != 7 && len(str) != 40 {
		return false
	}

	for _, r := range str {
		if !unicode.IsDigit(r) && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}

	return true
}

// NormalizeVersion returns the version string in a normalized format.
// It prefixes the version with "v" unless the version is stable, "nightly", or already a commit hash.
//
// Example usage:
//
//	normalized := NormalizeVersion("1.2.3")
//	fmt.Println("Normalized version:", normalized) // Output: v1.2.3
func NormalizeVersion(version string) string {
	if version == Stable || version == Nightly || IsCommitHash(version) {
		return version
	}

	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}

	return version
}

// FindLatestStable returns the latest stable release (non-prerelease) from the cached releases.
//
// Example usage:
//
//	release, err := FindLatestStable("/path/to/cache.json")
//	if err != nil {
//	    // handle error
//	}
//	fmt.Println("Latest stable release:", release.TagName)
func FindLatestStable(cachePath string) (Release, error) {
	releases, err := GetCachedReleases(false, cachePath)
	if err != nil {
		return Release{}, err
	}

	if len(releases) == 0 {
		return Release{}, ErrNoStableRelease
	}

	for _, r := range releases {
		if !r.Prerelease {
			return r, nil
		}
	}

	return Release{}, ErrNoStableRelease
}

// FindLatestNightly returns the latest nightly (prerelease) release from the cached releases.
//
// Example usage:
//
//	release, err := FindLatestNightly("/path/to/cache.json")
//	if err != nil {
//	    // handle error
//	}
//	fmt.Println("Latest Nightly release:", release.TagName)
func FindLatestNightly(cachePath string) (Release, error) {
	releases, err := GetCachedReleases(false, cachePath)
	if err != nil {
		return Release{}, err
	}

	if len(releases) == 0 {
		return Release{}, ErrNoNightlyRelease
	}

	for _, r := range releases {
		if r.Prerelease && strings.HasPrefix(r.TagName, "nightly") {
			return r, nil
		}
	}

	return Release{}, ErrNoNightlyRelease
}

// FindSpecificVersion returns the release that exactly matches the provided version tag.
//
// Example usage:
//
//	release, err := FindSpecificVersion("v0.6.0", "/path/to/cache.json")
//	if err != nil {
//	    // handle error
//	}
//	fmt.Println("Found release:", release.TagName)
func FindSpecificVersion(version, cachePath string) (Release, error) {
	releases, err := GetCachedReleases(false, cachePath)
	if err != nil {
		return Release{}, err
	}

	for _, r := range releases {
		if r.TagName == version {
			return r, nil
		}
	}

	return Release{}, fmt.Errorf("%w: %s", ErrVersionNotFound, version)
}

// GetAssetURL scans the release assets and returns the BrowserDownloadURL for the asset that
// matches the current OS/architecture along with the matched asset pattern.
//
// Example usage:
//
//	url, pattern, err := GetAssetURL(release)
//	if err != nil {
//	    // handle error
//	}
//	fmt.Println("Asset URL:", url, "Pattern:", pattern)
func GetAssetURL(release Release) (string, string, error) {
	var patterns []string
	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			patterns = []string{"linux-x86_64.tar.gz", "linux-64.tar.gz", "linux64.tar.gz"}
		case "arm64":
			patterns = []string{"linux-arm64.tar.gz", "linux-64.tar.gz", "linux64.tar.gz"}
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

	for _, asset := range release.Assets {
		for _, pattern := range patterns {
			if strings.Contains(asset.Name, pattern) {
				return asset.BrowserDownloadURL, pattern, nil
			}
		}
	}

	return "", "", fmt.Errorf("%w for %s/%s", ErrNoMatchingAsset, runtime.GOOS, runtime.GOARCH)
}

// GetInstalledReleaseIdentifier reads the version.txt file from the installed release directory
// and returns its content as the release identifier.
//
// Example usage:
//
//	id, err := GetInstalledReleaseIdentifier("/path/to/versions", "v0.6.0")
//	if err != nil {
//	    // handle error
//	}
//	fmt.Println("Installed release identifier:", id)
func GetInstalledReleaseIdentifier(versionsDir string, alias string) (string, error) {
	versionFile := filepath.Join(versionsDir, alias, "version.txt")

	data, err := os.ReadFile(versionFile)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

// GetChecksumURL returns the checksum URL for a given release by matching the asset whose name
// contains the assetPattern with ".sha256" appended.
//
// Example usage:
//
//	checksumURL, err := GetChecksumURL(release, "linux-x86_64.tar.gz")
//	if err != nil {
//	    // handle error
//	}
//	fmt.Println("Checksum URL:", checksumURL)
func GetChecksumURL(release Release, assetPattern string) (string, error) {
	checksumPattern := assetPattern + ".sha256"
	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, checksumPattern) {
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", nil
}

// GetReleaseIdentifier returns a string identifier for the release based on the alias.
// For nightly releases, it removes a "nightly-" prefix if present, or returns the first 7 characters of the commit hash.
// For other releases, it returns the tag name.
//
// Example usage:
//
//	identifier := GetReleaseIdentifier(release, Stable)
//	fmt.Println("Release identifier:", identifier)
func GetReleaseIdentifier(release Release, alias string) string {
	if alias == Nightly {
		if after, ok := strings.CutPrefix(release.TagName, Nightly+"-"); ok {
			return after
		}

		return release.CommitHash[:7]
	}

	return release.TagName
}

// FilterReleases filters the provided list of releases and returns only those releases
// whose version is greater than or equal to the specified minimum version.
// Non-semver versions are skipped.
//
// Example usage:
//
//	filtered, err := FilterReleases(releases, "0.5.0")
//	if err != nil {
//	    // handle error
//	}
//	fmt.Println("Filtered releases count:", len(filtered))
func FilterReleases(releases []Release, minVersion string) ([]Release, error) {
	constraints, err := semver.NewConstraint(">=" + minVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid version constraint: %w", err)
	}

	var filtered []Release
	for _, release := range releases {
		if release.TagName == Stable || release.TagName == Nightly {
			filtered = append(filtered, release)

			continue
		}

		versionStr := strings.TrimPrefix(release.TagName, "v")

		version, err := semver.NewVersion(versionStr)
		if err != nil {
			logrus.Debugf("Skipping invalid version: %s", release.TagName)

			continue
		}

		if constraints.Check(version) {
			filtered = append(filtered, release)
		}
	}

	return filtered, nil
}
