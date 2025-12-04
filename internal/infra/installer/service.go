// Package installer provides the infrastructure implementation for Neovim installation.
package installer

import (
	"context"
	"fmt"
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
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

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
	if checksumURL, err := rel.GetChecksumURL(); err == nil && checksumURL != "" {
		if progress != nil {
			progress("Verifying", 0)
		}
		if err := s.downloader.VerifyChecksum(ctx, tempFile, checksumURL); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	// 5. Extract
	if progress != nil {
		progress("Extracting", 0)
	}

	// Create destination directory
	installPath := filepath.Join(dest, rel.GetIdentifier())
	if err := os.MkdirAll(installPath, dirPerm); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	if err := s.extractor.Extract(tempFile, installPath); err != nil {
		// Cleanup on failure
		os.RemoveAll(installPath)
		return fmt.Errorf("extraction failed: %w", err)
	}

	// 6. Write version file
	versionFile := filepath.Join(installPath, "version.txt")
	if err := os.WriteFile(versionFile, []byte(rel.GetIdentifier()), filePerm); err != nil {
		logrus.Warnf("Failed to write version file: %v", err)
	}

	if progress != nil {
		progress("Complete", 100)
	}

	return nil
}

// BuildFromCommit builds Neovim from source.
func (s *Service) BuildFromCommit(ctx context.Context, commit string, dest string) error {
	return s.builder.BuildFromCommit(ctx, commit, dest)
}
