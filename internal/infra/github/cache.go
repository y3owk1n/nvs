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
	if err := json.Unmarshal(data, &apiReleases); err != nil {
		return nil, err
	}

	// Convert to domain releases
	releases := make([]release.Release, 0, len(apiReleases))
	for _, ar := range apiReleases {
		publishedAt, _ := time.Parse(time.RFC3339, ar.PublishedAt)

		assets := make([]release.Asset, 0, len(ar.Assets))
		for _, aa := range ar.Assets {
			assets = append(assets, release.NewAsset(aa.Name, aa.BrowserDownloadURL, aa.Size))
		}

		releases = append(releases, release.New(
			ar.TagName,
			ar.Prerelease,
			ar.CommitHash,
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

	for _, r := range releases {
		apiAssets := make([]apiAsset, 0, len(r.Assets()))
		for _, a := range r.Assets() {
			apiAssets = append(apiAssets, apiAsset{
				Name:               a.Name(),
				BrowserDownloadURL: a.DownloadURL(),
				Size:               a.Size(),
			})
		}

		apiReleases = append(apiReleases, apiRelease{
			TagName:     r.TagName(),
			Prerelease:  r.Prerelease(),
			Assets:      apiAssets,
			PublishedAt: r.PublishedAt().Format(time.RFC3339),
			CommitHash:  r.CommitHash(),
		})
	}

	data, err := json.Marshal(apiReleases)
	if err != nil {
		return err
	}

	if err := os.WriteFile(c.filePath, data, cacheFilePerm); err != nil {
		return err
	}

	logrus.Debugf("Cached %d releases to %s", len(releases), c.filePath)

	return nil
}
