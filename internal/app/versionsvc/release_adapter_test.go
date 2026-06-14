//nolint:testpackage // internal test: releaseAdapter and AssetResolveCount are unexported
package versionsvc

import (
	"runtime"
	"testing"
	"time"

	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/domain/release"
)

// TestReleaseAdapter_AssetCacheResolvesOnce pins the in-memory asset
// resolution cache. Each install pipeline (Install, Upgrade) calls
// GetAssetURL and GetChecksumURL back-to-back. The previous
// implementation re-invoked github.GetAssetURL inside
// GetChecksumURL just to recover the asset pattern, scanning the
// release's asset list twice per install.
//
// With the cache, both methods share a single underlying call: the
// second entry into resolveAsset must return the cached lookup
// without re-running github.GetAssetURL. AssetResolveCount is the
// only reliable way to assert that from outside the package.
func TestReleaseAdapter_AssetCacheResolvesOnce(t *testing.T) {
	t.Parallel()

	rel := release.New(
		"v0.10.0",
		false,
		"abc123",
		time.Now(),
		[]release.Asset{
			release.NewAsset(
				platformAssetName(),
				"https://example.com/release.tar.gz",
				1024,
			),
		},
	)

	adapter := &releaseAdapter{
		Release:   rel,
		mirrorURL: "",
	}

	// First call (GetAssetURL) must enter resolveAsset.
	assetURL, err := adapter.GetAssetURL()
	if err != nil {
		t.Fatalf("GetAssetURL failed: %v", err)
	}

	if assetURL == "" {
		t.Fatalf("GetAssetURL returned empty URL")
	}

	if count := adapter.AssetResolveCount(); count != 1 {
		t.Fatalf(
			"AssetResolveCount after first GetAssetURL = %d, want 1",
			count,
		)
	}

	// Second call (GetChecksumURL) MUST reuse the cached asset
	// lookup. If the cache were missing, this would push the count
	// to 2. The test release has no checksum asset, so
	// GetChecksumURL will return an error from
	// github.GetChecksumURL; that is fine - the only assertion is
	// on the resolve count.
	_, _ = adapter.GetChecksumURL()

	if count := adapter.AssetResolveCount(); count != 1 {
		t.Errorf(
			"AssetResolveCount after GetChecksumURL = %d, want 1 (cache should short-circuit the second call)",
			count,
		)
	}
}

// TestReleaseAdapter_AssetCacheRepeatedCalls verifies that the cache
// also short-circuits repeated GetAssetURL calls. This guards
// against a future refactor that accidentally re-resolves on every
// call.
func TestReleaseAdapter_AssetCacheRepeatedCalls(t *testing.T) {
	t.Parallel()

	rel := release.New(
		"v0.10.0",
		false,
		"abc123",
		time.Now(),
		[]release.Asset{
			release.NewAsset(
				platformAssetName(),
				"https://example.com/release.tar.gz",
				1024,
			),
		},
	)

	adapter := &releaseAdapter{
		Release:   rel,
		mirrorURL: "",
	}

	const callCount = 5

	for idx := range callCount {
		_, err := adapter.GetAssetURL()
		if err != nil {
			t.Fatalf("GetAssetURL call %d failed: %v", idx, err)
		}
	}

	if count := adapter.AssetResolveCount(); count != 1 {
		t.Errorf(
			"AssetResolveCount after %d GetAssetURL calls = %d, want 1",
			callCount,
			count,
		)
	}
}

// platformAssetName returns an asset name that matches the
// current platform's URL pattern in github.GetAssetURL. The test
// only requires a match for the local OS/arch; the actual URL is
// never downloaded.
func platformAssetName() string {
	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return "nvim-linux-x86_64.tar.gz"
		case constants.Arm64Arch:
			return "nvim-linux-arm64.tar.gz"
		}
	case "darwin":
		if runtime.GOARCH == constants.Arm64Arch {
			return "nvim-macos-arm64.tar.gz"
		}

		return "nvim-macos-x86_64.tar.gz"
	case "windows":
		if runtime.GOARCH == constants.Arm64Arch {
			return "nvim-win-arm64.zip"
		}

		return "nvim-win64.zip"
	}

	return "nvim-unknown.tar.gz"
}
