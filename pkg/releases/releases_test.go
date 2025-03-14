package releases

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Helper: create a fake releases slice.
func fakeReleases() []Release {
	return []Release{
		{
			TagName:    "v0.5.1",
			Prerelease: false,
			Assets: []Asset{
				{
					Name:               "linux-x86_64.tar.gz",
					BrowserDownloadURL: "http://example.com/linux.tar.gz",
				},
				{
					Name:               "linux-x86_64.tar.gz.sha256",
					BrowserDownloadURL: "http://example.com/linux.tar.gz.sha256",
				},
			},
			PublishedAt: "2023-01-01T00:00:00Z",
			CommitHash:  "abcdef1234567890",
		},
		{
			TagName:    "v0.6.0",
			Prerelease: false,
			Assets: []Asset{
				{
					Name:               "macos-x86_64.tar.gz",
					BrowserDownloadURL: "http://example.com/macos.tar.gz",
				},
			},
			PublishedAt: "2023-02-01T00:00:00Z",
			CommitHash:  "123456abcdef7890",
		},
		{
			TagName:    "nightly-20230315",
			Prerelease: true,
			Assets: []Asset{
				{
					Name:               "win64.zip",
					BrowserDownloadURL: "http://example.com/win64.zip",
				},
			},
			PublishedAt: "2023-03-15T00:00:00Z",
			CommitHash:  "deadbeefdeadbeef",
		},
		{
			TagName:     "invalid",
			Prerelease:  false,
			Assets:      []Asset{},
			PublishedAt: "2023-03-16T00:00:00Z",
			CommitHash:  "invalidhash",
		},
	}
}

// TestResolveVersion tests the ResolveVersion function for stable, nightly, and specific version.
func TestResolveVersion(t *testing.T) {
	// Create a temporary cache file with fake releases.
	cacheFile := createTempCache(t, fakeReleases())
	defer os.Remove(cacheFile)

	// Test stable.
	stable, err := ResolveVersion("stable", cacheFile)
	if err != nil {
		t.Fatalf("ResolveVersion(stable) failed: %v", err)
	}
	if stable.Prerelease {
		t.Errorf("expected stable release, got prerelease")
	}

	// Test nightly.
	nightly, err := ResolveVersion("nightly", cacheFile)
	if err != nil {
		t.Fatalf("ResolveVersion(nightly) failed: %v", err)
	}
	if !nightly.Prerelease {
		t.Errorf("expected nightly release, got stable")
	}

	// Test specific version.
	specific, err := ResolveVersion("v0.6.0", cacheFile)
	if err != nil {
		t.Fatalf("ResolveVersion(specific) failed: %v", err)
	}
	if specific.TagName != "v0.6.0" {
		t.Errorf("expected v0.6.0, got %s", specific.TagName)
	}

	// Test not found.
	_, err = ResolveVersion("v9.9.9", cacheFile)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error for non-existent version, got %v", err)
	}
}

// createTempCache serializes the given releases to a temporary cache file.
func createTempCache(t *testing.T, releases []Release) string {
	t.Helper()
	data, err := json.Marshal(releases)
	if err != nil {
		t.Fatalf("failed to marshal fake releases: %v", err)
	}
	tmpFile, err := os.CreateTemp("", "releases-cache-*.json")
	if err != nil {
		t.Fatalf("failed to create temp cache file: %v", err)
	}
	if _, err := tmpFile.Write(data); err != nil {
		t.Fatalf("failed to write to cache file: %v", err)
	}
	tmpFile.Close()
	return tmpFile.Name()
}

// TestGetCachedReleases_CacheHit tests that cached releases are used when TTL is not expired.
func TestGetCachedReleases_CacheHit(t *testing.T) {
	cacheFile := createTempCache(t, fakeReleases())
	defer os.Remove(cacheFile)

	// Set modTime to now.
	os.Chtimes(cacheFile, time.Now(), time.Now())

	rels, err := GetCachedReleases(false, cacheFile)
	if err != nil {
		t.Fatalf("GetCachedReleases failed: %v", err)
	}
	if len(rels) != len(fakeReleases()) {
		t.Errorf("expected %d releases, got %d", len(fakeReleases()), len(rels))
	}
}

// TestGetCachedReleases_ForceRefresh tests that a forced refresh fetches fresh releases.
func TestGetCachedReleases_ForceRefresh(t *testing.T) {
	// Create a temporary cache file with dummy data.
	cacheFile := createTempCache(t, []Release{})
	defer os.Remove(cacheFile)

	// Create a test server that returns fake releases.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		enc.Encode(fakeReleases())
	}))
	defer ts.Close()

	// Override the HTTP client's Transport so that any requests are redirected to our test server.
	origTransport := client.Transport
	client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = ts.Listener.Addr().String()
		return http.DefaultTransport.RoundTrip(req)
	})
	defer func() { client.Transport = origTransport }()

	rels, err := GetCachedReleases(true, cacheFile)
	if err != nil {
		t.Fatalf("GetCachedReleases(force) failed: %v", err)
	}
	if len(rels) == 0 {
		t.Errorf("expected non-empty releases after forced refresh")
	}
}

// roundTripperFunc type is a helper to override the Transport.
type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// TestGetReleases tests GetReleases for successful decoding and filtering.
func TestGetReleases(t *testing.T) {
	// Create a test server that returns fake releases in JSON.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		enc.Encode(fakeReleases())
	}))
	defer ts.Close()

	// Override client's Transport.
	origTransport := client.Transport
	client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = ts.Listener.Addr().String()
		return http.DefaultTransport.RoundTrip(req)
	})
	defer func() { client.Transport = origTransport }()

	rels, err := GetReleases()
	if err != nil {
		t.Fatalf("GetReleases failed: %v", err)
	}
	// fakeReleases has 4 items but one is invalid version "invalid" (skipped by FilterReleases)
	expected := 2
	if len(rels) != expected {
		t.Errorf("expected %d releases after filtering, got %d", expected, len(rels))
	}
}

// TestGetReleases_RateLimit tests that a 403 response with rate limit message is handled.
func TestGetReleases_RateLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		fmt.Fprintln(w, "API rate limit exceeded")
	}))
	defer ts.Close()

	origTransport := client.Transport
	client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = ts.Listener.Addr().String()
		return http.DefaultTransport.RoundTrip(req)
	})
	defer func() { client.Transport = origTransport }()

	_, err := GetReleases()
	if err == nil || !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("expected rate limit error, got: %v", err)
	}
}

// TestGetReleases_Non200 tests that non-200 response (other than 403) is handled.
func TestGetReleases_Non200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer ts.Close()

	origTransport := client.Transport
	client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = ts.Listener.Addr().String()
		return http.DefaultTransport.RoundTrip(req)
	})
	defer func() { client.Transport = origTransport }()

	_, err := GetReleases()
	if err == nil || !strings.Contains(err.Error(), "API returned status") {
		t.Errorf("expected API status error, got: %v", err)
	}
}

func TestIsCommitHash(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
		name     string
	}{
		// Special case: "master" should always return true.
		{"master", true, "master keyword"},

		// Invalid lengths (not 7 or 40)
		{"", false, "empty string"},
		{"abc", false, "too short"},
		{"12345678", false, "8 chars, invalid length"},
		{"123456789", false, "9 chars, invalid length"},

		// 7-character valid commit hash (all valid hex characters)
		{"abcdef0", true, "7-digit valid hash lowercase"},
		{"ABCDEF0", true, "7-digit valid hash uppercase"},
		{"a1B2c3D", true, "7-digit valid mix"},

		// 7-character invalid commit hash (contains invalid character)
		{"abcdeg0", false, "7-digit invalid hash, 'g' is not hex"},

		// 40-character valid commit hash
		{"0123456789abcdef0123456789abcdef01234567", true, "40-digit valid hash"},
		// 40-character valid commit hash with uppercase letters
		{"0123456789ABCDEF0123456789ABCDEF01234567", true, "40-digit valid hash uppercase"},

		// 40-character invalid commit hash (contains an invalid character)
		{"0123456789abcdef0123456789abcdef0123456g", false, "40-digit invalid hash, 'g' is not hex"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsCommitHash(tc.input)
			if result != tc.expected {
				t.Errorf("IsCommitHash(%q) = %v; want %v", tc.input, result, tc.expected)
			}
		})
	}
}

// TestNormalizeVersion tests NormalizeVersion behavior.
func TestNormalizeVersion(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"stable", "stable"},
		{"nightly", "nightly"},
		{"1.0.0", "v1.0.0"},
		{"v2.0.0", "v2.0.0"},
	}
	for _, tc := range cases {
		got := NormalizeVersion(tc.in)
		if got != tc.want {
			t.Errorf("NormalizeVersion(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestFindLatestStable tests FindLatestStable behavior.
func TestFindLatestStable(t *testing.T) {
	cacheFile := createTempCache(t, fakeReleases())
	defer os.Remove(cacheFile)
	release, err := FindLatestStable(cacheFile)
	if err != nil {
		t.Fatalf("FindLatestStable failed: %v", err)
	}
	if release.Prerelease {
		t.Errorf("expected a stable release, got prerelease")
	}
}

// TestFindLatestNightly tests FindLatestNightly behavior.
func TestFindLatestNightly(t *testing.T) {
	cacheFile := createTempCache(t, fakeReleases())
	defer os.Remove(cacheFile)
	release, err := FindLatestNightly(cacheFile)
	if err != nil {
		t.Fatalf("FindLatestNightly failed: %v", err)
	}
	if !release.Prerelease {
		t.Errorf("expected a nightly release, got stable")
	}
}

// TestFindSpecificVersion tests FindSpecificVersion behavior.
func TestFindSpecificVersion(t *testing.T) {
	cacheFile := createTempCache(t, fakeReleases())
	defer os.Remove(cacheFile)
	release, err := FindSpecificVersion("v0.5.1", cacheFile)
	if err != nil {
		t.Fatalf("FindSpecificVersion failed: %v", err)
	}
	if release.TagName != "v0.5.1" {
		t.Errorf("expected tag v0.5.1, got %s", release.TagName)
	}
}

// TestGetAssetURL tests GetAssetURL against current runtime values.
// It uses a release with assets that include expected substrings.
func TestGetAssetURL(t *testing.T) {
	var assetName, expectedPattern string
	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			assetName = "linux-x86_64.tar.gz"
			expectedPattern = "linux-x86_64.tar.gz"
		case "arm64":
			assetName = "linux-arm64.tar.gz"
			expectedPattern = "linux-arm64.tar.gz"
		default:
			t.Skipf("unsupported architecture %s for testing", runtime.GOARCH)
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			assetName = "macos-arm64.tar.gz"
			expectedPattern = "macos-arm64.tar.gz"
		} else {
			assetName = "macos-x86_64.tar.gz"
			expectedPattern = "macos-x86_64.tar.gz"
		}
	case "windows":
		assetName = "win64.zip"
		expectedPattern = "win64.zip"
	default:
		t.Skipf("unsupported OS %s for testing", runtime.GOOS)
	}

	release := Release{
		TagName:    "v1.2.3",
		Prerelease: false,
		Assets: []Asset{
			{
				Name:               assetName,
				BrowserDownloadURL: "http://example.com/download",
			},
		},
	}
	url, pattern, err := GetAssetURL(release)
	if err != nil {
		t.Fatalf("GetAssetURL failed: %v", err)
	}
	if url != "http://example.com/download" {
		t.Errorf("expected download URL to be %q, got %q", "http://example.com/download", url)
	}
	if pattern != expectedPattern {
		t.Errorf("expected pattern %q, got %q", expectedPattern, pattern)
	}
}

// TestGetInstalledReleaseIdentifier tests reading the version file.
func TestGetInstalledReleaseIdentifier(t *testing.T) {
	dir := t.TempDir()
	alias := "testalias"
	versionContent := "v1.2.3"
	versionFile := filepath.Join(dir, alias, "version.txt")
	os.MkdirAll(filepath.Dir(versionFile), 0755)
	if err := os.WriteFile(versionFile, []byte(versionContent), 0644); err != nil {
		t.Fatalf("failed to write version file: %v", err)
	}

	got, err := GetInstalledReleaseIdentifier(dir, alias)
	if err != nil {
		t.Fatalf("GetInstalledReleaseIdentifier failed: %v", err)
	}
	if got != versionContent {
		t.Errorf("expected %q, got %q", versionContent, got)
	}
}

// TestGetChecksumURL tests retrieval of the checksum URL.
func TestGetChecksumURL(t *testing.T) {
	release := Release{
		TagName:    "v1.2.3",
		Prerelease: false,
		Assets: []Asset{
			{
				Name:               "linux-x86_64.tar.gz",
				BrowserDownloadURL: "http://example.com/download",
			},
			{
				Name:               "linux-x86_64.tar.gz.sha256",
				BrowserDownloadURL: "http://example.com/download.sha256",
			},
		},
	}
	url, err := GetChecksumURL(release, "linux-x86_64.tar.gz")
	if err != nil {
		t.Fatalf("GetChecksumURL failed: %v", err)
	}
	if url != "http://example.com/download.sha256" {
		t.Errorf("expected checksum URL %q, got %q", "http://example.com/download.sha256", url)
	}

	// Test when no checksum asset exists.
	url, err = GetChecksumURL(release, "nonexistent")
	if err != nil {
		t.Fatalf("GetChecksumURL failed: %v", err)
	}
	if url != "" {
		t.Errorf("expected empty checksum URL, got %q", url)
	}
}

// TestGetReleaseIdentifier tests GetReleaseIdentifier behavior for nightly and others.
func TestGetReleaseIdentifier(t *testing.T) {
	nightly := Release{
		TagName:    "nightly-20230401",
		CommitHash: "1234567890abcdef",
	}
	if id := GetReleaseIdentifier(nightly, "nightly"); id != "20230401" {
		t.Errorf("expected nightly identifier '20230401', got %q", id)
	}

	stable := Release{
		TagName:    "v1.2.3",
		CommitHash: "abcdef1234567890",
	}
	if id := GetReleaseIdentifier(stable, "stable"); id != "v1.2.3" {
		t.Errorf("expected release identifier 'v1.2.3', got %q", id)
	}
}

// TestFilterReleases tests filtering based on a minimum semantic version.
func TestFilterReleases(t *testing.T) {
	// Create releases with various versions.
	releases := []Release{
		{TagName: "v0.4.0", Prerelease: false},
		{TagName: "v0.5.0", Prerelease: false},
		{TagName: "v0.6.0", Prerelease: false},
		{TagName: "stable", Prerelease: false},
		{TagName: "nightly", Prerelease: true},
		{TagName: "vinvalid", Prerelease: false},
	}

	filtered, err := FilterReleases(releases, "0.5.0")
	if err != nil {
		t.Fatalf("FilterReleases failed: %v", err)
	}

	// Expect to keep releases that are "stable", "nightly", and versions >= 0.5.0.
	// "v0.4.0" and "vinvalid" should be skipped.
	expectedTags := []string{"v0.5.0", "v0.6.0", "stable", "nightly"}
	if len(filtered) != len(expectedTags) {
		t.Errorf("expected %d releases, got %d", len(expectedTags), len(filtered))
	}
	tagSet := make(map[string]bool)
	for _, r := range filtered {
		tagSet[r.TagName] = true
	}
	for _, tag := range expectedTags {
		if !tagSet[tag] {
			t.Errorf("expected tag %q in filtered releases", tag)
		}
	}
}

// TestGetAssetURL_Unsupported tests that GetAssetURL returns an error for unsupported OS/ARCH.
func TestGetAssetURL_Unsupported(t *testing.T) {
	// Simulate an asset that won't match any pattern for the current OS/ARCH.
	release := Release{
		TagName:    "v1.0.0",
		Prerelease: false,
		Assets: []Asset{
			{
				Name:               "unsupported.asset",
				BrowserDownloadURL: "http://example.com/unsupported",
			},
		},
	}
	_, _, err := GetAssetURL(release)
	if err == nil {
		t.Errorf("expected error for unsupported asset, got none")
	}
}

// TestHTTPClientTimeout verifies that the package-level client timeout is set.
func TestHTTPClientTimeout(t *testing.T) {
	if client.Timeout < 15*time.Second {
		t.Errorf("expected client timeout >= 15 seconds, got %v", client.Timeout)
	}
}

// TestCacheWriteFailure simulates a failure when writing to the cache file.
func TestCacheWriteFailure(t *testing.T) {
	// Create a temporary file and remove write permissions.
	tmpFile, err := os.CreateTemp("", "cache-write-failure-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	cachePath := tmpFile.Name()
	tmpFile.Close()
	// Remove write permission.
	os.Chmod(cachePath, 0444)
	defer os.Remove(cachePath)

	// Create a test server that returns fake releases.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		enc.Encode(fakeReleases())
	}))
	defer ts.Close()

	origTransport := client.Transport
	client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = ts.Listener.Addr().String()
		return http.DefaultTransport.RoundTrip(req)
	})
	defer func() { client.Transport = origTransport }()

	// Call the function which should trigger the fatal.
	_, err = GetCachedReleases(true, cachePath)
	if err == nil {
		t.Errorf("expected error on cache write failure, but got none")
	} else {
		t.Logf("expected error occurred: %v", err)
	}
}
