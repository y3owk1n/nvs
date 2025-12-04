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
func (s *Service) Install(ctx context.Context, versionAlias string, progress installer.ProgressFunc) error {
	// Normalize version
	normalized := normalizeVersion(versionAlias)

	// Check if it's a commit hash
	if isCommitHash(normalized) {
		// This will be handled by a separate installer implementation
		return fmt.Errorf("commit hash installation not yet implemented in service layer")
	}

	// Resolve release
	var rel release.Release
	var err error

	switch normalized {
	case "stable":
		rel, err = s.releaseRepo.FindStable()
	case "nightly":
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

	return s.installer.InstallRelease(ctx, releaseInfo, s.config.VersionsDir, progress)
}

// releaseAdapter adapts release.Release to installer.ReleaseInfo
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
func (s *Service) Use(ctx context.Context, versionAlias string) error {
	normalized := normalizeVersion(versionAlias)

	// Determine target version
	var targetVersion version.Version

	if isCommitHash(normalized) {
		// For commit hash, the version name is the hash itself
		targetVersion = version.New(normalized, version.TypeCommit, normalized, "")
	} else {
		// Resolve from release
		var rel release.Release
		var err error

		switch normalized {
		case "stable":
			rel, err = s.releaseRepo.FindStable()
		case "nightly":
			rel, err = s.releaseRepo.FindNightly()
		default:
			rel, err = s.releaseRepo.FindByTag(normalized)
		}

		if err != nil {
			return fmt.Errorf("failed to resolve version: %w", err)
		}

		// Determine version type
		vType := version.TypeTag
		if rel.Prerelease() {
			vType = version.TypeNightly
		}

		targetVersion = version.New(rel.TagName(), vType, rel.TagName(), rel.CommitHash())
	}

	// Check if already installed
	if !s.versionManager.IsInstalled(targetVersion, s.config.VersionsDir) {
		return fmt.Errorf("%w: %s", version.ErrVersionNotFound, targetVersion.Name())
	}

	// Check if already current
	current, err := s.versionManager.Current(s.config.VersionsDir)
	if err == nil && current.Name() == targetVersion.Name() {
		logrus.Debugf("Already using version: %s", targetVersion.Name())
		return nil
	}

	// Switch version
	if err := s.versionManager.Switch(targetVersion, s.config.VersionsDir, s.config.GlobalBinDir); err != nil {
		return fmt.Errorf("failed to switch version: %w", err)
	}

	return nil
}

// List returns all installed versions.
func (s *Service) List() ([]version.Version, error) {
	versions, err := s.versionManager.List(s.config.VersionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}

	return versions, nil
}

// Current returns the currently active version.
func (s *Service) Current() (version.Version, error) {
	current, err := s.versionManager.Current(s.config.VersionsDir)
	if err != nil {
		return version.Version{}, fmt.Errorf("failed to get current version: %w", err)
	}

	return current, nil
}

// Uninstall removes an installed version.
func (s *Service) Uninstall(versionAlias string, force bool) error {
	normalized := normalizeVersion(versionAlias)

	// Find the version
	versions, err := s.versionManager.List(s.config.VersionsDir)
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
	if err := s.versionManager.Uninstall(targetVersion, s.config.VersionsDir, force); err != nil {
		return fmt.Errorf("failed to uninstall version: %w", err)
	}

	return nil
}

// ListRemote returns available remote releases.
func (s *Service) ListRemote(force bool) ([]release.Release, error) {
	// TODO: Pass force flag to repository if supported
	return s.releaseRepo.GetAll()
}

// Upgrade upgrades a version (stable or nightly).
func (s *Service) Upgrade(ctx context.Context, versionAlias string, progress installer.ProgressFunc) error {
	normalized := normalizeVersion(versionAlias)

	// Only stable and nightly can be upgraded
	if normalized != "stable" && normalized != "nightly" {
		return fmt.Errorf("only stable and nightly versions can be upgraded")
	}

	// Check if installed
	if !s.versionManager.IsInstalled(version.New(normalized, version.TypeTag, normalized, ""), s.config.VersionsDir) {
		return fmt.Errorf("not installed")
	}

	// Resolve remote release
	var rel release.Release
	var err error

	switch normalized {
	case "stable":
		rel, err = s.releaseRepo.FindStable()
	case "nightly":
		rel, err = s.releaseRepo.FindNightly()
	}

	if err != nil {
		return fmt.Errorf("failed to resolve version: %w", err)
	}

	// Check if update is needed
	currentIdentifier, err := s.versionManager.GetInstalledReleaseIdentifier(normalized, s.config.VersionsDir)
	if err == nil && currentIdentifier == rel.TagName() {
		return fmt.Errorf("already up-to-date")
	}

	// Backup existing version
	versionPath := filepath.Join(s.config.VersionsDir, normalized)
	backupPath := versionPath + ".backup"

	if err := os.Rename(versionPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup existing version: %w", err)
	}

	// Cleanup backup on success or restore on failure
	defer func() {
		if _, err := os.Stat(versionPath); err == nil {
			// Upgrade succeeded, remove backup
			_ = os.RemoveAll(backupPath)
		} else {
			// Upgrade failed, restore backup
			_ = os.Rename(backupPath, versionPath)
		}
	}()

	// Install new version
	releaseInfo := &releaseAdapter{
		Release: rel,
	}

	if err := s.installer.InstallRelease(ctx, releaseInfo, s.config.VersionsDir, progress); err != nil {
		return fmt.Errorf("failed to install upgrade: %w", err)
	}

	return nil
}

// normalizeVersion normalizes a version string.
func normalizeVersion(v string) string {
	if v == "stable" || v == "nightly" || isCommitHash(v) {
		return v
	}

	if !strings.HasPrefix(v, "v") {
		return "v" + v
	}

	return v
}

// isCommitHash checks if a string is a commit hash.
func isCommitHash(str string) bool {
	if str == "master" {
		return true
	}

	if len(str) != 7 && len(str) != 40 {
		return false
	}

	for _, r := range str {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}

	return true
}

// IsVersionInstalled checks if a version is installed.
func (s *Service) IsVersionInstalled(versionName string) bool {
	v := version.New(versionName, version.TypeTag, versionName, "")
	return s.versionManager.IsInstalled(v, s.config.VersionsDir)
}

// GetInstalledVersionIdentifier returns the identifier (commit hash) of an installed version.
func (s *Service) GetInstalledVersionIdentifier(versionName string) (string, error) {
	return s.versionManager.GetInstalledReleaseIdentifier(versionName, s.config.VersionsDir)
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
	return isCommitHash(str)
}
