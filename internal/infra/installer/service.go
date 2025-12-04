// Package installer provides the infrastructure implementation for Neovim installation.
package installer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/internal/domain/installer"
	"github.com/y3owk1n/nvs/internal/infra/archive"
	"github.com/y3owk1n/nvs/internal/infra/builder"
	"github.com/y3owk1n/nvs/internal/infra/downloader"
)

const (
	filePerm = 0o644
	dirPerm  = 0o755
	// ProgressComplete is the value for completed progress.
	ProgressComplete = 100
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

// InstallRelease installs a pre-built release.
func (s *Service) InstallRelease(
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

	// 4. Verify checksum (optional, if URL available)
	checksumURL, err := rel.GetChecksumURL()
	if err == nil && checksumURL != "" {
		if progress != nil {
			progress("Verifying", 0)
		}

		err := s.downloader.VerifyChecksum(ctx, tempFile, checksumURL)
		if err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	// 5. Extract
	if progress != nil {
		progress("Extracting", 0)
	}

	// Create destination directory
	installPath := filepath.Join(dest, installName)

	err = os.MkdirAll(installPath, dirPerm)
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

	// 6. Write version file
	versionFile := filepath.Join(installPath, "version.txt")

	err = os.WriteFile(versionFile, []byte(rel.GetIdentifier()), filePerm)
	if err != nil {
		logrus.Warnf("Failed to write version file: %v", err)
	}

	if progress != nil {
		progress("Complete", ProgressComplete)
	}

	return nil
}

// BuildFromCommit builds Neovim from source.
func (s *Service) BuildFromCommit(ctx context.Context, commit string, dest string) error {
	return s.builder.BuildFromCommit(ctx, commit, dest)
}
