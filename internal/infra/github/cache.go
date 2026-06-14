package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/domain/release"
)

// Cache handles caching of GitHub releases.
//
// Cache is safe for concurrent use. Set serializes the temp-file write
// and rename so two concurrent Set calls cannot corrupt the cache file
// by racing on the shared temp file path.
type Cache struct {
	filePath string
	ttl      time.Duration

	setMu sync.Mutex
}

// NewCache creates a new cache instance.
func NewCache(filePath string, ttl time.Duration) *Cache {
	return &Cache{
		filePath: filePath,
		ttl:      ttl,
	}
}

// Get retrieves releases from cache if valid.
func (c *Cache) Get() ([]release.Release, error) {
	info, err := os.Stat(c.filePath)
	if err != nil {
		return nil, err
	}

	// Check if cache is stale
	if time.Since(info.ModTime()) >= c.ttl {
		return nil, ErrCacheStale
	}

	return c.read()
}

// GetIgnoreStale retrieves releases from cache without checking the
// TTL. Use this as a last-resort fallback when a fresh fetch fails:
// the data may be old, but is better than no data at all.
func (c *Cache) GetIgnoreStale() ([]release.Release, error) {
	_, err := os.Stat(c.filePath)
	if err != nil {
		return nil, err
	}

	return c.read()
}

// Set stores releases in cache.
//
// If releases is empty, Set is a no-op: an empty network response must
// not clobber a valid on-disk cache. Concurrent Set calls are
// serialized to avoid the temp-file race.
func (c *Cache) Set(releases []release.Release) error {
	if len(releases) == 0 {
		logrus.Debug("Skipping cache write for empty release list")

		return nil
	}

	c.setMu.Lock()
	defer c.setMu.Unlock()

	// Convert domain releases to API format for caching
	apiReleases := make([]apiRelease, 0, len(releases))

	for _, rel := range releases {
		apiAssets := make([]apiAsset, 0, len(rel.Assets()))
		for _, a := range rel.Assets() {
			apiAssets = append(apiAssets, apiAsset{
				Name:               a.Name(),
				BrowserDownloadURL: a.DownloadURL(),
				Size:               a.Size(),
			})
		}

		apiReleases = append(apiReleases, apiRelease{
			TagName:     rel.TagName(),
			Prerelease:  rel.Prerelease(),
			Assets:      apiAssets,
			PublishedAt: rel.PublishedAt().Format(time.RFC3339),
			CommitHash:  rel.CommitHash(),
		})
	}

	data, err := json.Marshal(apiReleases)
	if err != nil {
		return err
	}

	// Write to temp file first for atomic operation
	tempFile := c.filePath + ".tmp"

	err = os.WriteFile(tempFile, data, constants.FilePerm)
	if err != nil {
		return err
	}

	// Best-effort cleanup of the temp file on any failure from here on.
	defer func() {
		_, statErr := os.Stat(tempFile)
		if statErr == nil {
			_ = os.Remove(tempFile)
		}
	}()

	// Sync to disk so a power loss between the temp file write and the
	// rename cannot leave the renamed cache file empty or partially
	// flushed.
	cacheFile, openErr := os.OpenFile(tempFile, os.O_RDWR, constants.FilePerm)
	if openErr == nil {
		syncErr := cacheFile.Sync()
		if syncErr != nil {
			logrus.Warnf("Failed to fsync cache temp file: %v", syncErr)
		}

		_ = cacheFile.Close()
	}

	// Atomically rename temp file to final location
	err = os.Rename(tempFile, c.filePath)
	if err != nil {
		return fmt.Errorf("rename cache temp file: %w", err)
	}

	logrus.Debugf("Cached %d releases to %s", len(releases), c.filePath)

	return nil
}

// read loads, decodes, and converts the on-disk cache file. It
// deletes the file if the JSON is corrupted.
func (c *Cache) read() ([]release.Release, error) {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		return nil, err
	}

	var apiReleases []apiRelease

	err = json.Unmarshal(data, &apiReleases)
	if err != nil {
		// Corrupted cache (truncated write, disk full mid-rename, etc.).
		// Delete the bad file so future calls don't keep paying the parse
		// cost and fall through to the network path instead.
		removeErr := os.Remove(c.filePath)
		if removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			logrus.Warnf(
				"Failed to remove corrupted cache file %s: %v",
				c.filePath,
				removeErr,
			)
		}

		return nil, fmt.Errorf("corrupted cache removed: %w", err)
	}

	releases := make([]release.Release, 0, len(apiReleases))
	for _, apiRelease := range apiReleases {
		publishedAt, err := time.Parse(time.RFC3339, apiRelease.PublishedAt)
		if err != nil {
			logrus.Warnf(
				"Skipping release %s due to invalid PublishedAt: %v",
				apiRelease.TagName,
				err,
			)

			continue
		}

		assets := make([]release.Asset, 0, len(apiRelease.Assets))
		for _, aa := range apiRelease.Assets {
			assets = append(assets, release.NewAsset(aa.Name, aa.BrowserDownloadURL, aa.Size))
		}

		releases = append(releases, release.New(
			apiRelease.TagName,
			apiRelease.Prerelease,
			apiRelease.CommitHash,
			publishedAt,
			assets,
		))
	}

	return releases, nil
}
