// Package version provides the application service for version management.
package version

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/internal/domain/installer"
	"github.com/y3owk1n/nvs/internal/domain/release"
	"github.com/y3owk1n/nvs/internal/domain/version"
	"github.com/y3owk1n/nvs/internal/infra/github"
)

// Service orchestrates version management operations.
type Service struct {
	releaseRepo    release.Repository
	versionManager version.Manager
	installer      installer.Installer
	config         *Config
}

// Constants for version names.
const (
	StableVersion  = "stable"
	NightlyVersion = "nightly"
)

// Config holds configuration for the version service.
type Config struct {
	VersionsDir   string
	CacheFilePath string
	GlobalBinDir  string
}

// New creates a new version Service.
func New(
	releaseRepo release.Repository,
	versionManager version.Manager,
	installer installer.Installer,
	config *Config,
) *Service {
	return &Service{
		releaseRepo:    releaseRepo,
		versionManager: versionManager,
		installer:      installer,
		config:         config,
	}
}

// Install installs a Neovim version.
// The versionAlias can be "stable", "nightly", a version tag, or a commit hash.
func (s *Service) Install(
	ctx context.Context,
	versionAlias string,
	progress installer.ProgressFunc,
) error {
	// Normalize version
	normalized := normalizeVersion(versionAlias)

	// Check if it's a commit hash
	if version.IsCommitHash(normalized) {
		// Build from source for commit hashes
		dest := filepath.Join(s.config.VersionsDir, normalized)

		return s.installer.BuildFromCommit(ctx, normalized, dest)
	}

	// Resolve release
	var (
		rel release.Release
		err error
	)

	switch normalized {
	case StableVersion:
		rel, err = s.releaseRepo.FindStable()
	case NightlyVersion:
		rel, err = s.releaseRepo.FindNightly()
	default:
		rel, err = s.releaseRepo.FindByTag(normalized)
	}

	if err != nil {
		return fmt.Errorf("failed to resolve version: %w", err)
	}

	// Get asset URL for current platform
	assetURL, _, err := github.GetAssetURL(rel)
	if err != nil {
		return fmt.Errorf("failed to get asset URL: %w", err)
	}

	logrus.Debugf("Asset URL: %s", assetURL)

	// Installation logic
	releaseInfo := &releaseAdapter{
		Release: rel,
	}

	return s.installer.InstallRelease(ctx, releaseInfo, s.config.VersionsDir, normalized, progress)
}

// releaseAdapter adapts release.Release to installer.ReleaseInfo.
type releaseAdapter struct {
	release.Release
}

func (r *releaseAdapter) GetAssetURL() (string, error) {
	url, _, err := github.GetAssetURL(r.Release)

	return url, err
}

func (r *releaseAdapter) GetChecksumURL() (string, error) {
	// Find asset name first to get pattern
	_, pattern, err := github.GetAssetURL(r.Release)
	if err != nil {
		return "", err
	}

	return github.GetChecksumURL(r.Release, pattern), nil
}

func (r *releaseAdapter) GetIdentifier() string {
	return r.TagName()
}

// Use switches to a specific version.
func (s *Service) Use(ctx context.Context, versionAlias string) (string, error) {
	normalized := normalizeVersion(versionAlias)

	// Determine target version
	var targetVersion version.Version

	if version.IsCommitHash(normalized) {
		// For commit hash, the version name is the hash itself
		targetVersion = version.New(normalized, version.TypeCommit, normalized, "")
	} else {
		// Resolve from release
		var (
			rel release.Release
			err error
		)

		switch normalized {
		case StableVersion:
			rel, err = s.releaseRepo.FindStable()
		case NightlyVersion:
			rel, err = s.releaseRepo.FindNightly()
		default:
			rel, err = s.releaseRepo.FindByTag(normalized)
		}

		if err != nil {
			return "", fmt.Errorf("failed to resolve version: %w", err)
		}

		// Determine version type
		vType := version.TypeTag
		if normalized == StableVersion {
			vType = version.TypeStable
		} else if rel.Prerelease() {
			vType = version.TypeNightly
		}

		targetVersion = version.New(normalized, vType, rel.TagName(), rel.CommitHash())
	}

	// Check if already installed
	if !s.versionManager.IsInstalled(targetVersion) {
		return "", fmt.Errorf("%w: %s", version.ErrVersionNotFound, targetVersion.Name())
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
func (s *Service) List() ([]version.Version, error) {
	versions, err := s.versionManager.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}

	return versions, nil
}

// Current returns the currently active version.
func (s *Service) Current() (version.Version, error) {
	current, err := s.versionManager.Current()
	if err != nil {
		return version.Version{}, fmt.Errorf("failed to get current version: %w", err)
	}

	return current, nil
}

// Uninstall removes an installed version.
func (s *Service) Uninstall(versionAlias string, force bool) error {
	normalized := normalizeVersion(versionAlias)

	// Find the version
	versions, err := s.versionManager.List()
	if err != nil {
		return fmt.Errorf("failed to list versions: %w", err)
	}

	var targetVersion version.Version

	found := false

	for _, v := range versions {
		if v.Name() == normalized {
			targetVersion = v
			found = true

			break
		}
	}

	if !found {
		return fmt.Errorf("%w: %s", version.ErrVersionNotFound, normalized)
	}

	// Uninstall
	err = s.versionManager.Uninstall(targetVersion, force)
	if err != nil {
		return fmt.Errorf("failed to uninstall version: %w", err)
	}

	return nil
}

// ListRemote returns available remote releases.
func (s *Service) ListRemote(force bool) ([]release.Release, error) {
	return s.releaseRepo.GetAll(force)
}

// Upgrade upgrades a version (stable or nightly).
func (s *Service) Upgrade(
	ctx context.Context,
	versionAlias string,
	progress installer.ProgressFunc,
) error {
	normalized := normalizeVersion(versionAlias)

	// Only stable and nightly can be upgraded
	if normalized != StableVersion && normalized != NightlyVersion {
		return ErrOnlyStableNightlyUpgrade
	}

	// Check if installed
	if !s.versionManager.IsInstalled(
		version.New(normalized, version.TypeTag, normalized, ""),
	) {
		return ErrNotInstalled
	}

	// Resolve remote release
	var (
		rel release.Release
		err error
	)

	switch normalized {
	case StableVersion:
		rel, err = s.releaseRepo.FindStable()
	case NightlyVersion:
		rel, err = s.releaseRepo.FindNightly()
	}

	if err != nil {
		return fmt.Errorf("failed to resolve version: %w", err)
	}

	// Check if update is needed
	currentIdentifier, err := s.versionManager.GetInstalledReleaseIdentifier(normalized)
	if err == nil && currentIdentifier == rel.TagName() {
		return ErrAlreadyUpToDate
	}

	// Backup existing version
	versionPath := filepath.Join(s.config.VersionsDir, normalized)
	backupPath := versionPath + ".backup"

	err = os.Rename(versionPath, backupPath)
	if err != nil {
		return fmt.Errorf("failed to backup version: %w", err)
	}

	upgradeSuccess := false

	// Cleanup backup on success or restore on failure
	defer func() {
		if upgradeSuccess {
			// Upgrade succeeded, remove backup
			_ = os.RemoveAll(backupPath)
		} else {
			// Upgrade failed, restore backup
			_ = os.RemoveAll(versionPath) // Clean partial install
			_ = os.Rename(backupPath, versionPath)
		}
	}()

	// Install new version
	releaseInfo := &releaseAdapter{
		Release: rel,
	}

	err = s.installer.InstallRelease(ctx, releaseInfo, s.config.VersionsDir, normalized, progress)
	if err != nil {
		return fmt.Errorf("failed to install release: %w", err)
	}

	upgradeSuccess = true

	return nil
}

// normalizeVersion normalizes a version string.
func normalizeVersion(versionStr string) string {
	if versionStr == StableVersion || versionStr == NightlyVersion ||
		version.IsCommitHash(versionStr) {
		return versionStr
	}

	if !strings.HasPrefix(versionStr, "v") {
		return "v" + versionStr
	}

	return versionStr
}

// IsVersionInstalled checks if a version is installed.
func (s *Service) IsVersionInstalled(versionName string) bool {
	v := version.New(versionName, version.TypeTag, versionName, "")

	return s.versionManager.IsInstalled(v)
}

// GetInstalledVersionIdentifier returns the identifier (commit hash) of an installed version.
func (s *Service) GetInstalledVersionIdentifier(versionName string) (string, error) {
	return s.versionManager.GetInstalledReleaseIdentifier(versionName)
}

// FindStable returns the latest stable release.
func (s *Service) FindStable() (release.Release, error) {
	return s.releaseRepo.FindStable()
}

// FindNightly returns the latest nightly release.
func (s *Service) FindNightly() (release.Release, error) {
	return s.releaseRepo.FindNightly()
}

// IsCommitHash checks if a string is a commit hash.
func (s *Service) IsCommitHash(str string) bool {
	return version.IsCommitHash(str)
}
