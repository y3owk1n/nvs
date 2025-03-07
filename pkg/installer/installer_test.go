package installer

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// createTestZipArchive creates a zip archive in memory with one file "dummy.txt"
// and returns its content as a byte slice.
func createTestZipArchive() ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	header := &zip.FileHeader{
		Name:   "dummy.txt",
		Method: zip.Deflate,
	}
	header.SetMode(0600)
	w, err := zw.CreateHeader(header)
	if err != nil {
		return nil, err
	}
	_, err = w.Write([]byte("Hello, installer!"))
	if err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// TestDownloadAndInstall tests the DownloadAndInstall function by using an httptest server.
func TestDownloadAndInstall(t *testing.T) {
	// Create a test zip archive.
	assetBytes, err := createTestZipArchive()
	if err != nil {
		t.Fatalf("failed to create test zip archive: %v", err)
	}

	// Compute the SHA256 checksum.
	hasher := sha256.New()
	hasher.Write(assetBytes)
	checksum := hex.EncodeToString(hasher.Sum(nil))
	// The checksum file content: you could include additional info (like filename) if needed.
	checksumContent := checksum + " dummy.txt"

	// Setup an HTTP test server.
	// We'll use a simple handler that returns the asset bytes on "/asset"
	// and returns the checksum on "/checksum".
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.URL.Path {
		case "/asset":
			w.Header().Set("Content-Length", string(rune(len(assetBytes))))
			// Write the asset content.
			_, err := w.Write(assetBytes)
			if err != nil {
				t.Logf("error writing asset: %v", err)
			}
		case "/checksum":
			_, err := w.Write([]byte(checksumContent))
			if err != nil {
				t.Logf("error writing checksum: %v", err)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Create a temporary directory to simulate the versionsDir.
	versionsDir, err := os.MkdirTemp("", "installer-test-")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(versionsDir)

	installName := "test-install"
	releaseIdentifier := "v1.0.0"

	// Dummy progress and phase callbacks that just record events.
	progressUpdates := []int{}
	phaseUpdates := []string{}
	progressCallback := func(p int) {
		progressUpdates = append(progressUpdates, p)
	}
	phaseCallback := func(phase string) {
		phaseUpdates = append(phaseUpdates, phase)
	}

	// Call DownloadAndInstall with the test server's URLs.
	assetURL := server.URL + "/asset"
	checksumURL := server.URL + "/checksum"

	err = DownloadAndInstall(versionsDir, installName, assetURL, checksumURL, releaseIdentifier, progressCallback, phaseCallback)
	if err != nil {
		t.Fatalf("DownloadAndInstall failed: %v", err)
	}

	// Verify that the version file was created correctly.
	versionFile := filepath.Join(versionsDir, installName, "version.txt")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatalf("failed to read version file: %v", err)
	}
	if string(data) != releaseIdentifier {
		t.Fatalf("expected version file content %q, got %q", releaseIdentifier, string(data))
	}

	// (Optional) Log the callbacks events for further debugging.
	t.Logf("Phase updates: %v", phaseUpdates)
	t.Logf("Progress updates: %v", progressUpdates)
}
