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

	// Perform the actual installation
	return s.installReleaseInternal(ctx, rel, dest, installName, progress)
}

// BuildFromCommit builds Neovim from source with per-version locking.
// Uses the same per-version lock as Install, Switch, and Uninstall for coordination.
//
// The lock uses the short hash version name to ensure consistency
// with Switch/Uninstall operations, which use the directory name.
func (s *Service) BuildFromCommit(
	ctx context.Context,
	commit string,
	dest string,
	progress installer.ProgressFunc,
) (string, error) {
	// Compute the short hash for the lock key.
	// The builder always creates the version directory with a 7-character short hash,
	// so we must use the same short hash for locking to coordinate with Switch/Uninstall.
	versionName := commit
	if len(commit) > constants.ShortCommitLen {
		versionName = commit[:constants.ShortCommitLen]
	}

	// Acquire per-version lock using the short hash
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

// UpgradeRelease upgrades an existing installation to a new release atomically.
// It acquires the per-version lock before renaming the existing version,
// ensuring no concurrent operations interfere with the upgrade.
func (s *Service) UpgradeRelease(
	ctx context.Context,
	release installer.ReleaseInfo,
	dest string,
	installName string,
	progress installer.ProgressFunc,
) error {
	// Acquire per-version lock BEFORE any filesystem operations
	// This ensures atomic upgrade with proper coordination
	lockPath := filepath.Join(dest, fmt.Sprintf(".nvs-version-%s.lock", installName))
	lock := filesystem.NewFileLock(lockPath)

	// Use context-aware lock with extended timeout (10 minutes)
	const upgradeLockTimeout = 10 * time.Minute

	upgradeCtx, cancel := context.WithTimeout(ctx, upgradeLockTimeout)
	defer cancel()

	err := lock.Lock(upgradeCtx)
	if err != nil {
		return fmt.Errorf("failed to acquire upgrade lock for %s: %w", installName, err)
	}

	defer func() {
		unlockErr := lock.Unlock()
		if unlockErr != nil {
			logrus.Warnf("failed to unlock upgrade lock for %s: %v", installName, unlockErr)
		}
	}()

	// Now perform the upgrade atomically while holding the lock
	return s.upgradeReleaseInternal(ctx, release, dest, installName, progress)
}

// upgradeReleaseInternal performs the actual upgrade after acquiring the lock.
func (s *Service) upgradeReleaseInternal(
	ctx context.Context,
	release installer.ReleaseInfo,
	dest string,
	installName string,
	progress installer.ProgressFunc,
) (retErr error) {
	versionPath := filepath.Join(dest, installName)
	backupPath := versionPath + ".backup"

	// Backup existing version
	retErr = os.Rename(versionPath, backupPath)
	if retErr != nil {
		return fmt.Errorf("failed to backup version: %w", retErr)
	}

	upgradeSuccess := false

	// Cleanup backup on success or restore on failure
	defer func() {
		if upgradeSuccess {
			// Upgrade succeeded, remove backup
			removeErr := os.RemoveAll(backupPath)
			if removeErr != nil {
				logrus.Errorf("Failed to remove backup after successful upgrade: %v", removeErr)
			}
		} else {
			// Upgrade failed, restore backup
			var rollbackErr error

			removeErr := os.RemoveAll(versionPath)
			if removeErr != nil {
				logrus.Errorf("Failed to clean partial install during rollback: %v", removeErr)
				rollbackErr = fmt.Errorf("failed to clean partial install: %w", removeErr)
			}

			// Only attempt rename if cleanup succeeded
			if removeErr == nil {
				renameErr := os.Rename(backupPath, versionPath)
				if renameErr != nil {
					logrus.Errorf("Failed to restore backup during rollback: %v", renameErr)
					rollbackErr = fmt.Errorf("failed to restore backup: %w", renameErr)
				}
			}

			// If upgrade failed and rollback also failed, wrap the original error
			if rollbackErr != nil && retErr != nil {
				retErr = fmt.Errorf(
					"%w (CRITICAL: rollback also failed: %w)",
					retErr,
					rollbackErr,
				)
			}
		}
	}()

	// Install new version (use internal method since lock is already held)
	retErr = s.installReleaseInternal(ctx, release, dest, installName, progress)
	if retErr != nil {
		return fmt.Errorf("failed to install release: %w", retErr)
	}

	upgradeSuccess = true

	return nil
}

// installReleaseInternal performs the actual installation without locking.
// This is called by InstallRelease (which handles locking) and UpgradeRelease (which already holds the lock).
func (s *Service) installReleaseInternal(
	ctx context.Context,
	rel installer.ReleaseInfo,
	dest string,
	installName string,
	progress installer.ProgressFunc,
) error {
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
		// Clean up the created directory on seek failure
		cleanupErr := os.RemoveAll(installPath)
		if cleanupErr != nil {
			logrus.Warnf("Failed to clean up directory after seek failure: %v", cleanupErr)
		}

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
	versionFile := filepath.Join(installPath, "version.txt")

	err = os.WriteFile(versionFile, []byte(rel.GetIdentifier()), constants.FilePerm)
	if err != nil {
		logrus.Warnf("Failed to write version file: %v", err)
	}

	if progress != nil {
		progress("Complete", constants.ProgressComplete)
	}

	return nil
}
