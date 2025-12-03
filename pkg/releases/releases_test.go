package releases_test

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

	rel "github.com/y3owk1n/nvs/pkg/releases"
)

const (
	HTTPSchemeScheme = "http"
	Arm64            = "arm64"
)

// Helper: create a fake releases slice.
func fakeReleases() []rel.Release {
	return []rel.Release{
		{
			TagName:    "v0.5.1",
			Prerelease: false,
			Assets: []rel.Asset{
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
			Assets: []rel.Asset{
				{
					Name:               "macos-x86_64.tar.gz",
					BrowserDownloadURL: "http://example.com/macos.tar.gz",
				},
			},
			PublishedAt: "2023-02-01T00:00:00Z",
			CommitHash:  "123456abcdef7890",
		},
		{
			TagName:    "stable",
			Prerelease: false,
			Assets: []rel.Asset{
				{
					Name:               "stable-linux-x86_64.tar.gz",
					BrowserDownloadURL: "http://example.com/stable-linux.tar.gz",
				},
				{
					Name:               "stable-macos-arm64.tar.gz",
					BrowserDownloadURL: "http://example.com/stable-macos-arm64.tar.gz",
				},
			},
			PublishedAt: "2023-03-01T00:00:00Z",
			CommitHash:  "stablecommit12345",
		},
		{
			TagName:    "nightly",
			Prerelease: false,
			Assets: []rel.Asset{
				{
					Name:               "nightly-linux-x86_64.tar.gz",
					BrowserDownloadURL: "http://example.com/nightly-linux.tar.gz",
				},
			},
			PublishedAt: "2023-03-02T00:00:00Z",
			CommitHash:  "nightlycommit12345",
		},
		{
			TagName:    "v0.9.0",
			Prerelease: false,
			Assets: []rel.Asset{
				{
					Name:               "v0.9.0-linux-x86_64.tar.gz",
					BrowserDownloadURL: "http://example.com/v0.9.0-linux.tar.gz",
				},
			},
			PublishedAt: "2023-03-03T00:00:00Z",
			CommitHash:  "v090commit12345",
		},
		{
			TagName:    "nightly-20230315",
			Prerelease: true,
			Assets: []rel.Asset{
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
			Assets:      []rel.Asset{},
			PublishedAt: "2023-03-16T00:00:00Z",
			CommitHash:  "invalidhash",
		},
	}
}

// TestResolveVersion tests the rel.ResolveVersion function for stable, nightly, and specific version.
func TestResolveVersion(t *testing.T) {
	// Create a temporary cache file with fake rel.
	cacheFile := createTempCache(t, fakeReleases())
	defer func() {
		err := os.Remove(cacheFile)
		if err != nil && !os.IsNotExist(err) {
			t.Errorf("Failed to remove %s: %v", cacheFile, err)
		}
	}()

	// Test stable
	stable, err := rel.ResolveVersion("stable", cacheFile)
	if err != nil {
		t.Fatalf("rel.ResolveVersion(stable) failed: %v", err)
	}

	t.Logf("stable: %+v", stable)

	// Test nightly
	nightly, err := rel.ResolveVersion("nightly", cacheFile)
	if err != nil {
		t.Fatalf("rel.ResolveVersion(nightly) failed: %v", err)
	}

	t.Logf("nightly: %+v", nightly)

	// Test specific version
	specific, err := rel.ResolveVersion("v0.9.0", cacheFile)
	if err != nil {
		t.Fatalf("rel.ResolveVersion(v0.9.0) failed: %v", err)
	}

	t.Logf("specific: %+v", specific)
}

// TestGetCachedReleases_CacheHit tests that cached releases are used when TTL is not expired.
func TestGetCachedReleases_CacheHit(t *testing.T) {
	cacheFile := createTempCache(t, fakeReleases())
	defer func() {
		err := os.Remove(cacheFile)
		if err != nil && !os.IsNotExist(err) {
			t.Errorf("Failed to remove %s: %v", cacheFile, err)
		}
	}()

	// Set modTime to now.
	err := os.Chtimes(cacheFile, time.Now(), time.Now())
	if err != nil {
		t.Errorf("failed to chtimes: %v", err)
	}

	rels, err := rel.GetCachedReleases(false, cacheFile)
	if err != nil {
		t.Fatalf("rel.GetCachedReleases failed: %v", err)
	}

	if len(rels) != len(fakeReleases()) {
		t.Errorf("expected %d releases, got %d", len(fakeReleases()), len(rels))
	}
}

// TestGetCachedReleases_ForceRefresh tests that a forced refresh fetches fresh rel.
func TestGetCachedReleases_ForceRefresh(t *testing.T) {
	// Create a temporary cache file with dummy data.
	cacheFile := createTempCache(t, []rel.Release{})
	defer func() {
		err := os.Remove(cacheFile)
		if err != nil && !os.IsNotExist(err) {
			t.Errorf("failed to remove %s: %v", cacheFile, err)
		}
	}()

	// Create a test server that returns fake rel.
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)

		err := enc.Encode(fakeReleases())
		if err != nil {
			panic(err)
		}
	}))
	defer testServer.Close()

	// Override the HTTP rel.Client's Transport so that any requests are redirected to our test server.
	origTransport := rel.Client.Transport

	rel.Client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = HTTPSchemeScheme
		req.URL.Host = testServer.Listener.Addr().String()

		return http.DefaultTransport.RoundTrip(req)
	})
	defer func() { rel.Client.Transport = origTransport }()

	rels, err := rel.GetCachedReleases(true, cacheFile)
	if err != nil {
		t.Fatalf("rel.GetCachedReleases(force) failed: %v", err)
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

// TestGetReleases tests rel.GetReleases for successful decoding and filtering.
func TestGetReleases(t *testing.T) {
	// Create a test server that returns fake releases in JSON.
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)

		err := enc.Encode(fakeReleases())
		if err != nil {
			panic(err)
		}
	}))
	defer testServer.Close()

	// Override rel.Client's Transport.
	origTransport := rel.Client.Transport

	rel.Client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = HTTPSchemeScheme
		req.URL.Host = testServer.Listener.Addr().String()

		return http.DefaultTransport.RoundTrip(req)
	})
	defer func() { rel.Client.Transport = origTransport }()

	rels, err := rel.GetReleases()
	if err != nil {
		t.Fatalf("rel.GetReleases failed: %v", err)
	}
	// fakeReleases has 7 items but 2 are invalid (nightly-20230315 and invalid)
	expected := 5
	if len(rels) != expected {
		t.Errorf("expected %d releases after filtering, got %d", expected, len(rels))
	}
}

// TestGetReleases_RateLimit tests that a 403 response with rate limit message is handled.
func TestGetReleases_RateLimit(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)

		_, err := fmt.Fprintln(w, "API rate limit exceeded")
		if err != nil {
			panic(err)
		}
	}))
	defer testServer.Close()

	origTransport := rel.Client.Transport

	rel.Client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = HTTPSchemeScheme
		req.URL.Host = testServer.Listener.Addr().String()

		return http.DefaultTransport.RoundTrip(req)
	})
	defer func() { rel.Client.Transport = origTransport }()

	_, err := rel.GetReleases()
	if err == nil || !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("expected rate limit error, got: %v", err)
	}
}

// TestGetReleases_Non200 tests that non-200 response (other than 403) is handled.
func TestGetReleases_Non200(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer testServer.Close()

	origTransport := rel.Client.Transport

	rel.Client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = HTTPSchemeScheme
		req.URL.Host = testServer.Listener.Addr().String()

		return http.DefaultTransport.RoundTrip(req)
	})
	defer func() { rel.Client.Transport = origTransport }()

	_, err := rel.GetReleases()
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
		{
			"0123456789abcdef0123456789abcdef0123456g",
			false,
			"40-digit invalid hash, 'g' is not hex",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := rel.IsCommitHash(tc.input)
			if result != tc.expected {
				t.Errorf("rel.IsCommitHash(%q) = %v; want %v", tc.input, result, tc.expected)
			}
		})
	}
}

// TestNormalizeVersion tests rel.NormalizeVersion behavior.
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
		got := rel.NormalizeVersion(tc.in)
		if got != tc.want {
			t.Errorf("rel.NormalizeVersion(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestFindLatestStable tests rel.FindLatestStable behavior.
func TestFindLatestStable(t *testing.T) {
	cacheFile := createTempCache(t, fakeReleases())
	defer func() {
		err := os.Remove(cacheFile)
		if err != nil && !os.IsNotExist(err) {
			t.Errorf("failed to remove %s: %v", cacheFile, err)
		}
	}()

	release, err := rel.FindLatestStable(cacheFile)
	if err != nil {
		t.Fatalf("rel.FindLatestStable failed: %v", err)
	}

	if release.Prerelease {
		t.Errorf("expected a stable release, got prerelease")
	}
}

// TestFindLatestNightly tests rel.FindLatestNightly behavior.
func TestFindLatestNightly(t *testing.T) {
	cacheFile := createTempCache(t, fakeReleases())
	defer func() {
		err := os.Remove(cacheFile)
		if err != nil && !os.IsNotExist(err) {
			t.Errorf("failed to remove %s: %v", cacheFile, err)
		}
	}()

	release, err := rel.FindLatestNightly(cacheFile)
	if err != nil {
		t.Fatalf("rel.FindLatestNightly failed: %v", err)
	}

	if !release.Prerelease {
		t.Errorf("expected a nightly release, got stable")
	}
}

// TestFindSpecificVersion tests rel.FindSpecificVersion behavior.
func TestFindSpecificVersion(t *testing.T) {
	cacheFile := createTempCache(t, fakeReleases())
	defer func() {
		err := os.Remove(cacheFile)
		if err != nil && !os.IsNotExist(err) {
			t.Errorf("failed to remove %s: %v", cacheFile, err)
		}
	}()

	release, err := rel.FindSpecificVersion("v0.5.1", cacheFile)
	if err != nil {
		t.Fatalf("rel.FindSpecificVersion failed: %v", err)
	}

	if release.TagName != "v0.5.1" {
		t.Errorf("expected tag v0.5.1, got %s", release.TagName)
	}
}

// TestGetAssetURL tests rel.GetAssetURL against current runtime values.
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

	release := rel.Release{
		TagName:    "v1.2.3",
		Prerelease: false,
		Assets: []rel.Asset{
			{
				Name:               assetName,
				BrowserDownloadURL: "http://example.com/download",
			},
		},
	}

	url, pattern, err := rel.GetAssetURL(release)
	if err != nil {
		t.Fatalf("rel.GetAssetURL failed: %v", err)
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

	err := os.MkdirAll(filepath.Dir(versionFile), 0o755)
	if err != nil {
		t.Fatalf("failed to mkdirall: %v", err)
	}

	err = os.WriteFile(versionFile, []byte(versionContent), 0o644)
	if err != nil {
		t.Fatalf("failed to write version file: %v", err)
	}

	got, err := rel.GetInstalledReleaseIdentifier(dir, alias)
	if err != nil {
		t.Fatalf("rel.GetInstalledReleaseIdentifier failed: %v", err)
	}

	if got != versionContent {
		t.Errorf("expected %q, got %q", versionContent, got)
	}
}

// TestGetChecksumURL tests retrieval of the checksum URL.
func TestGetChecksumURL(t *testing.T) {
	release := rel.Release{
		TagName:    "v1.2.3",
		Prerelease: false,
		Assets: []rel.Asset{
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

	url, err := rel.GetChecksumURL(release, "linux-x86_64.tar.gz")
	if err != nil {
		t.Fatalf("rel.GetChecksumURL failed: %v", err)
	}

	if url != "http://example.com/download.sha256" {
		t.Errorf("expected checksum URL %q, got %q", "http://example.com/download.sha256", url)
	}

	// Test when no checksum asset exists.
	url, err = rel.GetChecksumURL(release, "nonexistent")
	if err != nil {
		t.Fatalf("rel.GetChecksumURL failed: %v", err)
	}

	if url != "" {
		t.Errorf("expected empty checksum URL, got %q", url)
	}
}

// TestGetReleaseIdentifier tests rel.GetReleaseIdentifier behavior for nightly and others.
func TestGetReleaseIdentifier(t *testing.T) {
	nightly := rel.Release{
		TagName:    "nightly-20230401",
		CommitHash: "1234567890abcdef",
	}
	if id := rel.GetReleaseIdentifier(nightly, "nightly"); id != "20230401" {
		t.Errorf("expected nightly identifier '20230401', got %q", id)
	}

	stable := rel.Release{
		TagName:    "v1.2.3",
		CommitHash: "abcdef1234567890",
	}
	if id := rel.GetReleaseIdentifier(stable, "stable"); id != "v1.2.3" {
		t.Errorf("expected release identifier 'v1.2.3', got %q", id)
	}
}

// TestFilterReleases tests filtering based on a minimum semantic version.
func TestFilterReleases(t *testing.T) {
	// Create releases with various versions.
	releases := []rel.Release{
		{TagName: "v0.4.0", Prerelease: false},
		{TagName: "v0.5.0", Prerelease: false},
		{TagName: "v0.6.0", Prerelease: false},
		{TagName: "stable", Prerelease: false},
		{TagName: "nightly", Prerelease: true},
		{TagName: "vinvalid", Prerelease: false},
	}

	filtered, err := rel.FilterReleases(releases, "0.5.0")
	if err != nil {
		t.Fatalf("rel.FilterReleases failed: %v", err)
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

// TestGetAssetURL_Unsupported tests that rel.GetAssetURL returns an error for unsupported OS/ARCH.
func TestGetAssetURL_Unsupported(t *testing.T) {
	// Simulate an asset that won't match any pattern for the current OS/ARCH.
	release := rel.Release{
		TagName:    "v1.0.0",
		Prerelease: false,
		Assets: []rel.Asset{
			{
				Name:               "unsupported.asset",
				BrowserDownloadURL: "http://example.com/unsupported",
			},
		},
	}

	_, _, err := rel.GetAssetURL(release)
	if err == nil {
		t.Errorf("expected error for unsupported asset, got none")
	}
}

// TestHTTPClientTimeout verifies that the package-level rel.Client timeout is set.
func TestHTTPClientTimeout(t *testing.T) {
	if rel.Client.Timeout < 15*time.Second {
		t.Errorf("expected rel.Client timeout >= 15 seconds, got %v", rel.Client.Timeout)
	}
}

// TestCacheWriteFailure simulates a failure when writing to the cache file.
func TestCacheWriteFailure(t *testing.T) {
	// Create a temporary file and remove write permissions.
	tmpFile, err := os.CreateTemp(t.TempDir(), "cache-write-failure-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	cachePath := tmpFile.Name()

	err = tmpFile.Close()
	if err != nil {
		t.Errorf("failed to close tmpFile: %v", err)
	}

	// Remove write permission.
	err = os.Chmod(cachePath, 0o444)
	if err != nil {
		t.Errorf("failed to chmod: %v", err)
	}

	defer func() {
		err := os.Remove(cachePath)
		if err != nil && !os.IsNotExist(err) {
			t.Errorf("failed to remove %s: %v", cachePath, err)
		}
	}()

	// Create a test server that returns fake rel.
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)

		err := enc.Encode(fakeReleases())
		if err != nil {
			panic(err)
		}
	}))
	defer testServer.Close()

	origTransport := rel.Client.Transport

	rel.Client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = HTTPSchemeScheme
		req.URL.Host = testServer.Listener.Addr().String()

		return http.DefaultTransport.RoundTrip(req)
	})
	defer func() { rel.Client.Transport = origTransport }()

	// Call the function which should trigger the fatal.
	_, err = rel.GetCachedReleases(true, cachePath)
	if err == nil {
		t.Errorf("expected error on cache write failure, but got none")
	} else {
		t.Logf("expected error occurred: %v", err)
	}
}

// createTempCache creates a temporary cache file with the given releases encoded as JSON.
func createTempCache(t *testing.T, releases []rel.Release) string {
	t.Helper()

	tmpFile, err := os.CreateTemp(t.TempDir(), "cache-*.json")
	if err != nil {
		t.Fatalf("failed to create temp cache file: %v", err)
	}

	enc := json.NewEncoder(tmpFile)

	err = enc.Encode(releases)
	if err != nil {
		t.Fatalf("failed to encode releases: %v", err)
	}

	err = tmpFile.Close()
	if err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	return tmpFile.Name()
}
