package archive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// tarEntry defines an entry to be written into a tar archive.
type tarEntry struct {
	Name     string
	Body     string
	Mode     int64
	TypeFlag byte
}

// zipEntry defines an entry to be written into a zip archive.
type zipEntry struct {
	Name  string
	Body  string
	Mode  os.FileMode
	IsDir bool
}

// createTarGzFile is a helper that writes a tar.gz archive with the given entries
// to a temporary file and returns an open *os.File.
func createTarGzFile(t *testing.T, entries []tarEntry) *os.File {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test-archive-*.tar.gz")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for _, entry := range entries {
		header := &tar.Header{
			Name:     entry.Name,
			Mode:     entry.Mode,
			Size:     int64(len(entry.Body)),
			Typeflag: entry.TypeFlag,
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("failed to write tar header: %v", err)
		}
		if entry.TypeFlag == tar.TypeReg {
			if _, err := tw.Write([]byte(entry.Body)); err != nil {
				t.Fatalf("failed to write tar body: %v", err)
			}
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}

	if _, err := tmpFile.Write(buf.Bytes()); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}
	return tmpFile
}

// createZipFile is a helper that writes a zip archive with the given entries
// to a temporary file and returns an open *os.File.
func createZipFile(t *testing.T, entries []zipEntry) *os.File {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test-archive-*.zip")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	for _, entry := range entries {
		// Prepare file header.
		header := &zip.FileHeader{
			Name:   entry.Name,
			Method: zip.Deflate,
		}
		header.SetMode(entry.Mode)
		if entry.IsDir {
			// Ensure directory name ends with "/"
			header.Name = strings.TrimSuffix(entry.Name, "/") + "/"
		}

		writer, err := zw.CreateHeader(header)
		if err != nil {
			t.Fatalf("failed to create zip header: %v", err)
		}

		if !entry.IsDir {
			if _, err := writer.Write([]byte(entry.Body)); err != nil {
				t.Fatalf("failed to write zip entry: %v", err)
			}
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	if _, err := tmpFile.Write(buf.Bytes()); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}
	return tmpFile
}

// TestExtractArchive_TarGzSuccess creates a valid tar.gz archive, extracts it,
// and verifies that all files/directories are correctly created with the proper content.
func TestExtractArchive_TarGzSuccess(t *testing.T) {
	entries := []tarEntry{
		{Name: "dir/", Mode: 0755, TypeFlag: tar.TypeDir},
		{Name: "dir/test.txt", Body: "hello world", Mode: 0644, TypeFlag: tar.TypeReg},
	}
	srcFile := createTarGzFile(t, entries)
	defer os.Remove(srcFile.Name())

	destDir := t.TempDir()
	if err := ExtractArchive(srcFile, destDir); err != nil {
		t.Fatalf("ExtractArchive failed: %v", err)
	}

	// Verify directory exists.
	dirPath := filepath.Join(destDir, "dir")
	info, err := os.Stat(dirPath)
	if err != nil || !info.IsDir() {
		t.Fatalf("expected directory %s to exist", dirPath)
	}

	// Verify file exists and has the expected content.
	filePath := filepath.Join(destDir, "dir", "test.txt")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if string(data) != "hello world" {
		t.Fatalf("unexpected file content: got %q, want %q", string(data), "hello world")
	}
}

// TestExtractArchive_ZipSuccess creates a valid zip archive, extracts it,
// and verifies that all files/directories are correctly created with the proper content.
func TestExtractArchive_ZipSuccess(t *testing.T) {
	entries := []zipEntry{
		{Name: "folder/", IsDir: true, Mode: 0755},
		{Name: "folder/test.txt", Body: "zip content", IsDir: false, Mode: 0644},
	}
	srcFile := createZipFile(t, entries)
	defer os.Remove(srcFile.Name())

	destDir := t.TempDir()
	if err := ExtractArchive(srcFile, destDir); err != nil {
		t.Fatalf("ExtractArchive failed: %v", err)
	}

	// Verify directory exists.
	folderPath := filepath.Join(destDir, "folder")
	info, err := os.Stat(folderPath)
	if err != nil || !info.IsDir() {
		t.Fatalf("expected directory %s to exist", folderPath)
	}

	// Verify file exists and has the expected content.
	filePath := filepath.Join(destDir, "folder", "test.txt")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if string(data) != "zip content" {
		t.Fatalf("unexpected file content: got %q, want %q", string(data), "zip content")
	}
}

// TestExtractArchive_UnsupportedFormat writes a file with a PDF header
// to simulate an unsupported archive type.
func TestExtractArchive_UnsupportedFormat(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-unsupported-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write PDF header (a known file type but unsupported by our extractor).
	content := []byte("%PDF-1.4\n%âãÏÓ\n")
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}

	destDir := t.TempDir()
	err = ExtractArchive(tmpFile, destDir)
	if err == nil {
		t.Fatalf("expected error for unsupported archive format, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported archive format") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// TestDetectArchiveFormat_Zip creates a file with a ZIP signature and
// verifies that detectArchiveFormat returns "zip".
func TestDetectArchiveFormat_Zip(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-zip-detect-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write ZIP signature "PK\x03\x04" followed by arbitrary data.
	content := []byte("PK\x03\x04randomdata")
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}

	format, err := detectArchiveFormat(tmpFile)
	if err != nil {
		t.Fatalf("detectArchiveFormat failed: %v", err)
	}
	if format != "zip" {
		t.Fatalf("expected format 'zip', got %q", format)
	}
}

// TestDetectArchiveFormat_TarGz creates a file with valid gzip data and
// verifies that detectArchiveFormat returns "tar.gz".
func TestDetectArchiveFormat_TarGz(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-targz-detect-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Create a valid gzip stream.
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err = gw.Write([]byte("data"))
	if err != nil {
		t.Fatalf("failed to write gzip data: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}

	if _, err := tmpFile.Write(buf.Bytes()); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}

	format, err := detectArchiveFormat(tmpFile)
	if err != nil {
		t.Fatalf("detectArchiveFormat failed: %v", err)
	}
	if format != "tar.gz" {
		t.Fatalf("expected format 'tar.gz', got %q", format)
	}
}

// TestDetectArchiveFormat_Unknown writes data that does not match any known file type,
// so that detectArchiveFormat returns an error.
func TestDetectArchiveFormat_Unknown(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-unknown-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := []byte("abcdefg")
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}

	_, err = detectArchiveFormat(tmpFile)
	if err == nil || !strings.Contains(err.Error(), "unknown file type") {
		t.Fatalf("expected unknown file type error, got: %v", err)
	}
}

// TestDetectArchiveFormat_EmptyBuffer creates an empty file and verifies that
// detectArchiveFormat returns an error indicating an empty buffer.
func TestDetectArchiveFormat_EmptyBuffer(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-empty-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Ensure the file is empty.
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}

	_, err = detectArchiveFormat(tmpFile)
	if err == nil || !strings.Contains(err.Error(), "empty buffer") {
		t.Fatalf("expected empty buffer error, got: %v", err)
	}
}

// TestExtractTarGz_InvalidGzip writes non-gzip data and ensures extractTarGz fails
// when trying to create a gzip reader.
func TestExtractTarGz_InvalidGzip(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-invalid-gzip-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte("not a gzip")); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}

	destDir := t.TempDir()
	err = extractTarGz(tmpFile, destDir)
	if err == nil || !strings.Contains(err.Error(), "failed to create gzip reader") {
		t.Fatalf("expected gzip reader creation error, got: %v", err)
	}
}

// TestExtractTarGz_InvalidTar creates a valid gzip stream containing invalid tar data,
// so that extractTarGz fails when reading the tar archive.
func TestExtractTarGz_InvalidTar(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-invalid-tar-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err = gw.Write([]byte("not a tar archive"))
	if err != nil {
		t.Fatalf("failed to write gzip data: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}

	if _, err := tmpFile.Write(buf.Bytes()); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}

	destDir := t.TempDir()
	err = extractTarGz(tmpFile, destDir)
	if err == nil || !strings.Contains(err.Error(), "error reading tar archive") {
		t.Fatalf("expected tar archive reading error, got: %v", err)
	}
}

// TestExtractZip_InvalidZip writes non-zip data and ensures extractZip fails when
// attempting to create a zip reader.
func TestExtractZip_InvalidZip(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-invalid-zip-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte("not a zip archive")); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}

	destDir := t.TempDir()
	err = extractZip(tmpFile, destDir)
	if err == nil || !strings.Contains(err.Error(), "failed to create zip reader") {
		t.Fatalf("expected zip reader creation error, got: %v", err)
	}
}

// TestExtractArchive_SeekError simulates a seek failure by closing the source file
// before calling ExtractArchive.
func TestExtractArchive_SeekError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-seek-error-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	// Close the file to force a seek error.
	tmpFile.Close()

	destDir := t.TempDir()
	err = ExtractArchive(tmpFile, destDir)
	if err == nil || !strings.Contains(err.Error(), "failed to seek to start of file") {
		t.Fatalf("expected seek error, got: %v", err)
	}
}

// TestExtractZip_DestNotWritable creates a valid zip archive and attempts to extract it
// into a destination directory with no write permissions.
func TestExtractZip_DestNotWritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping non-writable destination test on Windows")
	}
	entries := []zipEntry{
		{Name: "file.txt", Body: "data", IsDir: false, Mode: 0644},
	}
	srcFile := createZipFile(t, entries)
	defer os.Remove(srcFile.Name())

	destDir := t.TempDir()
	// Make destDir non-writable.
	if err := os.Chmod(destDir, 0555); err != nil {
		t.Fatalf("failed to chmod destDir: %v", err)
	}
	defer os.Chmod(destDir, 0755) // Restore permissions for cleanup.

	err := ExtractArchive(srcFile, destDir)
	if err == nil {
		t.Fatalf("expected error due to non-writable destination, got nil")
	}
}

// TestExtractTarGz_DestNotWritable creates a valid tar.gz archive and attempts to extract it
// into a destination directory with no write permissions.
func TestExtractTarGz_DestNotWritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping non-writable destination test on Windows")
	}
	entries := []tarEntry{
		{Name: "file.txt", Body: "data", Mode: 0644, TypeFlag: tar.TypeReg},
	}
	srcFile := createTarGzFile(t, entries)
	defer os.Remove(srcFile.Name())

	destDir := t.TempDir()
	// Make destDir non-writable.
	if err := os.Chmod(destDir, 0555); err != nil {
		t.Fatalf("failed to chmod destDir: %v", err)
	}
	defer os.Chmod(destDir, 0755) // Restore permissions for cleanup.

	err := ExtractArchive(srcFile, destDir)
	if err == nil {
		t.Fatalf("expected error due to non-writable destination, got nil")
	}
}
