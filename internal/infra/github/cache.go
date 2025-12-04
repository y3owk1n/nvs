package github

import (
	"encoding/json"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/internal/domain/release"
)

const cacheFilePerm = 0o644

// Cache handles caching of GitHub releases.
type Cache struct {
	filePath string
	ttl      time.Duration
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
		return nil, os.ErrNotExist
	}

	data, err := os.ReadFile(c.filePath)
	if err != nil {
		return nil, err
	}

	// We need to unmarshal to API format first, then convert
	var apiReleases []apiRelease

	err = json.Unmarshal(data, &apiReleases)
	if err != nil {
		return nil, err
	}

	// Convert to domain releases
	releases := make([]release.Release, 0, len(apiReleases))
	for _, apiRelease := range apiReleases {
		publishedAt, _ := time.Parse(time.RFC3339, apiRelease.PublishedAt)

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

// Set stores releases in cache.
func (c *Cache) Set(releases []release.Release) error {
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

	err = os.WriteFile(c.filePath, data, cacheFilePerm)
	if err != nil {
		return err
	}

	logrus.Debugf("Cached %d releases to %s", len(releases), c.filePath)

	return nil
}
