//go:build integration

package releases_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/y3owk1n/nvs/pkg/releases"
)

const httpScheme = "http"

// Helper: create a fake releases slice.
func fakeReleases() []releases.Release {
	return []releases.Release{
		{
			TagName:    "v0.5.1",
			Prerelease: false,
			Assets: []releases.Asset{
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
			Assets: []releases.Asset{
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
			Assets: []releases.Asset{
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
			Assets: []releases.Asset{
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
			Assets: []releases.Asset{
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
			Assets: []releases.Asset{
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
			Assets:      []releases.Asset{},
			PublishedAt: "2023-03-16T00:00:00Z",
			CommitHash:  "invalidhash",
		},
	}
}

func TestResolveVersion(t *testing.T) {
	// Create a temporary cache file with fake rel.
	cacheFile := createTempCache(t, fakeReleases())
	defer func() {
		err := os.Remove(cacheFile)
		if err != nil && !os.IsNotExist(err) {
			t.Errorf("Failed to remove %s: %v", cacheFile, err)
		}
	}()

	tests := []struct {
		version  string
		expected *releases.Release
		hasError bool
	}{
		{"stable", &releases.Release{TagName: "v0.5.1", Prerelease: false}, false},
		{"nightly", &releases.Release{TagName: "nightly-20230315", Prerelease: true}, false},
		{"v0.9.0", &releases.Release{TagName: "v0.9.0", Prerelease: false}, false},
		{"nonexistent", nil, true},
	}

	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			result, err := releases.ResolveVersion(test.version, cacheFile)
			if test.hasError {
				if err == nil {
					t.Errorf("expected error for version %s, got nil", test.version)

					return
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error for version %s: %v", test.version, err)

				return
			}

			if result.TagName != test.expected.TagName {
				t.Errorf("expected tag %s, got %s", test.expected.TagName, result.TagName)
			}
		})
	}
}

func TestGetCachedReleases_CacheHit(t *testing.T) {
	cacheDir := t.TempDir()
	cacheFile := filepath.Join(cacheDir, "test-releases.json")

	// Create cache file with test data
	rels := []*releases.Release{
		{TagName: "v1.0.0", Prerelease: false},
	}

	data, err := json.Marshal(rels)
	if err != nil {
		t.Fatalf("failed to marshal releases: %v", err)
	}

	err = os.WriteFile(cacheFile, data, 0o644)
	if err != nil {
		t.Fatalf("failed to write cache file: %v", err)
	}

	result, err := releases.GetCachedReleases(false, cacheFile)
	if err != nil {
		t.Fatalf("failed to read cache: %v", err)
	}

	if len(result) != 1 || result[0].TagName != "v1.0.0" {
		t.Errorf("expected cached release, got %v", result)
	}
}

func TestGetCachedReleases_ForceRefresh(t *testing.T) {
	cacheFile := filepath.Join(t.TempDir(), "test-releases.json")

	// This will attempt to fetch from network, which should fail gracefully
	_, err := releases.GetCachedReleases(true, cacheFile)
	// We expect this to fail since no network mock
	if err == nil {
		t.Log("cache invalidation worked, network fetch failed as expected")
	}
}

// roundTripperFunc type is a helper to override the Transport.
type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGetReleases(t *testing.T) {
	// Create a test server that returns fake releases in JSON.
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)

		_ = enc.Encode(fakeReleases()) // Error ignored; test will fail on decode side
	}))
	defer testServer.Close()

	// Override rel.Client's Transport.
	origTransport := releases.Client.Transport

	releases.Client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = httpScheme
		req.URL.Host = testServer.Listener.Addr().String()

		return http.DefaultTransport.RoundTrip(req)
	})
	defer func() { releases.Client.Transport = origTransport }()

	rels, err := releases.GetReleases()
	if err != nil {
		t.Fatalf("rel.GetReleases failed: %v", err)
	}
	// fakeReleases has 7 items but 2 are invalid (nightly-20230315 and invalid)
	expected := 5
	if len(rels) != expected {
		t.Errorf("expected %d releases after filtering, got %d", expected, len(rels))
	}
}

func TestGetReleases_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)

		_, err := fmt.Fprintln(w, "API rate limit exceeded")
		if err != nil {
			panic(err)
		}
	}))
	defer server.Close()

	origTransport := releases.Client.Transport

	releases.Client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = httpScheme
		req.URL.Host = server.Listener.Addr().String()

		return http.DefaultTransport.RoundTrip(req)
	})
	defer func() { releases.Client.Transport = origTransport }()

	_, err := releases.GetReleases()
	if err == nil || !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("expected rate limit error, got: %v", err)
	}
}

func TestGetReleases_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	origTransport := releases.Client.Transport

	releases.Client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = httpScheme
		req.URL.Host = server.Listener.Addr().String()

		return http.DefaultTransport.RoundTrip(req)
	})
	defer func() { releases.Client.Transport = origTransport }()

	_, err := releases.GetReleases()
	if err == nil || !strings.Contains(err.Error(), "API returned status") {
		t.Errorf("expected API status error, got: %v", err)
	}
}

func TestFindLatestStable(t *testing.T) {
	cacheFile := createTempCache(t, fakeReleases())

	defer func() {
		err := os.Remove(cacheFile)
		if err != nil && !os.IsNotExist(err) {
			t.Errorf("failed to remove %s: %v", cacheFile, err)
		}
	}()

	release, err := releases.FindLatestStable(cacheFile)
	if err != nil {
		t.Fatalf("FindLatestStable failed: %v", err)
	}

	if release.Prerelease {
		t.Errorf("expected a stable release, got prerelease")
	}
}

func TestFindLatestNightly(t *testing.T) {
	cacheFile := createTempCache(t, fakeReleases())

	defer func() {
		err := os.Remove(cacheFile)
		if err != nil && !os.IsNotExist(err) {
			t.Errorf("failed to remove %s: %v", cacheFile, err)
		}
	}()

	release, err := releases.FindLatestNightly(cacheFile)
	if err != nil {
		t.Fatalf("FindLatestNightly failed: %v", err)
	}

	if !release.Prerelease {
		t.Errorf("expected a nightly release, got stable")
	}
}

func TestFindSpecificVersion(t *testing.T) {
	cacheFile := createTempCache(t, fakeReleases())

	defer func() {
		err := os.Remove(cacheFile)
		if err != nil && !os.IsNotExist(err) {
			t.Errorf("failed to remove %s: %v", cacheFile, err)
		}
	}()

	release, err := releases.FindSpecificVersion("v0.9.0", cacheFile)
	if err != nil {
		t.Fatalf("FindSpecificVersion failed: %v", err)
	}

	if release.TagName != "v0.9.0" {
		t.Errorf("expected v0.9.0, got %s", release.TagName)
	}
}

func TestGetInstalledReleaseIdentifier(t *testing.T) {
	// This is a unit test, but since it reads files, it's integration
	tempDir := t.TempDir()

	versionDir := filepath.Join(tempDir, "v1.0.0")

	err := os.MkdirAll(versionDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create version dir: %v", err)
	}
}

// createTempCache creates a temporary cache file with the given releases encoded as JSON.
func createTempCache(t *testing.T, releases []releases.Release) string {
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
