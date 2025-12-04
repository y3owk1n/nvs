// Package downloader provides file download functionality with progress tracking.
package downloader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	clientTimeoutSec = 30
	progressDiv      = 100
)

// Downloader handles file downloads with progress tracking.
type Downloader struct {
	httpClient *http.Client
}

// New creates a new Downloader instance.
func New() *Downloader {
	return &Downloader{
		httpClient: &http.Client{Timeout: clientTimeoutSec * time.Second},
	}
}

// ProgressFunc is a callback for download progress updates.
type ProgressFunc func(percent int)

// Download downloads a file from the given URL to the destination file.
func (d *Downloader) Download(
	ctx context.Context,
	url string,
	dest *os.File,
	progress ProgressFunc,
) error {
	logrus.Debugf("Downloading from URL: %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "nvs")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrDownloadFailed, resp.StatusCode)
	}

	totalSize := resp.ContentLength
	progressReader := &progressReader{
		reader:   resp.Body,
		total:    totalSize,
		callback: progress,
	}

	_, err = io.Copy(dest, progressReader)
	if err != nil {
		return fmt.Errorf("failed to copy download content: %w", err)
	}

	return nil
}

// VerifyChecksum downloads the checksum file and verifies the file's SHA256 hash.
func (d *Downloader) VerifyChecksum(ctx context.Context, file *os.File, checksumURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create checksum request: %w", err)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download checksum: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrChecksumDownloadFailed, resp.StatusCode)
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

	// Compute actual hash
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	hasher := sha256.New()

	_, err = io.Copy(hasher, file)
	if err != nil {
		return fmt.Errorf("failed to hash file: %w", err)
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))

	// Reset file pointer
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	if actualHash != expectedHash {
		return fmt.Errorf("%w: expected %s, got %s", ErrChecksumMismatch, expectedHash, actualHash)
	}

	return nil
}

// progressReader wraps an io.Reader to report progress.
type progressReader struct {
	reader   io.Reader
	total    int64
	read     int64
	callback ProgressFunc
}

func (pr *progressReader) Read(p []byte) (int, error) {
	bytesRead, err := pr.reader.Read(p)

	pr.read += int64(bytesRead)
	if pr.callback != nil && pr.total > 0 {
		percent := int((pr.read * progressDiv) / pr.total)
		pr.callback(percent)
	}

	return bytesRead, err
}
