package releases

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

var sampleReleases = []Release{
	{
		TagName:    "v0.5.1",
		Prerelease: false,
		Assets: []Asset{
			{
				Name:               "linux-x86_64.tar.gz",
				BrowserDownloadURL: "http://example.com/linux-x86_64.tar.gz",
			},
			{
				Name:               "linux-x86_64.tar.gz.sha256",
				BrowserDownloadURL: "http://example.com/linux-x86_64.tar.gz.sha256",
			},
		},
		PublishedAt: "2025-03-07T00:00:00Z",
		CommitHash:  "abcdef1234567890",
	},
	{
		TagName:    "v0.5.2",
		Prerelease: true,
		Assets: []Asset{
			{
				Name:               "linux-x86_64.tar.gz",
				BrowserDownloadURL: "http://example.com/linux-x86_64-nightly.tar.gz",
			},
			{
				Name:               "linux-x86_64.tar.gz.sha256",
				BrowserDownloadURL: "http://example.com/linux-x86_64-nightly.tar.gz.sha256",
			},
		},
		PublishedAt: "2025-03-08T00:00:00Z",
		CommitHash:  "123456abcdef7890",
	},
	{
		TagName:    "v0.4.9",
		Prerelease: false,
		Assets: []Asset{
			{
				Name:               "linux-x86_64.tar.gz",
				BrowserDownloadURL: "http://example.com/old-linux-x86_64.tar.gz",
			},
		},
		PublishedAt: "2025-02-01T00:00:00Z",
		CommitHash:  "oldcommit1234",
	},
}

// writeCacheFile writes sample release data to a temporary cache file and
// returns its path. It also updates the file's modification time to now.
func writeCacheFile(t *testing.T, releases []Release) string {
	t.Helper()
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "releases.json")
	data, err := json.Marshal(releases)
	if err != nil {
		t.Fatalf("failed to marshal sample releases: %v", err)
	}
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		t.Fatalf("failed to write cache file: %v", err)
	}
	// Update mod time so the cache is considered fresh.
	if err := os.Chtimes(cachePath, time.Now(), time.Now()); err != nil {
		t.Fatalf("failed to update mod time: %v", err)
	}
	return cachePath
}

func TestNormalizeVersion(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"stable", "stable"},
		{"nightly", "nightly"},
		{"1.2.3", "v1.2.3"},
		{"v1.2.3", "v1.2.3"},
	}
	for _, c := range cases {
		result := NormalizeVersion(c.input)
		if result != c.expected {
			t.Errorf("NormalizeVersion(%q) = %q; want %q", c.input, result, c.expected)
		}
	}
}

func TestFilterReleases(t *testing.T) {
	filtered, err := FilterReleases(sampleReleases, "0.5.0")
	if err != nil {
		t.Fatalf("FilterReleases error: %v", err)
	}
	// Expect to include v0.5.1 and v0.5.2, but not v0.4.9.
	for _, r := range filtered {
		if r.TagName == "v0.4.9" {
			t.Errorf("FilterReleases included release %q which should be filtered out", r.TagName)
		}
	}
}

func TestFindLatestStable(t *testing.T) {
	cachePath := writeCacheFile(t, sampleReleases)
	release, err := FindLatestStable(cachePath)
	if err != nil {
		t.Fatalf("FindLatestStable error: %v", err)
	}
	// The first non-prerelease in sampleReleases is v0.5.1.
	if release.TagName != "v0.5.1" {
		t.Errorf("FindLatestStable = %q; want %q", release.TagName, "v0.5.1")
	}
}

func TestFindLatestNightly(t *testing.T) {
	cachePath := writeCacheFile(t, sampleReleases)
	release, err := FindLatestNightly(cachePath)
	if err != nil {
		t.Fatalf("FindLatestNightly error: %v", err)
	}
	if release.TagName != "v0.5.2" {
		t.Errorf("FindLatestNightly = %q; want %q", release.TagName, "v0.5.2")
	}
}

func TestFindSpecificVersion(t *testing.T) {
	cachePath := writeCacheFile(t, sampleReleases)
	release, err := FindSpecificVersion("v0.5.1", cachePath)
	if err != nil {
		t.Fatalf("FindSpecificVersion error: %v", err)
	}
	if release.TagName != "v0.5.1" {
		t.Errorf("FindSpecificVersion = %q; want %q", release.TagName, "v0.5.1")
	}

	// Test for a version that doesn't exist.
	_, err = FindSpecificVersion("v1.0.0", cachePath)
	if err == nil {
		t.Errorf("FindSpecificVersion expected error for non-existent version, got nil")
	}
}

func TestGetAssetURL(t *testing.T) {
	// Create a sample release with an asset that matches the current OS/ARCH.
	var expectedPattern string
	switch runtime.GOOS {
	case "linux":
		if runtime.GOARCH == "amd64" {
			expectedPattern = "linux-x86_64.tar.gz"
		} else if runtime.GOARCH == "arm64" {
			expectedPattern = "linux-arm64.tar.gz"
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			expectedPattern = "macos-arm64.tar.gz"
		} else {
			expectedPattern = "macos-x86_64.tar.gz"
		}
	case "windows":
		expectedPattern = "win64.zip"
	default:
		t.Skip("unsupported OS for this test")
	}
	release := Release{
		TagName:    "v0.5.1",
		Prerelease: false,
		Assets: []Asset{
			{
				Name:               expectedPattern,
				BrowserDownloadURL: "http://example.com/asset",
			},
		},
	}
	url, pattern, err := GetAssetURL(release)
	if err != nil {
		t.Fatalf("GetAssetURL error: %v", err)
	}
	if url != "http://example.com/asset" {
		t.Errorf("GetAssetURL returned url %q; want %q", url, "http://example.com/asset")
	}
	if pattern != expectedPattern {
		t.Errorf("GetAssetURL returned pattern %q; want %q", pattern, expectedPattern)
	}
}

func TestGetChecksumURL(t *testing.T) {
	assetPattern := "linux-x86_64.tar.gz"
	release := Release{
		TagName:    "v0.5.1",
		Prerelease: false,
		Assets: []Asset{
			{
				Name:               "linux-x86_64.tar.gz",
				BrowserDownloadURL: "http://example.com/asset",
			},
			{
				Name:               "linux-x86_64.tar.gz.sha256",
				BrowserDownloadURL: "http://example.com/asset.sha256",
			},
		},
	}
	url, err := GetChecksumURL(release, assetPattern)
	if err != nil {
		t.Fatalf("GetChecksumURL error: %v", err)
	}
	if url != "http://example.com/asset.sha256" {
		t.Errorf("GetChecksumURL returned url %q; want %q", url, "http://example.com/asset.sha256")
	}
}

func TestGetInstalledReleaseIdentifier(t *testing.T) {
	tmpDir := t.TempDir()
	alias := "test-install"
	releaseIdentifier := "v0.5.1"
	versionDir := filepath.Join(tmpDir, alias)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	versionFile := filepath.Join(versionDir, "version.txt")
	if err := os.WriteFile(versionFile, []byte(releaseIdentifier+"\n"), 0644); err != nil {
		t.Fatalf("failed to write version file: %v", err)
	}
	id, err := GetInstalledReleaseIdentifier(tmpDir, alias)
	if err != nil {
		t.Fatalf("GetInstalledReleaseIdentifier error: %v", err)
	}
	if strings.TrimSpace(id) != releaseIdentifier {
		t.Errorf("GetInstalledReleaseIdentifier = %q; want %q", id, releaseIdentifier)
	}
}

func TestGetReleaseIdentifier(t *testing.T) {
	// For a stable release.
	release := Release{
		TagName: "v0.5.1",
	}
	id := GetReleaseIdentifier(release, "stable")
	if id != "v0.5.1" {
		t.Errorf("GetReleaseIdentifier = %q; want %q", id, "v0.5.1")
	}

	// For a nightly release with a tag starting with "nightly-".
	release = Release{
		TagName:    "nightly-2025-03-07",
		CommitHash: "abcdef1234567890",
	}
	id = GetReleaseIdentifier(release, "nightly")
	if id != "2025-03-07" {
		t.Errorf("GetReleaseIdentifier = %q; want %q", id, "2025-03-07")
	}

	// For a nightly release without the "nightly-" prefix.
	release = Release{
		TagName:    "random",
		CommitHash: "abcdef1234567890",
	}
	id = GetReleaseIdentifier(release, "nightly")
	if id != "abcdef1234" { // first 10 characters of commit hash
		t.Errorf("GetReleaseIdentifier = %q; want %q", id, "abcdef1234")
	}
}
