package releases_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/y3owk1n/nvs/pkg/releases"
)

const testVersion = "v1.0.0"

func TestIsCommitHash(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"master", true},
		{"", false},
		{"abc", false},
		{"123456789", false},
		{"abcdef1234567890abcdef1234567890ab", false},
		{"abcdef1234567890abcdef1234567890abc", false},
		{"abcdefg", false},
		{"1234567", true},
		{"abcdef0", true},
		{"ABCDEF0", true},
		{"abcdefg", false},
		{"1234567890123456789012345678901234567890", true},
		{"12345678901234567890123456789012345678901", false},
		{"g234567890123456789012345678901234567890", false},
	}

	for _, test := range tests {
		result := releases.IsCommitHash(test.input)
		if result != test.expected {
			t.Errorf("IsCommitHash(%q) = %v; expected %v", test.input, result, test.expected)
		}
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"v1.0.0", "v1.0.0"},
		{"1.0.0", "v1.0.0"},
		{"stable", "stable"},
		{"nightly", "nightly"},
		{"master", "master"},
		{"abc123", "vabc123"},
	}

	for _, test := range tests {
		result := releases.NormalizeVersion(test.input)
		if result != test.expected {
			t.Errorf("NormalizeVersion(%q) = %q; expected %q", test.input, result, test.expected)
		}
	}
}

func TestGetAssetURL(t *testing.T) {
	release := &releases.Release{
		Assets: []releases.Asset{
			{Name: "linux-x86_64.tar.gz", BrowserDownloadURL: "http://example.com/linux.tar.gz"},
			{
				Name:               "nvim-macos-arm64.tar.gz",
				BrowserDownloadURL: "http://example.com/macos.tar.gz",
			},
			{
				Name:               "win64.zip",
				BrowserDownloadURL: "http://example.com/win64.zip",
			},
		},
	}

	url, _, err := releases.GetAssetURL(*release)
	if err != nil {
		t.Fatalf("GetAssetURL failed: %v", err)
	}

	var expected string

	switch runtime.GOOS {
	case "darwin":
		expected = "http://example.com/macos.tar.gz"
	case "linux":
		expected = "http://example.com/linux.tar.gz"
	case "windows":
		expected = "http://example.com/win64.zip"
	default:
		t.Fatalf("unsupported OS: %s", runtime.GOOS)
	}

	if url != expected {
		t.Errorf("expected %s, got %s", expected, url)
	}
}

func TestGetChecksumURL(t *testing.T) {
	release := &releases.Release{
		Assets: []releases.Asset{
			{Name: "linux-x86_64.tar.gz.sha256", BrowserDownloadURL: "http://example.com/checksum"},
		},
	}

	url, err := releases.GetChecksumURL(*release, "linux-x86_64.tar.gz")
	if err != nil {
		t.Fatalf("GetChecksumURL failed: %v", err)
	}

	expected := "http://example.com/checksum"
	if url != expected {
		t.Errorf("expected %s, got %s", expected, url)
	}
}

func TestGetReleaseIdentifier(t *testing.T) {
	release := &releases.Release{TagName: "v1.0.0"}
	identifier := releases.GetReleaseIdentifier(*release, "")

	expected := testVersion
	if identifier != expected {
		t.Errorf("expected %s, got %s", expected, identifier)
	}
}

func TestFilterReleases(t *testing.T) {
	relPtrs := []*releases.Release{
		{TagName: "v1.0.0", Prerelease: false},
		{TagName: "v1.0.0-rc1", Prerelease: true},
	}

	rels := make([]releases.Release, len(relPtrs))
	for i, r := range relPtrs {
		rels[i] = *r
	}

	filtered, err := releases.FilterReleases(rels, "0.5.0")
	if err != nil {
		t.Fatalf("FilterReleases failed: %v", err)
	}

	if len(filtered) != 1 || filtered[0].TagName != "v1.0.0" {
		t.Errorf("expected stable release, got %v", filtered)
	}
}

func TestGetAssetURL_Unsupported(t *testing.T) {
	release := &releases.Release{}

	url, _, _ := releases.GetAssetURL(*release)
	if url != "" {
		t.Errorf("expected empty URL for unsupported asset, got %s", url)
	}
}

func TestHTTPClientTimeout(t *testing.T) {
	if releases.Client.Timeout < 15*time.Second {
		t.Errorf("expected timeout >= 15s, got %v", releases.Client.Timeout)
	}
}
