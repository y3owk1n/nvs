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

type progressReader struct {
	reader   io.Reader
	total    int64
	current  int64
	callback func(progress int)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.current += int64(n)
	if pr.total > 0 && pr.callback != nil {
		percent := int(float64(pr.current) / float64(pr.total) * 100)
		pr.callback(percent)
	}
	return n, err
}

// DownloadAndInstall downloads the asset, verifies its checksum (if available),
// extracts the archive to the proper directory and writes a version file.
func DownloadAndInstall(versionsDir, installName, assetURL, checksumURL, releaseIdentifier string, progressCallback func(progress int), phaseCallback func(phase string)) error {
	tmpFile, err := os.CreateTemp("", "nvim-*.archive")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if phaseCallback != nil {
		phaseCallback("Downloading asset...")
	}

	if err := downloadFile(assetURL, tmpFile, progressCallback); err != nil {
		return fmt.Errorf("download error: %w", err)
	}

	if phaseCallback != nil {
		phaseCallback("Verifying checksum...")
	}

	if checksumURL != "" {
		logrus.Debug("Verifying checksum...")
		if err := verifyChecksum(tmpFile, checksumURL); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		logrus.Debug("Checksum verified successfully")
	}

	if phaseCallback != nil {
		phaseCallback("Extracting Archive...")
	}

	versionDir := filepath.Join(versionsDir, installName)
	if err := archive.ExtractArchive(tmpFile, versionDir); err != nil {
		return fmt.Errorf("extraction error: %w", err)
	}

	if phaseCallback != nil {
		phaseCallback("Writing version file...")
	}

	versionFile := filepath.Join(versionDir, "version.txt")
	if err := os.WriteFile(versionFile, []byte(releaseIdentifier), 0644); err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}
	return nil
}

func downloadFile(url string, dest *os.File, callback func(progress int)) error {
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

	total := resp.ContentLength
	pr := &progressReader{
		reader:   resp.Body,
		total:    total,
		callback: callback,
	}

	if _, err := io.Copy(dest, pr); err != nil {
		return fmt.Errorf("failed to copy download content: %w", err)
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
