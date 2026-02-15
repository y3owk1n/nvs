// Package installer provides the infrastructure implementation for Neovim installation.
package installer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/domain/installer"
	"github.com/y3owk1n/nvs/internal/infra/archive"
	"github.com/y3owk1n/nvs/internal/infra/builder"
	"github.com/y3owk1n/nvs/internal/infra/downloader"
	"github.com/y3owk1n/nvs/internal/infra/filesystem"
)

// Service implements installer.Installer.
type Service struct {
	downloader *downloader.Downloader
	extractor  *archive.Extractor
	builder    *builder.SourceBuilder
}

// New creates a new installer Service.
func New(
	d *downloader.Downloader,
	e *archive.Extractor,
	b *builder.SourceBuilder,
) *Service {
	return &Service{
		downloader: d,
		extractor:  e,
		builder:    b,
	}
}

// InstallRelease installs a pre-built release with per-version locking.
// Uses the same per-version lock as Switch and Uninstall for coordination.
func (s *Service) InstallRelease(
	ctx context.Context,
	rel installer.ReleaseInfo,
	dest string,
	installName string,
	progress installer.ProgressFunc,
) error {
	// Fast path: check if already installed before acquiring lock
	// Check for version.txt to ensure installation was complete
	versionPath := filepath.Join(dest, installName)
	versionFile := filepath.Join(versionPath, "version.txt")

	_, err := os.Stat(versionFile)
	if err == nil {
		logrus.Debugf("Version %s already exists, skipping install", installName)

		return nil
	}

	// Acquire per-version lock to prevent concurrent operations on the same version
	lockPath := filepath.Join(dest, fmt.Sprintf(".nvs-version-%s.lock", installName))
	lock := filesystem.NewFileLock(lockPath)

	// Use context-aware lock with extended timeout (10 minutes)
	// This accommodates slow downloads while respecting caller cancellation
	const installLockTimeout = 10 * time.Minute

	installCtx, cancel := context.WithTimeout(ctx, installLockTimeout)
	defer cancel()

	err = lock.Lock(installCtx)
	if err != nil {
		return fmt.Errorf("failed to acquire install lock for %s: %w", installName, err)
	}

	defer func() {
		unlockErr := lock.Unlock()
		if unlockErr != nil {
			logrus.Warnf("failed to unlock install lock for %s: %v", installName, unlockErr)
		}
	}()

	// Double-check after acquiring lock (another process may have installed it)
	// Check for version.txt to ensure installation was complete
	_, err = os.Stat(versionFile)
	if err == nil {
		logrus.Debugf("Version %s was installed by another process", installName)

		return nil
	}

	// 1. Get asset URL
	assetURL, err := rel.GetAssetURL()
	if err != nil {
		return fmt.Errorf("failed to get asset URL: %w", err)
	}

	// 2. Create temp file for download
	tempFile, err := os.CreateTemp("", "nvim-release-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	defer func() { _ = os.Remove(tempFile.Name()) }()
	defer func() { _ = tempFile.Close() }()

	// 3. Download
	// Check if we have a checksum URL for streaming verification
	checksumURL, err := rel.GetChecksumURL()
	hasChecksum := err == nil && checksumURL != ""

	if hasChecksum {
		if progress != nil {
			progress("Downloading & Verifying", 0)
		}

		assetName := filepath.Base(assetURL)

		err = s.downloader.DownloadWithChecksumVerification(
			ctx,
			assetURL,
			checksumURL,
			assetName,
			tempFile,
			func(p int) {
				if progress != nil {
					progress("Downloading & Verifying", p)
				}
			},
		)
		if err != nil {
			if errors.Is(err, downloader.ErrChecksumMismatch) {
				return fmt.Errorf("checksum verification failed: %w", err)
			}

			return fmt.Errorf("download failed: %w", err)
		}
	} else {
		if progress != nil {
			progress("Downloading", 0)
		}

		err = s.downloader.Download(ctx, assetURL, tempFile, func(p int) {
			if progress != nil {
				progress("Downloading", p)
			}
		})
		if err != nil {
			return fmt.Errorf("download failed: %w", err)
		}
	}

	// 4. Extract
	if progress != nil {
		progress("Extracting", 0)
	}

	// Create destination directory
	installPath := filepath.Join(dest, installName)

	err = os.MkdirAll(installPath, constants.DirPerm)
	if err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	// Reset file position for extraction
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek temp file: %w", err)
	}

	err = s.extractor.Extract(tempFile, installPath)
	if err != nil {
		// Clean up partial installation directory on failure
		cleanupErr := os.RemoveAll(installPath)
		if cleanupErr != nil {
			logrus.Warnf("Failed to clean up partial install directory: %v", cleanupErr)
		}

		return fmt.Errorf("extraction failed: %w", err)
	}

	// 5. Write version file
	versionFile = filepath.Join(installPath, "version.txt")

	err = os.WriteFile(versionFile, []byte(rel.GetIdentifier()), constants.FilePerm)
	if err != nil {
		logrus.Warnf("Failed to write version file: %v", err)
	}

	if progress != nil {
		progress("Complete", constants.ProgressComplete)
	}

	return nil
}

// BuildFromCommit builds Neovim from source with per-version locking.
// Uses the same per-version lock as Install, Switch, and Uninstall for coordination.
//
// The lock uses the resolved version name (short hash) to ensure consistency
// with Switch/Uninstall operations, which use the directory name.
func (s *Service) BuildFromCommit(
	ctx context.Context,
	commit string,
	dest string,
	progress installer.ProgressFunc,
) (string, error) {
	// Resolve the commit to a version name by scanning for matching entries
	// This ensures the lock key matches what Switch/Uninstall will use
	versionName := s.resolveCommitToVersionName(dest, commit)

	// Acquire per-version lock using the resolved version name
	lockPath := filepath.Join(dest, fmt.Sprintf(".nvs-version-%s.lock", versionName))
	lock := filesystem.NewFileLock(lockPath)

	// Use extended timeout for build operations (15 minutes)
	// Builds can take several minutes with multiple retry attempts
	const buildLockTimeout = 15 * time.Minute

	buildCtx, cancel := context.WithTimeout(ctx, buildLockTimeout)
	defer cancel()

	err := lock.Lock(buildCtx)
	if err != nil {
		return "", fmt.Errorf("failed to acquire build lock for %s: %w", versionName, err)
	}

	defer func() {
		unlockErr := lock.Unlock()
		if unlockErr != nil {
			logrus.Warnf("failed to unlock build lock for %s: %v", versionName, unlockErr)
		}
	}()

	return s.builder.BuildFromCommit(ctx, commit, dest, progress)
}

// resolveCommitToVersionName finds an existing version directory that matches the commit.
// It scans the dest directory for entries where version.txt contains the commit string.
func (s *Service) resolveCommitToVersionName(dest, commit string) string {
	// First, try using the commit as-is (for short hashes or already-resolved names)
	entries, err := os.ReadDir(dest)
	if err != nil {
		return commit
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Skip special directories
		if name == "current" || name == "nightly" || strings.HasPrefix(name, ".") {
			continue
		}

		// Check if version.txt contains the commit
		versionFile := filepath.Join(dest, name, "version.txt")

		data, err := os.ReadFile(versionFile)
		if err != nil {
			continue
		}

		// Check if this version matches the commit (full or partial hash)
		storedCommit := strings.TrimSpace(string(data))
		if strings.HasPrefix(storedCommit, commit) || strings.HasPrefix(commit, storedCommit) {
			return name
		}
	}

	// If no match found, return the original commit
	// The builder will resolve it to a short hash
	return commit
}
