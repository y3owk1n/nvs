package archive_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/y3owk1n/nvs/internal/infra/archive"
)

// buildTarGz creates a tar.gz archive in memory with the given
// (name, content) entries. The returned bytes are the full
// archive ready to be written to disk or fed to a reader.
func buildTarGz(t *testing.T, entries map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer

	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	for name, content := range entries {
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}

		err := tarWriter.WriteHeader(header)
		if err != nil {
			t.Fatalf("Failed to write tar header for %s: %v", name, err)
		}

		_, err = tarWriter.Write([]byte(content))
		if err != nil {
			t.Fatalf("Failed to write tar content for %s: %v", name, err)
		}
	}

	err := tarWriter.Close()
	if err != nil {
		t.Fatalf("Failed to close tar writer: %v", err)
	}

	err = gzWriter.Close()
	if err != nil {
		t.Fatalf("Failed to close gzip writer: %v", err)
	}

	return buf.Bytes()
}

// buildZip creates a zip archive in memory with the given
// (name, content) entries.
func buildZip(t *testing.T, entries map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer

	zipWriter := zip.NewWriter(&buf)

	for name, content := range entries {
		writer, err := zipWriter.Create(name)
		if err != nil {
			t.Fatalf("Failed to create zip entry %s: %v", name, err)
		}

		_, err = writer.Write([]byte(content))
		if err != nil {
			t.Fatalf("Failed to write zip content for %s: %v", name, err)
		}
	}

	err := zipWriter.Close()
	if err != nil {
		t.Fatalf("Failed to close zip writer: %v", err)
	}

	return buf.Bytes()
}

// progressTracker captures the sequence of percent values
// reported by the extractor. It is safe for concurrent use
// (the extractor's callback is invoked from the extraction
// goroutine, while tests read from the main goroutine).
type progressTracker struct {
	mu       sync.Mutex
	percents []int
}

func (p *progressTracker) callback(percent int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.percents = append(p.percents, percent)
}

func (p *progressTracker) snapshot() []int {
	p.mu.Lock()
	defer p.mu.Unlock()

	out := make([]int, len(p.percents))
	copy(out, p.percents)

	return out
}

// writeArchiveAndOpen writes archiveBytes to a file named name
// inside tempDir and returns the open file. The caller is
// responsible for closing the returned file.
func writeArchiveAndOpen(t *testing.T, tempDir, name string, archiveBytes []byte) *os.File {
	t.Helper()

	archivePath := filepath.Join(tempDir, name)

	err := os.WriteFile(archivePath, archiveBytes, 0o644)
	if err != nil {
		t.Fatalf("Failed to write archive: %v", err)
	}

	file, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("Failed to open archive: %v", err)
	}

	return file
}

func TestExtractor_ExtractTarGz_ProgressReaches100(t *testing.T) {
	t.Parallel()

	entries := map[string]string{
		"a.txt": "alpha",
		"b.txt": "bravo",
		"c.txt": "charlie",
		"d.txt": "delta",
		"e.txt": "echo",
	}
	archiveBytes := buildTarGz(t, entries)

	tempDir := t.TempDir()
	destDir := filepath.Join(tempDir, "extract")

	file := writeArchiveAndOpen(t, tempDir, "test.tar.gz", archiveBytes)

	defer func() { _ = file.Close() }()

	tracker := &progressTracker{}

	err := archive.New().Extract(file, destDir, tracker.callback)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	percents := tracker.snapshot()
	if len(percents) == 0 {
		t.Fatal("progress callback was never invoked")
	}

	last := percents[len(percents)-1]
	if last != 100 {
		t.Errorf("final progress = %d%%, want 100%% (full sequence: %v)", last, percents)
	}

	// The progress values must be monotonically non-decreasing.
	// Spinner code assumes the percent only grows.
	for i := 1; i < len(percents); i++ {
		if percents[i] < percents[i-1] {
			t.Errorf(
				"progress regressed at index %d: %d -> %d (full: %v)",
				i, percents[i-1], percents[i], percents,
			)
		}
	}
}

func TestExtractor_ExtractZip_ProgressReaches100(t *testing.T) {
	t.Parallel()

	entries := map[string]string{
		"a.txt": "alpha",
		"b.txt": "bravo",
		"c.txt": "charlie",
		"d.txt": "delta",
	}
	archiveBytes := buildZip(t, entries)

	tempDir := t.TempDir()
	destDir := filepath.Join(tempDir, "extract")

	file := writeArchiveAndOpen(t, tempDir, "test.zip", archiveBytes)

	defer func() { _ = file.Close() }()

	tracker := &progressTracker{}

	err := archive.New().Extract(file, destDir, tracker.callback)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	percents := tracker.snapshot()
	if len(percents) == 0 {
		t.Fatal("progress callback was never invoked")
	}

	last := percents[len(percents)-1]
	if last != 100 {
		t.Errorf("final progress = %d%%, want 100%% (full sequence: %v)", last, percents)
	}

	for i := 1; i < len(percents); i++ {
		if percents[i] < percents[i-1] {
			t.Errorf(
				"progress regressed at index %d: %d -> %d (full: %v)",
				i, percents[i-1], percents[i], percents,
			)
		}
	}
}

func TestExtractor_ExtractTarGz_ProgressNilSafe(t *testing.T) {
	t.Parallel()

	entries := map[string]string{
		"a.txt": "alpha",
		"b.txt": "bravo",
	}
	archiveBytes := buildTarGz(t, entries)

	tempDir := t.TempDir()
	destDir := filepath.Join(tempDir, "extract")

	file := writeArchiveAndOpen(t, tempDir, "test.tar.gz", archiveBytes)

	defer func() { _ = file.Close() }()

	// nil progress callback must not panic. This is the
	// contract every internal caller relies on: a missing
	// callback degrades to "no progress reporting", not an
	// error.
	err := archive.New().Extract(file, destDir, nil)
	if err != nil {
		t.Fatalf("Extract with nil progress failed: %v", err)
	}
}

func TestExtractor_ExtractTarGz_ProgressAllFiles(t *testing.T) {
	t.Parallel()

	// 10 entries. After extraction, the progress callback
	// should have been called at least 10 times (once per
	// file written), and the final value must be 100%.
	entries := make(map[string]string, 10)
	for i := range 10 {
		name := filepath.Join("sub", string(rune('a'+i))+".txt")
		entries[name] = "x"
	}

	archiveBytes := buildTarGz(t, entries)

	tempDir := t.TempDir()
	destDir := filepath.Join(tempDir, "extract")

	file := writeArchiveAndOpen(t, tempDir, "test.tar.gz", archiveBytes)

	defer func() { _ = file.Close() }()

	tracker := &progressTracker{}

	err := archive.New().Extract(file, destDir, tracker.callback)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	percents := tracker.snapshot()
	if len(percents) < 10 {
		t.Errorf(
			"progress called %d times, want at least 10 (one per file): %v",
			len(percents), percents,
		)
	}

	if last := percents[len(percents)-1]; last != 100 {
		t.Errorf("final progress = %d%%, want 100%%", last)
	}
}
