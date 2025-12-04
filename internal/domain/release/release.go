// Package release provides the domain model for Neovim releases.
package release

import (
	"context"
	"time"
)

// Release represents a Neovim release from GitHub.
type Release struct {
	tagName     string
	prerelease  bool
	commitHash  string
	publishedAt time.Time
	assets      []Asset
}

// Asset represents a downloadable asset attached to a release.
type Asset struct {
	name        string
	downloadURL string
	size        int64
}

// New creates a new Release instance.
func New(
	tagName string,
	prerelease bool,
	commitHash string,
	publishedAt time.Time,
	assets []Asset,
) Release {
	return Release{
		tagName:     tagName,
		prerelease:  prerelease,
		commitHash:  commitHash,
		publishedAt: publishedAt,
		assets:      assets,
	}
}

// TagName returns the release tag name.
func (r Release) TagName() string {
	return r.tagName
}

// Prerelease returns whether this is a pre-release.
func (r Release) Prerelease() bool {
	return r.prerelease
}

// CommitHash returns the commit hash for this release.
func (r Release) CommitHash() string {
	return r.commitHash
}

// PublishedAt returns the publication timestamp.
func (r Release) PublishedAt() time.Time {
	return r.publishedAt
}

// Assets returns the list of downloadable assets.
func (r Release) Assets() []Asset {
	if r.assets == nil {
		return nil
	}

	result := make([]Asset, len(r.assets))
	copy(result, r.assets)

	return result
}

// NewAsset creates a new Asset instance.
func NewAsset(name, downloadURL string, size int64) Asset {
	return Asset{
		name:        name,
		downloadURL: downloadURL,
		size:        size,
	}
}

// Name returns the asset name.
func (a Asset) Name() string {
	return a.name
}

// DownloadURL returns the asset download URL.
func (a Asset) DownloadURL() string {
	return a.downloadURL
}

// Size returns the asset size in bytes.
func (a Asset) Size() int64 {
	return a.size
}

// Repository fetches releases from a remote source.
type Repository interface {
	// GetAll fetches all available releases.
	// If force is true, bypasses any caching.
	GetAll(ctx context.Context, force bool) ([]Release, error)

	// FindStable returns the latest stable release.
	FindStable() (Release, error)

	// FindNightly returns the latest nightly release.
	FindNightly() (Release, error)

	// FindByTag returns a specific release by tag.
	FindByTag(tag string) (Release, error)
}
