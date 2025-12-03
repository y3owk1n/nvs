//go:build integration

package archive_test

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

	"github.com/y3owk1n/nvs/pkg/archive"
)

// tarEntry defines an entry to be written into a tar.
type tarEntry struct {
	Name     string
	Body     string
	Mode     int64
	TypeFlag byte
}

// ZipEntry defines an entry to be written into a archive.Zip.
type ZipEntry struct {
	Name  string
	Body  string
	Mode  os.FileMode
	IsDir bool
}

// createTarGzFile is a helper that writes a tar.gz archive with the given entries
// to a temporary file and returns an open *os.File.
func createTarGzFile(t *testing.T, entries []tarEntry) *os.File {
	t.Helper()

	tmpFile, err := os.CreateTemp(t.TempDir(), "test-archive-*.tar.gz")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	var buf bytes.Buffer

	gzipWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzipWriter)

	for _, entry := range entries {
		header := &tar.Header{
			Name:     entry.Name,
			Mode:     entry.Mode,
			Size:     int64(len(entry.Body)),
			Typeflag: entry.TypeFlag,
		}

		err := tarWriter.WriteHeader(header)
		if err != nil {
			t.Fatalf("failed to write tar header: %v", err)
		}

		if entry.TypeFlag == tar.TypeReg {
			_, err = tarWriter.Write([]byte(entry.Body))
			if err != nil {
				t.Fatalf("failed to write tar body: %v", err)
			}
		}
	}

	err = tarWriter.Close()
	if err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}

	err = gzipWriter.Close()
	if err != nil {
		t.Fatalf("failed to close 	gzip writer: %v", err)
	}

	_, err = tmpFile.Write(buf.Bytes())
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}

	return tmpFile
}

// createZipFile is a helper that writes a archive.Zip archive with the given entries
// to a temporary file and returns an open *os.File.
func createZipFile(t *testing.T, entries []ZipEntry) *os.File {
	t.Helper()

	tmpFile, err := os.CreateTemp(t.TempDir(), "test-archive-*.archive.Zip")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

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

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			t.Fatalf("failed to create archive.Zip header: %v", err)
		}

		if !entry.IsDir {
			_, err = writer.Write([]byte(entry.Body))
			if err != nil {
				t.Fatalf("failed to write archive.Zip entry: %v", err)
			}
		}
	}

	err = zipWriter.Close()
	if err != nil {
		t.Fatalf("failed to close archive.Zip writer: %v", err)
	}

	_, err = tmpFile.Write(buf.Bytes())
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}

	return tmpFile
}

// Test	ExtractArchive_TarGzSuccess creates a valid tar.gz archive, extracts it,
// and verifies that all files/directories are correctly created with the proper content.
func TestExtractArchive_TarGzSuccess(t *testing.T) {
	entries := []tarEntry{
		{Name: "dir/", Mode: 0o755, TypeFlag: tar.TypeDir},
		{Name: "dir/test.txt", Body: "hello world", Mode: 0o644, TypeFlag: tar.TypeReg},
	}

	srcFile := createTarGzFile(t, entries)
	defer func() { _ = srcFile.Close() }()
	defer func() { _ = os.Remove(srcFile.Name()) }()

	destDir := t.TempDir()

	err := archive.ExtractArchive(srcFile, destDir)
	if err != nil {
		t.Fatalf("	archive.ExtractArchive failed: %v", err)
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

// TestExtractArchive_ZipSuccess creates a valid archive.Zip archive, extracts it,
// and verifies that all files/directories are correctly created with the proper content.
func TestExtractArchive_ZipSuccess(t *testing.T) {
	entries := []ZipEntry{
		{Name: "folder/", IsDir: true, Mode: 0o755},
		{Name: "folder/test.txt", Body: "archive.Zip content", IsDir: false, Mode: 0o644},
	}

	srcFile := createZipFile(t, entries)
	defer func() { _ = srcFile.Close() }()
	defer func() { _ = os.Remove(srcFile.Name()) }()

	destDir := t.TempDir()

	err := archive.ExtractArchive(srcFile, destDir)
	if err != nil {
		t.Fatalf("	archive.ExtractArchive failed: %v", err)
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

	if string(data) != "archive.Zip content" {
		t.Fatalf("unexpected file content: got %q, want %q", string(data), "archive.Zip content")
	}
}

// TestExtractArchive_UnsupportedFormat writes a file with a PDF header
// to simulate an unsupported archive type.
func TestExtractArchive_UnsupportedFormat(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-unsupported-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	defer func() { _ = tmpFile.Close() }()
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// Write PDF header (a known file type but unsupported by our extractor).
	content := []byte("%PDF-1.4\n%âãÏÓ\n")

	_, err = tmpFile.Write(content)
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}

	destDir := t.TempDir()

	err = archive.ExtractArchive(tmpFile, destDir)
	if err == nil {
		t.Fatalf("expected error for unsupported archive format, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported archive format") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// TestDetectArchiveFormat_Zip creates a file with a ZIP signature and
// verifies that archive.DetectArchiveFormat returns "archive.Zip".
func TestDetectArchiveFormat_Zip(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-archive.Zip-detect-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	defer func() { _ = tmpFile.Close() }()
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// Write ZIP signature "PK\x03\x04" followed by arbitrary data.
	content := []byte("PK\x03\x04randomdata")

	_, err = tmpFile.Write(content)
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}

	format, err := archive.DetectArchiveFormat(tmpFile)
	if err != nil {
		t.Fatalf("archive.DetectArchiveFormat failed: %v", err)
	}

	if format != archive.Zip {
		t.Fatalf("expected format %q, got %q", archive.Zip, format)
	}
}

// TestDetectArchiveFormat_TarGz creates a file with valid 	gzip data and
// verifies that archive.DetectArchiveFormat returns "tar.gz".
func TestDetectArchiveFormat_TarGz(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-targz-detect-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	defer func() { _ = tmpFile.Close() }()
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// Create a valid 	gzip stream.
	var buf bytes.Buffer

	gzipWriter := gzip.NewWriter(&buf)

	_, err = gzipWriter.Write([]byte("data"))
	if err != nil {
		t.Fatalf("failed to write 	gzip data: %v", err)
	}

	err = gzipWriter.Close()
	if err != nil {
		t.Fatalf("failed to close 	gzip writer: %v", err)
	}

	_, err = tmpFile.Write(buf.Bytes())
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}

	format, err := archive.DetectArchiveFormat(tmpFile)
	if err != nil {
		t.Fatalf("archive.DetectArchiveFormat failed: %v", err)
	}

	if format != archive.TarGz {
		t.Fatalf("expected format %q, got %q", archive.TarGz, format)
	}
}

// TestDetectArchiveFormat_Unknown writes data that does not match any known file type,
// so that archive.DetectArchiveFormat returns an error.
func TestDetectArchiveFormat_Unknown(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-unknown-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	defer func() { _ = tmpFile.Close() }()
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	content := []byte("abcdefg")

	_, err = tmpFile.Write(content)
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}

	_, err = archive.DetectArchiveFormat(tmpFile)
	if err == nil || !strings.Contains(err.Error(), "unknown file type") {
		t.Fatalf("expected unknown file type error, got: %v", err)
	}
}

// TestDetectArchiveFormat_EmptyBuffer creates an empty file and verifies that
// archive.DetectArchiveFormat returns an error indicating an empty buffer.
func TestDetectArchiveFormat_EmptyBuffer(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-empty-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	defer func() { _ = tmpFile.Close() }()
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// Ensure the file is empty.
	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}

	_, err = archive.DetectArchiveFormat(tmpFile)
	if err == nil || !strings.Contains(err.Error(), "empty buffer") {
		t.Fatalf("expected empty buffer error, got: %v", err)
	}
}

// TestExtractTarGz_InvalidGZip writes non-	gzip data and ensures extractTarGz fails
// when trying to create a 	gzip reader.
func TestExtractTarGz_InvalidGZip(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-invalid-gzip-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	defer func() { _ = tmpFile.Close() }()

	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString("not a 	gzip")
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}

	destDir := t.TempDir()

	err = archive.ExtractTarGz(tmpFile, destDir)
	if err == nil || !strings.Contains(err.Error(), "failed to create gzip reader") {
		t.Fatalf("expected gzip reader creation error, got: %v", err)
	}
}

// TestExtractTarGz_InvalidTar creates a valid 	gzip stream containing invalid tar data,
// so that extractTarGz fails when reading the tar.
func TestExtractTarGz_InvalidTar(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-invalid-tar-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	defer func() { _ = tmpFile.Close() }()
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	var buf bytes.Buffer

	gzipWriter := gzip.NewWriter(&buf)

	_, err = gzipWriter.Write([]byte("not a tar archive"))
	if err != nil {
		t.Fatalf("failed to write 	gzip data: %v", err)
	}

	err = gzipWriter.Close()
	if err != nil {
		t.Fatalf("failed to close 	gzip writer: %v", err)
	}

	_, err = tmpFile.Write(buf.Bytes())
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}

	destDir := t.TempDir()

	err = archive.ExtractTarGz(tmpFile, destDir)
	if err == nil || !strings.Contains(err.Error(), "error reading tar archive") {
		t.Fatalf("expected tar archive reading error, got: %v", err)
	}
}

// TestExtractZip_InvalidZip writes non-archive.Zip data and ensures extractZip fails when
// attempting to create a archive.Zip reader.
func TestExtractZip_InvalidZip(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-invalid-archive.Zip-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	defer func() { _ = tmpFile.Close() }()
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString("not a archive.Zip archive")
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("failed to seek temp file: %v", err)
	}

	destDir := t.TempDir()

	err = archive.ExtractZip(tmpFile, destDir)
	if err == nil || !strings.Contains(err.Error(), "failed to create zip reader") {
		t.Fatalf("expected zip reader creation error, got: %v", err)
	}
}

// TestExtractArchive_SeekError simulates a seek failure by closing the source file
// before calling archive.ExtractArchive.
func TestExtractArchive_SeekError(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-seek-error-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	// Close the file to force a seek error.
	_ = tmpFile.Close()

	destDir := t.TempDir()

	err = archive.ExtractArchive(tmpFile, destDir)
	if err == nil || !strings.Contains(err.Error(), "failed to seek to start of file") {
		t.Fatalf("expected seek error, got: %v", err)
	}
}

// TestExtractZip_DestNotWritable creates a valid archive.Zip archive and attempts to extract it
// into a destination directory with no write permissions.
func TestExtractZip_DestNotWritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping non-writable destination test on Windows")
	}

	entries := []ZipEntry{
		{Name: "file.txt", Body: "data", IsDir: false, Mode: 0o644},
	}

	srcFile := createZipFile(t, entries)
	defer func() { _ = srcFile.Close() }()
	defer func() { _ = os.Remove(srcFile.Name()) }()

	destDir := t.TempDir()
	// Make destDir non-writable.
	chmodErr := os.Chmod(destDir, 0o555)
	if chmodErr != nil {
		t.Fatalf("failed to chmod destDir: %v", chmodErr)
	}

	defer func() { _ = os.Chmod(destDir, 0o755) }() // Restore permissions for cleanup.

	err := archive.ExtractArchive(srcFile, destDir)
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
		{Name: "file.txt", Body: "data", Mode: 0o644, TypeFlag: tar.TypeReg},
	}

	srcFile := createTarGzFile(t, entries)
	defer func() { _ = srcFile.Close() }()
	defer func() { _ = os.Remove(srcFile.Name()) }()

	destDir := t.TempDir()
	// Make destDir non-writable.
	chmodErr := os.Chmod(destDir, 0o555)
	if chmodErr != nil {
		t.Fatalf("failed to chmod destDir: %v", chmodErr)
	}

	defer func() { _ = os.Chmod(destDir, 0o755) }() // Restore permissions for cleanup.

	err := archive.ExtractArchive(srcFile, destDir)
	if err == nil {
		t.Fatalf("expected error due to non-writable destination, got nil")
	}
}
