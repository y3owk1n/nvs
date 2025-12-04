package archive_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/y3owk1n/nvs/internal/infra/archive"
)

func TestExtractor_ExtractTarGz_PathTraversal(t *testing.T) {
	// Test that tar.gz extraction prevents path traversal attacks
	tempDir := t.TempDir()
	extractor := archive.New()

	// Create a malicious tar.gz with path traversal
	var buf bytes.Buffer

	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	// Add a malicious file that tries to escape the destination
	maliciousPath := "../../../etc/passwd"

	err := tarWriter.WriteHeader(&tar.Header{
		Name: maliciousPath,
		Mode: 0o644,
		Size: 0,
	})
	if err != nil {
		t.Fatalf("Failed to write malicious tar header: %v", err)
	}

	_ = tarWriter.Close()
	_ = gzWriter.Close()

	err = gzWriter.Close()
	if err != nil {
		t.Fatalf("Failed to close gzip writer: %v", err)
	}

	// Write to temp file
	archivePath := filepath.Join(tempDir, "malicious.tar.gz")

	err = os.WriteFile(archivePath, buf.Bytes(), 0o644)
	if err != nil {
		t.Fatalf("Failed to write archive: %v", err)
	}

	// Try to extract
	destDir := filepath.Join(tempDir, "extract")

	file, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("Failed to open archive: %v", err)
	}

	defer func() { _ = file.Close() }()

	err = extractor.Extract(file, destDir)
	if err == nil {
		t.Error("Expected extraction to fail due to path traversal")
	}

	// Check that the error mentions the illegal path
	if err != nil && !strings.Contains(err.Error(), maliciousPath) {
		t.Errorf("Expected error to mention illegal path %s, got: %v", maliciousPath, err)
	}

	// Verify that no files were extracted to the destination (since extraction should fail)
	entries, err := os.ReadDir(destDir)
	if err != nil && !os.IsNotExist(err) {
		t.Errorf("Failed to read destination directory: %v", err)
	}

	if len(entries) > 0 {
		t.Errorf("Files were extracted despite path traversal error: %v", entries)
	}
}

func TestExtractor_ExtractZip_PathTraversal(t *testing.T) {
	// Test that zip extraction prevents path traversal attacks
	tempDir := t.TempDir()
	extractor := archive.New()

	// Create a malicious zip with path traversal
	var buf bytes.Buffer

	zipWriter := zip.NewWriter(&buf)

	// Add a malicious file that tries to escape the destination
	maliciousPath := "../../../etc/passwd"

	writer, err := zipWriter.Create(maliciousPath)
	if err != nil {
		t.Fatalf("Failed to create malicious zip entry: %v", err)
	}

	_, _ = writer.Write([]byte("malicious content"))

	// Add a legitimate file
	writer, err = zipWriter.Create("safe.txt")
	if err != nil {
		t.Fatalf("Failed to create safe zip entry: %v", err)
	}

	_, _ = writer.Write([]byte("safe content"))

	_ = zipWriter.Close()

	// Write to temp file
	archivePath := filepath.Join(tempDir, "malicious.zip")

	err = os.WriteFile(archivePath, buf.Bytes(), 0o644)
	if err != nil {
		t.Fatalf("Failed to write archive: %v", err)
	}

	// Try to extract
	destDir := filepath.Join(tempDir, "extract")

	file, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("Failed to open archive: %v", err)
	}

	defer func() { _ = file.Close() }()

	err = extractor.Extract(file, destDir)
	if err == nil {
		t.Error("Expected extraction to fail due to path traversal")
	}

	// Check that the error mentions the illegal path
	if !strings.Contains(err.Error(), maliciousPath) {
		t.Errorf("Expected error to mention illegal path %s, got: %v", maliciousPath, err)
	}

	// Verify that no files were extracted to the destination (since extraction should fail)
	entries, err := os.ReadDir(destDir)
	if err != nil && !os.IsNotExist(err) {
		t.Errorf("Failed to read destination directory: %v", err)
	}

	if len(entries) > 0 {
		t.Errorf("Files were extracted despite path traversal error: %v", entries)
	}
}

func TestExtractor_ExtractTarGz_ValidPaths(t *testing.T) {
	// Test that legitimate tar.gz archives extract correctly
	tempDir := t.TempDir()
	extractor := archive.New()

	// Create a valid tar.gz
	var buf bytes.Buffer

	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	// Add a file
	content := "test content"

	err := tarWriter.WriteHeader(&tar.Header{
		Name: "test.txt",
		Mode: 0o644,
		Size: int64(len(content)),
	})
	if err != nil {
		t.Fatalf("Failed to write tar header: %v", err)
	}

	_, err = tarWriter.Write([]byte(content))
	if err != nil {
		t.Fatalf("Failed to write tar content: %v", err)
	}

	// Add a directory
	err = tarWriter.WriteHeader(&tar.Header{
		Name:     "subdir/",
		Mode:     0o755,
		Typeflag: tar.TypeDir,
	})
	if err != nil {
		t.Fatalf("Failed to write tar directory header: %v", err)
	}

	_ = tarWriter.Close()
	_ = gzWriter.Close()

	// Write to temp file
	archivePath := filepath.Join(tempDir, "valid.tar.gz")

	err = os.WriteFile(archivePath, buf.Bytes(), 0o644)
	if err != nil {
		t.Fatalf("Failed to write archive: %v", err)
	}

	// Extract
	destDir := filepath.Join(tempDir, "extract")

	file, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("Failed to open archive: %v", err)
	}

	defer func() { _ = file.Close() }()

	err = extractor.Extract(file, destDir)
	if err != nil {
		t.Errorf("Expected extraction to succeed, got: %v", err)
	}

	// Verify files were extracted
	testFile := filepath.Join(destDir, "test.txt")

	_, err = os.Stat(testFile)
	if os.IsNotExist(err) {
		t.Error("Test file was not extracted")
	}

	// Verify content
	extractedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	if string(extractedContent) != content {
		t.Errorf("Expected content %q, got %q", content, string(extractedContent))
	}

	// Verify directory was created
	testDir := filepath.Join(destDir, "subdir")

	_, err = os.Stat(testDir)
	if os.IsNotExist(err) {
		t.Error("Test directory was not extracted")
	}
}

func TestExtractor_ExtractZip_ValidPaths(t *testing.T) {
	// Test that legitimate zip archives extract correctly
	tempDir := t.TempDir()
	extractor := archive.New()

	// Create a valid zip
	var buf bytes.Buffer

	zipWriter := zip.NewWriter(&buf)

	// Add a file
	content := "test content"

	writer, err := zipWriter.Create("test.txt")
	if err != nil {
		t.Fatalf("Failed to create zip entry: %v", err)
	}

	_, err = writer.Write([]byte(content))
	if err != nil {
		t.Fatalf("Failed to write zip content: %v", err)
	}

	_ = zipWriter.Close()

	// Write to temp file
	archivePath := filepath.Join(tempDir, "valid.zip")

	err = os.WriteFile(archivePath, buf.Bytes(), 0o644)
	if err != nil {
		t.Fatalf("Failed to write archive: %v", err)
	}

	// Extract
	destDir := filepath.Join(tempDir, "extract")

	file, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("Failed to open archive: %v", err)
	}

	defer func() { _ = file.Close() }()

	err = extractor.Extract(file, destDir)
	if err != nil {
		t.Errorf("Expected extraction to succeed, got: %v", err)
	}

	// Verify file was extracted
	testFile := filepath.Join(destDir, "test.txt")

	_, err = os.Stat(testFile)
	if os.IsNotExist(err) {
		t.Error("Test file was not extracted")
	}

	// Verify content
	extractedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	if string(extractedContent) != content {
		t.Errorf("Expected content %q, got %q", content, string(extractedContent))
	}
}
