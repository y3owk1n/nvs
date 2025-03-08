package installer

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// createValidZipArchive creates a valid ZIP archive and returns its bytes.
func createValidZipArchive() []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	// Create at least one file entry (its content is irrelevant).
	f, _ := zw.Create("dummy.txt")
	f.Write([]byte("dummy content"))
	zw.Close()
	return buf.Bytes()
}

// createInvalidArchive returns bytes that do not represent a valid archive.
func createInvalidArchive() []byte {
	return []byte("not a valid archive")
}

// computeSHA256 returns the hex SHA256 sum of data.
func computeSHA256(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

// newAssetServer returns an httptest.Server that serves the given assetBytes.
func newAssetServer(assetBytes []byte, status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write(assetBytes)
	}))
}

// newChecksumServer returns an httptest.Server that serves the given checksum string.
func newChecksumServer(checksum string, status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		// The checksum file is assumed to have the checksum as the first field.
		fmt.Fprintln(w, checksum)
	}))
}

// TestDownloadAndInstall_Success_WithChecksum verifies that DownloadAndInstall works
// as expected when a valid asset and checksum are provided.
func TestDownloadAndInstall_Success_WithChecksum(t *testing.T) {
	assetBytes := createValidZipArchive()
	hash := computeSHA256(assetBytes)

	assetSrv := newAssetServer(assetBytes, http.StatusOK)
	defer assetSrv.Close()
	checksumSrv := newChecksumServer(hash, http.StatusOK)
	defer checksumSrv.Close()

	var progressUpdates []int
	var phaseUpdates []string
	var mu sync.Mutex

	progressCb := func(p int) {
		mu.Lock()
		progressUpdates = append(progressUpdates, p)
		mu.Unlock()
	}
	phaseCb := func(phase string) {
		mu.Lock()
		phaseUpdates = append(phaseUpdates, phase)
		mu.Unlock()
	}

	// Use a temporary versions directory.
	versionsDir := t.TempDir()
	installName := "test-install"
	releaseID := "v1.0.0"

	// Call DownloadAndInstall.
	err := DownloadAndInstall(versionsDir, installName, assetSrv.URL, checksumSrv.URL, releaseID, progressCb, phaseCb)
	if err != nil {
		t.Fatalf("DownloadAndInstall failed: %v", err)
	}

	// Verify that the version file is written correctly.
	versionFile := filepath.Join(versionsDir, installName, "version.txt")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatalf("failed to read version file: %v", err)
	}
	if string(data) != releaseID {
		t.Errorf("version file content = %q, want %q", string(data), releaseID)
	}

	// Check that at least one progress update was made.
	mu.Lock()
	if len(progressUpdates) == 0 {
		t.Errorf("expected progress updates, got none")
	}
	// And verify the phases were reported in order.
	expectedPhases := []string{
		"Downloading asset...",
		"Verifying checksum...",
		"Extracting Archive...",
		"Writing version file...",
	}
	if len(phaseUpdates) != len(expectedPhases) {
		t.Errorf("phase callback updates length = %d, want %d", len(phaseUpdates), len(expectedPhases))
	} else {
		for i, phase := range expectedPhases {
			if phaseUpdates[i] != phase {
				t.Errorf("phase update[%d] = %q, want %q", i, phaseUpdates[i], phase)
			}
		}
	}
	mu.Unlock()
}

// TestDownloadAndInstall_Success_NoChecksum verifies that DownloadAndInstall works
// when no checksum URL is provided.
func TestDownloadAndInstall_Success_NoChecksum(t *testing.T) {
	assetBytes := createValidZipArchive()
	assetSrv := newAssetServer(assetBytes, http.StatusOK)
	defer assetSrv.Close()

	versionsDir := t.TempDir()
	installName := "no-checksum-install"
	releaseID := "v2.0.0"

	phaseCalled := false
	phaseCb := func(phase string) {
		// We expect no "Verifying checksum..." phase if checksumURL is empty.
		if strings.Contains(phase, "Verifying checksum") {
			t.Errorf("checksum verification phase should not be called")
		}
		phaseCalled = true
	}

	err := DownloadAndInstall(versionsDir, installName, assetSrv.URL, "", releaseID, nil, phaseCb)
	if err != nil {
		t.Fatalf("DownloadAndInstall failed: %v", err)
	}

	versionFile := filepath.Join(versionsDir, installName, "version.txt")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatalf("failed to read version file: %v", err)
	}
	if string(data) != releaseID {
		t.Errorf("version file content = %q, want %q", string(data), releaseID)
	}

	if !phaseCalled {
		t.Errorf("phase callback was not called")
	}
}

// TestDownloadAndInstall_AssetDownloadError simulates a non-200 response when downloading asset.
func TestDownloadAndInstall_AssetDownloadError(t *testing.T) {
	assetSrv := newAssetServer([]byte("dummy"), http.StatusInternalServerError)
	defer assetSrv.Close()

	versionsDir := t.TempDir()
	installName := "error-install"
	releaseID := "v3.0.0"

	err := DownloadAndInstall(versionsDir, installName, assetSrv.URL, "", releaseID, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "download failed with status") {
		t.Errorf("expected asset download error, got: %v", err)
	}
}

// TestDownloadAndInstall_ChecksumDownloadError simulates a non-200 response when downloading checksum.
func TestDownloadAndInstall_ChecksumDownloadError(t *testing.T) {
	assetBytes := createValidZipArchive()
	assetSrv := newAssetServer(assetBytes, http.StatusOK)
	defer assetSrv.Close()

	// Create a checksum server that returns an error.
	checksumSrv := newChecksumServer("dummy", http.StatusInternalServerError)
	defer checksumSrv.Close()

	versionsDir := t.TempDir()
	installName := "checksum-error-install"
	releaseID := "v4.0.0"

	err := DownloadAndInstall(versionsDir, installName, assetSrv.URL, checksumSrv.URL, releaseID, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "checksum download failed") {
		t.Errorf("expected checksum download error, got: %v", err)
	}
}

// TestDownloadAndInstall_ChecksumMismatch simulates a checksum mismatch.
func TestDownloadAndInstall_ChecksumMismatch(t *testing.T) {
	assetBytes := createValidZipArchive()
	assetSrv := newAssetServer(assetBytes, http.StatusOK)
	defer assetSrv.Close()

	// Provide a wrong checksum.
	checksumSrv := newChecksumServer("wrongchecksum", http.StatusOK)
	defer checksumSrv.Close()

	versionsDir := t.TempDir()
	installName := "mismatch-install"
	releaseID := "v5.0.0"

	err := DownloadAndInstall(versionsDir, installName, assetSrv.URL, checksumSrv.URL, releaseID, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("expected checksum mismatch error, got: %v", err)
	}
}

// TestDownloadAndInstall_ExtractionError simulates an extraction error by serving an invalid archive.
func TestDownloadAndInstall_ExtractionError(t *testing.T) {
	// Use invalid archive bytes so that archive.ExtractArchive fails.
	assetBytes := createInvalidArchive()
	hash := computeSHA256(assetBytes)

	assetSrv := newAssetServer(assetBytes, http.StatusOK)
	defer assetSrv.Close()
	checksumSrv := newChecksumServer(hash, http.StatusOK)
	defer checksumSrv.Close()

	versionsDir := t.TempDir()
	installName := "extract-error-install"
	releaseID := "v6.0.0"

	err := DownloadAndInstall(versionsDir, installName, assetSrv.URL, checksumSrv.URL, releaseID, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "extraction error") {
		t.Errorf("expected extraction error, got: %v", err)
	}
}

// TestDownloadFile_CopyError simulates an error during file copy by closing the destination file early.
func TestDownloadFile_CopyError(t *testing.T) {
	// Create a test server that writes valid asset content.
	assetBytes := createValidZipArchive()
	ts := newAssetServer(assetBytes, http.StatusOK)
	defer ts.Close()

	tmpFile, err := os.CreateTemp("", "download-error-*.tmp")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	// Close the destination file so that io.Copy fails.
	tmpFile.Close()

	err = downloadFile(ts.URL, tmpFile, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to copy download content") {
		t.Errorf("expected copy error, got: %v", err)
	}
}

// TestVerifyChecksum_ReadError simulates a failure in reading the checksum data.
func TestVerifyChecksum_ReadError(t *testing.T) {
	assetBytes := createValidZipArchive()
	tmpFile, err := os.CreateTemp("", "verify-read-error-*.tmp")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write(assetBytes)
	tmpFile.Seek(0, io.SeekStart)

	// Create a checksum server that returns no data.
	checksumSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write nothing.
	}))
	defer checksumSrv.Close()

	err = verifyChecksum(tmpFile, checksumSrv.URL)
	if err == nil || !strings.Contains(err.Error(), "checksum file is empty") {
		t.Errorf("expected empty checksum error, got: %v", err)
	}
}

// TestVerifyChecksum_FileSeekError simulates a seek error by closing the file before checksum verification.
func TestVerifyChecksum_FileSeekError(t *testing.T) {
	assetBytes := createValidZipArchive()
	tmpFile, err := os.CreateTemp("", "verify-seek-error-*.tmp")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Write(assetBytes)
	// Close file to cause seek error.
	tmpFile.Close()

	checksumSrv := newChecksumServer("dummy", http.StatusOK)
	defer checksumSrv.Close()

	err = verifyChecksum(tmpFile, checksumSrv.URL)
	if err == nil || !strings.Contains(err.Error(), "failed to seek file for checksum computation") {
		t.Errorf("expected seek error, got: %v", err)
	}
}

// TestProgressReader verifies that progressReader calls the callback with proper percentages.
func TestProgressReader(t *testing.T) {
	data := []byte("0123456789") // 10 bytes
	reader := bytes.NewReader(data)
	var updates []int
	pr := &progressReader{
		reader: reader,
		total:  int64(len(data)),
		callback: func(progress int) {
			updates = append(updates, progress)
		},
	}
	buf := make([]byte, 4)
	// Read in chunks.
	for {
		_, err := pr.Read(buf)
		if err == io.EOF {
			break
		}
	}
	// Expect that the progress updates eventually reached 100.
	if len(updates) == 0 || updates[len(updates)-1] != 100 {
		t.Errorf("expected final progress to be 100, got %v", updates)
	}
}

// Since DownloadAndInstall uses a package-level http.Client, we ensure that its timeout is set.
func TestHTTPClientTimeout(t *testing.T) {
	if client.Timeout < 15*time.Second {
		t.Errorf("expected http.Client Timeout to be at least 15 seconds, got %v", client.Timeout)
	}
}
