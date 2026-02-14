package downloader_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/y3owk1n/nvs/internal/infra/downloader"
)

// TestDownloader_Download tests the Download function with a mock HTTP server.
func TestDownloader_Download(t *testing.T) {
	// Create a mock HTTP server
	expectedContent := "test file content for download"

	server := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, r *http.Request) {
			if r.Header.Get("User-Agent") != "nvs" {
				t.Errorf("Expected User-Agent 'nvs', got '%s'", r.Header.Get("User-Agent"))
			}

			responseWriter.Header().Set("Content-Length", "30")
			_, _ = responseWriter.Write([]byte(expectedContent))
		}),
	)
	defer server.Close()

	downloaderInstance := downloader.New()
	ctx := context.Background()

	// Create temp file
	tempFile, err := os.CreateTemp(t.TempDir(), "download-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer func() {
		err := tempFile.Close()
		if err != nil {
			t.Logf("close error: %v", err)
		}
	}()

	// Track progress
	var progressUpdates []int

	progressFn := func(percent int) {
		progressUpdates = append(progressUpdates, percent)
	}

	// Call Download
	err = downloaderInstance.Download(ctx, server.URL, tempFile, progressFn)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	// Verify content
	_, _ = tempFile.Seek(0, 0)
	content := make([]byte, len(expectedContent))

	_, err = tempFile.Read(content)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	if string(content) != expectedContent {
		t.Errorf("Downloaded content = %q, want %q", string(content), expectedContent)
	}

	// Verify progress was called
	if len(progressUpdates) == 0 {
		t.Error("Progress callback was never called")
	}
}

// TestDownloader_Download_HTTPError tests Download with HTTP error status.
func TestDownloader_Download_HTTPError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, r *http.Request) {
			responseWriter.WriteHeader(http.StatusNotFound)
		}),
	)
	defer server.Close()

	downloaderInstance := downloader.New()
	ctx := context.Background()

	tempFile, err := os.CreateTemp(t.TempDir(), "download-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer func() {
		err := tempFile.Close()
		if err != nil {
			t.Logf("close error: %v", err)
		}
	}()

	err = downloaderInstance.Download(ctx, server.URL, tempFile, nil)
	if err == nil {
		t.Error("Download() expected error for 404 status, got nil")
	}
}

// TestDownloader_Download_ContextCancellation tests Download with canceled context.
func TestDownloader_Download_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, r *http.Request) {
			// Slow response to allow context cancellation
			<-r.Context().Done()
		}),
	)
	defer server.Close()

	downloaderInstance := downloader.New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	tempFile, err := os.CreateTemp(t.TempDir(), "download-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer func() {
		err := tempFile.Close()
		if err != nil {
			t.Logf("close error: %v", err)
		}
	}()

	err = downloaderInstance.Download(ctx, server.URL, tempFile, nil)
	if err == nil {
		t.Error("Download() expected error for canceled context, got nil")
	}
}

// TestDownloader_VerifyChecksum tests checksum verification with a mock server.
func TestDownloader_VerifyChecksum(t *testing.T) {
	// Create test file content
	testContent := []byte("test content for checksum verification")

	// Calculate actual SHA256
	hasher := sha256.New()
	hasher.Write(testContent)
	expectedHash := hex.EncodeToString(hasher.Sum(nil))

	// Create mock checksum server
	server := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, r *http.Request) {
			// Return checksum in "hash filename" format
			_, _ = responseWriter.Write([]byte(expectedHash + "  test-file.tar.gz"))
		}),
	)
	defer server.Close()

	downloaderInstance := downloader.New()
	ctx := context.Background()

	// Create temp file with test content
	tempFile, err := os.CreateTemp(t.TempDir(), "checksum-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer func() {
		err := tempFile.Close()
		if err != nil {
			t.Logf("close error: %v", err)
		}
	}()

	_, err = tempFile.Write(testContent)
	if err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}

	// Verify checksum
	err = downloaderInstance.VerifyChecksum(ctx, tempFile, server.URL, "test-file.tar.gz")
	if err != nil {
		t.Errorf("VerifyChecksum() error = %v", err)
	}
}

// TestDownloader_VerifyChecksum_Mismatch tests checksum mismatch detection.
func TestDownloader_VerifyChecksum_Mismatch(t *testing.T) {
	// Create test file content
	testContent := []byte("test content")

	// Return wrong hash
	wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"

	server := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, r *http.Request) {
			_, _ = responseWriter.Write([]byte(wrongHash + "  test-file.tar.gz"))
		}),
	)
	defer server.Close()

	downloaderInstance := downloader.New()
	ctx := context.Background()

	tempFile, err := os.CreateTemp(t.TempDir(), "checksum-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer func() {
		err := tempFile.Close()
		if err != nil {
			t.Logf("close error: %v", err)
		}
	}()

	_, _ = tempFile.Write(testContent)

	err = downloaderInstance.VerifyChecksum(ctx, tempFile, server.URL, "test-file.tar.gz")
	if err == nil {
		t.Error("VerifyChecksum() expected error for checksum mismatch, got nil")
	}
}

// TestDownloader_VerifyChecksum_ShasumTxt tests shasum.txt format parsing.
func TestDownloader_VerifyChecksum_ShasumTxt(t *testing.T) {
	testContent := []byte("test content for shasum")

	hasher := sha256.New()
	hasher.Write(testContent)
	expectedHash := hex.EncodeToString(hasher.Sum(nil))

	// shasum.txt format with multiple entries
	shasumContent := "0000000000000000000000000000000000000000000000000000000000000001  other-file.tar.gz\n" +
		expectedHash + "  target-file.tar.gz\n" +
		"0000000000000000000000000000000000000000000000000000000000000002  another-file.tar.gz"

	server := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, r *http.Request) {
			_, _ = responseWriter.Write([]byte(shasumContent))
		}),
	)
	defer server.Close()

	downloaderInstance := downloader.New()
	ctx := context.Background()

	tempFile, err := os.CreateTemp(t.TempDir(), "checksum-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer func() {
		err := tempFile.Close()
		if err != nil {
			t.Logf("close error: %v", err)
		}
	}()

	_, _ = tempFile.Write(testContent)

	// Use URL ending in shasum.txt to trigger shasum.txt parsing
	err = downloaderInstance.VerifyChecksum(
		ctx,
		tempFile,
		server.URL+"/shasum.txt",
		"target-file.tar.gz",
	)
	if err != nil {
		t.Errorf("VerifyChecksum() error = %v", err)
	}
}

// TestDownloader_VerifyChecksum_EmptyFile tests empty checksum file handling.
func TestDownloader_VerifyChecksum_EmptyFile(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, r *http.Request) {
			// Return empty response
		}),
	)
	defer server.Close()

	downloaderInstance := downloader.New()
	ctx := context.Background()

	tempFile, err := os.CreateTemp(t.TempDir(), "checksum-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer func() {
		err := tempFile.Close()
		if err != nil {
			t.Logf("close error: %v", err)
		}
	}()

	_, _ = tempFile.WriteString("test")

	err = downloaderInstance.VerifyChecksum(ctx, tempFile, server.URL, "test.tar.gz")
	if err == nil {
		t.Error("VerifyChecksum() expected error for empty checksum file, got nil")
	}
}

// TestNew tests the New constructor.
func TestNew(t *testing.T) {
	downloaderInstance := downloader.New()
	if downloaderInstance == nil {
		t.Error("New() returned nil")
	}
}

// TestDownloader_DownloadWithChecksumVerification tests successful download with checksum verification.
func TestDownloader_DownloadWithChecksumVerification(t *testing.T) {
	expectedContent := "test file content for download with checksum"

	hasher := sha256.New()
	hasher.Write([]byte(expectedContent))
	expectedHash := hex.EncodeToString(hasher.Sum(nil))

	server := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, r *http.Request) {
			if r.Header.Get("User-Agent") != "nvs" {
				t.Errorf("Expected User-Agent 'nvs', got '%s'", r.Header.Get("User-Agent"))
			}

			_, _ = responseWriter.Write([]byte(expectedContent))
		}),
	)
	defer server.Close()

	checksumServer := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, r *http.Request) {
			_, _ = responseWriter.Write([]byte(expectedHash + "  test-file.tar.gz"))
		}),
	)
	defer checksumServer.Close()

	downloaderInstance := downloader.New()
	ctx := context.Background()

	tempFile, err := os.CreateTemp(t.TempDir(), "download-checksum-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer func() {
		err := tempFile.Close()
		if err != nil {
			t.Logf("close error: %v", err)
		}
	}()

	var progressUpdates []int

	progressFn := func(percent int) {
		progressUpdates = append(progressUpdates, percent)
	}

	err = downloaderInstance.DownloadWithChecksumVerification(
		ctx,
		server.URL,
		checksumServer.URL,
		"test-file.tar.gz",
		tempFile,
		progressFn,
	)
	if err != nil {
		t.Fatalf("DownloadWithChecksumVerification() error = %v", err)
	}

	_, _ = tempFile.Seek(0, 0)

	content, err := os.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	if string(content) != expectedContent {
		t.Errorf("Downloaded content = %q, want %q", string(content), expectedContent)
	}

	if len(progressUpdates) == 0 {
		t.Error("Progress callback was never called")
	}
}

// TestDownloader_DownloadWithChecksumVerification_Mismatch tests checksum mismatch detection.
func TestDownloader_DownloadWithChecksumVerification_Mismatch(t *testing.T) {
	expectedContent := "test file content"

	wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"

	server := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, r *http.Request) {
			_, _ = responseWriter.Write([]byte(expectedContent))
		}),
	)
	defer server.Close()

	checksumServer := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, r *http.Request) {
			_, _ = responseWriter.Write([]byte(wrongHash + "  test-file.tar.gz"))
		}),
	)
	defer checksumServer.Close()

	downloaderInstance := downloader.New()
	ctx := context.Background()

	tempFile, err := os.CreateTemp(t.TempDir(), "download-checksum-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer func() {
		err := tempFile.Close()
		if err != nil {
			t.Logf("close error: %v", err)
		}
	}()

	err = downloaderInstance.DownloadWithChecksumVerification(
		ctx,
		server.URL,
		checksumServer.URL,
		"test-file.tar.gz",
		tempFile,
		nil,
	)
	if err == nil {
		t.Error("DownloadWithChecksumVerification() expected error for checksum mismatch, got nil")
	}
}

// TestDownloader_DownloadWithChecksumVerification_HTTPError tests HTTP error handling.
func TestDownloader_DownloadWithChecksumVerification_HTTPError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, r *http.Request) {
			responseWriter.WriteHeader(http.StatusNotFound)
		}),
	)
	defer server.Close()

	checksumServer := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, r *http.Request) {
			_, _ = responseWriter.Write(
				[]byte(
					"abc123def456789012345678901234567890123456789012345678901234  test-file.tar.gz",
				),
			)
		}),
	)
	defer checksumServer.Close()

	downloaderInstance := downloader.New()
	ctx := context.Background()

	tempFile, err := os.CreateTemp(t.TempDir(), "download-checksum-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer func() {
		err := tempFile.Close()
		if err != nil {
			t.Logf("close error: %v", err)
		}
	}()

	err = downloaderInstance.DownloadWithChecksumVerification(
		ctx,
		server.URL,
		checksumServer.URL,
		"test-file.tar.gz",
		tempFile,
		nil,
	)
	if err == nil {
		t.Error("DownloadWithChecksumVerification() expected error for 404 status, got nil")
	}
}

// TestDownloader_DownloadWithChecksumVerification_ChecksumHTTPError tests checksum server HTTP error.
func TestDownloader_DownloadWithChecksumVerification_ChecksumHTTPError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, r *http.Request) {
			_, _ = responseWriter.Write([]byte("test content"))
		}),
	)
	defer server.Close()

	checksumServer := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, r *http.Request) {
			responseWriter.WriteHeader(http.StatusNotFound)
		}),
	)
	defer checksumServer.Close()

	downloaderInstance := downloader.New()
	ctx := context.Background()

	tempFile, err := os.CreateTemp(t.TempDir(), "download-checksum-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer func() {
		err := tempFile.Close()
		if err != nil {
			t.Logf("close error: %v", err)
		}
	}()

	err = downloaderInstance.DownloadWithChecksumVerification(
		ctx,
		server.URL,
		checksumServer.URL,
		"test-file.tar.gz",
		tempFile,
		nil,
	)
	if err == nil {
		t.Error(
			"DownloadWithChecksumVerification() expected error for checksum server 404, got nil",
		)
	}
}

// TestDownloader_DownloadWithChecksumVerification_ContextCancellation tests context cancellation.
func TestDownloader_DownloadWithChecksumVerification_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}),
	)
	defer server.Close()

	checksumServer := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, r *http.Request) {
			_, _ = responseWriter.Write(
				[]byte(
					"abc123def456789012345678901234567890123456789012345678901234  test-file.tar.gz",
				),
			)
		}),
	)
	defer checksumServer.Close()

	downloaderInstance := downloader.New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tempFile, err := os.CreateTemp(t.TempDir(), "download-checksum-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer func() {
		err := tempFile.Close()
		if err != nil {
			t.Logf("close error: %v", err)
		}
	}()

	err = downloaderInstance.DownloadWithChecksumVerification(
		ctx,
		server.URL,
		checksumServer.URL,
		"test-file.tar.gz",
		tempFile,
		nil,
	)
	if err == nil {
		t.Error("DownloadWithChecksumVerification() expected error for canceled context, got nil")
	}
}
