package github_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/y3owk1n/nvs/internal/domain/release"
	"github.com/y3owk1n/nvs/internal/infra/github"
)

func TestApplyMirrorToURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		mirrorURL string
		want      string
	}{
		{
			name:      "no mirror",
			url:       "https://github.com/neovim/neovim/releases/download/v0.10.0/nvim.tar.gz",
			mirrorURL: "",
			want:      "https://github.com/neovim/neovim/releases/download/v0.10.0/nvim.tar.gz",
		},
		{
			name:      "with mirror",
			url:       "https://github.com/neovim/neovim/releases/download/v0.10.0/nvim.tar.gz",
			mirrorURL: "https://mirror.example.com",
			want:      "https://mirror.example.com/neovim/neovim/releases/download/v0.10.0/nvim.tar.gz",
		},
		{
			name:      "non-github url unchanged",
			url:       "https://example.com/file.tar.gz",
			mirrorURL: "https://mirror.example.com",
			want:      "https://example.com/file.tar.gz",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := github.ApplyMirrorToURL(testCase.url, testCase.mirrorURL)
			if got != testCase.want {
				t.Errorf("ApplyMirrorToURL() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestGetAssetURL(t *testing.T) {
	// Create releases with different asset patterns
	tests := []struct {
		name       string
		assets     []release.Asset
		wantErr    bool
		errContain string
		wantURL    string
	}{
		{
			name: "linux amd64 asset found",
			assets: []release.Asset{
				release.NewAsset(
					"nvim-linux-x86_64.tar.gz",
					"https://example.com/linux.tar.gz",
					1000,
				),
			},
			wantErr: runtime.GOOS != "linux" || runtime.GOARCH != "amd64",
			wantURL: "https://example.com/linux.tar.gz",
		},
		{
			name: "macos asset found",
			assets: []release.Asset{
				release.NewAsset("nvim-macos.tar.gz", "https://example.com/macos.tar.gz", 1000),
			},
			wantErr: runtime.GOOS != "darwin",
			wantURL: "https://example.com/macos.tar.gz",
		},
		{
			name: "windows asset found",
			assets: []release.Asset{
				release.NewAsset("nvim-win64.zip", "https://example.com/win.zip", 1000),
			},
			wantErr: runtime.GOOS != "windows",
			wantURL: "https://example.com/win.zip",
		},
		{
			name:       "no matching asset",
			assets:     []release.Asset{},
			wantErr:    true,
			errContain: "no matching asset",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			rel := release.New("v0.10.0", false, "abc123", time.Now(), testCase.assets)
			url, _, err := github.GetAssetURL(rel)

			if testCase.wantErr {
				if err == nil {
					t.Errorf("GetAssetURL() expected error, got nil")
				} else if testCase.errContain != "" && !strings.Contains(err.Error(), testCase.errContain) {
					t.Errorf("GetAssetURL() error = %q, want to contain %q", err.Error(), testCase.errContain)
				}
			} else {
				if err != nil {
					t.Errorf("GetAssetURL() expected no error, got %v", err)
				}

				if url == "" {
					t.Errorf("GetAssetURL() expected URL, got empty")
				}

				if testCase.wantURL != "" && url != testCase.wantURL {
					t.Errorf("GetAssetURL() = %q, want %q", url, testCase.wantURL)
				}
			}
			// Note: Platform-specific asset matching depends on runtime.GOOS/GOARCH
		})
	}
}

// writeCacheFile is a helper to create a cache file with test data.
func writeCacheFile(t *testing.T, cacheData []map[string]any) string {
	t.Helper()

	tempDir := t.TempDir()
	cacheFile := filepath.Join(tempDir, "cache.json")

	data, err := json.Marshal(cacheData)
	if err != nil {
		t.Fatalf("Failed to marshal cache data: %v", err)
	}

	err = os.WriteFile(cacheFile, data, 0o644)
	if err != nil {
		t.Fatalf("Failed to write cache file: %v", err)
	}

	return cacheFile
}

func TestGetChecksumURL(t *testing.T) {
	tests := []struct {
		name         string
		assets       []release.Asset
		assetPattern string
		wantURL      string
		wantErr      bool
	}{
		{
			name: "sha256 file found",
			assets: []release.Asset{
				release.NewAsset(
					"nvim-linux-x86_64.tar.gz",
					"https://example.com/linux.tar.gz",
					1000,
				),
				release.NewAsset(
					"nvim-linux-x86_64.tar.gz.sha256",
					"https://example.com/linux.tar.gz.sha256",
					64,
				),
			},
			assetPattern: "linux-x86_64.tar.gz",
			wantURL:      "https://example.com/linux.tar.gz.sha256",
			wantErr:      false,
		},
		{
			name: "shasum.txt fallback",
			assets: []release.Asset{
				release.NewAsset(
					"nvim-linux-x86_64.tar.gz",
					"https://example.com/linux.tar.gz",
					1000,
				),
				release.NewAsset("shasum.txt", "https://example.com/shasum.txt", 256),
			},
			assetPattern: "linux-x86_64.tar.gz",
			wantURL:      "https://example.com/shasum.txt",
			wantErr:      false,
		},
		{
			name: "no checksum found",
			assets: []release.Asset{
				release.NewAsset(
					"nvim-linux-x86_64.tar.gz",
					"https://example.com/linux.tar.gz",
					1000,
				),
			},
			assetPattern: "linux-x86_64.tar.gz",
			wantErr:      true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			rel := release.New("v0.10.0", false, "abc123", time.Now(), testCase.assets)
			got, err := github.GetChecksumURL(rel, testCase.assetPattern)

			if testCase.wantErr {
				if err == nil {
					t.Errorf("GetChecksumURL() expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Errorf("GetChecksumURL() unexpected error: %v", err)

				return
			}

			if got != testCase.wantURL {
				t.Errorf("GetChecksumURL() = %v, want %v", got, testCase.wantURL)
			}
		})
	}
}

// TestClient_FindStable tests finding the latest stable release.
// Uses pre-populated cache file in temp directory - no network requests.
func TestClient_FindStable(t *testing.T) {
	// Pre-populate cache with test data
	cacheData := []map[string]any{
		{
			"tag_name":         "v0.10.0",
			"prerelease":       false,
			"target_commitish": "abc123",
			"published_at":     "2024-12-01T10:00:00Z",
			"assets": []map[string]any{
				{
					"name":                 "nvim-linux-x86_64.tar.gz",
					"browser_download_url": "https://example.com/nvim.tar.gz",
					"size":                 1000000,
				},
			},
		},
		{
			"tag_name":         "nightly",
			"prerelease":       true,
			"target_commitish": "def456",
			"published_at":     "2024-12-02T10:00:00Z",
			"assets":           []map[string]any{},
		},
	}

	cacheFile := writeCacheFile(t, cacheData)
	client := github.NewClient(cacheFile, time.Hour, "", "", false)
	ctx := context.Background()

	stable, err := client.FindStable(ctx)
	if err != nil {
		t.Fatalf("FindStable() error = %v", err)
	}

	if stable.TagName() != "v0.10.0" {
		t.Errorf("FindStable() TagName = %v, want v0.10.0", stable.TagName())
	}

	if stable.Prerelease() {
		t.Error("FindStable() returned a prerelease")
	}
}

// TestClient_FindNightly tests finding the latest nightly release.
// Uses pre-populated cache file in temp directory - no network requests.
func TestClient_FindNightly(t *testing.T) {
	// Pre-populate cache with test data
	cacheData := []map[string]any{
		{
			"tag_name":         "v0.10.0",
			"prerelease":       false,
			"target_commitish": "abc123",
			"published_at":     "2024-12-01T10:00:00Z",
			"assets":           []map[string]any{},
		},
		{
			"tag_name":         "nightly",
			"prerelease":       true,
			"target_commitish": "def456",
			"published_at":     "2024-12-02T10:00:00Z",
			"assets":           []map[string]any{},
		},
	}

	cacheFile := writeCacheFile(t, cacheData)
	client := github.NewClient(cacheFile, time.Hour, "", "", false)
	ctx := context.Background()

	nightly, err := client.FindNightly(ctx)
	if err != nil {
		t.Fatalf("FindNightly() error = %v", err)
	}

	if nightly.TagName() != "nightly" {
		t.Errorf("FindNightly() TagName = %v, want nightly", nightly.TagName())
	}

	if !nightly.Prerelease() {
		t.Error("FindNightly() returned a non-prerelease")
	}
}

// TestClient_FindByTag tests finding a specific release by tag.
// Uses pre-populated cache file in temp directory - no network requests.
func TestClient_FindByTag(t *testing.T) {
	// Pre-populate cache with test data
	cacheData := []map[string]any{
		{
			"tag_name":         "v0.10.0",
			"prerelease":       false,
			"target_commitish": "abc123",
			"published_at":     "2024-12-01T10:00:00Z",
			"assets":           []map[string]any{},
		},
		{
			"tag_name":         "v0.9.0",
			"prerelease":       false,
			"target_commitish": "xyz789",
			"published_at":     "2024-11-01T10:00:00Z",
			"assets":           []map[string]any{},
		},
	}

	cacheFile := writeCacheFile(t, cacheData)
	client := github.NewClient(cacheFile, time.Hour, "", "", false)
	ctx := context.Background()

	tests := []struct {
		name    string
		tag     string
		wantTag string
		wantErr bool
	}{
		{"existing tag", "v0.10.0", "v0.10.0", false},
		{"older tag", "v0.9.0", "v0.9.0", false},
		{"non-existent tag", "v0.8.0", "", true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			rel, err := client.FindByTag(ctx, testCase.tag)

			if testCase.wantErr {
				if err == nil {
					t.Errorf("FindByTag(%s) expected error, got nil", testCase.tag)
				}

				return
			}

			if err != nil {
				t.Errorf("FindByTag(%s) unexpected error: %v", testCase.tag, err)

				return
			}

			if rel.TagName() != testCase.wantTag {
				t.Errorf(
					"FindByTag(%s) TagName = %v, want %v",
					testCase.tag,
					rel.TagName(),
					testCase.wantTag,
				)
			}
		})
	}
}

// TestClient_MirrorURL tests the mirror URL functionality.
func TestClient_MirrorURL(t *testing.T) {
	tempDir := t.TempDir()
	cacheFile := filepath.Join(tempDir, "cache.json")

	tests := []struct {
		name      string
		mirrorURL string
		want      string
	}{
		{"no mirror", "", ""},
		{"with mirror", "https://mirror.example.com", "https://mirror.example.com"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			client := github.NewClient(cacheFile, time.Hour, "", testCase.mirrorURL, false)

			if got := client.MirrorURL(); got != testCase.want {
				t.Errorf("MirrorURL() = %v, want %v", got, testCase.want)
			}
		})
	}
}

// TestClient_ApplyMirror tests the instance method for applying mirror.
func TestClient_ApplyMirror(t *testing.T) {
	tempDir := t.TempDir()
	cacheFile := filepath.Join(tempDir, "cache.json")

	client := github.NewClient(cacheFile, time.Hour, "", "https://mirror.example.com", false)

	url := "https://github.com/neovim/neovim/releases/download/v0.10.0/nvim.tar.gz"
	want := "https://mirror.example.com/neovim/neovim/releases/download/v0.10.0/nvim.tar.gz"

	if got := client.ApplyMirror(url); got != want {
		t.Errorf("ApplyMirror() = %v, want %v", got, want)
	}
}
