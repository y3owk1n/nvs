// Package installer provides functions for downloading and installing Neovim.
package installer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/pkg/archive"
	"github.com/y3owk1n/nvs/pkg/releases"
)

// Constants for installer operations.
const (
	SpinnerSpeed   = 100
	TimeoutSeconds = 30
	ProgressDiv    = 100
	FilePerm       = 0o644
)

// Errors for installer operations.
var (
	ErrDownloadFailed         = errors.New("download failed")
	ErrChecksumDownloadFailed = errors.New("checksum download failed")
	ErrChecksumFileEmpty      = errors.New("checksum file is empty")
	ErrChecksumMismatch       = errors.New("checksum mismatch")
)

// Client is the HTTP client for downloads.
var Client = &http.Client{
	Timeout: TimeoutSeconds * time.Second,
}

// ProgressReader wraps an io.Reader to report progress.
type ProgressReader struct {
	Reader   io.Reader
	Total    int64
	read     int64
	Callback func(int)
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	numBytes, err := pr.Reader.Read(p)

	pr.read += int64(numBytes)
	if pr.Callback != nil && pr.Total > 0 {
		progress := int((pr.read * ProgressDiv) / pr.Total)
		pr.Callback(progress)
	}

	return numBytes, err
}

// DownloadAndInstall downloads and installs a Neovim release.
func DownloadAndInstall(
	ctx context.Context,
	versionsDir, installName, assetURL, checksumURL, releaseIdentifier string,
	progressCallback func(progress int),
	phaseCallback func(phase string),
) error {
	tmpFile, err := os.CreateTemp("", "nvim-*.archive")
	// Return err
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		err := tmpFile.Close()
		if err != nil {
			logrus.Errorf("warning: failed to close tmp file: %v", err)
		}

		err = os.Remove(tmpFile.Name())
		if err != nil {
			logrus.Warnf("Failed to remove temporary file %s: %v", tmpFile.Name(), err)
		}
	}()

	if phaseCallback != nil {
		phaseCallback("Downloading asset...")
	}

	err = DownloadFile(ctx, assetURL, tmpFile, progressCallback)
	if err != nil {
		return fmt.Errorf("download error: %w", err)
	}

	if checksumURL != "" {
		if phaseCallback != nil {
			phaseCallback("Verifying checksum...")
		}

		logrus.Debug("Verifying checksum...")

		err := VerifyChecksum(ctx, tmpFile, checksumURL)
		if err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}

		logrus.Debug("Checksum verified successfully")
	}

	if phaseCallback != nil {
		phaseCallback("Extracting Archive...")
	}

	versionDir := filepath.Join(versionsDir, installName)

	err = archive.ExtractArchive(tmpFile, versionDir)
	if err != nil {
		return fmt.Errorf("extraction error: %w", err)
	}

	if phaseCallback != nil {
		phaseCallback("Writing version file...")
	}

	versionFile := filepath.Join(versionDir, "version.txt")

	err = os.WriteFile(versionFile, []byte(releaseIdentifier), FilePerm)
	if err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	return nil
}

// DownloadFile downloads the content from the given URL and writes it to dest.
// It uses a ProgressReader to report progress via the callback.
//
// Example usage:
//
//	ctx := context.Background()
//	dest, _ := os.Create("downloaded.archive")
//	defer dest.Close()
//	err := DownloadFile(ctx, "https://example.com/neovim.archive", dest, func(progress int) {
//	    fmt.Fprintf(os.Stdout,"Progress: %d%%\n", progress)
//	})
//	if err != nil {
//	    // handle error
//	}
func DownloadFile(
	ctx context.Context,
	url string,
	dest *os.File,
	callback func(progress int),
) error {
	logrus.Debugf("Downloading asset from URL: %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	req.Header.Set("User-Agent", "nvs")

	resp, err := Client.Do(req)
	// Return err
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			logrus.Warnf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w with status %d", ErrDownloadFailed, resp.StatusCode)
	}

	totalSize := resp.ContentLength
	progressReader := &ProgressReader{
		Reader:   resp.Body,
		Total:    totalSize,
		Callback: callback,
	}

	_, err = io.Copy(dest, progressReader)
	if err != nil {
		return fmt.Errorf("failed to copy download content: %w", err)
	}

	return nil
}

// VerifyChecksum downloads the expected checksum from checksumURL,
// computes the SHA256 checksum of the provided file, and compares them.
// The file pointer is reset after the computation.
//
// Example usage:
//
//	ctx := context.Background()
//	file, _ := os.Open("downloaded.archive")
//	defer file.Close()
//	err := VerifyChecksum(ctx, file, "https://example.com/neovim.archive.sha256")
//	if err != nil {
//	    // handle error
//	}
func VerifyChecksum(ctx context.Context, file *os.File, checksumURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create checksum request: %w", err)
	}

	resp, err := Client.Do(req)
	// Return err
	if err != nil {
		return fmt.Errorf("failed to download checksum file: %w", err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			logrus.Warnf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w with status %d", ErrChecksumDownloadFailed, resp.StatusCode)
	}

	checksumData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read checksum data: %w", err)
	}

	expectedFields := strings.Fields(string(checksumData))
	if len(expectedFields) == 0 {
		return ErrChecksumFileEmpty
	}

	expectedHash := expectedFields[0]

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek file for checksum computation: %w", err)
	}

	hasher := sha256.New()

	_, err = io.Copy(hasher, file)
	if err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to reset file pointer: %w", err)
	}

	if actualHash != expectedHash {
		return fmt.Errorf("%w: expected %s, got %s", ErrChecksumMismatch, expectedHash, actualHash)
	}

	return nil
}

// InstallVersion installs a version from a cached archive file.
// It extracts the archive to the version directory and writes the version file.
//
// Example usage:
//
//	err := InstallVersion(context.Background(), "v0.9.0", "/path/to/versions", "/path/to/cache/neovim-v0.9.0.tar.gz")
//	if err != nil {
//	    // handle error
//	}
func InstallVersion(
	ctx context.Context,
	alias, versionsDir, cacheFilePath string,
	progressCallback func(int),
) error {
	// Resolve the version
	release, err := releases.ResolveVersion(alias, cacheFilePath)
	if err != nil {
		return fmt.Errorf("failed to resolve version: %w", err)
	}

	// Get the asset URL
	assetURL, _, err := releases.GetAssetURL(release)
	if err != nil {
		return fmt.Errorf("failed to get asset URL: %w", err)
	}

	logrus.Debugf("Asset URL: %s", assetURL)

	// Create temporary file for download
	tmpFile, err := os.CreateTemp("", "nvim-*.archive")
	// Return err
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		err := tmpFile.Close()
		if err != nil {
			logrus.Warnf("Failed to close tmp file: %v", err)
		}

		err = os.Remove(tmpFile.Name())
		if err != nil {
			logrus.Warnf("Failed to remove temporary file %s: %v", tmpFile.Name(), err)
		}
	}()

	// Download the asset
	err = DownloadFile(ctx, assetURL, tmpFile, progressCallback)
	if err != nil {
		return fmt.Errorf("download error: %w", err)
	}

	// Extract the archive
	versionDir := filepath.Join(versionsDir, alias)

	err = archive.ExtractArchive(tmpFile, versionDir)
	if err != nil {
		return fmt.Errorf("extraction error: %w", err)
	}

	// Write version file
	versionFile := filepath.Join(versionDir, "version.txt")
	releaseIdentifier := releases.GetReleaseIdentifier(release, alias)

	err = os.WriteFile(versionFile, []byte(releaseIdentifier), FilePerm)
	if err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	return nil
}
