package releases

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/sirupsen/logrus"
)

// Release represents a GitHub release.
type Release struct {
	TagName     string  `json:"tag_name"`
	Prerelease  bool    `json:"prerelease"`
	Assets      []Asset `json:"assets"`
	PublishedAt string  `json:"published_at"`
	CommitHash  string  `json:"target_commitish"`
}

// Asset represents an asset in a release.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

var (
	client           = &http.Client{Timeout: 15 * time.Second}
	releasesCacheTTL = 5 * time.Minute
)

// ResolveVersion determines which release to use based on the alias.
func ResolveVersion(version, cachePath string) (Release, error) {
	switch version {
	case "stable":
		return FindLatestStable(cachePath)
	case "nightly":
		return FindLatestNightly(cachePath)
	default:
		return FindSpecificVersion(version, cachePath)
	}
}

func GetCachedReleases(force bool, cachePath string) ([]Release, error) {
	if !force {
		if info, err := os.Stat(cachePath); err == nil {
			if time.Since(info.ModTime()) < releasesCacheTTL {
				data, err := os.ReadFile(cachePath)
				if err == nil {
					var releases []Release
					if err = json.Unmarshal(data, &releases); err == nil {
						logrus.Debug("Using cached releases")
						return releases, nil
					}
				}
			}
		}
	}
	logrus.Info("Fetching fresh releases from GitHub")
	releases, err := GetReleases()
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(releases)
	if err == nil {
		os.WriteFile(cachePath, data, 0644)
	}
	return releases, nil
}

func GetReleases() ([]Release, error) {
	req, err := http.NewRequestWithContext(context.Background(), "GET", "https://api.github.com/repos/neovim/neovim/releases", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "nvs")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 403 {
		body, _ := io.ReadAll(resp.Body)
		if strings.Contains(string(body), "rate limit") {
			return nil, fmt.Errorf("GitHub API rate limit exceeded. Please try again later")
		}
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return FilterReleases(releases, "0.5.0")
}

func NormalizeVersion(version string) string {
	if version == "stable" || version == "nightly" {
		return version
	}
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

func FindLatestStable(cachePath string) (Release, error) {
	releases, err := GetCachedReleases(false, cachePath)
	if err != nil {
		return Release{}, err
	}
	for _, r := range releases {
		if !r.Prerelease {
			return r, nil
		}
	}
	return Release{}, fmt.Errorf("no stable release found")
}

func FindLatestNightly(cachePath string) (Release, error) {
	releases, err := GetCachedReleases(false, cachePath)
	if err != nil {
		return Release{}, err
	}
	for _, r := range releases {
		if r.Prerelease {
			return r, nil
		}
	}
	return Release{}, fmt.Errorf("no nightly release found")
}

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
	return Release{}, fmt.Errorf("version %s not found", version)
}

func GetAssetURL(release Release) (string, string, error) {
	var patterns []string
	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			patterns = []string{"linux-x86_64.tar.gz"}
		case "arm64":
			patterns = []string{"linux-arm64.tar.gz"}
		default:
			return "", "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			patterns = []string{"macos-arm64.tar.gz"}
		} else {
			patterns = []string{"macos-x86_64.tar.gz"}
		}
	case "windows":
		patterns = []string{"win64.zip"}
	default:
		return "", "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	for _, asset := range release.Assets {
		for _, pattern := range patterns {
			if strings.Contains(asset.Name, pattern) {
				return asset.BrowserDownloadURL, pattern, nil
			}
		}
	}
	return "", "", fmt.Errorf("no matching asset found for %s/%s", runtime.GOOS, runtime.GOARCH)
}

func GetInstalledReleaseIdentifier(versionsDir string, alias string) (string, error) {
	versionFile := filepath.Join(versionsDir, alias, "version.txt")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func GetChecksumURL(release Release, assetPattern string) (string, error) {
	checksumPattern := assetPattern + ".sha256"
	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, checksumPattern) {
			return asset.BrowserDownloadURL, nil
		}
	}
	return "", nil
}

func GetReleaseIdentifier(release Release, alias string) string {
	if alias == "nightly" {
		if strings.HasPrefix(release.TagName, "nightly-") {
			return strings.TrimPrefix(release.TagName, "nightly-")
		}
		return release.CommitHash[:10]
	}
	return release.TagName
}

func FilterReleases(releases []Release, minVersion string) ([]Release, error) {
	constraints, err := semver.NewConstraint(">=" + minVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid version constraint: %w", err)
	}

	var filtered []Release
	for _, r := range releases {
		// Keep "stable" and "nightly" tags
		if r.TagName == "stable" || r.TagName == "nightly" {
			filtered = append(filtered, r)
			continue
		}

		// Normalize version: remove 'v' prefix if present
		versionStr := strings.TrimPrefix(r.TagName, "v")

		v, err := semver.NewVersion(versionStr)
		if err != nil {
			fmt.Printf("Skipping invalid version: %s\n", r.TagName)
			continue
		}

		if constraints.Check(v) {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}
