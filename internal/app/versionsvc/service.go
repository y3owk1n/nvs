// Package versionsvc provides the application service for version management.
package versionsvc

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/domain/installer"
	"github.com/y3owk1n/nvs/internal/domain/release"
	"github.com/y3owk1n/nvs/internal/domain/vtypes"
	"github.com/y3owk1n/nvs/internal/infra/github"
)

// Service orchestrates version management operations.
type Service struct {
	releaseRepo    release.Repository
	versionManager vtypes.Manager
	installer      installer.Installer
	config         *Config
}

// Config holds configuration for the version service.
type Config struct {
	VersionsDir    string
	CacheFilePath  string
	GlobalBinDir   string
	MirrorURL      string // Optional GitHub mirror URL for downloads
	UseGlobalCache bool   // Whether to use global cache for releases (passed to GitHub client)
}

// New creates a new version Service.
func New(
	releaseRepo release.Repository,
	versionManager vtypes.Manager,
	installer installer.Installer,
	config *Config,
) (*Service, error) {
	if config == nil {
		return nil, fmt.Errorf("%w", ErrConfigNil)
	}

	if config.VersionsDir == "" {
		return nil, fmt.Errorf("%w", ErrVersionsDirEmpty)
	}

	return &Service{
		releaseRepo:    releaseRepo,
		versionManager: versionManager,
		installer:      installer,
		config:         config,
	}, nil
}

// Install installs a Neovim version.
// The versionAlias can be "stable", "nightly", a version tag, or a commit hash.
func (s *Service) Install(
	ctx context.Context,
	versionAlias string,
	progress installer.ProgressFunc,
) error {
	// Reject path-traversal or otherwise malformed input before
	// it reaches filepath.Join(VersionsDir, ...) or any installer
	// syscall. Without this check, an attacker who can place a
	// crafted .nvs-version file in a project directory could
	// coerce 'nvs install' into creating or writing outside the
	// versions root, or into treating a non-version path as a
	// build source.
	validateErr := vtypes.ValidateVersionName(versionAlias)
	if validateErr != nil {
		return validateErr
	}

	// Normalize version
	normalized := normalizeVersion(versionAlias)

	// Check if it's a commit hash
	if vtypes.IsCommitReference(normalized) {
		// Build from source for commit hashes
		_, err := s.installer.BuildFromCommit(
			ctx,
			normalized,
			s.config.VersionsDir,
			progress,
		)
		if err != nil {
			return err
		}

		return nil
	}

	// Resolve release
	var (
		rel release.Release
		err error
	)

	switch normalized {
	case constants.Stable:
		rel, err = s.releaseRepo.FindStable(ctx)
	case constants.Nightly:
		rel, err = s.releaseRepo.FindNightly(ctx)
	default:
		rel, err = s.releaseRepo.FindByTag(ctx, normalized)
	}

	if err != nil {
		return fmt.Errorf("failed to resolve version: %w", err)
	}

	// Installation logic
	releaseInfo := &releaseAdapter{
		Release:   rel,
		mirrorURL: s.config.MirrorURL,
	}

	return s.installer.InstallRelease(ctx, releaseInfo, s.config.VersionsDir, normalized, progress)
}

// releaseAdapter adapts release.Release to installer.ReleaseInfo.
type releaseAdapter struct {
	release.Release

	mirrorURL string

	// assetOnce + assetResult memoize the platform-specific asset
	// resolution so that GetAssetURL and GetChecksumURL share a
	// single underlying github.GetAssetURL call. The previous
	// implementation re-invoked github.GetAssetURL on every
	// GetChecksumURL call just to recover the asset pattern, which
	// re-scanned the release's assets list per install. Per the
	// installer pipeline (internal/infra/installer/service.go),
	// each install calls GetAssetURL once and GetChecksumURL once,
	// so caching turns 2 asset-list scans into 1.
	assetOnce   sync.Once
	assetResult assetLookup

	// assetResolveCount tracks how many times the asset resolution
	// actually ran (i.e. how many cache misses occurred). It is
	// not used for caching (sync.Once is), but it lets tests verify
	// that the cache actually short-circuits the second call
	// instead of re-running github.GetAssetURL. The count is
	// incremented inside the once.Do block, so once the first
	// resolution completes, the count remains 1 for the lifetime
	// of the adapter.
	assetResolveCount atomic.Int64
}

func (r *releaseAdapter) GetAssetURL() (string, error) {
	result := r.resolveAsset()
	if result.Err != nil {
		return "", result.Err
	}

	return result.URL, nil
}

func (r *releaseAdapter) GetChecksumURL() (string, error) {
	// Reuse the cached asset resolution: we need the pattern to find
	// the matching checksum file, but the URL itself is not relevant
	// here. If asset resolution already failed, surface that error
	// before attempting a second lookup.
	asset := r.resolveAsset()
	if asset.Err != nil {
		return "", asset.Err
	}

	url, err := github.GetChecksumURL(r.Release, asset.Pattern)
	if err != nil {
		return "", err
	}

	return r.applyMirror(url), nil
}

func (r *releaseAdapter) GetIdentifier() string {
	// For nightly releases, use commit hash as identifier; for stable, use tag name
	if r.Prerelease() && strings.HasPrefix(strings.ToLower(r.TagName()), "nightly") {
		return r.CommitHash()
	}

	return r.TagName()
}

// AssetResolveCount returns the number of times the underlying
// asset resolution function (github.GetAssetURL) has actually been
// invoked on this adapter. The count is incremented inside the
// sync.Once block, so it stays at 1 for the lifetime of the
// adapter after the first call. It exists solely to support
// regression tests that verify the asset cache actually
// short-circuits the second call.
func (r *releaseAdapter) AssetResolveCount() int64 {
	return r.assetResolveCount.Load()
}

// applyMirror replaces the default GitHub URL with the mirror URL if configured.
func (r *releaseAdapter) applyMirror(url string) string {
	return github.ApplyMirrorToURL(url, r.mirrorURL)
}

// assetLookup is the cached result of resolving a release's
// platform-specific asset: the mirror-applied download URL, the
// original asset name pattern (needed to look up the checksum file),
// and any error from the resolution. The fields are populated
// together by a single sync.Once, so callers always see a consistent
// snapshot.
type assetLookup struct {
	URL     string
	Pattern string
	Err     error
}

// resolveAsset returns the cached asset lookup, computing it on the
// first call. The error from the underlying github.GetAssetURL is
// captured in the result and returned to subsequent callers as-is,
// preserving the pre-existing error-propagation semantics.
//
// assetResolveCount is incremented inside the sync.Once block, so
// it counts the number of times github.GetAssetURL was actually
// invoked (i.e. the number of times the cache missed). After the
// first call, the count stays at 1 forever.
func (r *releaseAdapter) resolveAsset() assetLookup {
	r.assetOnce.Do(func() {
		url, pattern, err := github.GetAssetURL(r.Release)
		r.assetResult = assetLookup{
			URL:     r.applyMirror(url),
			Pattern: pattern,
			Err:     err,
		}
		r.assetResolveCount.Add(1)
	})

	return r.assetResult
}

// Use switches to a specific version.
func (s *Service) Use(ctx context.Context, versionAlias string) (string, error) {
	// Reject path-traversal input before any filepath operation.
	// This guards the .nvs-version file path: a malicious value
	// in that file would otherwise flow into the symlink target
	// and end up exec'd by downstream tools that respect the
	// 'current' link.
	err := vtypes.ValidateVersionName(versionAlias)
	if err != nil {
		return "", err
	}

	normalized := normalizeVersion(versionAlias)

	// Determine target version
	var targetVersion vtypes.Version

	if vtypes.IsCommitReference(normalized) {
		// For commit hash, the version name is the hash itself
		targetVersion = vtypes.New(normalized, vtypes.TypeCommit, normalized, "")
	} else {
		// Resolve from release
		var (
			rel release.Release
			err error
		)

		switch normalized {
		case constants.Stable:
			rel, err = s.releaseRepo.FindStable(ctx)
		case constants.Nightly:
			rel, err = s.releaseRepo.FindNightly(ctx)
		default:
			rel, err = s.releaseRepo.FindByTag(ctx, normalized)
		}

		if err != nil {
			return "", fmt.Errorf("failed to resolve version: %w", err)
		}

		// Determine version type
		vType := determineVersionType(normalized)

		targetVersion = vtypes.New(normalized, vType, rel.TagName(), rel.CommitHash())
	}

	// Check if already installed
	if !s.versionManager.IsInstalled(targetVersion) {
		return "", fmt.Errorf("%w: %s", vtypes.ErrVersionNotFound, targetVersion.Name())
	}

	// Check if already current
	current, err := s.versionManager.Current()
	if err == nil && current.Name() == targetVersion.Name() {
		logrus.Debugf("Already using version: %s", targetVersion.Name())

		return targetVersion.Identifier(), nil
	}

	// Switch version
	err = s.versionManager.Switch(targetVersion)
	if err != nil {
		return "", fmt.Errorf("failed to switch version: %w", err)
	}

	return targetVersion.Identifier(), nil
}

// List returns all installed versions.
func (s *Service) List() ([]vtypes.Version, error) {
	versions, err := s.versionManager.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}

	return versions, nil
}

// Current returns the currently active version.
func (s *Service) Current() (vtypes.Version, error) {
	current, err := s.versionManager.Current()
	if err != nil {
		return vtypes.Version{}, fmt.Errorf("failed to get current version: %w", err)
	}

	return current, nil
}

// Uninstall removes an installed version.
func (s *Service) Uninstall(versionAlias string, force bool) error {
	// Reject path-traversal input before any filepath operation
	// or filesystem deletion.
	err := vtypes.ValidateVersionName(versionAlias)
	if err != nil {
		return err
	}

	normalized := normalizeVersion(versionAlias)

	// Find the version
	versions, err := s.versionManager.List()
	if err != nil {
		return fmt.Errorf("failed to list versions: %w", err)
	}

	var targetVersion vtypes.Version

	found := false

	for _, v := range versions {
		if v.Name() == normalized {
			targetVersion = v
			found = true

			break
		}
	}

	if !found {
		return fmt.Errorf("%w: %s", vtypes.ErrVersionNotFound, normalized)
	}

	// Uninstall
	err = s.versionManager.Uninstall(targetVersion, force)
	if err != nil {
		return fmt.Errorf("failed to uninstall version: %w", err)
	}

	return nil
}

// ListRemote returns available remote releases.
func (s *Service) ListRemote(ctx context.Context, force bool) ([]release.Release, error) {
	return s.releaseRepo.GetAll(ctx, force)
}

// Upgrade upgrades a version (stable or nightly).
func (s *Service) Upgrade(
	ctx context.Context,
	versionAlias string,
	progress installer.ProgressFunc,
) error {
	// Reject path-traversal input before any filepath operation.
	validateErr := vtypes.ValidateVersionName(versionAlias)
	if validateErr != nil {
		return validateErr
	}

	normalized := normalizeVersion(versionAlias)

	// Only stable and nightly can be upgraded
	if normalized != constants.Stable && normalized != constants.Nightly {
		return ErrOnlyStableNightlyUpgrade
	}

	// Check if installed
	if !s.versionManager.IsInstalled(
		vtypes.New(normalized, determineVersionType(normalized), normalized, ""),
	) {
		return ErrNotInstalled
	}

	// Resolve remote release
	var (
		rel release.Release
		err error
	)

	switch normalized {
	case constants.Stable:
		rel, err = s.releaseRepo.FindStable(ctx)
	case constants.Nightly:
		rel, err = s.releaseRepo.FindNightly(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to resolve version: %w", err)
	}

	// Check if update is needed
	currentIdentifier, err := s.versionManager.GetInstalledReleaseIdentifier(normalized)
	if err != nil {
		return fmt.Errorf("failed to get installed release identifier: %w", err)
	}

	expectedIdentifier := rel.TagName()
	if normalized == constants.Nightly {
		expectedIdentifier = rel.CommitHash()
	}

	if currentIdentifier == expectedIdentifier {
		return ErrAlreadyUpToDate
	}

	// Use installer.UpgradeRelease for atomic upgrade with proper locking
	releaseInfo := &releaseAdapter{
		Release:   rel,
		mirrorURL: s.config.MirrorURL,
	}

	err = s.installer.UpgradeRelease(ctx, releaseInfo, s.config.VersionsDir, normalized, progress)
	if err != nil {
		return fmt.Errorf("failed to upgrade: %w", err)
	}

	return nil
}

// normalizeVersion normalizes a version string.
func normalizeVersion(versionStr string) string {
	return vtypes.NormalizeVersionForPath(versionStr)
}

// determineVersionType determines the version type from the name.
func determineVersionType(name string) vtypes.Type {
	switch {
	case name == constants.Stable:
		return vtypes.TypeStable
	case strings.HasPrefix(strings.ToLower(name), "nightly"):
		return vtypes.TypeNightly
	case vtypes.IsCommitReference(name):
		return vtypes.TypeCommit
	default:
		return vtypes.TypeTag
	}
}

// IsVersionInstalled checks if a version is installed.
func (s *Service) IsVersionInstalled(versionName string) bool {
	// Reject path-traversal input up front: an invalid name
	// cannot match an installed version, so the answer is
	// unambiguous (false) without needing to perform the lookup.
	err := vtypes.ValidateVersionName(versionName)
	if err != nil {
		return false
	}

	normalized := normalizeVersion(versionName)
	versionType := determineVersionType(normalized)
	v := vtypes.New(normalized, versionType, normalized, "")

	return s.versionManager.IsInstalled(v)
}

// InstalledVersionNames returns the names of all currently installed
// versions. It is intended for callers that need to test membership
// in a loop (e.g. cmd/list-remote.go), where repeatedly calling
// IsVersionInstalled would issue an os.Stat per iteration. Callers
// should build a set from the result and look up keys in O(1).
//
// Note: this delegates to the version manager's List(), which already
// filters out the "current" sentinel and nightly backup directories
// (prefix "nightly-"). Callers asking about a tag like "nightly" or
// "v0.10.0" will see the expected true/false.
func (s *Service) InstalledVersionNames() ([]string, error) {
	versions, err := s.versionManager.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list installed versions: %w", err)
	}

	names := make([]string, 0, len(versions))
	for _, v := range versions {
		names = append(names, v.Name())
	}

	return names, nil
}

// InstalledVersionIdentifiers returns a map of installed version
// name -> commit identifier (the contents of <VersionsDir>/<name>
// /version.txt) in a single pass. It is intended for callers that
// loop over the installed set and previously called
// GetInstalledVersionIdentifier per iteration, issuing one
// os.ReadFile per version (N+1 syscalls). With this method the
// cost is one os.ReadDir plus one os.ReadFile per installed
// version, which is the minimum possible work for the
// information returned.
//
// An entry with an empty value means the version has no
// version.txt (e.g. a pre-existing install that was not produced
// by nvs, or a version.txt that failed to read). Callers should
// treat the empty string the same way they would treat a
// GetInstalledVersionIdentifier error.
func (s *Service) InstalledVersionIdentifiers() (map[string]string, error) {
	versions, err := s.versionManager.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list installed versions: %w", err)
	}

	identifiers := make(map[string]string, len(versions))
	for _, v := range versions {
		identifiers[v.Name()] = v.CommitHash()
	}

	return identifiers, nil
}

// GetInstalledVersionIdentifier returns the identifier (commit hash) of an installed version.
func (s *Service) GetInstalledVersionIdentifier(versionName string) (string, error) {
	// Reject path-traversal input before any filepath operation.
	err := vtypes.ValidateVersionName(versionName)
	if err != nil {
		return "", err
	}

	normalized := normalizeVersion(versionName)

	return s.versionManager.GetInstalledReleaseIdentifier(normalized)
}

// FindStable returns the latest stable release.
func (s *Service) FindStable(ctx context.Context) (release.Release, error) {
	return s.releaseRepo.FindStable(ctx)
}

// FindNightly returns the latest nightly release.
func (s *Service) FindNightly(ctx context.Context) (release.Release, error) {
	return s.releaseRepo.FindNightly(ctx)
}

// IsCommitReference checks if a string is a commit reference.
func (s *Service) IsCommitReference(str string) bool {
	return vtypes.IsCommitReference(str)
}
