package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/pkg/archive"
)

var client = &http.Client{Timeout: 15 * time.Second}

// DownloadAndInstall downloads the asset, verifies its checksum (if available),
// extracts the archive to the proper directory and writes a version file.
func DownloadAndInstall(versionsDir, installName, assetURL, checksumURL, releaseIdentifier string) error {
	tmpFile, err := os.CreateTemp("", "nvim-*.archive")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := downloadFile(assetURL, tmpFile); err != nil {
		return fmt.Errorf("download error: %w", err)
	}

	if checksumURL != "" {
		logrus.Info("Verifying checksum...")
		if err := verifyChecksum(tmpFile, checksumURL); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		logrus.Info("Checksum verified successfully")
	}

	versionDir := filepath.Join(versionsDir, installName)
	if err := archive.ExtractArchive(tmpFile, versionDir); err != nil {
		return fmt.Errorf("extraction error: %w", err)
	}

	versionFile := filepath.Join(versionDir, "version.txt")
	if err := os.WriteFile(versionFile, []byte(releaseIdentifier), 0644); err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}
	return nil
}

func downloadFile(url string, dest *os.File) error {
	logrus.Debugf("Downloading asset from URL: %s", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}
	n, err := io.Copy(dest, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy download content: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("downloaded file is empty; check asset URL: %s", url)
	}
	return nil
}

func verifyChecksum(file *os.File, checksumURL string) error {
	req, err := http.NewRequest("GET", checksumURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create checksum request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download checksum file: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("checksum download failed with status %d", resp.StatusCode)
	}
	checksumData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read checksum data: %w", err)
	}
	expectedFields := strings.Fields(string(checksumData))
	if len(expectedFields) == 0 {
		return fmt.Errorf("checksum file is empty")
	}
	expectedHash := expectedFields[0]
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek file for checksum computation: %w", err)
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to reset file pointer: %w", err)
	}
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}
	return nil
}
