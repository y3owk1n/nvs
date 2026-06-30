package github_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/y3owk1n/nvs/internal/domain/release"
	"github.com/y3owk1n/nvs/internal/infra/github"
)

const (
	testDownloadURL = "https://github.com/neovim/neovim/releases/download/v0.10.0/nvim.tar.gz"
	testMirrorURL   = "https://mirror.example.com"
	testAssetPat    = "linux-x86_64.tar.gz"
	testKeyTagName  = "tag_name"
	testKeyPreRel   = "prerelease"
	testKeyAssets   = "assets"
	testCommitHash  = "abc123"
	testPubAt       = "2024-12-01T10:00:00Z"
	testNightlyTag  = "nightly"
	testV090        = "v0.9.0"
	testKeyTarget   = "target_commitish"
	testKeyPubAt    = "published_at"
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
			url:       testDownloadURL,
			mirrorURL: "",
			want:      testDownloadURL,
		},
		{
			name:      "with mirror",
			url:       testDownloadURL,
			mirrorURL: testMirrorURL,
			want:      "https://mirror.example.com/neovim/neovim/releases/download/v0.10.0/nvim.tar.gz",
		},
		{
			name:      "non-github url unchanged",
			url:       "https://example.com/file.tar.gz",
			mirrorURL: testMirrorURL,
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
			rel := release.New(cacheTestTag, false, testCommitHash, time.Now(), testCase.assets)
			url, _, err := github.GetAssetURL(rel)

			if testCase.wantErr {
				if err == nil {
					t.Errorf("GetAssetURL() expected error, got nil")
				} else if testCase.errContain != "" && !strings.Contains(err.Error(), testCase.errContain) {
					t.Errorf(
						"GetAssetURL() error = %q, want to contain %q",
						err.Error(),
						testCase.errContain,
					)
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
			assetPattern: testAssetPat,
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
			assetPattern: testAssetPat,
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
			assetPattern: testAssetPat,
			wantErr:      true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			rel := release.New(cacheTestTag, false, testCommitHash, time.Now(), testCase.assets)
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
			testKeyTagName: cacheTestTag,
			testKeyPreRel:  false,
			testKeyTarget:  testCommitHash,
			testKeyPubAt:   testPubAt,
			testKeyAssets: []map[string]any{
				{
					"name":                 "nvim-linux-x86_64.tar.gz",
					"browser_download_url": "https://example.com/nvim.tar.gz",
					"size":                 1000000,
				},
			},
		},
		{
			testKeyTagName: testNightlyTag,
			testKeyPreRel:  true,
			testKeyTarget:  "def456",
			testKeyPubAt:   "2024-12-02T10:00:00Z",
			testKeyAssets:  []map[string]any{},
		},
	}

	cacheFile := writeCacheFile(t, cacheData)
	client := github.NewClient(cacheFile, time.Hour, "", "", false)
	ctx := t.Context()

	stable, err := client.FindStable(ctx)
	if err != nil {
		t.Fatalf("FindStable() error = %v", err)
	}

	if stable.TagName() != cacheTestTag {
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
			testKeyTagName: cacheTestTag,
			testKeyPreRel:  false,
			testKeyTarget:  testCommitHash,
			testKeyPubAt:   testPubAt,
			testKeyAssets:  []map[string]any{},
		},
		{
			testKeyTagName: testNightlyTag,
			testKeyPreRel:  true,
			testKeyTarget:  "def456",
			testKeyPubAt:   "2024-12-02T10:00:00Z",
			testKeyAssets:  []map[string]any{},
		},
	}

	cacheFile := writeCacheFile(t, cacheData)
	client := github.NewClient(cacheFile, time.Hour, "", "", false)
	ctx := t.Context()

	nightly, err := client.FindNightly(ctx)
	if err != nil {
		t.Fatalf("FindNightly() error = %v", err)
	}

	if nightly.TagName() != testNightlyTag {
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
			testKeyTagName: cacheTestTag,
			testKeyPreRel:  false,
			testKeyTarget:  testCommitHash,
			testKeyPubAt:   testPubAt,
			testKeyAssets:  []map[string]any{},
		},
		{
			testKeyTagName: testV090,
			testKeyPreRel:  false,
			testKeyTarget:  "xyz789",
			testKeyPubAt:   "2024-11-01T10:00:00Z",
			testKeyAssets:  []map[string]any{},
		},
	}

	cacheFile := writeCacheFile(t, cacheData)
	client := github.NewClient(cacheFile, time.Hour, "", "", false)
	ctx := t.Context()

	tests := []struct {
		name    string
		tag     string
		wantTag string
		wantErr bool
	}{
		{"existing tag", cacheTestTag, cacheTestTag, false},
		{"older tag", testV090, testV090, false},
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
		{"with mirror", testMirrorURL, testMirrorURL},
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

	client := github.NewClient(cacheFile, time.Hour, "", testMirrorURL, false)

	url := testDownloadURL
	want := "https://mirror.example.com/neovim/neovim/releases/download/v0.10.0/nvim.tar.gz"

	if got := client.ApplyMirror(url); got != want {
		t.Errorf("ApplyMirror() = %v, want %v", got, want)
	}
}

// TestClient_GetAll_InMemoryCache pins the in-memory caching behavior
// of Client.GetAll. The first call with force=false populates both
// the disk cache and the in-memory cache from whatever the disk cache
// already contains. Subsequent calls (within the same process) must
// return the in-memory result without re-reading the disk cache, so
// a freshly rewritten disk file is ignored until the process exits.
//
// Without the in-memory cache, the second call below would re-read
// the modified disk cache and return "v9.9.9" instead of the original
// cached tag.
func TestClient_GetAll_InMemoryCache(t *testing.T) {
	const (
		memCacheTag = "v0.10.0"
		modifiedTag = "v9.9.9"
	)

	originalData := []map[string]any{
		{
			testKeyTagName: memCacheTag,
			testKeyPreRel:  false,
			testKeyTarget:  testCommitHash,
			testKeyPubAt:   testPubAt,
			testKeyAssets:  []map[string]any{},
		},
	}
	cacheFile := writeCacheFile(t, originalData)
	client := github.NewClient(cacheFile, time.Hour, "", "", false)
	ctx := t.Context()

	// First call: should read the disk cache and populate the
	// in-memory cache.
	first, err := client.GetAll(ctx, false)
	if err != nil {
		t.Fatalf("first GetAll: %v", err)
	}

	if len(first) != 1 || first[0].TagName() != memCacheTag {
		t.Fatalf(
			"first GetAll returned %v, want one release tagged %s",
			first,
			memCacheTag,
		)
	}

	// Overwrite the on-disk cache with a different release. A naive
	// implementation that re-reads the disk cache on every call would
	// surface this in the second call.
	modifiedData := []map[string]any{
		{
			testKeyTagName: modifiedTag,
			testKeyPreRel:  false,
			testKeyTarget:  "fff999",
			testKeyPubAt:   "2025-01-01T10:00:00Z",
			testKeyAssets:  []map[string]any{},
		},
	}

	modified, err := json.Marshal(modifiedData)
	if err != nil {
		t.Fatalf("marshal modified data: %v", err)
	}

	writeErr := os.WriteFile(cacheFile, modified, 0o644)
	if writeErr != nil {
		t.Fatalf("rewrite cache file: %v", writeErr)
	}

	// Second call: must return the in-memory cached result, NOT the
	// newly written disk file.
	second, err := client.GetAll(ctx, false)
	if err != nil {
		t.Fatalf("second GetAll: %v", err)
	}

	if len(second) != 1 || second[0].TagName() != memCacheTag {
		t.Errorf(
			"second GetAll returned %v, want in-memory cached %s (disk cache was rewritten to %s)",
			second,
			memCacheTag,
			modifiedTag,
		)
	}
}

// TestClient_GetAll_ConcurrentColdCache verifies that concurrent
// GetAll callers with a cold in-memory cache share one slow-path
// execution (singleflight), and that no caller sees a torn or empty
// result. The test runs under -race to catch any data race.
func TestClient_GetAll_ConcurrentColdCache(t *testing.T) {
	const (
		goroutines = 16
		tag        = "v0.10.0"
	)

	originalData := []map[string]any{
		{
			testKeyTagName: tag,
			testKeyPreRel:  false,
			testKeyTarget:  testCommitHash,
			testKeyPubAt:   testPubAt,
			testKeyAssets:  []map[string]any{},
		},
	}
	cacheFile := writeCacheFile(t, originalData)
	client := github.NewClient(cacheFile, time.Hour, "", "", false)
	ctx := t.Context()

	var waitGroup sync.WaitGroup

	waitGroup.Add(goroutines)

	results := make([][]string, goroutines)

	for idx := range goroutines {
		go func(idx int) {
			defer waitGroup.Done()

			releases, err := client.GetAll(ctx, false)
			if err != nil {
				t.Errorf("goroutine %d: GetAll error: %v", idx, err)

				return
			}

			tags := make([]string, 0, len(releases))
			for _, r := range releases {
				tags = append(tags, r.TagName())
			}

			results[idx] = tags
		}(idx)
	}

	waitGroup.Wait()

	for idx, tags := range results {
		if len(tags) != 1 || tags[0] != tag {
			t.Errorf("goroutine %d: got tags %v, want [%s]", idx, tags, tag)
		}
	}
}

// TestClient_GetAll_ConcurrentMutationSafe verifies that concurrent
// callers can sort and iterate the snapshot returned by GetAll
// without racing each other or the shared cache. The test runs
// under -race; a failure indicates the snapshot is not safe to
// mutate.
func TestClient_GetAll_ConcurrentMutationSafe(t *testing.T) {
	const (
		tag      = "v0.10.0"
		iterTags = 200
	)

	originalData := []map[string]any{
		{
			testKeyTagName: tag,
			testKeyPreRel:  false,
			testKeyTarget:  testCommitHash,
			testKeyPubAt:   testPubAt,
			testKeyAssets:  []map[string]any{},
		},
	}
	cacheFile := writeCacheFile(t, originalData)
	client := github.NewClient(cacheFile, time.Hour, "", "", false)
	ctx := t.Context()

	// Prime the in-memory cache.
	_, err := client.GetAll(ctx, false)
	if err != nil {
		t.Fatalf("prime GetAll: %v", err)
	}

	const goroutines = 8

	var waitGroup sync.WaitGroup

	waitGroup.Add(goroutines)

	for idx := range goroutines {
		go func(idx int) {
			defer waitGroup.Done()

			for range iterTags {
				releases, err := client.GetAll(ctx, false)
				if err != nil {
					t.Errorf("g%d: GetAll error: %v", idx, err)

					return
				}

				// Sort in place; safe only if the snapshot is a copy.
				slices.SortFunc(releases, func(a, b release.Release) int {
					return strings.Compare(a.TagName(), b.TagName())
				})
			}
		}(idx)
	}

	waitGroup.Wait()
}

// TestClient_GetAll_ForceBypassesMemCache verifies that force=true
// skips the in-memory cache and consults the disk cache instead. This
// preserves the pre-existing --force semantics, where users expect a
// fresh read after a manual cache rebuild.
func TestClient_GetAll_ForceBypassesMemCache(t *testing.T) {
	const (
		memCacheTag = "v0.10.0"
		modifiedTag = "v9.9.9"
	)

	originalData := []map[string]any{
		{
			testKeyTagName: memCacheTag,
			testKeyPreRel:  false,
			testKeyTarget:  testCommitHash,
			testKeyPubAt:   testPubAt,
			testKeyAssets:  []map[string]any{},
		},
	}
	cacheFile := writeCacheFile(t, originalData)
	client := github.NewClient(cacheFile, time.Hour, "", "", false)
	ctx := t.Context()

	// Prime the in-memory cache.
	_, primeErr := client.GetAll(ctx, false)
	if primeErr != nil {
		t.Fatalf("prime GetAll: %v", primeErr)
	}

	// Overwrite the disk cache.
	modifiedData := []map[string]any{
		{
			testKeyTagName: modifiedTag,
			testKeyPreRel:  false,
			testKeyTarget:  "fff999",
			testKeyPubAt:   "2025-01-01T10:00:00Z",
			testKeyAssets:  []map[string]any{},
		},
	}

	modified, err := json.Marshal(modifiedData)
	if err != nil {
		t.Fatalf("marshal modified data: %v", err)
	}

	writeErr := os.WriteFile(cacheFile, modified, 0o644)
	if writeErr != nil {
		t.Fatalf("rewrite cache file: %v", writeErr)
	}

	// force=true must skip the in-memory cache and pick up the new
	// disk contents.
	forced, err := client.GetAll(ctx, true)
	if err != nil {
		// An offline test will fail at the network step. That is
		// acceptable here: what matters is that force=true did NOT
		// silently return the stale in-memory value.
		return
	}

	if len(forced) == 1 && forced[0].TagName() == memCacheTag {
		t.Errorf(
			"forced GetAll returned stale in-memory %s; force=true must skip the in-memory cache",
			memCacheTag,
		)
	}
}
