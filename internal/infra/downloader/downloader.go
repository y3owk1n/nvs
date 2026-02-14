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

	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/internal/constants"
)

// Downloader handles file downloads with progress tracking.
type Downloader struct {
	httpClient *http.Client
}

// New creates a new Downloader instance.
func New() *Downloader {
	return &Downloader{
		httpClient: &http.Client{
			Timeout: constants.DefaultTimeout,
		},
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

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close response body: %v", err)
		}
	}()

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

// DownloadWithChecksumVerification downloads a file and verifies its checksum.
// The hash is computed in a single pass during download, avoiding a separate
// file read to compute the hash afterwards.
func (d *Downloader) DownloadWithChecksumVerification(
	ctx context.Context,
	url string,
	checksumURL string,
	assetName string,
	dest *os.File,
	progress ProgressFunc,
) error {
	logrus.Debugf("Downloading from URL: %s", url)

	expectedHash, err := d.fetchExpectedHash(ctx, checksumURL, assetName)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "nvs")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrDownloadFailed, resp.StatusCode)
	}

	totalSize := resp.ContentLength
	hasher := sha256.New()

	progressReader := &progressReader{
		reader:   io.TeeReader(resp.Body, hasher),
		total:    totalSize,
		callback: progress,
	}

	_, err = io.Copy(dest, progressReader)
	if err != nil {
		return fmt.Errorf("failed to copy download content: %w", err)
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))

	if !strings.EqualFold(actualHash, expectedHash) {
		_ = dest.Truncate(0)
		_, _ = dest.Seek(0, io.SeekStart)

		return fmt.Errorf("%w: expected %s, got %s", ErrChecksumMismatch, expectedHash, actualHash)
	}

	return nil
}

// VerifyChecksum verifies the file's SHA256 hash against a downloaded checksum file.
func (d *Downloader) VerifyChecksum(
	ctx context.Context,
	file *os.File,
	checksumURL string,
	assetName string,
) error {
	expectedHash, err := d.fetchExpectedHash(ctx, checksumURL, assetName)
	if err != nil {
		return err
	}

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

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	if !strings.EqualFold(actualHash, expectedHash) {
		return fmt.Errorf("%w: expected %s, got %s", ErrChecksumMismatch, expectedHash, actualHash)
	}

	return nil
}

func (d *Downloader) fetchExpectedHash(
	ctx context.Context,
	checksumURL string,
	assetName string,
) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create checksum request: %w", err)
	}

	req.Header.Set("User-Agent", "nvs")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download checksum: %w", err)
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			logrus.Warnf("failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status %d", ErrChecksumDownloadFailed, resp.StatusCode)
	}

	checksumData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read checksum data: %w", err)
	}

	var expectedHash string

	if strings.HasSuffix(checksumURL, "shasum.txt") {
		lines := strings.SplitSeq(strings.TrimSpace(string(checksumData)), "\n")
		for line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[1] == assetName {
				expectedHash = fields[0]

				break
			}
		}

		if expectedHash == "" {
			return "", fmt.Errorf("%w: %s not found in shasum.txt", ErrChecksumNotFound, assetName)
		}
	} else {
		expectedFields := strings.Fields(string(checksumData))
		if len(expectedFields) == 0 {
			return "", ErrChecksumFileEmpty
		}

		if len(expectedFields) >= 2 && expectedFields[1] != assetName {
			return "", fmt.Errorf(
				"%w: expected %s, got %s",
				ErrChecksumNotFound,
				assetName,
				expectedFields[1],
			)
		}

		expectedHash = expectedFields[0]
	}

	if len(expectedHash) != constants.Sha256HashLen {
		return "", fmt.Errorf(
			"invalid checksum format: expected %d characters, got %d: %w",
			constants.Sha256HashLen,
			len(expectedHash),
			ErrInvalidChecksumFormat,
		)
	}

	return expectedHash, nil
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
		percent := int((pr.read * 100) / pr.total) //nolint:mnd // 100 for percentage calculation
		pr.callback(percent)
	}

	return bytesRead, err
}
