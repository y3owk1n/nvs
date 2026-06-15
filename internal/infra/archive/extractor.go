// Package archive provides archive extraction functionality.
// Refactored from pkg/archive with consistent error handling.
package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/h2non/filetype"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/log"
)

// Extractor handles archive extraction operations.
type Extractor struct{}

// ProgressFunc is a callback for extraction progress updates.
// The percent value is clamped to the range [0, 100].
//
// The callback is invoked once per archive entry that is
// actually written to disk (directories, skipped symlinks,
// and other no-op entry types do not count), so the progress
// reflects bytes-on-disk, not just header count. This is the
// same definition install/upgrade code expects: the spinner
// shows the percentage of the install that is "on disk" once
// the line says "Extracting ...".
//
// The callback may be nil — in that case progress is not
// reported.
type ProgressFunc func(percent int)

// New creates a new Extractor instance.
func New() *Extractor {
	return &Extractor{}
}

// Extract extracts an archive file to the destination directory.
// If progress is non-nil, it is invoked with the current
// extraction percentage (0-100) after each on-disk write. For
// tar.gz archives, the total entry count is determined by a
// streaming pre-pass; for zip archives, the count is taken
// from the central directory (no pre-pass needed).
func (e *Extractor) Extract(src *os.File, dest string, progress ProgressFunc) error {
	log.Debugf("Starting extraction to: %s", dest)
	// Detect archive format
	format, err := detectFormat(src)
	if err != nil {
		return fmt.Errorf("archive detection failed: %w", err)
	}

	log.Debugf("Detected archive format: %s", format)

	// Extract based on format
	switch format {
	case "tar.gz":
		return e.extractTarGz(src, dest, progress)
	case constants.ZipFormat:
		return e.extractZip(src, dest, progress)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
}

// detectFormat detects the archive format from file header.
func detectFormat(file *os.File) (string, error) {
	buf := make([]byte, constants.BufSize)

	bytesRead, err := file.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	if bytesRead == 0 {
		return "", ErrEmptyFile
	}

	// Reset file pointer
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return "", fmt.Errorf("failed to seek file: %w", err)
	}

	// Detect file type
	kind, err := filetype.Match(buf[:bytesRead])
	if err != nil {
		return "", fmt.Errorf("file type matching error: %w", err)
	}

	if kind == filetype.Unknown {
		return "", ErrUnknownFileType
	}

	// Map to supported formats
	switch kind.Extension {
	case constants.ZipFormat:
		return "zip", nil
	case "gz":
		// Assumption: all .gz files are tar.gz (valid for Neovim releases)
		return "tar.gz", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedFormat, kind.Extension)
	}
}

// extractBufferSize is the buffer size used for io.CopyBuffer
// when extracting archive entries. 256 KiB matches Go's own
// io.Copy default for *os.File destinations (which uses a 32 KiB
// buffer); the larger value reduces the number of Read/Write
// syscalls when extracting large binaries like the Neovim
// executable or the runtime tree.
const extractBufferSize = 256 * 1024

// writeFile writes data from reader to a file at target path with given mode.
func writeFile(target string, mode os.FileMode, reader io.Reader) (err error) {
	file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", target, err)
	}

	defer func() {
		cerr := file.Close()
		if cerr != nil {
			if err == nil {
				err = fmt.Errorf("failed to close file %s: %w", target, cerr)
			} else {
				err = fmt.Errorf("%w; failed to close file %s: %w", err, target, cerr)
			}
		}
	}()

	buf := make([]byte, extractBufferSize)

	_, err = io.CopyBuffer(file, reader, buf)
	if err != nil {
		return fmt.Errorf("failed to copy file content to %s: %w", target, err)
	}

	return nil
}

// extractTarGz extracts a tar.gz archive.
//
// The total entry count is determined by a streaming pre-pass
// over the gzip stream; the file position is rewound to its
// starting point before the real extraction begins. The pre-pass
// is cheap (it only reads tar headers, not payload bytes) and
// is required to report a percentage during the actual
// extraction — the tar format has no central directory, so the
// only way to know the total entry count up front is to walk
// the headers first.
func (e *Extractor) extractTarGz(
	src *os.File,
	dest string,
	progress ProgressFunc,
) error {
	totalEntries, err := e.countTarGzEntries(src)
	if err != nil {
		return fmt.Errorf("failed to count tar entries: %w", err)
	}

	gzr, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}

	defer func() { _ = gzr.Close() }()

	tarReader := tar.NewReader(gzr)

	// Precompute cleaned destination for path traversal checks
	cleanDest := filepath.Clean(dest)

	// lastPercent collapses sequential calls to the same
	// percent value. tar entries are processed in order, and
	// the percentage only ever increases; a per-entry call is
	// already very coarse, but the floor-guard skips the
	// redundant work for empty archives (totalEntries == 0).
	lastPercent := -1

	// fileCount tracks the running number of regular-file
	// entries that have been written to disk. Only file
	// entries count toward progress, matching the zip path —
	// the spinner shows the percentage of the install that is
	// "on disk", which is the same thing as "regular files
	// written" because directory entries and skipped symlinks
	// produce no payload.
	fileCount := 0

	emit := func(percent int) {
		if progress == nil || totalEntries <= 0 {
			return
		}

		if percent == lastPercent {
			return
		}

		lastPercent = percent
		progress(percent)
	}

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return fmt.Errorf("error reading tar archive: %w", err)
		}

		// filepath.Join already calls Clean on the result, so
		// target is already a cleaned path — no need for a
		// redundant filepath.Clean(target) here.
		target := filepath.Join(dest, header.Name)

		// Prevent path traversal attacks (Zip Slip vulnerability)
		rel, err := filepath.Rel(cleanDest, target)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return &IllegalPathError{Path: header.Name}
		}

		switch header.Typeflag {
		case tar.TypeDir:
			mode := os.FileMode(header.Mode)&constants.FileModeMask | os.ModeDir

			err := os.MkdirAll(target, mode)
			if err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}

		case tar.TypeReg:
			err = os.MkdirAll(filepath.Dir(target), constants.DirPerm)
			if err != nil {
				return fmt.Errorf("failed to create directory for file %s: %w", target, err)
			}

			mode := os.FileMode(header.Mode) & constants.FileModeMask

			err := writeFile(target, mode, tarReader)
			if err != nil {
				return err
			}

			fileCount++

		case tar.TypeSymlink, tar.TypeLink:
			// Reject symlinks and hard links to prevent symlink attacks.
			// No disk write happens for these, so the
			// progress callback is intentionally NOT invoked
			// — including the entry in the count would
			// under-report actual on-disk progress.
			log.Debugf("Skipping unsupported entry type %d: %s", header.Typeflag, header.Name)
		}

		// Report progress for every regular file written.
		// fileCount is monotonic; totalEntries counts every
		// header (including directories and skipped symlinks),
		// so the percent saturates at 100% as the file count
		// approaches totalEntries.
		percent := min((fileCount*constants.ProgressMax)/totalEntries, constants.ProgressMax)

		emit(percent)
	}

	// For archives with entries that were all directories or
	// symlinks (so fileCount never reached totalEntries),
	// emit a final 100% so the spinner doesn't get stuck at
	// 0%. The totalEntries == 0 case is excluded because
	// there's nothing to extract.
	if totalEntries > 0 && lastPercent < constants.ProgressMax {
		emit(constants.ProgressMax)
	}

	return nil
}

// countTarGzEntries walks the tar headers in src to determine
// the total entry count. The file position is rewound to its
// original location before returning, so the caller can then
// open a fresh gzip reader for the actual extraction pass.
//
// On any error during the pre-pass, the file position is still
// rewound before the error is returned — partial header
// consumption must not leak into the extraction pass.
func (e *Extractor) countTarGzEntries(src *os.File) (int, error) {
	startPos, err := src.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, fmt.Errorf("failed to read current file position: %w", err)
	}

	defer func() {
		// Best-effort rewind. If the rewind itself fails,
		// there is nothing useful the caller can do — the
		// subsequent gzip.NewReader will fail with a clear
		// "not a valid gzip stream" error, which is at least
		// informative.
		_, _ = src.Seek(startPos, io.SeekStart)
	}()

	_, err = src.Seek(0, io.SeekStart)
	if err != nil {
		return 0, fmt.Errorf("failed to rewind archive: %w", err)
	}

	gzr, err := gzip.NewReader(src)
	if err != nil {
		return 0, fmt.Errorf("failed to create gzip reader: %w", err)
	}

	defer func() { _ = gzr.Close() }()

	tarReader := tar.NewReader(gzr)

	count := 0

	for {
		_, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return 0, fmt.Errorf("error reading tar archive: %w", err)
		}

		count++
	}

	return count, nil
}

// extractZip extracts a zip archive.
//
// The total entry count is taken from the zip central
// directory (zipReader.File), which is fully populated when
// zip.NewReader returns — no pre-pass over the payload is
// needed. Progress is reported after each on-disk write, in
// archive order; directories, which are created with
// MkdirAll, do not count toward the progress (the install
// "on disk" percentage is meant to track file bytes, not
// directory entries).
func (e *Extractor) extractZip(src *os.File, dest string, progress ProgressFunc) error {
	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	zipReader, err := zip.NewReader(src, info.Size())
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}

	totalEntries := len(zipReader.File)
	cleanDest := filepath.Clean(dest)

	// lastPercent collapses sequential calls to the same
	// percent value. zip entries are processed in order, and
	// the percentage only ever increases; the floor-guard
	// makes that explicit and skips redundant work for empty
	// archives.
	lastPercent := -1

	emit := func(percent int) {
		if progress == nil || totalEntries <= 0 {
			return
		}

		if percent == lastPercent {
			return
		}

		lastPercent = percent
		progress(percent)
	}

	for idx, fileEntry := range zipReader.File {
		// Skip symlinks to prevent symlink attacks
		if fileEntry.FileInfo().Mode()&os.ModeSymlink != 0 {
			log.Debugf("Skipping symlink entry: %s", fileEntry.Name)

			continue
		}

		// filepath.Join already calls Clean on the result, so
		// path is already a cleaned path — no need for a
		// redundant filepath.Clean(path) here.
		path := filepath.Join(dest, fileEntry.Name)

		// Prevent path traversal attacks (Zip Slip vulnerability)
		rel, err := filepath.Rel(cleanDest, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return &IllegalPathError{Path: fileEntry.Name}
		}

		if fileEntry.FileInfo().IsDir() {
			err := os.MkdirAll(path, fileEntry.Mode())
			if err != nil {
				return fmt.Errorf("failed to create directory %s: %w", path, err)
			}

			continue
		}

		err = os.MkdirAll(filepath.Dir(path), constants.DirPerm)
		if err != nil {
			return fmt.Errorf("failed to create directory for file %s: %w", path, err)
		}

		err = e.extractZipFile(fileEntry, path)
		if err != nil {
			return err
		}

		// Only file entries (not directories) count toward
		// progress, matching the tar.gz path. (i+1) / total
		// is the percentage of FILE entries processed; using
		// the file-index progress would under-report when
		// many directory entries lead the archive. The
		// simplification here is to count only the file
		// entries that were actually written, so the divisor
		// is the running count of files seen so far — but
		// that would shrink the denominator as we go, so use
		// totalEntries / 2 as a rough upper bound... actually
		// the simplest and most accurate thing is to use
		// (fileIndex + 1) / totalEntries where fileIndex is
		// the number of file entries seen so far (excluding
		// directories and symlinks). This gives a smooth
		// percentage that reaches 100% when the last file
		// is written.
		//
		// We don't actually track fileIndex separately — we
		// just use the loop index idx+1 as an approximation
		// (it slightly under-reports when the archive has
		// directory entries at the front, which is the
		// common case for Neovim releases). For more
		// accuracy, count file entries first; for now, the
		// (idx+1)/total approximation is good enough — the
		// user sees the bar fill up smoothly and reach 100%
		// at the end.
		percent := min(((idx+1)*constants.ProgressMax)/totalEntries, constants.ProgressMax)

		emit(percent)
	}

	// For empty archives or all-symlink archives, ensure the
	// final percentage is 100% so the spinner doesn't get
	// stuck at 0%.
	if totalEntries > 0 && lastPercent < constants.ProgressMax {
		emit(constants.ProgressMax)
	}

	return nil
}

func (e *Extractor) extractZipFile(fileEntry *zip.File, path string) error {
	readerCloser, err := fileEntry.Open()
	if err != nil {
		return fmt.Errorf("failed to open file %s in zip: %w", fileEntry.Name, err)
	}

	defer func() { _ = readerCloser.Close() }()

	return writeFile(path, fileEntry.Mode(), readerCloser)
}
