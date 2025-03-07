package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// createTarGzArchive creates a temporary tar.gz archive with one file "test.txt".
func createTarGzArchive() (*os.File, error) {
	tmpFile, err := os.CreateTemp("", "test-archive-*.tar.gz")
	if err != nil {
		return nil, err
	}

	// Create gzip and tar writers.
	gw := gzip.NewWriter(tmpFile)
	tw := tar.NewWriter(gw)

	// File content and header.
	content := []byte("Hello, tar.gz!")
	hdr := &tar.Header{
		Name: "test.txt",
		Mode: 0600,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write(content); err != nil {
		return nil, err
	}

	// Close writers to flush the archive.
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}

	// Seek back to the beginning for reading.
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	return tmpFile, nil
}

// createZipArchive creates a temporary zip archive with one file "test.txt".
func createZipArchive() (*os.File, error) {
	tmpFile, err := os.CreateTemp("", "test-archive-*.zip")
	if err != nil {
		return nil, err
	}

	zw := zip.NewWriter(tmpFile)
	// Prepare file header.
	header := &zip.FileHeader{
		Name:   "test.txt",
		Method: zip.Deflate,
	}
	header.SetMode(0600)
	w, err := zw.CreateHeader(header)
	if err != nil {
		return nil, err
	}
	// Write file content.
	if _, err := w.Write([]byte("Hello, zip!")); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}

	// Seek back to the beginning.
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	return tmpFile, nil
}

// TestExtractArchiveTarGz tests extraction of a tar.gz archive.
func TestExtractArchiveTarGz(t *testing.T) {
	// Create a temporary tar.gz archive.
	f, err := createTarGzArchive()
	if err != nil {
		t.Fatalf("Failed to create tar.gz archive: %v", err)
	}
	defer os.Remove(f.Name())

	// Create a temporary directory to extract to.
	destDir, err := os.MkdirTemp("", "extract-test-")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Call the extraction function.
	if err := ExtractArchive(f, destDir); err != nil {
		t.Fatalf("ExtractArchive failed: %v", err)
	}

	// Verify that "test.txt" exists and has the expected content.
	extractedPath := filepath.Join(destDir, "test.txt")
	data, err := os.ReadFile(extractedPath)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	expected := "Hello, tar.gz!"
	if string(data) != expected {
		t.Fatalf("Expected file content %q, got %q", expected, string(data))
	}
}

// TestExtractArchiveZip tests extraction of a zip archive.
func TestExtractArchiveZip(t *testing.T) {
	// Create a temporary zip archive.
	f, err := createZipArchive()
	if err != nil {
		t.Fatalf("Failed to create zip archive: %v", err)
	}
	defer os.Remove(f.Name())

	// Create a temporary directory to extract to.
	destDir, err := os.MkdirTemp("", "extract-test-")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Call the extraction function.
	if err := ExtractArchive(f, destDir); err != nil {
		t.Fatalf("ExtractArchive failed: %v", err)
	}

	// Verify that "test.txt" exists and has the expected content.
	extractedPath := filepath.Join(destDir, "test.txt")
	data, err := os.ReadFile(extractedPath)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	expected := "Hello, zip!"
	if string(data) != expected {
		t.Fatalf("Expected file content %q, got %q", expected, string(data))
	}
}

// TestExtractArchiveUnsupported tests extraction on a file with unsupported content.
func TestExtractArchiveUnsupported(t *testing.T) {
	// Create a temporary file with non-archive content.
	f, err := os.CreateTemp("", "test-unsupported-*")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(f.Name())

	content := "This is not an archive"
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("Failed to seek temporary file: %v", err)
	}

	destDir, err := os.MkdirTemp("", "extract-test-")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Call the extraction function expecting an error.
	if err := ExtractArchive(f, destDir); err == nil {
		t.Fatal("Expected error for unsupported archive format, got nil")
	}
}
