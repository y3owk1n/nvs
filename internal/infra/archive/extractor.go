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
	"github.com/sirupsen/logrus"
)

const (
	bufSize   = 262
	dirPerm   = 0o755
	zipFormat = "zip"
)

// Extractor handles archive extraction operations.
type Extractor struct{}

// New creates a new Extractor instance.
func New() *Extractor {
	return &Extractor{}
}

// Extract extracts an archive file to the destination directory.
func (e *Extractor) Extract(src *os.File, dest string) error {
	logrus.Debugf("Starting extraction to: %s", dest)
	// Detect archive format
	_, err := src.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	format, err := detectFormat(src)
	if err != nil {
		return fmt.Errorf("archive detection failed: %w", err)
	}

	logrus.Debugf("Detected archive format: %s", format)

	// Extract based on format
	switch format {
	case "tar.gz":
		return e.extractTarGz(src, dest)
	case zipFormat:
		return e.extractZip(src, dest)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
}

// detectFormat detects the archive format from file header.
func detectFormat(file *os.File) (string, error) {
	buf := make([]byte, bufSize)

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
	case zipFormat:
		return "zip", nil
	case "gz":
		return "tar.gz", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedFormat, kind.Extension)
	}
}

// writeFile writes data from reader to a file at target path with given mode.
func writeFile(target string, mode os.FileMode, reader io.Reader) error {
	file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", target, err)
	}

	defer func() { _ = file.Close() }()

	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("failed to copy file content to %s: %w", target, err)
	}

	return nil
}

// extractTarGz extracts a tar.gz archive.
func (e *Extractor) extractTarGz(src *os.File, dest string) error {
	gzr, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}

	defer func() { _ = gzr.Close() }()

	tarReader := tar.NewReader(gzr)

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return fmt.Errorf("error reading tar archive: %w", err)
		}

		target := filepath.Join(dest, header.Name)

		// Prevent path traversal attacks (Zip Slip vulnerability)
		cleanDest := filepath.Clean(dest)

		cleanTarget := filepath.Clean(target)

		rel, err := filepath.Rel(cleanDest, cleanTarget)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return &IllegalPathError{Path: header.Name}
		}

		switch header.Typeflag {
		case tar.TypeDir:
			err := os.MkdirAll(target, dirPerm)
			if err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}

		case tar.TypeReg:
			err = os.MkdirAll(filepath.Dir(target), dirPerm)
			if err != nil {
				return fmt.Errorf("failed to create directory for file %s: %w", target, err)
			}

			err := writeFile(target, os.FileMode(header.Mode), tarReader)
			if err != nil {
				return err
			}

		case tar.TypeSymlink, tar.TypeLink:
			// Reject symlinks and hard links to prevent symlink attacks
			logrus.Debugf("Skipping unsupported entry type %d: %s", header.Typeflag, header.Name)
		}
	}

	return nil
}

// extractZip extracts a zip archive.
func (e *Extractor) extractZip(src *os.File, dest string) error {
	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	r, err := zip.NewReader(src, info.Size())
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}

	for _, fileEntry := range r.File {
		path := filepath.Join(dest, fileEntry.Name)

		// Prevent path traversal attacks (Zip Slip vulnerability)
		cleanDest := filepath.Clean(dest)

		cleanPath := filepath.Clean(path)

		rel, err := filepath.Rel(cleanDest, cleanPath)
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

		err = os.MkdirAll(filepath.Dir(path), dirPerm)
		if err != nil {
			return fmt.Errorf("failed to create directory for file %s: %w", path, err)
		}

		err = e.extractZipFile(fileEntry, path)
		if err != nil {
			return err
		}
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
