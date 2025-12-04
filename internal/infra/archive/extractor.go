// Package archive provides archive extraction functionality.
// Refactored from pkg/archive with consistent error handling.
package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/h2non/filetype"
	"github.com/sirupsen/logrus"
)

const (
	bufSize = 262
	dirPerm = 0o755
)

// Extractor handles archive extraction operations.
type Extractor struct{}

// New creates a new Extractor instance.
func New() *Extractor {
	return &Extractor{}
}

// Extract extracts an archive file to the destination directory.
func (e *Extractor) Extract(src *os.File, dest string) error {
	// Detect archive format
	if _, err := src.Seek(0, io.SeekStart); err != nil {
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
	case "zip":
		return e.extractZip(src, dest)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
}

// detectFormat detects the archive format from file header.
func detectFormat(file *os.File) (string, error) {
	buf := make([]byte, bufSize)

	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	if n == 0 {
		return "", ErrEmptyFile
	}

	// Reset file pointer
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to seek file: %w", err)
	}

	// Detect file type
	kind, err := filetype.Match(buf[:n])
	if err != nil {
		return "", fmt.Errorf("file type matching error: %w", err)
	}

	if kind == filetype.Unknown {
		return "", ErrUnknownFileType
	}

	// Map to supported formats
	switch kind.Extension {
	case "zip":
		return "zip", nil
	case "gz":
		return "tar.gz", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedFormat, kind.Extension)
	}
}

// extractTarGz extracts a tar.gz archive.
func (e *Extractor) extractTarGz(src *os.File, dest string) error {
	gzr, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tarReader := tar.NewReader(gzr)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar archive: %w", err)
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, dirPerm); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), dirPerm); err != nil {
				return fmt.Errorf("failed to create directory for file %s: %w", target, err)
			}

			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", target, err)
			}

			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return fmt.Errorf("failed to copy file content to %s: %w", target, err)
			}

			file.Close()
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

		if fileEntry.FileInfo().IsDir() {
			if err := os.MkdirAll(path, fileEntry.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", path, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
			return fmt.Errorf("failed to create directory for file %s: %w", path, err)
		}

		rc, err := fileEntry.Open()
		if err != nil {
			return fmt.Errorf("failed to open file %s in zip: %w", fileEntry.Name, err)
		}

		out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileEntry.Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create output file %s: %w", path, err)
		}

		if _, err := io.Copy(out, rc); err != nil {
			rc.Close()
			out.Close()
			return fmt.Errorf("failed to copy file %s: %w", path, err)
		}

		rc.Close()
		out.Close()
	}

	return nil
}
