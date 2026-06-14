package github_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/y3owk1n/nvs/internal/domain/release"
	"github.com/y3owk1n/nvs/internal/infra/github"
)

func TestCache_GetSet(t *testing.T) {
	tempDir := t.TempDir()
	cacheFile := filepath.Join(tempDir, "cache.json")

	cache := github.NewCache(cacheFile, time.Hour)

	// Create test releases
	releases := []release.Release{
		release.New("v0.10.0", false, "abc123", time.Now(), []release.Asset{
			release.NewAsset("nvim-linux.tar.gz", "https://example.com/nvim.tar.gz", 1000),
		}),
		release.New("nightly", true, "def456", time.Now(), []release.Asset{}),
	}

	// Test Set
	err := cache.Set(releases)
	if err != nil {
		t.Fatalf("Cache.Set() error = %v", err)
	}

	// Verify file was created
	_, statErr := os.Stat(cacheFile)
	if os.IsNotExist(statErr) {
		t.Fatal("Cache file was not created")
	}

	// Test Get
	got, err := cache.Get()
	if err != nil {
		t.Fatalf("Cache.Get() error = %v", err)
	}

	if len(got) != len(releases) {
		t.Errorf("Cache.Get() returned %d releases, want %d", len(got), len(releases))
	}

	// Verify release data
	if got[0].TagName() != "v0.10.0" {
		t.Errorf("First release TagName = %v, want v0.10.0", got[0].TagName())
	}

	if got[1].TagName() != "nightly" {
		t.Errorf("Second release TagName = %v, want nightly", got[1].TagName())
	}
}

func TestCache_Get_NonExistent(t *testing.T) {
	tempDir := t.TempDir()
	cacheFile := filepath.Join(tempDir, "nonexistent.json")

	cache := github.NewCache(cacheFile, time.Hour)

	_, err := cache.Get()
	if err == nil {
		t.Error("Cache.Get() expected error for non-existent file, got nil")
	}
}

func TestCache_Get_Stale(t *testing.T) {
	tempDir := t.TempDir()
	cacheFile := filepath.Join(tempDir, "cache.json")

	// Create cache with very short TTL
	cache := github.NewCache(cacheFile, time.Nanosecond)

	releases := []release.Release{
		release.New("v0.10.0", false, "abc123", time.Now(), []release.Asset{}),
	}

	err := cache.Set(releases)
	if err != nil {
		t.Fatalf("Cache.Set() error = %v", err)
	}

	// Wait for cache to become stale
	time.Sleep(time.Millisecond)

	_, err = cache.Get()
	if err == nil {
		t.Error("Cache.Get() expected stale error, got nil")
	}
}

func TestCache_Get_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	cacheFile := filepath.Join(tempDir, "cache.json")

	// Write invalid JSON
	err := os.WriteFile(cacheFile, []byte("invalid json"), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cache := github.NewCache(cacheFile, time.Hour)

	_, err = cache.Get()
	if err == nil {
		t.Error("Cache.Get() expected error for invalid JSON, got nil")
	}
}

func TestCache_Set_AtomicWrite(t *testing.T) {
	tempDir := t.TempDir()
	cacheFile := filepath.Join(tempDir, "cache.json")

	cache := github.NewCache(cacheFile, time.Hour)

	releases := []release.Release{
		release.New("v0.10.0", false, "abc123", time.Now(), []release.Asset{}),
	}

	err := cache.Set(releases)
	if err != nil {
		t.Fatalf("Cache.Set() error = %v", err)
	}

	// Verify temp file is cleaned up
	tempFile := cacheFile + ".tmp"

	_, statErr := os.Stat(tempFile)
	if !os.IsNotExist(statErr) {
		t.Error("Temp file was not cleaned up after Set()")
	}
}

func TestCache_Set_PreservesAssets(t *testing.T) {
	tempDir := t.TempDir()
	cacheFile := filepath.Join(tempDir, "cache.json")

	cache := github.NewCache(cacheFile, time.Hour)

	assets := []release.Asset{
		release.NewAsset("nvim-linux.tar.gz", "https://example.com/linux.tar.gz", 1000),
		release.NewAsset("nvim-macos.tar.gz", "https://example.com/macos.tar.gz", 2000),
		release.NewAsset("shasum.txt", "https://example.com/shasum.txt", 100),
	}

	releases := []release.Release{
		release.New("v0.10.0", false, "abc123", time.Now(), assets),
	}

	err := cache.Set(releases)
	if err != nil {
		t.Fatalf("Cache.Set() error = %v", err)
	}

	got, err := cache.Get()
	if err != nil {
		t.Fatalf("Cache.Get() error = %v", err)
	}

	gotAssets := got[0].Assets()
	if len(gotAssets) != len(assets) {
		t.Errorf("Got %d assets, want %d", len(gotAssets), len(assets))
	}

	// Verify asset data is preserved
	for idx, asset := range gotAssets {
		if asset.Name() != assets[idx].Name() {
			t.Errorf("Asset[%d].Name() = %v, want %v", idx, asset.Name(), assets[idx].Name())
		}

		if asset.DownloadURL() != assets[idx].DownloadURL() {
			t.Errorf(
				"Asset[%d].DownloadURL() = %v, want %v",
				idx,
				asset.DownloadURL(),
				assets[idx].DownloadURL(),
			)
		}

		if asset.Size() != assets[idx].Size() {
			t.Errorf("Asset[%d].Size() = %v, want %v", idx, asset.Size(), assets[idx].Size())
		}
	}
}

func TestNewCache(t *testing.T) {
	tempDir := t.TempDir()
	cacheFile := filepath.Join(tempDir, "cache.json")
	ttl := time.Hour * 2

	cache := github.NewCache(cacheFile, ttl)

	if cache == nil {
		t.Error("NewCache() returned nil")
	}
}

func TestCache_Get_CorruptedCacheDeleted(t *testing.T) {
	tempDir := t.TempDir()
	cacheFile := filepath.Join(tempDir, "cache.json")

	err := os.WriteFile(cacheFile, []byte("not valid json {{{"), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cache := github.NewCache(cacheFile, time.Hour)

	_, err = cache.Get()
	if err == nil {
		t.Fatal("Cache.Get() expected error for corrupted JSON, got nil")
	}

	_, statErr := os.Stat(cacheFile)
	if !os.IsNotExist(statErr) {
		t.Errorf("Corrupted cache file was not removed (stat err = %v)", statErr)
	}
}

func TestCache_Set_EmptyIsNoop(t *testing.T) {
	tempDir := t.TempDir()
	cacheFile := filepath.Join(tempDir, "cache.json")

	const seedTag = "v0.11.0"

	cache := github.NewCache(cacheFile, time.Hour)

	original := []release.Release{
		release.New(seedTag, false, "abc123", time.Now(), []release.Asset{}),
	}

	err := cache.Set(original)
	if err != nil {
		t.Fatalf("Seed Cache.Set() error = %v", err)
	}

	// An empty Set must not clobber the valid on-disk cache.
	err = cache.Set(nil)
	if err != nil {
		t.Fatalf("Cache.Set(nil) error = %v", err)
	}

	got, err := cache.Get()
	if err != nil {
		t.Fatalf("Cache.Get() error = %v", err)
	}

	if len(got) != 1 || got[0].TagName() != seedTag {
		t.Errorf("Empty Set clobbered cache; got %d releases, want 1 (tag %q)", len(got), seedTag)
	}
}

func TestCache_Set_ConcurrentDoesNotCorrupt(t *testing.T) {
	tempDir := t.TempDir()
	cacheFile := filepath.Join(tempDir, "cache.json")

	cache := github.NewCache(cacheFile, time.Hour)

	const goroutines = 8

	var waitGroup sync.WaitGroup

	waitGroup.Add(goroutines)

	for idx := range goroutines {
		go func(idx int) {
			defer waitGroup.Done()

			batch := make([]release.Release, 0, 10)
			for range 10 {
				batch = append(batch, release.New(
					"v0.0.0",
					false,
					"abc",
					time.Now(),
					[]release.Asset{},
				))
			}

			setErr := cache.Set(batch)
			if setErr != nil {
				t.Errorf("goroutine %d: Cache.Set() error = %v", idx, setErr)
			}
		}(idx)
	}

	waitGroup.Wait()

	_, statErr := os.Stat(cacheFile + ".tmp")
	if !os.IsNotExist(statErr) {
		t.Errorf("Temp file was not cleaned up (err = %v)", statErr)
	}

	got, err := cache.Get()
	if err != nil {
		t.Fatalf("Cache.Get() after concurrent Set error = %v", err)
	}

	if len(got) != 10 {
		t.Errorf("Got %d releases, want 10", len(got))
	}
}
