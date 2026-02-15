// Package installer provides the infrastructure implementation for Neovim installation.
package installer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

	err = lock.LockWithDefaultTimeout()
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
// Note: No fast-path check is performed here because the builder resolves the
// commit to a short hash internally, making it difficult to check for existing
// builds without performing git operations. The lock provides sufficient coordination.
func (s *Service) BuildFromCommit(
	ctx context.Context,
	commit string,
	dest string,
	progress installer.ProgressFunc,
) (string, error) {
	// Acquire per-version lock to prevent concurrent operations on the same commit
	// The commit hash becomes the version name
	lockPath := filepath.Join(dest, fmt.Sprintf(".nvs-version-%s.lock", commit))
	lock := filesystem.NewFileLock(lockPath)

	// Use extended timeout for build operations (15 minutes)
	// Builds can take several minutes with multiple retry attempts
	const buildLockTimeout = 15 * time.Minute

	buildCtx, cancel := context.WithTimeout(ctx, buildLockTimeout)
	defer cancel()

	err := lock.Lock(buildCtx)
	if err != nil {
		return "", fmt.Errorf("failed to acquire build lock for commit %s: %w", commit, err)
	}

	defer func() {
		unlockErr := lock.Unlock()
		if unlockErr != nil {
			logrus.Warnf("failed to unlock build lock for commit %s: %v", commit, unlockErr)
		}
	}()

	return s.builder.BuildFromCommit(ctx, commit, dest, progress)
}
